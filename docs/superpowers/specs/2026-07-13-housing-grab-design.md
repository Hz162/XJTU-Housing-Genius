# Housing-Genius 抢床功能设计文档

## 1. 架构总览

```
┌──────────────────────────────────────────────────┐
│                  Flutter 前端                      │
│  login → bed_page (sidebar + 床位面板 + 收藏 + 抢床)   │
│                      ↕ HTTP (127.0.0.1:18721)     │
│                  Go 后端                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │
│  │ auth/    │ │ bed/     │ │ bed/             │  │
│  │ login.go │ │ proxy.go │ │ collection.go    │  │
│  │ mfa.go   │ │          │ │ engine.go (抢床)  │  │
│  └──────────┘ └──────────┘ └──────────────────┘  │
│                      ↕ HTTPS                      │
│             housing2021.xjtu.edu.cn               │
└──────────────────────────────────────────────────┘
```

### 后端新模块

| 文件 | 职责 |
|------|------|
| `internal/bed/proxy.go` | 代理 housing bed API，透传请求并处理 AES 加密 |
| `internal/bed/collection.go` | 收藏 JSON 文件读写，优先级校验 |
| `internal/bed/engine.go` | 抢床引擎：并发池、优先级调度、状态机 |

### Session 传递（关键）

```
api.Server
  ├─ client *resty.Client    ← 登录后的认证 client（CAS cookies + token）
  ├─ engine *bed.Engine      ← 抢床引擎（共享同一个 client 引用）
  └─ engine 内部：
       ├─ getClient()  → 返回 *resty.Client（Server.client 的指针）
       ├─ 每次 distributeBed 前检查 IsSessionAlive(client)
       └─ session 过期 → auth.ReloginIfNeeded(client) → 更新 client header
```

**Engine 不持有 client 副本，而是持有 `*resty.Client` 指针**——Server 和 Engine 指向同一个 client 实例。Engine 内部的 goroutine 通过这个指针访问认证状态，session 刷新后所有 goroutine 自动使用新 session。

### 前端新文件

| 文件 | 职责 |
|------|------|
| `pages/bed_page.dart` | 主页面（替代当前 home_page 占位） |
| `widgets/bed_sidebar.dart` | 楼栋树导航（折叠面板） |
| `widgets/bed_content.dart` | 床位展示（表格+状态） |
| `widgets/collection_panel.dart` | 收藏列表（优先级下拉+并发设置） |
| `widgets/grab_panel.dart` | 抢床控制（开始/停止+进度+日志） |

## 2. 后端 API

### 已有机遇（复用）

登录流程不变：`/api/login` → CAS → OAuth → bedAuthenLogin → token。
登录成功或 access_denied 后 session cookies 均保存，可代理 API。

### 新增端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/bed/divideId` | GET | 获取 divideId（代理 `getDivideIdBySn`） |
| `/api/bed/tree` | GET | 床位树（代理 `getBunkTreeByDivideId`） |
| `/api/bed/room-beds` | GET | 房间床位列表（代理 `getBedInfoByDivideId`） |
| `/api/bed/check` | GET | 检查是否已有床位（代理 `checkMyBed`） |
| `/api/bed/collection` | GET | 获取收藏列表（读本地文件） |
| `/api/bed/collection` | POST | 保存收藏列表（写本地文件） |
| `/api/bed/grab/start` | POST | 开始抢床 |
| `/api/bed/grab/stop` | POST | 停止抢床 |
| `/api/bed/grab/status` | GET | 抢床实时状态 |

### 抢床引擎（`engine.go`）

**并发模型：** 每个床位 N 路 goroutine，所有床位同时跑。

```
对于每个收藏床位（按 priority 分组）：
  并发数 = max(1, floor(totalConcurrency × weight / sumWeights))
  其中 weight = 6 - priority

启动所有 goroutine → 
  任一路返回 success → 发信号取消所有其他 → engine 停止
  全部返回 fail → 等待 retryInterval 后重试
  连续 N 轮全部 fail → 引擎停止
```

**状态机：** `idle → running → success | stopped | exhausted`

**AES 加密：** `distributeBed` 请求体中的 `bedPlaceCode` 使用 AES-ECB-PKCS7 加密，
key = `"shu" + timestamp`，与 web 端 `h.a(bedCode, "shu"+timestamp)` 一致。

## 3. 前端 UI（仿 Course-Genius）

### 3.1 页面布局

```
┌─ WindowBar ──────────────────────────────────────┐
│ ☰ XJTU Housing Genius              [抢床中 ⏳]    │
├────────┬──────────────────────────────────────────┤
│ Sidebar│        主内容区                           │
│ (可折叠)│  ┌─ 工具栏 ──────────────────────────┐  │
│        │  │ [楼栋 ▼] [楼层 ▼]              │  │
│  B1栋  │  ├──────────────────────────────────┤  │
│  1层   │  │ 床位表格                          │  │
│   301  │  │ # │ 床位号 │ 状态 │ 操作          │  │
│   302  │  │ 1 │ A床    │ 空闲 │ [+收藏]       │  │
│  B2栋  │  │ 2 │ B床    │ 已占 │ ─            │  │
│        │  └──────────────────────────────────┘  │
│        │  ┌─ 收藏列表 (可折叠) ───────────────┐  │
│        │  │ # │ 床位        │ 优先级▼ │ 操作  │  │
│        │  │ 1 │ B1-301-A   │ 1 ▾    │ 🗑    │  │
│        │  │ 2 │ B2-201-B   │ 3 ▾    │ 🗑    │  │
│        │  └──────────────────────────────────┘  │
│        │  ┌─ 抢床控制 ────────────────────────┐  │
│        │  │ 总并发: [10] 重试间隔: [500ms]     │  │
│        │  │ [═══ 开始抢床 ═══]                 │  │
│        │  │ A001: ████░░ 3/5 成功4 失败1       │  │
│        │  │ B002: ██░░░░ 2/3 成功0 失败2       │  │
│        │  │ 日志: [12:00:01] A001 req1: 失败   │  │
│        │  └──────────────────────────────────┘  │
└────────┴──────────────────────────────────────────┘
```

