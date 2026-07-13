# XJTU Housing Genius v1.0

> 西安交通大学宿舍抢床助手 — 基于 Flutter + Go 的跨平台桌面应用

## 功能

- **统一认证登录** — XJTU CAS/OAuth 登录，支持 MFA 二次验证、验证码
- **楼栋-楼层-房间树形浏览** — 7 级树形结构（校区→园区→楼栋→单元→楼层→房间）
- **床位收藏** — 浏览房间床位，一键收藏，自动同步到服务器（`saveBed` / `getBedCollectList`）
- **收藏列表** — 显示收藏人数、床位状态，支持优先级设置、删除
- **并发抢床** — 基于优先级的自动并发分配，50ms 重试间隔，无限制循环直到抢到
- **跨平台** — Windows（.exe / 安装包）、Linux（.deb / .tar.gz）、macOS（.dmg）

## 安装

### Windows
- **安装包**: 运行 `XJTU-Housing-Genius-Setup-v1.0.exe`，自动创建桌面快捷方式和开始菜单项
- **便携版**: 解压 `XJTU-Housing-Genius-windows-x64.zip`，运行 `xjtu_housing_genius.exe`

### macOS
- 挂载 `XJTU-Housing-Genius-macos.dmg`，拖入 Applications 文件夹
- 首次运行如提示"无法验证开发者"，在系统设置 → 隐私与安全性中点击"仍要打开"

### Linux
- Debian/Ubuntu: `sudo dpkg -i XJTU-Housing-Genius-linux-x64.deb`
- 其他发行版: 解压 `XJTU-Housing-Genius-linux-x64.tar.gz`，运行 `xjtu_housing_genius`

## 使用说明

1. 启动应用，输入学号和 CAS 密码登录
2. 如需 MFA 验证码，输入短信/邮箱收到的验证码
3. 左侧树形结构浏览宿舍楼栋，点击房间查看床位
4. 点击「+ 收藏」将床位加入收藏列表，设置优先级（1-5）
5. 在收藏列表中设置总并发数
6. 点击「开始」启动抢床引擎，实时查看进度和日志
7. 抢到后弹窗提示，点击「确定」

### 并发分配策略

总并发数按优先级权重自动分配：权重 = 6 − 优先级。例如 10 总并发、3 个床位（优先级 1/2/3），分配比例为 5:4:3 → 5 路 + 3 路 + 2 路 ≈ 10。

### 注意事项

- 抢床需在选宿时段内进行，建议提前 30 秒启动引擎
- 已有床位的账号无法再次抢床（服务器限制）
- 每个账号最多收藏 5 个床位（服务器限制）
- 收藏列表会同步到学校服务器，删除收藏也会同步

## 技术栈

| 层 | 技术 |
|---|------|
| 前端 | Flutter 3.44 + Material 3 |
| 后端 | Go 1.21 + chi router + resty |
| 登录 | CAS/OAuth + AES-ECB 密码加密 |
| 床 API | 完全复刻 housing2021.xjtu.edu.cn 接口 |
| 打包 | Inno Setup (Win) / dpkg-deb (Linux) / hdiutil (Mac) |

## 开发

```bash
# 启动后端
cd backend && go run .

# 启动前端（会自动启动后端）
cd frontend && flutter run -d windows

# Mock 后端（离线测试 UI）
cd backend && go run ./cmd/mock
```
