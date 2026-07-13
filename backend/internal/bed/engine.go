package bed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	stdlog "log"
	"strings"
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

func (s EngineState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StateSuccess:
		return "success"
	case StateStopped:
		return "stopped"
	case StateExhausted:
		return "exhausted"
	default:
		return "unknown"
	}
}

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
	client  *resty.Client
	state   EngineState
	mu      sync.Mutex
	cancel  context.CancelFunc
	status  GrabStatus
	logs    []string
	logsMu  sync.Mutex

	RetryInterval time.Duration
	MaxRetries    int
	personsn      string
	divideId      string
}

func NewEngine() *Engine {
	return &Engine{
		state:         StateIdle,
		RetryInterval: 500 * time.Millisecond,
		MaxRetries:    3,
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
	e.status = GrabStatus{
		Running:  true,
		Progress: make(map[string]BedProgress),
		Log:      []string{},
	}
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
	if e.state != StateRunning {
		return
	}
	e.state = StateStopped
	if e.cancel != nil {
		e.cancel()
	}
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
	for _, b := range col.Beds {
		weightSum += 6 - b.Priority
	}

	type grabTask struct {
		bed         CollectedBed
		concurrency int
	}
	var tasks []grabTask
	for _, b := range col.Beds {
		weight := 6 - b.Priority
		n := int(math.Max(1, math.Floor(float64(totalConcurrency)*float64(weight)/float64(weightSum))))
		tasks = append(tasks, grabTask{bed: b, concurrency: n})
	}

	e.log(fmt.Sprintf("开始抢床，总并发=%d，%d个床位", totalConcurrency, len(tasks)))

	var successCount int32
	var wg sync.WaitGroup

	for _, t := range tasks {
		e.initProgress(t.bed.BedCode, t.concurrency)
		e.log(fmt.Sprintf("%s: %d路并发 (优先级%d)", t.bed.BedName, t.concurrency, t.bed.Priority))

		for i := 0; i < t.concurrency; i++ {
			wg.Add(1)
			go func(bed CollectedBed) {
				defer wg.Done()
				for round := 0; round < e.MaxRetries; round++ {
					select {
					case <-ctx.Done():
						return
					default:
					}

					if !auth.IsSessionAlive(e.client) {
						e.log("session 过期，重新登录...")
						if err := auth.ReloginIfNeeded(e.client); err != nil {
							e.log(fmt.Sprintf("relogin 失败: %v", err))
							continue
						}
						e.client.SetHeader("Token", session.Get().Token)
					}

					body := BuildDistributeBedBody(e.personsn, bed.BedCode, e.divideId, "")
					e.log(fmt.Sprintf("%s 第%d轮: 发送请求", bed.BedName, round+1))

					resp, err := e.client.R().
						SetHeader("Content-Type", "application/x-www-form-urlencoded").
						SetFormData(body).
						Post(housingAPI + "/appdm/freshman/bunk/distributeBed")

					if err != nil {
						e.recordFail(bed.BedCode)
						e.log(fmt.Sprintf("%s: 网络错误 %v", bed.BedName, err))
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
					e.log(fmt.Sprintf("[DEBUG] %s raw=%s", bed.BedName, string(resp.Body())[:min(200, len(resp.Body()))]))

					// code != 0 → 服务端错误
					if j.Code != 0 {
						e.recordFail(bed.BedCode)
						e.log(fmt.Sprintf("❌ %s: 服务端错误 code=%d msg=%s", bed.BedName, j.Code, j.Msg))
						continue
					}

					switch j.Status {
					case 0:
						// status=0: 需要看 promptMsg 判断
						msg := j.PromptMsg
						if strings.Contains(msg, "成功") || strings.Contains(msg, "选床") && !strings.Contains(msg, "还未") && !strings.Contains(msg, "未开始") && !strings.Contains(msg, "结束") {
							e.recordOK(bed.BedCode)
							atomic.AddInt32(&successCount, 1)
							e.log(fmt.Sprintf("✅ %s: 抢到! %s", bed.BedName, msg))
							e.mu.Lock()
							e.status.Success = true
							e.status.SuccessBed = bed.BedCode
							e.mu.Unlock()
							e.cancel()
							return
						}
						e.recordFail(bed.BedCode)
						e.log(fmt.Sprintf("⚠️ %s: status=0 但未成功: %s", bed.BedName, msg))

					case 1:
						e.log(fmt.Sprintf("🔄 %s: session过期, relogin...", bed.BedName))
						auth.ReloginIfNeeded(e.client)
						e.client.SetHeader("Token", session.Get().Token)

					case 5:
						e.log(fmt.Sprintf("⏰ %s: 选床还未开始: %s", bed.BedName, j.PromptMsg))
						e.recordFail(bed.BedCode)

					default:
						e.recordFail(bed.BedCode)
						e.log(fmt.Sprintf("❓ %s: 未知status=%d msg=%s prompt=%s", bed.BedName, j.Status, j.Msg, j.PromptMsg))
					}

					if round < e.MaxRetries-1 {
						time.Sleep(e.RetryInterval)
					}
				}
			}(t.bed)
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
	e.mu.Lock()
	e.status.Progress[bedCode] = BedProgress{Total: total}
	e.mu.Unlock()
}

func (e *Engine) recordOK(bedCode string) {
	e.mu.Lock()
	p := e.status.Progress[bedCode]
	p.Done++
	p.OK++
	e.status.Progress[bedCode] = p
	e.mu.Unlock()
}

func (e *Engine) recordFail(bedCode string) {
	e.mu.Lock()
	p := e.status.Progress[bedCode]
	p.Done++
	p.Fail++
	e.status.Progress[bedCode] = p
	e.mu.Unlock()
}

func (e *Engine) log(msg string) {
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)
	stdlog.Println(line)
	e.logsMu.Lock()
	e.logs = append(e.logs, line)
	if len(e.logs) > 200 {
		e.logs = e.logs[len(e.logs)-200:]
	}
	e.mu.Lock()
	e.status.Log = e.logs
	e.mu.Unlock()
}

func (e *Engine) done(state EngineState) {
	e.mu.Lock()
	e.state = state
	e.status.Running = false
	e.mu.Unlock()
}
