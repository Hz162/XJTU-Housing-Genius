# Housing-Genius 抢床功能 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现床位浏览、收藏管理、多优先级并发抢床的完整前后端功能。

**Architecture:** Go 后端新增 `internal/bed/` 模块（proxy/collection/engine），Flutter 前端新增 `bed_page` + 5 个 widget 组件。Engine 通过共享 `*resty.Client` 指针与 Server 共用认证 session。

**Tech Stack:** Go (chi, resty), Flutter (Material 3, http), AES-ECB-PKCS7

---

## File Map

| 操作 | 文件 | 职责 |
|------|------|------|
| Create | `backend/internal/bed/proxy.go` | 代理 housing bed API，AES 加密 distributeBed 请求 |
| Create | `backend/internal/bed/collection.go` | 收藏 JSON 读写，优先级/并发校验 |
| Create | `backend/internal/bed/engine.go` | 抢床引擎：并发池、优先级调度、状态机 |
| Modify | `backend/internal/api/router.go` | 新增 `/api/bed/*` 路由 |
| Modify | `backend/internal/api/handlers.go` | 新增 bed handler 方法 + 注入 engine |
| Create | `frontend/lib/pages/bed_page.dart` | 主页面（sidebar + 内容 + 收藏 + 抢床） |
| Create | `frontend/lib/widgets/bed_sidebar.dart` | 楼栋→楼层→房间树形导航 |
| Create | `frontend/lib/widgets/bed_content.dart` | 房间床位表格 + 收藏按钮 |
| Create | `frontend/lib/widgets/collection_panel.dart` | 收藏列表 + 优先级下拉 + 拖拽排序 |
| Create | `frontend/lib/widgets/grab_panel.dart` | 抢床控制 + 进度条 + 日志 |
| Modify | `frontend/lib/services/api_service.dart` | 新增 bed API 方法 |
| Modify | `frontend/lib/pages/login_page.dart` | 登录成功跳转 bed_page |

**Stage 1: 后端 bed 模块**
- Task 1: `bed/proxy.go` — 代理 + AES
- Task 2: `bed/collection.go` — 收藏文件
- Task 3: `bed/engine.go` — 抢床引擎
- Task 4: `api/router.go` + `api/handlers.go` — 路由 + handler

**Stage 2: 前端页面**
- Task 5: `api_service.dart` — bed API 方法
- Task 6: `bed_page.dart` + `bed_sidebar.dart` — 页面框架 + 侧边栏
- Task 7: `bed_content.dart` — 床位表格
- Task 8: `collection_panel.dart` — 收藏面板
- Task 9: `grab_panel.dart` — 抢床面板
- Task 10: `login_page.dart` — 登录后跳转

---

### Task 1: `bed/proxy.go` — 代理 + AES 加密

**Files:**
- Create: `backend/internal/bed/proxy.go`

- [ ] **Step 1: 写 proxy.go**

```go
package bed

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

const housingAPI = "http://housing2021.xjtu.edu.cn"

// ProxyGet 代理 GET 请求到 housing API（带认证 client）
func ProxyGet(client *resty.Client, path string, params map[string]string, token string) ([]byte, error) {
	req := client.R()
	if token != "" {
		req.SetHeader("Token", token)
	}
	if params != nil {
		req.SetQueryParams(params)
	}
	resp, err := req.Get(housingAPI + path)
	if err != nil {
		return nil, fmt.Errorf("proxy GET %s: %w", path, err)
	}
	return resp.Body(), nil
}

// ProxyPost 代理 POST 请求到 housing API
func ProxyPost(client *resty.Client, path string, params map[string]string, bodyType string, token string) ([]byte, error) {
	req := client.R()
	if token != "" {
		req.SetHeader("Token", token)
	}
	if params != nil {
		if bodyType == "query" {
			req.SetQueryParams(params)
		} else {
			req.SetFormData(params)
		}
	}
	resp, err := req.Post(housingAPI + path)
	if err != nil {
		return nil, fmt.Errorf("proxy POST %s: %w", path, err)
	}
	return resp.Body(), nil
}

// EncryptBedCode AES-ECB-PKCS7 加密床位编码
// key = "shu" + timestamp，与 web 端 h.a(bedCode, "shu"+timestamp) 一致
func EncryptBedCode(bedCode string, timestamp int64) string {
	key := []byte(fmt.Sprintf("shu%d", timestamp))
	padded := make([]byte, 16)
	copy(padded, key)
	key = padded

	block, err := aes.NewCipher(key)
	if err != nil {
		return bedCode
	}

	plaintext := []byte(bedCode)
	padLen := aes.BlockSize - len(plaintext)%aes.BlockSize
	buf := make([]byte, len(plaintext)+padLen)
	copy(buf, plaintext)
	for i := len(plaintext); i < len(buf); i++ {
		buf[i] = byte(padLen)
	}

	encrypted := make([]byte, len(buf))
	for i := 0; i < len(buf); i += aes.BlockSize {
		block.Encrypt(encrypted[i:i+aes.BlockSize], buf[i:i+aes.BlockSize])
	}

	return base64.StdEncoding.EncodeToString(encrypted)
}

// BuildDistributeBedBody 构造 distributeBed 请求体
func BuildDistributeBedBody(personsn, bedCode, divideId string) map[string]string {
	ts := time.Now().UnixMilli()
	return map[string]string{
		"personsn":     personsn,
		"bedPlaceCode": EncryptBedCode(bedCode, ts),
		"divideId":     divideId,
		"aircondition": "0",
		"beddingInfo":  "",
		"chooseWay":    "2",
		"t":            fmt.Sprintf("%d", ts),
	}
}
```

- [ ] **Step 2: 编译验证**

