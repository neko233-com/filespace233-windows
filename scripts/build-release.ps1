param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $false)][string]$OutputRoot
)

$ErrorActionPreference = 'Stop'
if ($Version.StartsWith('v', [System.StringComparison]::OrdinalIgnoreCase)) { $Version = $Version.Substring(1) }
if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw 'Version must be a three-part semantic version, for example 0.1.0.' }
$root = Resolve-Path (Join-Path $PSScriptRoot '..')
$outputDirectory = if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
    [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\artifacts\release'))
} else {
    [System.IO.Path]::GetFullPath($OutputRoot)
}
New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null
Get-ChildItem -LiteralPath $outputDirectory -Force | Remove-Item -Recurse -Force

foreach ($runtime in @('win-x64', 'win-arm64')) {
    & (Join-Path $PSScriptRoot 'build-msi.ps1') -Version $Version -Runtime $runtime -OutputRoot $outputDirectory
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

$assets = foreach ($runtime in @('win-x64', 'win-arm64')) {
    $name = "Filespace-$Version-$runtime.msi"
    $path = Join-Path $outputDirectory $name
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $path).Hash.ToLowerInvariant()
    [ordered]@{
        runtime = $runtime
        file = $name
        sha256 = $hash
        size = (Get-Item -LiteralPath $path).Length
    }
}

$manifest = [ordered]@{
    schemaVersion = 1
    version = $Version
    tag = "v$Version"
    releasePage = "https://github.com/neko233-com/filespace233-windows/releases/tag/v$Version"
    assets = @($assets)
}
$manifest | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $outputDirectory 'latest.json') -Encoding utf8
Write-Output "Release artifacts written to $outputDirectory"
