# XJTU Housing Genius

西安交通大学抢宿舍辅助工具 — 浏览床位、收藏管理、多优先级并发抢床。

## 功能

- **统一身份认证登录** — CAS/OAuth 登录，支持 MFA 验证码
- **床位浏览** — 楼栋→楼层→房间树形导航，查看床位状态
- **收藏管理** — 收藏意向床位，设置 1-5 级优先级
- **并发抢床** — 多床位同时抢，优先级自动分配并发数，任一路成功即停止
- **三端支持** — Windows / macOS / Linux

## 架构

```
Flutter 前端 (login → bed_page)
    ↕ HTTP (127.0.0.1:18721)
Go 后端
    ├─ auth/  CAS/OAuth 登录
    ├─ bed/   床位代理 + 收藏引擎 + 抢床引擎
    └─ session/ 认证状态管理
    ↕ HTTPS
housing2021.xjtu.edu.cn
```

## 开发

### 后端

```bash
cd backend
go build -o xjtu-housing-genius.exe .
./xjtu-housing-genius.exe    # 端口 18721
```

Mock 测试后端（无真实 housing 连接）：

```bash
go build -o cmd/mock/mock-backend.exe ./cmd/mock/
./cmd/mock/mock-backend.exe  # 端口 18730
```

### 前端

```bash
cd frontend
flutter run -d windows   # 自动启动后端
flutter run -d macos
flutter run -d linux
```

> 前端通过 stdout `PORT=xxxxx` 自动发现后端端口，多实例互不干扰。

## 打包发布

### Windows (Inno Setup)

```bash
cd frontend && flutter build windows --release
iscc scripts/setupconfig.iss
```

### macOS

```bash
cd frontend && flutter build macos --release
# 生成 .dmg 见 .github/workflows/release.yml
```

### Linux

```bash
cd frontend && flutter build linux --release
# 生成 .deb 见 .github/workflows/release.yml
```

## GitHub Actions

推送 `v*` tag 或手动触发 workflow 自动构建三端并发布 Release。

## License

MIT