```bash
cd D:\XJTU-Housing-Genius\backend && go build ./...
```

Expected: 编译通过。

---

### Task 2: `bed/collection.go` — 收藏文件读写

**Files:**
- Create: `backend/internal/bed/collection.go`

- [ ] **Step 1: 写 collection.go**

```go
package bed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// CollectedBed 收藏的床位
type CollectedBed struct {
	BedCode      string `json:"bedCode"`
	BedName      string `json:"bedName"`
	RoomCode     string `json:"roomCode"`
	BuildingCode string `json:"buildingCode"`
	Priority     int    `json:"priority"` // 1-5, 1=最高
}

// Collection 收藏配置
type Collection struct {
	Beds             []CollectedBed `json:"beds"`
	TotalConcurrency int            `json:"totalConcurrency"`
}

var (
	collection Collection
	colMu      sync.RWMutex
)

// configDir 返回配置文件目录
func configDir() string {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "xjtu-housing-genius")
}

// collectionPath 按学号返回收藏文件路径
func collectionPath(studentCode string) string {
	return filepath.Join(configDir(), fmt.Sprintf("housing-config-%s.json", studentCode))
}

// LoadCollection 加载收藏文件
func LoadCollection(studentCode string) error {
	colMu.Lock()
	defer colMu.Unlock()

	collection = Collection{Beds: []CollectedBed{}, TotalConcurrency: 10}
	data, err := os.ReadFile(collectionPath(studentCode))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &collection)
}

// GetCollection 获取当前收藏（线程安全）
func GetCollection() Collection {
	colMu.RLock()
	defer colMu.RUnlock()
	return collection
}

// SaveCollection 保存收藏文件
func SaveCollection(c Collection, studentCode string) error {
	colMu.Lock()
	collection = c
	colMu.Unlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(collectionPath(studentCode), data, 0644)
}

// ValidatePriority 校验优先级
func ValidatePriority(p int) error {
	if p < 1 || p > 5 {
		return fmt.Errorf("优先级必须在1-5之间")
	}
	return nil
}

// MinConcurrency 计算最小总并发数
func MinConcurrency() int {
	c := GetCollection()
	return len(c.Beds) // 至少每个床位1路
}
```

- [ ] **Step 2: 编译验证**

```bash
cd D:\XJTU-Housing-Genius\backend && go build ./...
```

Expected: 编译通过。

---

### Task 3: `bed/engine.go` — 抢床引擎

**Files:**
- Create: `backend/internal/bed/engine.go`

- [ ] **Step 1: 写 engine.go**

```go
package bed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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

func (s EngineState) String() string {
	switch s {
	case StateIdle: return "idle"
	case StateRunning: return "running"
	case StateSuccess: return "success"
	case StateStopped: return "stopped"
	case StateExhausted: return "exhausted"
	default: return "unknown"
	}
}

// BedProgress 单个床位的抢床进度
type BedProgress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
	OK    int `json:"ok"`
	Fail  int `json:"fail"`
}

// GrabStatus 抢床状态（通过 /api/bed/grab/status 返回）
type GrabStatus struct {
	Running    bool                  `json:"running"`
	Success    bool                  `json:"success"`
	SuccessBed string                `json:"successBed"`
	Progress   map[string]BedProgress `json:"progress"`
	Log        []string              `json:"log"`
}

// Engine 抢床引擎
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

// NewEngine 创建引擎
func NewEngine() *Engine {
	return &Engine{
		state:         StateIdle,
		RetryInterval: 500 * time.Millisecond,
		MaxRetries:    3,
	}
}

// SetClient 注入认证 client（登录后调用）
func (e *Engine) SetClient(client *resty.Client, personsn, divideId string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.client = client
	e.personsn = personsn
	e.divideId = divideId
}

// Start 开始抢床
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

// Stop 停止抢床
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

// Status 获取当前状态
func (e *Engine) Status() GrabStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.status
	s.Running = e.state == StateRunning
	return s
}

// run 主循环
func (e *Engine) run(ctx context.Context, totalConcurrency int) {
	col := GetCollection()
	if len(col.Beds) == 0 {
		e.log("没有收藏的床位")
		e.done(StateExhausted)
		return
	}

	// 确保每个床位至少1路并发
	if totalConcurrency < len(col.Beds) {
		totalConcurrency = len(col.Beds)
	}

	// 计算权重和
	weightSum := 0
	for _, b := range col.Beds {
		weightSum += 6 - b.Priority
	}

	// 分配并发数
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

					// 检查 session
					if !auth.IsSessionAlive(e.client) {
						e.log("session 过期，重新登录...")
						if err := auth.ReloginIfNeeded(e.client); err != nil {
							e.log(fmt.Sprintf("relogin 失败: %v", err))
							continue
						}
					}

					body := BuildDistributeBedBody(e.personsn, bed.BedCode, e.divideId)
					e.log(fmt.Sprintf("%s 第%d轮: 发送请求", bed.BedName, round+1))

					resp, err := e.client.R().
						SetHeader("Content-Type", "application/x-www-form-urlencoded").
						SetFormData(body).
						Post(housingAPI + "/appdm/freshman/bunk/distributeBed")

					if err != nil {
						e.recordFail(bed.BedCode)
						e.log(fmt.Sprintf("%s: 网络错误 %v", bed.BedName, err))
						continue
					}

					var j struct {
						Code   int    `json:"code"`
						Status int    `json:"status"`
						Msg    string `json:"msg"`
						PromptMsg string `json:"promptMsg"`
					}
					json.Unmarshal(resp.Body(), &j)

					if j.Code == 0 && j.Status == 0 {
						// 检查是否真的抢到了
						if resp.StatusCode() == 200 {
							e.recordOK(bed.BedCode)
							atomic.AddInt32(&successCount, 1)
							e.log(fmt.Sprintf("✅ %s: 成功! %s", bed.BedName, j.PromptMsg))
							e.mu.Lock()
							e.status.Success = true
							e.status.SuccessBed = bed.BedCode
							e.mu.Unlock()
							e.cancel() // 取消其他 goroutine
							return
						}
					} else if j.Status == 1 {
						e.log(fmt.Sprintf("%s: session过期, 重试", bed.BedName))
						auth.ReloginIfNeeded(e.client)
					} else {
						e.recordFail(bed.BedCode)
						e.log(fmt.Sprintf("%s: 失败 code=%d msg=%s", bed.BedName, j.Code, j.Msg))
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
```

