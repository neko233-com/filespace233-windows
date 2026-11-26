param(
    [switch]$NoPaw
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$env:CGO_ENABLED = "0"

$goos = (& go env GOOS).Trim()
$goarch = (& go env GOARCH).Trim()
if ($goos -ne "windows" -or $goarch -ne "amd64") {
    throw "verify.ps1 is for Windows/amd64 only (host: $goos/$goarch)"
}

Write-Host "== gofmt =="
$unformatted = & gofmt -l . 2>$null
if ($LASTEXITCODE -ne 0) { throw "gofmt failed" }
if ($unformatted) {
    $unformatted | ForEach-Object { Write-Host "  needs gofmt: $_" }
    throw "gofmt found unformatted files"
}

Write-Host "== go vet =="
& go vet ./...
if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

Write-Host "== go test =="
& go test -count=1 ./...
if ($LASTEXITCODE -ne 0) { throw "go test failed" }

Write-Host "== build + MSI =="
& powershell -NoProfile -ExecutionPolicy Bypass -File "$root\scripts\build-msi.ps1"
if ($LASTEXITCODE -ne 0) { throw "build-msi failed" }

Write-Host "== first frame =="
$shot = Join-Path $root "build\shot.png"
if (Test-Path -LiteralPath $shot) { Remove-Item -LiteralPath $shot -Force }
$p = Start-Process -FilePath "$root\dist\windows-amd64\FileSpace233.exe" `
    -ArgumentList "-capture", $shot -PassThru
if (-not $p.WaitForExit(25000)) {
    $p.Kill()
    throw "app did not finish -capture in time"
}
if (-not (Test-Path -LiteralPath $shot)) { throw "no screenshot produced" }

New-Item -ItemType Directory -Force -Path "$root\docs" | Out-Null
Copy-Item -LiteralPath $shot -Destination "$root\docs\shot.png" -Force

if (-not $NoPaw) {
    $cfg = Get-Content -LiteralPath "mygo.json" -Raw | ConvertFrom-Json
    $stamp = Get-Date -Format "yyyy-MM-dd HH:mm"
    Add-Content -LiteralPath "$root\AGENTS.md" -Encoding utf8 `
        -Value "| 🐾 $stamp | v$($cfg.version) | PASS | verify.ps1 |"
}

Write-Host "All checks passed. Screenshot: docs/shot.png"
