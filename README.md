# Filespace

Filespace is a high-performance WinUI 3 file workspace for Windows power users. It is designed as a native, async-first alternative to Q-Dir and a foundation for QSpace Pro-style workflows: tabs, dual panes, fast navigation, keyboard-first search, and Windows-specific integration.

## Current capabilities

- WinUI 3 on Windows App SDK 2.3.1, targeting Windows 10 build 17763 and newer.
- Async, cancellable file enumeration with bounded back-pressure so large folders do not block the UI thread.
- Tabs, address navigation, back/forward/up/refresh, details and compact views.
- Dual-pane layout toggle as the base for synchronized pane workflows.
- `Ctrl+K` search overlay with a local asynchronous search engine.
- Mouse-wheel scrolling for file lists, search results, the navigation sidebar, and horizontal tab overflow.
- Optional Everything bridge through `es.exe`; if Everything is unavailable, search falls back to the local engine.
- `Win+F` global hotkey to activate Filespace. The installed MSIX exposes a user-controlled Windows StartupTask so the hotkey can work after sign-in; it can be disabled in Settings. `Win+E` is intentionally untouched and remains Windows Explorer.
- AOT-oriented `Aot` configuration for self-contained publishing.
- Single-project MSIX packaging for x64 and arm64, with StoreUpload output and package validation scripts.

## Build prerequisites

Install the .NET 10 SDK, a Windows 10/11 SDK, and a current Visual Studio WinUI workload with:

- .NET desktop development
- Windows App SDK / WinUI application development

Then build:

```powershell
dotnet restore
dotnet build -c Release -p:Platform=x64
```

The packaged development path is the recommended local run path because it
loads the Windows App SDK runtime and app identity correctly. Developer Mode
must be enabled for local package deployment.

For a self-contained AOT-oriented publish:

```powershell
dotnet publish -c Aot -r win-x64 -p:Platform=x64
```

The Store package can be built with the Windows SDK and Windows App SDK NuGet tooling. A complete Microsoft Store submission still requires a Partner Center identity and publisher account.

## Search providers

The default provider is the built-in local traversal engine. It searches configured user folders and stops at 300 matches. When `Everything` is selected in Settings, Filespace looks for `es.exe` in the standard Everything install locations and `PATH`, then falls back automatically if it is not available.

## Local MSI releases and updates

The supported public distribution is an unsigned, self-contained per-user MSI. It does not require Microsoft Store certification or administrator privileges. The MSI installs under the current user's Local AppData directory and creates a Start Menu shortcut.

Build a release locally after installing WiX Toolset 7:

```powershell
dotnet tool install --global wix --version 7.0.0
wix eula accept wix7
./scripts/build-release.ps1 -Version 0.1.0
./scripts/validate-release.ps1 -ReleaseDirectory artifacts/release
```

The GitHub Actions `Release` workflow builds x64 and arm64 MSI files, SHA-256 sidecars, and `latest.json`, then publishes them as a GitHub Release. Push a tag such as `v0.1.0`, or run the workflow manually with a semantic version.

Filespace checks `UpdateMirrorPrefix` first, then a user-supplied manifest URL, then the public GitHub release manifest. This allows a trusted HTTPS mirror or domestic relay to be configured when GitHub is slow or unreachable. The manifest and MSI are verified by HTTPS, exact file size, and SHA-256 before Windows Installer starts.

When automatic checking cannot reach any source, download the MSI and matching `.sha256` file from the release page or a trusted mirror, then use Settings > Install an MSI downloaded manually. Filespace never silently installs an update.

GitHub Actions artifacts and release metadata contain only build outputs, hashes, and public repository URLs. Do not include private paths, user names, tokens, or personal files in issues, release notes, or commits.

Microsoft Store packaging remains optional and is not required for the MSI distribution path.

## Roadmap

The architecture intentionally leaves room for the remaining QSpace-class features: shell context menus, drag and drop, previews, bookmarks, archive providers, custom columns, split-pane state, command palette, plugins, and a richer resident companion for low-memory background activation.

Filespace is a foundation and working prototype; it does not yet claim feature parity with QSpace Pro.