- [ ] **Step 2: 编译验证**

```bash
cd D:\XJTU-Housing-Genius\backend && go build ./...
```

Expected: 编译通过。

---

### Task 4: 路由 + Handler

**Files:**
- Modify: `backend/internal/api/router.go`
- Modify: `backend/internal/api/handlers.go`

- [ ] **Step 1: 更新 router.go — 添加 bed 路由**

在 `NewRouter()` 的 `r.Route("/api", ...)` 块内，MFA 路由后面添加：

```go
r.Route("/bed", func(r chi.Router) {
    r.Get("/divideId", s.HandleBedDivideId)
    r.Get("/tree", s.HandleBedTree)
    r.Get("/room-beds", s.HandleBedRoomBeds)
    r.Get("/check", s.HandleBedCheck)
    r.Get("/collection", s.HandleBedCollectionGet)
    r.Post("/collection", s.HandleBedCollectionSave)
    r.Get("/grab/status", s.HandleBedGrabStatus)
    r.Post("/grab/start", s.HandleBedGrabStart)
    r.Post("/grab/stop", s.HandleBedGrabStop)
})
```

- [ ] **Step 2: 更新 handlers.go — 添加 Server 字段和 bed handler**

在 `type Server struct` 中新增 engine 字段：

```go
type Server struct {
    client *resty.Client
    engine *bed.Engine
}
```

在 `NewServer()` 中初始化 engine：

```go
func NewServer() *Server {
    client := session.NewClient()
    return &Server{
        client: client,
        engine: bed.NewEngine(),
    }
}
```

在文件末尾（`HandleProxyAppsys` 后面）添加所有 bed handler。add `"xjtu-housing-genius/internal/bed"` import，然后添加：

```go
func (s *Server) HandleBedDivideId(w http.ResponseWriter, r *http.Request) {
    personsn := r.URL.Query().Get("personsn")
    if personsn == "" {
        personsn = session.Get().StudentCode
    }
    body, err := bed.ProxyPost(s.client, "/appdm/freshman/resident/getDivideIdBySn",
        map[string]string{"personsn": personsn}, "query", session.Get().Token)
    if err != nil {
        writeJSON(w, 500, map[string]string{"error": err.Error()})
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(body)
}

func (s *Server) HandleBedTree(w http.ResponseWriter, r *http.Request) {
    divideId := r.URL.Query().Get("divideId")
    body, err := bed.ProxyGet(s.client, "/appdm/freshman/divide/getBunkTreeByDivideId",
        map[string]string{"modelId": "dm", "type": "ROOM", "divideId": divideId}, session.Get().Token)
    if err != nil {
        writeJSON(w, 500, map[string]string{"error": err.Error()})
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(body)
}

func (s *Server) HandleBedRoomBeds(w http.ResponseWriter, r *http.Request) {
    divideId := r.URL.Query().Get("divideId")
    roomCode := r.URL.Query().Get("roomCode")
    body, err := bed.ProxyPost(s.client, "/appdm/freshman/divide/getBedInfoByDivideId",
        map[string]string{"modelId": "dm", "roomCode": roomCode, "divideId": divideId}, "query", session.Get().Token)
    if err != nil {
        writeJSON(w, 500, map[string]string{"error": err.Error()})
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(body)
}

func (s *Server) HandleBedCheck(w http.ResponseWriter, r *http.Request) {
    personsn := r.URL.Query().Get("personsn")
    divideId := r.URL.Query().Get("divideId")
    body, err := bed.ProxyGet(s.client, "/appdm/freshman/bunk/checkMyBed",
        map[string]string{"personsn": personsn, "divideId": divideId}, session.Get().Token)
    if err != nil {
        writeJSON(w, 500, map[string]string{"error": err.Error()})
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(body)
}

func (s *Server) HandleBedCollectionGet(w http.ResponseWriter, r *http.Request) {
    studentCode := session.Get().StudentCode
    if studentCode == "" {
        studentCode = r.URL.Query().Get("studentCode")
    }
    bed.LoadCollection(studentCode)
    writeJSON(w, 200, bed.GetCollection())
}

func (s *Server) HandleBedCollectionSave(w http.ResponseWriter, r *http.Request) {
    var col bed.Collection
    if err := json.NewDecoder(r.Body).Decode(&col); err != nil {
        writeJSON(w, 400, map[string]string{"error": "参数错误"})
        return
    }
    for _, b := range col.Beds {
        if err := bed.ValidatePriority(b.Priority); err != nil {
            writeJSON(w, 400, map[string]string{"error": err.Error()})
            return
        }
    }
    studentCode := session.Get().StudentCode
    if err := bed.SaveCollection(col, studentCode); err != nil {
        writeJSON(w, 500, map[string]string{"error": err.Error()})
        return
    }
    writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) HandleBedGrabStart(w http.ResponseWriter, r *http.Request) {
    studentCode := session.Get().StudentCode
    personsn := r.URL.Query().Get("personsn")
    if personsn == "" {
        personsn = studentCode
    }
    divideId := r.URL.Query().Get("divideId")

    // 注入认证 client 到 engine
    s.engine.SetClient(s.client, personsn, divideId)

    var req struct {
        TotalConcurrency int `json:"totalConcurrency"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    if req.TotalConcurrency == 0 {
        req.TotalConcurrency = bed.GetCollection().TotalConcurrency
    }

    if err := s.engine.Start(req.TotalConcurrency); err != nil {
        writeJSON(w, 400, map[string]string{"error": err.Error()})
        return
    }
    writeJSON(w, 200, map[string]string{"status": "started"})
}

