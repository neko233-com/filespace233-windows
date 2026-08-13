# Filespace Privacy Policy

Last updated: 2026-08-13

Filespace is a local Windows file-management application. It does not require an account and does not send file names, file contents, search queries, or usage telemetry to Filespace servers.

## Local data

Filespace reads directories and files that the user chooses to browse. Settings such as search provider, startup preference, update source, and configured search roots are stored locally in the Windows local application settings container or the per-user Filespace settings file for MSI installs. Filespace does not upload that data.

## Optional Everything integration

When the user selects Everything, Filespace starts the locally installed `es.exe` command-line client and reads its standard output to display matching paths. The query is sent only to the local Everything installation; Filespace does not proxy it through a network service.

## Windows permissions

Filespace uses normal user permissions. Access to protected directories is subject to Windows access checks. Filespace does not request administrator privileges and does not modify the Windows Explorer `Win+E` shortcut.

## Updates

Automatic update checks request only the configured HTTPS manifest and MSI asset. The default source is the public Filespace GitHub release; users may configure a trusted HTTPS mirror. Update requests do not include file names, search queries, account identifiers, or telemetry. Before installation, Filespace checks the expected asset size and SHA-256 hash from the manifest. Users can also download an MSI themselves and start it from Settings.

## Contact

For privacy questions, open an issue at https://github.com/neko233-com/filespace233-windows/issues.
