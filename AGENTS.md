# AGENTS.md

FileSpace233 的工程规则。改代码前先读这一页。

## 栈与硬约束

- **Go 1.27，纯 Go，`CGO_ENABLED=0`**。不允许 `import "C"`，不允许 C/C++ 构建依赖。
- **原生 UI**：`github.com/egoist/mygo`（GPUI 思路的 Go 原生 UI，`ui.View` / `ui.Context`）+ `github.com/lobehub/lobe-mygo`（Lobe 风格组件与 `theme.TokensOf`，即本项目的 "awesome" 组件层）。不用 WebView、HTML、CSS、JS。
- 所有构建、测试、CI 都必须显式 `CGO_ENABLED=0`，验证时 `go env CGO_ENABLED` 必须是 `0`。
- **发布目标只有 `windows/amd64`（x86-64）+ MSI**。不产出 NSIS `.exe` 安装包，不产出 macOS / Linux / arm64 包。
- 图标与发布资源：`resources/icon.png`（1024×1024，`mygo build` 生成 exe 图标）、`resources/tray.png`（32×32 托盘图标）、`resources/app.ico`（WiX 快捷方式 / ARP 图标）。

## 后台常驻与托盘（必须保持）

- 关闭主窗口时 **`PreventDefault` + `Hide`**，进程留在托盘，不能退出。
- 托盘图标必须在任务栏通知区可见（参照用户提供的图 1），ToolTip 写 "FileSpace233 — 文件夹管理器（后台运行中）"。
- 托盘菜单至少包含 `打开 FileSpace233` 和 `退出`；只有菜单里的 `退出` 才调用 `mygo.App.Quit()`。
- `mygo.App.OnWindowAllClosed(func(){})` 保活；再次启动用 `RequestSingleInstanceLock` 聚焦已有窗口，不再起第二个进程。
- 不允许把"关闭到托盘"改成"关闭即退出"，也不允许隐藏托盘图标。

## 分层

| 包 / 文件 | 职责 | 不许做 |
| --- | --- | --- |
| `internal/fsx` | 目录读取、驱动器枚举、排序、大小统计 | 依赖 `mygo` / `ui` |
| `app.go` | 状态、导航、历史、持久化、托盘 | 直接画 UI |
| `view.go` | 视图与交互 | 阻塞 I/O、直接 `os.ReadDir` |
| `scripts/` | 构建、MSI、验收 | 隐藏失败退出码 |

- 目录读取一律走 `fsx`；UI 线程不做阻塞 I/O。
- 导航结果必须通过 `win.Update` 回 UI 线程，并用 `gen` 拒绝过期结果。
- 列表用 `ui.List` 虚拟化，不要一次性构建全部行。
- `view.go` 里不要创建 `ui.Bitmap`，logo 只在包级 `ui.DecodeBitmap` 创建一次。

## 本机验收（每次改动后按爪）

仅当机器是 **Windows + `go env GOARCH=amd64`** 时执行。每次改完代码、文档或打包脚本，必须运行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\verify.ps1
```

脚本会依次做 `gofmt` 检查、`go vet`、`go test`、`CGO_ENABLED=0` 构建、`mygo build`、WiX MSI、首帧截图，并在全部通过后向本文件末尾的「🐾 爪印记录」追加一行。**不要手写爪印，也不要跳过失败项伪造 PASS。**

非 Windows/amd64 环境只跑 `go test ./...` 和 `go vet ./...`，并在 PR 里写明未做本机 MSI 验收。

## 构建与发布

```powershell
# 开发
CGO_ENABLED=0 go run .

# 生产 exe + 资源（不生成 NSIS）
CGO_ENABLED=0 go run github.com/egoist/mygo/cmd/mygo@v0.10.0 build -platform windows/amd64 -skip-installer

# MSI（WiX 5，必须 x64）
powershell -File .\scripts\build-msi.ps1
```

产物：

- `dist/windows-amd64/FileSpace233.exe`（mygo 生产构建，已内嵌图标/版本信息）
- `dist/FileSpace233-<version>-x64.msi`（唯一安装包）
- `dist/SHA256SUMS.txt`

版本号同时改 `main.go` 的 `version`、`mygo.json` 的 `version`、`build/wix/app.wxs` 的 `Package Version`。

## MyGo 相关 skill

随 CLI 安装到 `.agents/skills/`：

- `mygo-maintenance`：元素生命周期、稳定 Key、线程与 `ui.Tester`。
- `mygo-native-modules`：本项目不需要原生模块，保持纯 Go。

升级 MyGo 后刷新：

```powershell
go run github.com/egoist/mygo/cmd/mygo@v0.10.0 install-skills .
```

改 UI 前先对照固定版本源码（`go list -m -f '{{.Dir}}' github.com/egoist/mygo`），不要凭记忆写 API。

## 数据诚实

- 目录读不到就显示错误行，不要显示空列表假装成功。
- 文件夹大小显示 `—`（未递归统计），不要编一个数字。
- 托盘图标创建失败要在 stderr 留日志，不许静默吞掉。

## 🐾 爪印记录

| 时间 | 版本 | 结果 | 执行者 |
| --- | --- | --- | --- |
| 🐾 2026-11-26 14:58 | v0.1.0 | PASS | verify.ps1 |
| 🐾 2026-11-26 15:14 | v0.1.0 | PASS | verify.ps1 |
