# Filespace

Filespace is a high-performance WinUI 3 file workspace for Windows power users. It is designed as a native, async-first alternative to Q-Dir and a foundation for QSpace Pro-style workflows: tabs, dual panes, fast navigation, keyboard-first search, and Windows-specific integration.

## Current capabilities

- WinUI 3 on Windows App SDK, targeting Windows 10 19041 and newer.
- Async, cancellable file enumeration with bounded back-pressure so large folders do not block the UI thread.
- Tabs, address navigation, back/forward/up/refresh, details and compact views.
- Dual-pane layout toggle as the base for synchronized pane workflows.
- `Ctrl+K` search overlay with a local asynchronous search engine.
- Optional Everything bridge through `es.exe`; if Everything is unavailable, search falls back to the local engine.
- `Win+F` global hotkey to activate Filespace. On first launch, a current-user startup entry keeps a lightweight Filespace process ready for the hotkey; it can be disabled in Settings. `Win+E` is intentionally untouched and remains Windows Explorer.
- AOT-oriented `Aot` configuration for self-contained publishing once the Windows App SDK toolchain is installed.

## Build prerequisites

Install Visual Studio 2022 17.10 or newer with:

- .NET desktop development
- Windows App SDK / WinUI application development
- Windows 10/11 SDK 10.0.19041.0 or newer

Then build:

```powershell
dotnet restore
dotnet build -c Release -p:Platform=x64
```

For a self-contained AOT-oriented publish:

```powershell
dotnet publish -c Aot -r win-x64 -p:Platform=x64
```

The current machine used to scaffold this repository does not have Visual Studio or the Windows App SDK workload installed, so full native compilation must be run on a Windows development machine with those prerequisites.

## Search providers

The default provider is the built-in local traversal engine. It searches configured user folders and stops at 300 matches. When `Everything` is selected in settings in a later configuration surface, Filespace looks for `es.exe` in the standard Everything install locations and `PATH`, then falls back automatically if it is not available.

## Roadmap

The architecture intentionally leaves room for the remaining QSpace-class features: shell context menus, drag and drop, previews, bookmarks, archive providers, custom columns, split-pane state, command palette, plugins, and a richer resident companion for low-memory background activation.

Filespace is a foundation and working prototype; it does not yet claim feature parity with QSpace Pro.