func (s *Server) HandleBedGrabStop(w http.ResponseWriter, r *http.Request) {
    s.engine.Stop()
    writeJSON(w, 200, map[string]string{"status": "stopped"})
}

func (s *Server) HandleBedGrabStatus(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, 200, s.engine.Status())
}
```

- [ ] **Step 3: 编译验证**

```bash
cd D:\XJTU-Housing-Genius\backend && go build -o xjtu-housing-genius.exe .
```

Expected: 编译通过。

---

### Task 5: `api_service.dart` — 新增 bed API 方法

**Files:**
- Modify: `frontend/lib/services/api_service.dart`

- [ ] **Step 1: 在 `api_service.dart` 末尾（`getBuildingList` 之后）添加 bed API 方法**

```dart
  // ── Bed ──

  Future<Map<String, dynamic>> getDivideId(String personsn) async {
    final data = await _request('GET', '/bed/divideId?personsn=$personsn');
    return data as Map<String, dynamic>;
  }

  Future<List<dynamic>> getBedTree(String divideId) async {
    final data = await _request('GET', '/bed/tree?divideId=$divideId');
    return (data is List) ? data : <dynamic>[];
  }

  Future<Map<String, dynamic>> getRoomBeds(String divideId, String roomCode) async {
    final data = await _request(
        'GET', '/bed/room-beds?divideId=$divideId&roomCode=$roomCode');
    return data as Map<String, dynamic>;
  }

  Future<Map<String, dynamic>> checkMyBed(String personsn, String divideId) async {
    final data = await _request(
        'GET', '/bed/check?personsn=$personsn&divideId=$divideId');
    return data as Map<String, dynamic>;
  }

  Future<Map<String, dynamic>> getCollection() async {
    final data = await _request('GET', '/bed/collection');
    return data as Map<String, dynamic>;
  }

  Future<void> saveCollection(Map<String, dynamic> collection) =>
      _request('POST', '/bed/collection', body: collection);

  Future<Map<String, dynamic>> grabStart(
      {required String personsn, required String divideId, required int totalConcurrency}) async {
    final data = await _request('POST',
        '/bed/grab/start?personsn=$personsn&divideId=$divideId',
        body: {'totalConcurrency': totalConcurrency});
    return data as Map<String, dynamic>;
  }

  Future<void> grabStop() => _request('POST', '/bed/grab/stop');

  Future<Map<String, dynamic>> grabStatus() async {
    final data = await _request('GET', '/bed/grab/status');
    return data as Map<String, dynamic>;
  }
```

- [ ] **Step 2: 验证无语法错误**

Read `api_service.dart` 确认无重复方法名。Expected: 无错误。

---

### Task 6: `bed_page.dart` + `bed_sidebar.dart` — 页面框架 + 侧边栏

**Files:**
- Create: `frontend/lib/pages/bed_page.dart`
- Create: `frontend/lib/widgets/bed_sidebar.dart`

- [ ] **Step 1: 写 bed_page.dart**

```dart
import 'dart:async';
import 'package:flutter/material.dart';
import '../main.dart';
import '../services/api_service.dart';
import '../theme/app_theme.dart';
import '../widgets/window_bar.dart';
import '../widgets/bed_sidebar.dart';
import '../widgets/bed_content.dart';
import '../widgets/collection_panel.dart';
import '../widgets/grab_panel.dart';

class BedPage extends StatefulWidget {
  final ApiService api;
  final String studentCode;
  const BedPage({super.key, required this.api, required this.studentCode});

  @override
  State<BedPage> createState() => _BedPageState();
}

class _BedPageState extends State<BedPage> {
  late ApiService api;

  bool _sidebarCollapsed = false;
  String _divideId = '';
  String _personsn = '';
  String? _selectedRoomCode;
  bool _isMyBed = false;
  bool _initialized = false;

  List<Map<String, dynamic>> _collection = [];
  int _totalConcurrency = 10;
  Map<String, dynamic>? _grabStatus;
  Timer? _grabTimer;

  @override
  void initState() {
    super.initState();
    api = widget.api;
    _initData();
    _startSessionMonitor();
  }

  @override
  void dispose() {
    _grabTimer?.cancel();
    _sessionTimer?.cancel();
    super.dispose();
  }

