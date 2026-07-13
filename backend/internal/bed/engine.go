package bed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	stdlog "log"
	"sync"
	"sync/atomic"
	"time"

	"xjtu-housing-genius/internal/auth"
	"xjtu-housing-genius/internal/session"

	"github.com/go-resty/resty/v2"
)

type EngineState int

const (
	StateIdle EngineState = iota
	StateRunning
	StateSuccess
	StateStopped
	StateExhausted
)

type BedProgress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
	OK    int `json:"ok"`
	Fail  int `json:"fail"`
}

type GrabStatus struct {
	Running    bool                   `json:"running"`
	Success    bool                   `json:"success"`
	SuccessBed string                 `json:"successBed"`
	Progress   map[string]BedProgress `json:"progress"`
	Log        []string               `json:"log"`
}

type Engine struct {
	client        *resty.Client
	state         EngineState
	mu            sync.Mutex
	cancel        context.CancelFunc
	status        GrabStatus
	logs          []string
	logsMu        sync.Mutex
	RetryInterval time.Duration
	personsn      string
	divideId      string
}

func NewEngine() *Engine {
	return &Engine{
		state:         StateIdle,
		RetryInterval: 50 * time.Millisecond,
	}
}

func (e *Engine) SetClient(client *resty.Client, personsn, divideId string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.client = client
	e.personsn = personsn
	e.divideId = divideId
}

func (e *Engine) Start(totalConcurrency int) error {
	e.mu.Lock()
	if e.state == StateRunning {
		e.mu.Unlock()
		return fmt.Errorf("抢床引擎已在运行")
	}
	if e.client == nil {
		e.mu.Unlock()
		return fmt.Errorf("未登录，请先登录")
	}
	e.state = StateRunning
	e.status = GrabStatus{Running: true, Progress: make(map[string]BedProgress), Log: []string{}}
	e.logs = []string{}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.mu.Unlock()
	go e.run(ctx, totalConcurrency)
	return nil
}

func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != StateRunning { return }
	e.state = StateStopped
	if e.cancel != nil { e.cancel() }
}

func (e *Engine) Status() GrabStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.status
	s.Running = e.state == StateRunning
	return s
}

func (e *Engine) run(ctx context.Context, totalConcurrency int) {
	col := GetCollection()
	if len(col.Beds) == 0 {
		e.log("没有收藏的床位")
		e.done(StateExhausted)
		return
	}
	if totalConcurrency < len(col.Beds) {
		totalConcurrency = len(col.Beds)
	}
	weightSum := 0
	for _, b := range col.Beds { weightSum += 6 - b.Priority }
	type grabTask struct { bed CollectedBed; concurrency int }
	var tasks []grabTask
	for _, b := range col.Beds {
		weight := 6 - b.Priority
		n := int(math.Max(1, math.Floor(float64(totalConcurrency)*float64(weight)/float64(weightSum))))
		tasks = append(tasks, grabTask{bed: b, concurrency: n})
	}
	e.log(fmt.Sprintf("开始抢床 总并发=%d %d个床位", totalConcurrency, len(tasks)))
	var successCount int32
	var wg sync.WaitGroup
	for _, tk := range tasks {
		e.initProgress(tk.bed.BedCode, tk.concurrency)
		e.log(fmt.Sprintf("%s: %d路(优先级%d)", tk.bed.BedName, tk.concurrency, tk.bed.Priority))
		for i := 0; i < tk.concurrency; i++ {
			wg.Add(1)
			go func(bed CollectedBed) {
				fmt.Printf("GOROUTINE-START %s\n", bed.BedName)
				stdlog.Printf("GOROUTINE-START %s", bed.BedName)
				defer wg.Done()
				tok := session.Get().Token
				stdlog.Printf("[grab] token=%s personsn=%s divideId=%s", tok[:min(8, len(tok))], e.personsn, e.divideId)
				for round := 0; ; round++ {
					select {
					case <-ctx.Done(): return
					default:
					}
					body := BuildDistributeBedBody(e.personsn, bed.BedCode, e.divideId, "")
					stdlog.Printf("[grab] %s round=%d posting...", bed.BedName, round)
					resp, err := e.client.R().
						SetHeader("Content-Type", "application/json").
						SetHeader("Token", tok).
						SetCookie(&http.Cookie{Name: "token", Value: tok}).
						SetBody(body).
						Post(housingAPI + "/appdm/freshman/bunk/distributeBed")
					if err != nil {
						e.recordFail(bed.BedCode)
						e.log(fmt.Sprintf("%s: err=%v", bed.BedName, err))
						time.Sleep(e.RetryInterval)
						continue
					}
					var j struct {
						Code      int    `json:"code"`
						Status    int    `json:"status"`
						Msg       string `json:"msg"`
						PromptMsg string `json:"promptMsg"`
					}
					json.Unmarshal(resp.Body(), &j)
					e.log(fmt.Sprintf("%s round=%d resp: code=%d status=%d prompt=%s", bed.BedName, round, j.Code, j.Status, j.PromptMsg))
					if j.Code != 0 {
						e.recordFail(bed.BedCode)
						time.Sleep(e.RetryInterval)
						continue
					}
					if j.Status == 0 {
						e.recordOK(bed.BedCode)
						atomic.AddInt32(&successCount, 1)
						e.log(fmt.Sprintf("✅ %s: 抢到! %s", bed.BedName, j.PromptMsg))
						e.mu.Lock()
						e.status.Success = true
						e.status.SuccessBed = bed.BedName
						e.mu.Unlock()
						e.cancel()
						return
					}
					e.recordFail(bed.BedCode)
					if j.Status == 1 {
						e.log(fmt.Sprintf("🔄 %s: session过期", bed.BedName))
						auth.ReloginIfNeeded(e.client)
						tok = session.Get().Token
						e.client.SetHeader("Token", tok)
					}
					time.Sleep(e.RetryInterval)
				}
			}(tk.bed)
		}
	}
	wg.Wait()
	if atomic.LoadInt32(&successCount) > 0 {
		e.done(StateSuccess)
	} else {
		e.log("所有床位均失败")
		e.done(StateExhausted)
	}
}

func (e *Engine) initProgress(bedCode string, total int) {
	e.mu.Lock(); e.status.Progress[bedCode] = BedProgress{Total: total}; e.mu.Unlock()
}
func (e *Engine) recordOK(bedCode string) {
	e.mu.Lock(); p := e.status.Progress[bedCode]; p.Done++; p.OK++; e.status.Progress[bedCode] = p; e.mu.Unlock()
}
func (e *Engine) recordFail(bedCode string) {
	e.mu.Lock(); p := e.status.Progress[bedCode]; p.Done++; p.Fail++; e.status.Progress[bedCode] = p; e.mu.Unlock()
}
func (e *Engine) log(msg string) {
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)
	stdlog.Println(line)
	e.logsMu.Lock()
	e.logs = append(e.logs, line)
	if len(e.logs) > 200 { e.logs = e.logs[len(e.logs)-200:] }
	e.mu.Lock(); e.status.Log = e.logs; e.mu.Unlock()
}
func (e *Engine) done(state EngineState) {
	e.mu.Lock(); e.state = state; e.status.Running = false; e.mu.Unlock()
}
