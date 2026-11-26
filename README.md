# FileSpace233

Windows 文件夹管理器：Go 1.27 + [MyGo](https://github.com/egoist/mygo) 原生 GPU UI + [Lobe MyGo](https://github.com/lobehub/lobe-mygo) 主题组件，纯 Go、`CGO_ENABLED=0`、无 WebView。

![FileSpace233](docs/shot.png)

## 功能

- 快速访问：主目录、桌面、文档、下载、图片、音乐、视频；自动列出可用盘符。
- 浏览与导航：后退 / 前进 / 上一级、面包屑跳转、路径复制、在资源管理器打开。
- 文件列表：虚拟化列表、目录优先、名称 / 大小 / 修改日期 / 类型排序、搜索过滤、隐藏文件开关。
- 状态栏：项目数、文件 / 文件夹数、文件总大小、当前选中项。
- **后台常驻**：关闭主窗口只隐藏，托盘图标始终可见；托盘菜单提供"打开"和"退出"。

![托盘常驻](docs/tray.png)
- 单实例：重复启动会聚焦已打开的窗口，不会出现第二个托盘图标。
- 记住上次目录、排序和隐藏文件设置。

## 安装

只发布 **Windows x86-64 MSI**：

```powershell
msiexec /i FileSpace233-0.1.0-x64.msi
```

安装到 `%LOCALAPPDATA%\Programs\FileSpace233`，无需管理员权限；创建开始菜单和桌面快捷方式，卸载保留用户数据。

## 开发

```powershell
$env:CGO_ENABLED = "0"
go run .

# 验收（Windows/amd64 本机）
powershell -ExecutionPolicy Bypass -File .\scripts\verify.ps1

# 只打 MSI
powershell -ExecutionPolicy Bypass -File .\scripts\build-msi.ps1
```

产物：

- `dist/windows-amd64/FileSpace233.exe`
- `dist/FileSpace233-<version>-x64.msi`
- `dist/SHA256SUMS.txt`

## 结构

```
app.go              状态、导航、托盘、持久化
view.go             MyGo 原生 UI
internal/fsx/       目录读取与排序（无 UI 依赖）
resources/          icon.png / tray.png / app.ico
build/wix/app.wxs   WiX 5 MSI 定义
scripts/            build-msi.ps1 / verify.ps1
```

工程约束与"每次处理后按爪"的验收规则见 [AGENTS.md](AGENTS.md)。

旧 WinUI 3 实现保存在分支 `archive/winui-0.1`。

MIT License.