  Future<void> _initData() async {
    try {
      _personsn = widget.studentCode;

      final divResp = await api.getDivideId(studentCode);
      if (divResp['code'] == 0 && divResp['map'] != null) {
        _divideId = divResp['map']['divideId'] ?? '';
      }

      final checkResp = await api.checkMyBed(_personsn, _divideId);
      _isMyBed = checkResp['isMybed'] == true;

      final colResp = await api.getCollection();
      if (colResp['beds'] != null) {
        _collection = List<Map<String, dynamic>>.from(colResp['beds']);
      }
      _totalConcurrency = colResp['totalConcurrency'] ?? 10;

      setState(() => _initialized = true);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('初始化失败: $e'), backgroundColor: dangerColor),
        );
      }
    }
  }

  Timer? _sessionTimer;
  bool _sessionDead = false;

  void _startSessionMonitor() {
    _sessionTimer = Timer.periodic(const Duration(seconds: 30), (_) async {
      try {
        final alive = await api.checkSession();
        if (!alive && mounted && !_sessionDead) {
          setState(() => _sessionDead = true);
          final ok = await api.relogin().then((_) => true).catchError((_) => false);
          if (ok) {
            setState(() => _sessionDead = false);
          } else {
            if (mounted) redirectToLogin();
          }
        }
      } catch (_) {}
    });
  }

  void _onRoomSelected(String roomCode) {
    setState(() => _selectedRoomCode = roomCode);
  }

  void _onCollectionChanged(List<Map<String, dynamic>> collection, int concurrency) {
    setState(() {
      _collection = collection;
      _totalConcurrency = concurrency;
    });
  }

  void _onGrabStatus(Map<String, dynamic>? status) {
    setState(() => _grabStatus = status);
  }

  bool get _isGrabbing => _grabStatus != null && _grabStatus!['running'] == true;

  @override
  Widget build(BuildContext context) {
    if (!_initialized) {
      return Scaffold(
        backgroundColor: bgColor,
        body: const Center(child: CircularProgressIndicator()),
      );
    }

    return Scaffold(
      backgroundColor: bgColor,
      body: Column(
        children: [
          WindowBar(
            leading: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                IconButton(
                  icon: Icon(
                    _sidebarCollapsed ? Icons.menu_rounded : Icons.menu_open_rounded,
                    size: 20,
                  ),
                  onPressed: () => setState(() => _sidebarCollapsed = !_sidebarCollapsed),
                ),
                const Text('XJTU Housing Genius',
                    style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: textPrimary)),
                if (_isGrabbing)
                  Padding(
                    padding: const EdgeInsets.only(left: 8),
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        gradient: primaryGradient,
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          SizedBox(width: 10, height: 10,
                              child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white)),
                          SizedBox(width: 6),
                          Text('抢床中', style: TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.w600)),
                        ],
                      ),
                    ),
                  ),
              ],
            ),
          ),
          Expanded(
            child: Row(
              children: [
                BedSidebar(
                  collapsed: _sidebarCollapsed,
                  api: api,
                  divideId: _divideId,
                  onRoomSelected: _onRoomSelected,
                ),
                Expanded(
                  child: Column(
                    children: [
                      Expanded(
                        child: _isMyBed
                            ? Center(
                                child: Column(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    Icon(Icons.bed_rounded, size: 64, color: successColor.withAlpha(150)),
                                    const SizedBox(height: 16),
                                    const Text('您已有床位', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600, color: textPrimary)),
                                    const SizedBox(height: 8),
                                    const Text('选床系统已为您分配床位', style: TextStyle(fontSize: 14, color: textSecondary)),
                                  ],
                                ),
                              )
                            : _selectedRoomCode != null
                                ? BedContent(
                                    api: api,
                                    divideId: _divideId,
                                    roomCode: _selectedRoomCode!,
                                    personsn: _personsn,
                                    collection: _collection,
                                    onCollectionChanged: _onCollectionChanged,
                                    readOnly: _isGrabbing,
                                  )
                                : Center(
                                    child: Column(
                                      mainAxisSize: MainAxisSize.min,
                                      children: [
                                        Icon(Icons.arrow_back_rounded, size: 48, color: textMuted.withAlpha(100)),
                                        const SizedBox(height: 12),
                                        const Text('从左侧选择一个房间', style: TextStyle(fontSize: 14, color: textMuted)),
                                      ],
                                    ),
                                  ),
                      ),
                      CollectionPanel(
                        collection: _collection,
                        totalConcurrency: _totalConcurrency,
                        readOnly: _isGrabbing,
                        onChanged: (col, concurrency) {
                          _onCollectionChanged(col, concurrency);
                          api.saveCollection({
                            'beds': col,
                            'totalConcurrency': concurrency,
                          });
                        },
                      ),
                      GrabPanel(
                        api: api,
                        personsn: _personsn,
                        divideId: _divideId,
                        totalConcurrency: _totalConcurrency,
                        grabStatus: _grabStatus,
                        collectionCount: _collection.length,
                        onStatusChanged: _onGrabStatus,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
```

- [ ] **Step 2: 写 bed_sidebar.dart**

```dart
import 'package:flutter/material.dart';
import '../services/api_service.dart';
import '../theme/app_theme.dart';

class BedSidebar extends StatefulWidget {
  final bool collapsed;
  final ApiService api;
  final String divideId;
  final void Function(String roomCode) onRoomSelected;

  const BedSidebar({
    super.key,
    required this.collapsed,
    required this.api,
    required this.divideId,
    required this.onRoomSelected,
  });

  @override
  State<BedSidebar> createState() => _BedSidebarState();
}

class _BedSidebarState extends State<BedSidebar> {
  List<dynamic> _treeData = [];
  bool _loading = true;
  String? _selectedRoom;

  @override
  void didUpdateWidget(BedSidebar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.divideId != widget.divideId && widget.divideId.isNotEmpty) {
      _loadTree();
    }
  }

  @override
  void initState() {
    super.initState();
    if (widget.divideId.isNotEmpty) _loadTree();
  }

  Future<void> _loadTree() async {
    setState(() => _loading = true);
    try {
      _treeData = await widget.api.getBedTree(widget.divideId);
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    if (widget.collapsed) return const SizedBox(width: 0);

    return Container(
      width: 240,
      decoration: BoxDecoration(
        color: surfaceColor,
        border: const Border(right: BorderSide(color: borderColor)),
      ),
      child: _loading
          ? const Center(child: CircularProgressIndicator(strokeWidth: 2))
          : ListView(
              padding: const EdgeInsets.all(8),
              children: _treeData.map((building) => _buildBuilding(building)).toList(),
            ),
    );
  }

  Widget _buildBuilding(dynamic building) {
    final name = building['name'] ?? building['buildingName'] ?? '?';
    final floors = building['children'] ?? building['floors'] ?? [];
    return ExpansionTile(
      tilePadding: const EdgeInsets.symmetric(horizontal: 8),
      childrenPadding: const EdgeInsets.only(left: 16),
      leading: const Icon(Icons.apartment_rounded, size: 18, color: primaryColor),
      title: Text(name.toString(),
          style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: textPrimary)),
      children: (floors as List).map((floor) => _buildFloor(floor)).toList(),
    );
  }

  Widget _buildFloor(dynamic floor) {
    final name = floor['name'] ?? floor['floorName'] ?? '?';
    final rooms = floor['children'] ?? floor['rooms'] ?? [];
    return ExpansionTile(
      tilePadding: const EdgeInsets.symmetric(horizontal: 8),
      leading: const Icon(Icons.layers_rounded, size: 16, color: textSecondary),
      title: Text(name.toString(),
          style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: textSecondary)),
      children: (rooms as List).map((room) => _buildRoom(room)).toList(),
    );
  }

  Widget _buildRoom(dynamic room) {
    final name = room['name'] ?? room['roomName'] ?? '?';
    final code = room['code'] ?? room['roomCode'] ?? '';
    final isSelected = _selectedRoom == code;
    return ListTile(
      dense: true,
      selected: isSelected,
      selectedTileColor: primaryColor.withAlpha(20),
      leading: Icon(Icons.meeting_room_rounded,
          size: 14, color: isSelected ? primaryColor : textMuted),
      title: Text(name.toString(),
          style: TextStyle(
              fontSize: 12,
              fontWeight: isSelected ? FontWeight.w600 : FontWeight.w400,
              color: isSelected ? primaryColor : textSecondary)),
      onTap: () {
        setState(() => _selectedRoom = code);
        widget.onRoomSelected(code);
      },
    );
  }
}
```

---

### Task 7: `bed_content.dart` — 床位表格

**Files:**
- Create: `frontend/lib/widgets/bed_content.dart`

- [ ] **Step 1: 写 bed_content.dart**

```dart
import 'package:flutter/material.dart';
import '../services/api_service.dart';
import '../theme/app_theme.dart';

class BedContent extends StatefulWidget {
  final ApiService api;
  final String divideId;
  final String roomCode;
  final String personsn;
  final List<Map<String, dynamic>> collection;
  final bool readOnly;
  final void Function(List<Map<String, dynamic>>, int) onCollectionChanged;

  const BedContent({
    super.key,
    required this.api,
    required this.divideId,
    required this.roomCode,
    required this.personsn,
    required this.collection,
    required this.readOnly,
    required this.onCollectionChanged,
  });

  @override
  State<BedContent> createState() => _BedContentState();
}

class _BedContentState extends State<BedContent> {
  List<dynamic> _beds = [];
  bool _loading = false;

  @override
  void didUpdateWidget(BedContent oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.roomCode != widget.roomCode) {
      _loadBeds();
    }
  }

  @override
  void initState() {
    super.initState();
    _loadBeds();
  }

  Future<void> _loadBeds() async {
    setState(() => _loading = true);
    try {
      final resp = await widget.api.getRoomBeds(widget.divideId, widget.roomCode);
      if (resp['code'] == 0 && resp['bedsInfo'] != null) {
        _beds = List.from(resp['bedsInfo']);
      }
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  bool _isCollected(String bedCode) {
    return widget.collection.any((c) => c['bedCode'] == bedCode);
  }

  void _addToCollection(dynamic bed) {
    final bedCode = (bed['bedCode'] ?? bed['code'] ?? '').toString();
    final bedName = (bed['bedName'] ?? bed['name'] ?? bedCode).toString();
    final newCol = List<Map<String, dynamic>>.from(widget.collection);
    newCol.add({
      'bedCode': bedCode,
      'bedName': bedName,
      'roomCode': widget.roomCode,
      'buildingCode': '',
      'priority': newCol.length + 1, // 默认优先级递增
    });
    widget.onCollectionChanged(newCol, 10);
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) return const Center(child: CircularProgressIndicator());

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.meeting_room_rounded, size: 18, color: primaryColor),
              const SizedBox(width: 8),
              Text('房间 ${widget.roomCode}',
                  style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: textPrimary)),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.refresh_rounded, size: 20),
                onPressed: _loadBeds,
              ),
            ],
          ),
          const SizedBox(height: 12),
          Expanded(
            child: ListView.builder(
              itemCount: _beds.length,
              itemBuilder: (_, i) {
                final bed = _beds[i];
                final code = (bed['bedCode'] ?? bed['code'] ?? '').toString();
                final name = (bed['bedName'] ?? bed['name'] ?? code).toString();
                final status = (bed['status'] ?? '').toString();
                final collected = _isCollected(code);
                final isOccupied = status != '0' && status != '空闲';

                return Container(
                  margin: const EdgeInsets.only(bottom: 8),
                  decoration: BoxDecoration(
                    color: surfaceColor,
                    borderRadius: BorderRadius.circular(radiusLg),
                    border: Border.all(color: borderColor),
                  ),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                    child: Row(
                      children: [
                        Container(
                          width: 3, height: 36,
                          decoration: BoxDecoration(
                            color: isOccupied ? textMuted : (collected ? successColor : primaryColor),
                            borderRadius: BorderRadius.circular(2),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(name, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: textPrimary)),
                              Text(code, style: const TextStyle(fontSize: 11, fontFamily: 'monospace', color: textSecondary)),
                            ],
                          ),
                        ),
                        if (isOccupied)
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                            decoration: BoxDecoration(
                              color: textMuted.withAlpha(25),
                              borderRadius: BorderRadius.circular(radiusSm),
                            ),
                            child: const Text('已占', style: TextStyle(fontSize: 12, color: textMuted)),
                          )
                        else if (collected)
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                            decoration: BoxDecoration(
                              color: successColor.withAlpha(25),
                              borderRadius: BorderRadius.circular(radiusSm),
                            ),
                            child: const Text('已收藏', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: successColor)),
                          )
                        else if (!widget.readOnly)
                          Container(
                            decoration: BoxDecoration(
                              gradient: primaryGradient,
                              borderRadius: BorderRadius.circular(radiusMd),
                            ),
                            child: InkWell(
                              borderRadius: BorderRadius.circular(radiusMd),
                              onTap: () => _addToCollection(bed),
                              child: const Padding(
                                padding: EdgeInsets.symmetric(horizontal: 14, vertical: 6),
                                child: Text('+ 收藏', style: TextStyle(color: Colors.white, fontSize: 12.5, fontWeight: FontWeight.w600)),
                              ),
                            ),
                          ),
                      ],
                    ),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