### 3.2 Sidebar

- 树形结构：楼栋 → 楼层 → 房间
- 可折叠（同 Course-Genius sidebarFold）
- 选中房间后右侧加载床位
- 通过 `/api/bed/tree` 获取数据

### 3.3 床位表格

- 列：序号、床位号、**状态（空闲/已占/已选）**、操作
- 空闲床位显示 `[+收藏]` 按钮（style 同 Course-Genius 的 "+抢课"）
- 已在收藏中的床位显示 `已收藏` 标记
- 已占床位灰色显示

### 3.4 收藏面板

- 下方可折叠面板
- 每行：床位名、**优先级下拉框（1-5）**、删除按钮
- 优先级下拉框样式**完全照抄 Course-Genius 的下拉菜单样式**（`PopupMenuButton` + `Container` 圆角卡片风格）
- 可拖拽排序（`ReorderableListView`），拖拽自动更新优先级顺序
- 总并发数设置（Number input，最小值 = 收藏数）

### 3.5 抢床面板

- 开始/停止按钮（style 同 Course-Genius 抢课按钮）
- 进度条：每个床位独立进度（`LinearProgressIndicator`）
- 实时日志：滚动列表，最新日志在上
- 状态轮询：前端 1 秒间隔 GET `/api/bed/grab/status`

## 4. 数据流

### 登录后初始化

```
1. GET /api/bed/divideId → {divideId, disabled, time, endtime}
2. GET /api/bed/check?personsn=xxx&divideId=xxx → {isMybed: bool}
   - isMybed=true → 已有床位，禁止抢
3. GET /api/bed/collection → 加载本地收藏文件
4. GET /api/bed/tree?divideId=xxx → 楼栋树
```

### 浏览 + 收藏

```
选中房间 → GET /api/bed/room-beds?roomCode=xxx&divideId=xxx
         → bedsInfo: [{bedCode, bedName, status, ...}]
点击[+收藏] → POST /api/bed/collection（追加到本地文件）
```

### 抢床循环

```
POST /api/bed/grab/start
  → engine 读取本地 collection.json
  → 按 priority 分配并发
  → 每个床位 N 路 goroutine 并发 POST distributeBed
  → 任一路成功 → 取消所有 → engine 停止
  → 记录日志到内存

前端轮询 GET /api/bed/grab/status（1s间隔）
  → 显示进度、成功/失败数、日志
```

## 5. 错误处理

| 场景 | 处理 |
|------|------|
| session 过期 | 自动 relogin（复用已保存密码），成功后恢复抢床 |
| divideId 失效 | 重新获取 divideId，提示用户 |
| distributeBed 返回 status=1 | relogin 后重试 |
| 网络超时 | goroutine 重试（单请求超时 5s，最多重试 3 次） |
| 所有床位全部失败 N 轮 | 引擎自动停止，提示"抢床失败" |
| 配置 JSON 损坏 | 重置为空收藏列表 |

## 6. 多实例隔离（已实现，复用 Course-Genius 机制）

每个 Flutter 实例启动独立后端进程，`PORT=` 机制确保不混淆：

```
Flutter App A → spawn backend_A → PORT=18721 → App A 连 18721
Flutter App B → spawn backend_B → PORT=18722 → App B 连 18722
```

- `findAvailablePort("18721")` 从 18721 开始找，最多试 100 个端口
- 后端通过 stdout `PORT=xxxxx` 通知前端，无需读 port 文件
- 多实例共享 config dir（`%APPDATA%/xjtu-housing-genius/`），但各自独立
- 收藏文件按账号存储：`housing-config-{studentCode}.json`，不同账号互不干扰

## 7. Engine 初始化 & divideId

```
Server 启动时:
  engine = bed.NewEngine(client)  // client 尚未认证（nil or 未登录）

登录成功后:
  s.client = authenticatedClient  // CAS cookies + token
  engine.SetClient(s.client)      // 注入已认证 client

用户进入 bed_page:
  GET /api/bed/divideId → 后端代理 getDivideIdBySn
    → 返回 {divideId, disabled, time, endtime}
    → 前端保存 divideId，传给后续所有 bed API
  GET /api/bed/check → {isMybed}
    → isMybed=true → 显示"已有床位"，禁止抢床
```

## 8. 抢床中修改收藏

- 抢床运行时：收藏面板**只读**（禁止添加/删除/改优先级）
- 停止抢床后：恢复可编辑
- 总并发数修改：同样抢床中禁止修改

## 9. 登录后自动跳转

- login_page 登录成功 → `Navigator.pushReplacement(bed_page)`
- relogin 成功后 engine 自动恢复（无需用户重新操作）
- home_page.dart 移除，替换为 bed_page.dart

## 10. 依赖约束

- AES 加密依赖 Go `crypto/aes`（已存在）
- 收藏文件路径：`{configDir}/housing-config.json`（与后端 exe 同目录）
- 前端 Flutter 依赖：现有 `http` 包即够用，无需新增
- 总并发 goroutine 上限：100（防止系统资源耗尽）
- 单个 goroutine 请求间隔：可配置（默认无间隔，即连续发送）
