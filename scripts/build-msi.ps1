param(
    [switch]$SkipAppBuild
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$env:CGO_ENABLED = "0"
$cfg = Get-Content -LiteralPath "mygo.json" -Raw | ConvertFrom-Json
$version = $cfg.version

if (-not $SkipAppBuild) {
    & go run github.com/egoist/mygo/cmd/mygo@v0.10.0 build -platform windows/amd64 -skip-installer
    if ($LASTEXITCODE -ne 0) { throw "mygo build failed ($LASTEXITCODE)" }
}

$msi = "dist\FileSpace233-$version-x64.msi"
& wix build "build\wix\app.wxs" -arch x64 -o $msi
if ($LASTEXITCODE -ne 0) { throw "wix build failed ($LASTEXITCODE)" }

$hashes = Get-ChildItem -LiteralPath "dist" -File |
    Where-Object { $_.Extension -in ".msi", ".exe" } |
    ForEach-Object {
        $h = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        "$h  $($_.Name)"
    }
$hashes | Set-Content -LiteralPath "dist\SHA256SUMS.txt" -Encoding utf8

$size = [math]::Round((Get-Item -LiteralPath $msi).Length / 1MB, 1)
Write-Host "MSI: $msi ($size MB)"
Write-Host "SHA256SUMS.txt written"