```

---

### Task 8: `collection_panel.dart` — 收藏面板

**Files:**
- Create: `frontend/lib/widgets/collection_panel.dart`

- [ ] **Step 1: 写 collection_panel.dart**

```dart
import 'package:flutter/material.dart';
import '../theme/app_theme.dart';

class CollectionPanel extends StatelessWidget {
  final List<Map<String, dynamic>> collection;
  final int totalConcurrency;
  final bool readOnly;
  final void Function(List<Map<String, dynamic>>, int) onChanged;

  const CollectionPanel({
    super.key,
    required this.collection,
    required this.totalConcurrency,
    required this.readOnly,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      constraints: const BoxConstraints(maxHeight: 200),
      decoration: BoxDecoration(
        color: surfaceColor,
        border: const Border(top: BorderSide(color: borderColor)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Row(
              children: [
                const Text('收藏列表', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: textPrimary)),
                const Spacer(),
                SizedBox(
                  width: 60,
                  child: TextField(
                    decoration: const InputDecoration(
                      labelText: '并发',
                      isDense: true,
                      contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 8),
                    ),
                    keyboardType: TextInputType.number,
                    controller: TextEditingController(text: '$totalConcurrency'),
                    readOnly: readOnly,
                    onSubmitted: (v) {
                      final n = int.tryParse(v);
                      if (n != null && n >= collection.length) {
                        onChanged(collection, n);
                      }
                    },
                  ),
                ),
              ],
            ),
          ),
          Expanded(
            child: collection.isEmpty
                ? const Center(child: Text('暂无收藏', style: TextStyle(fontSize: 12, color: textMuted)))
                : ListView.builder(
                    itemCount: collection.length,
                    itemBuilder: (_, i) {
                      final bed = collection[i];
                      return Container(
                        margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                        decoration: BoxDecoration(
                          color: surfaceSecondary,
                          borderRadius: BorderRadius.circular(radiusSm),
                          border: Border.all(color: borderLight),
                        ),
                        child: Row(
                          children: [
                            Container(
                              width: 24, height: 24,
                              decoration: BoxDecoration(
                                color: primaryColor.withAlpha(20),
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: Center(
                                child: Text('${i + 1}',
                                    style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w700, color: primaryColor)),
                              ),
                            ),
                            const SizedBox(width: 8),
                            Expanded(
                              child: Text(bed['bedName'] ?? bed['bedCode'] ?? '?',
                                  style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: textPrimary)),
                            ),
                            _buildPriorityDropdown(bed, i),
                            const SizedBox(width: 4),
                            if (!readOnly)
                              InkWell(
                                onTap: () {
                                  final newCol = List<Map<String, dynamic>>.from(collection);
                                  newCol.removeAt(i);
                                  onChanged(newCol, totalConcurrency);
                                },
                                child: const Icon(Icons.delete_outline_rounded, size: 18, color: dangerColor),
                              ),
                          ],
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }

  Widget _buildPriorityDropdown(Map<String, dynamic> bed, int index) {
    final current = bed['priority'] ?? 1;
    return PopupMenuButton<int>(
      initialValue: current,
      enabled: !readOnly,
      onSelected: (v) {
        final newCol = List<Map<String, dynamic>>.from(collection);
        newCol[index]['priority'] = v;
        onChanged(newCol, totalConcurrency);
      },
      itemBuilder: (_) => List.generate(5, (i) => i + 1).map((p) =>
        PopupMenuItem(value: p, height: 32,
          child: Text('优先级 $p', style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: textPrimary)),
        ),
      ).toList(),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: surfaceSecondary,
          borderRadius: BorderRadius.circular(radiusSm),
          border: Border.all(color: borderColor),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('$current', style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: textPrimary)),
            const SizedBox(width: 2),
            const Icon(Icons.arrow_drop_down_rounded, size: 15, color: textSecondary),
          ],
        ),
      ),
    );
  }
}
```

---

### Task 9: `grab_panel.dart` — 抢床控制

**Files:**
- Create: `frontend/lib/widgets/grab_panel.dart`

- [ ] **Step 1: 写 grab_panel.dart**

```dart
import 'dart:async';
import 'package:flutter/material.dart';
import '../services/api_service.dart';
import '../theme/app_theme.dart';

