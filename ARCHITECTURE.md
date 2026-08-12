# Architecture

## Principles

1. Keep filesystem and process work off the UI thread.
2. Make every long-running operation cancellable.
3. Deliver bounded incremental results instead of waiting for a full directory snapshot.
4. Keep Windows-specific integration behind small services so the UI remains testable.
5. Keep AOT constraints visible: avoid runtime code generation and use explicit interop boundaries.

## Components

- `FileSystemService`: shallow folder enumeration using `IAsyncEnumerable<FileItem>` and a bounded `Channel<FileItem>`.
- `SearchService`: provider selection and fallback policy.
- `EverythingBridge`: optional `es.exe` process integration with cancellation and no shell execution.
- `SettingsService`: local app settings and default search roots.
- `GlobalHotkeyService`: Win32 `RegisterHotKey` for Win+F only. There is no Win+E hook.
- `StartupService`: optional current-user Run entry that starts the app with `--background`, allowing Win+F to work after sign-in without replacing the Windows Explorer shortcut.
- `MainWindow`: composition layer for navigation state and UI interaction.

## Performance notes

Directory enumeration reports items as they arrive, uses `IgnoreInaccessible`, and skips system entries. Search is bounded to 300 results and uses a cancellation token that is replaced for every new query, preventing stale searches from competing with the active query. The next implementation stage should add a persistent index (SQLite or Windows Search API) for instant global search without walking folders.
