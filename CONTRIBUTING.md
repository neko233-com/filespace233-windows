# Contributing

## Development setup

Filespace is a C# WinUI 3 desktop application targeting .NET 10 and Windows App SDK 2.3.1.

1. Install the .NET 10 SDK, a Windows 10/11 SDK, and the WinUI workload in Visual Studio.
2. Enable Windows Developer Mode for local unpackaged or packaged development.
3. Restore and build with `dotnet restore` and `dotnet build -c Debug -p:Platform=x64`.
4. Run the packaged development app with `dotnet run --project Filespace233.csproj -c Debug -p:Platform=x64`.

Do not commit `bin`, `obj`, `artifacts`, `AppPackages`, `BundleArtifacts`, screenshots, or local settings.

## Pull requests

- Keep changes focused and explain user-visible behavior.
- Preserve the privacy guarantees in `PRIVACY.md`.
- Add or update validation when changing packaging, update delivery, or Windows integration.
- Run the Debug build and the relevant release validation before opening a pull request.
- Do not claim QSpace Pro feature parity unless the feature is implemented and tested.

## Release builds

Install WiX Toolset 7, then run `./scripts/build-release.ps1 -Version x.y.z` and
`./scripts/validate-release.ps1 -ReleaseDirectory artifacts/release`.