class GrabPanel extends StatefulWidget {
  final ApiService api;
  final String personsn;
  final String divideId;
  final int totalConcurrency;
  final int collectionCount;
  final Map<String, dynamic>? grabStatus;
  final void Function(Map<String, dynamic>?) onStatusChanged;

  const GrabPanel({
    super.key,
    required this.api,
    required this.personsn,
    required this.divideId,
    required this.totalConcurrency,
    required this.collectionCount,
    required this.grabStatus,
    required this.onStatusChanged,
  });

  @override
  State<GrabPanel> createState() => _GrabPanelState();
}

class _GrabPanelState extends State<GrabPanel> {
  Timer? _pollTimer;

  bool get _isGrabbing => widget.grabStatus != null && widget.grabStatus!['running'] == true;

  @override
  void dispose() {
    _pollTimer?.cancel();
    super.dispose();
  }

  void _startPolling() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(const Duration(seconds: 1), (_) async {
      if (!mounted) return;
      try {
        final status = await widget.api.grabStatus();
        if (mounted) {
          widget.onStatusChanged(status);
          if (status['running'] == false) {
            _pollTimer?.cancel();
            _showResult(status);
          }
        }
      } catch (_) {}
    });
  }

  void _showResult(Map<String, dynamic> status) {
    if (!mounted) return;
    final success = status['success'] == true;
    final bed = status['successBed'] ?? '';
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(success ? '抢床成功!' : '抢床结束'),
        content: Text(success ? '成功抢到: $bed' : '所有床位均未抢到'),
        actions: [
          FilledButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('确定'),
          ),
        ],
      ),
    );
  }

  Future<void> _toggleGrab() async {
    if (_isGrabbing) {
      await widget.api.grabStop();
      _pollTimer?.cancel();
      widget.onStatusChanged(null);
    } else {
      if (widget.collectionCount == 0) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('请先收藏床位'), backgroundColor: warningColor),
          );
        }
        return;
      }
      await widget.api.grabStart(
        personsn: widget.personsn,
        divideId: widget.divideId,
        totalConcurrency: widget.totalConcurrency,
      );
      _startPolling();
      widget.onStatusChanged({'running': true, 'progress': {}, 'log': []});
    }
  }

  @override
  Widget build(BuildContext context) {
    final status = widget.grabStatus;
    final progress = status?['progress'] as Map<String, dynamic>? ?? {};
    final logs = (status?['log'] as List?)?.cast<String>() ?? [];

    return Container(
      decoration: BoxDecoration(
        color: surfaceColor,
        border: const Border(top: BorderSide(color: borderColor)),
      ),
      padding: const EdgeInsets.all(12),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              const Text('抢床控制', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: textPrimary)),
              const Spacer(),
              SizedBox(
                height: 36,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(radiusMd),
                    gradient: _isGrabbing
                        ? const LinearGradient(colors: [dangerColor, Color(0xFFDC2626)])
                        : primaryGradient,
                  ),
                  child: FilledButton(
                    onPressed: _toggleGrab,
                    style: FilledButton.styleFrom(
                      backgroundColor: Colors.transparent,
                      shadowColor: Colors.transparent,
                      padding: const EdgeInsets.symmetric(horizontal: 16),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(_isGrabbing ? Icons.stop_rounded : Icons.play_arrow_rounded, size: 20),
                        const SizedBox(width: 4),
                        Text(_isGrabbing ? '停 止' : '开 始',
                            style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600)),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
          if (progress.isNotEmpty) ...[
            const SizedBox(height: 8),
            ...progress.entries.map((e) {
              final bp = e.value as Map<String, dynamic>? ?? {};
              final done = bp['done'] ?? 0;
              final total = bp['total'] ?? 0;
              final ok = bp['ok'] ?? 0;
              final fail = bp['fail'] ?? 0;
              final ratio = total > 0 ? done / total : 0.0;
              return Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Row(
                  children: [
                    SizedBox(width: 60, child: Text(e.key, style: const TextStyle(fontSize: 11, fontFamily: 'monospace', color: textSecondary))),
                    Expanded(
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(2),
                        child: LinearProgressIndicator(
                          value: ratio, minHeight: 6,
                          backgroundColor: borderColor,
                          valueColor: const AlwaysStoppedAnimation<Color>(primaryColor),
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Text('$ok/$fail', style: const TextStyle(fontSize: 10, color: textMuted)),
                  ],
                ),
              );
            }),
          ],
          if (logs.isNotEmpty) ...[
            const SizedBox(height: 8),
            SizedBox(
              height: 80,
              child: ListView.builder(
                itemCount: logs.length,
                itemBuilder: (_, i) => Text(logs[i],
                    style: const TextStyle(fontSize: 11, color: textSecondary, fontFamily: 'monospace')),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
```

---

### Task 10: 登录后跳转 bed_page

**Files:**
- Modify: `frontend/lib/pages/login_page.dart`

- [ ] **Step 1: 修改 `_gotoHome` → `_gotoBed`，跳转 bed_page**

找到 `login_page.dart` 中的 `_gotoHome` 方法和 `_login` 中所有调用，替换为：

```dart
Future<void> _gotoBed() async {
  if (!mounted) return;
  final studentCode = _accountCtl.text.trim();
  Navigator.pushReplacement(
    context,
    MaterialPageRoute(builder: (_) => BedPage(api: _api, studentCode: studentCode)),
  );
}
```

把所有 `_gotoHome()` 改为 `_gotoBed()`。import 中添加 `import 'bed_page.dart';` 并删除 `import 'home_page.dart';`。

- [ ] **Step 2: 删除 home_page.dart**

```bash
rm D:\XJTU-Housing-Genius\frontend\lib\pages\home_page.dart
```

- [ ] **Step 3: 验证前端无语法错误**

```bash
cd D:\XJTU-Housing-Genius\frontend && flutter analyze lib/
```

Expected: 无错误。

---

