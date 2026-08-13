param(
    [Parameter(Mandatory = $true)][string]$ReleaseDirectory
)

$ErrorActionPreference = 'Stop'
$directory = Resolve-Path -LiteralPath $ReleaseDirectory
$manifestPath = Join-Path $directory 'latest.json'
if (-not (Test-Path -LiteralPath $manifestPath)) { throw 'latest.json was not found.' }
$manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json
if ($manifest.schemaVersion -ne 1) { throw 'Unsupported latest.json schema version.' }
if ($manifest.assets.Count -ne 2) { throw 'The release must contain x64 and arm64 assets.' }

foreach ($asset in $manifest.assets) {
    if ($asset.runtime -notin @('win-x64', 'win-arm64')) { throw "Unsupported runtime: $($asset.runtime)" }
    $path = Join-Path $directory $asset.file
    if (-not (Test-Path -LiteralPath $path)) { throw "Missing release asset: $($asset.file)" }
    $actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $path).Hash.ToLowerInvariant()
    if ($actualHash -ne $asset.sha256.ToLowerInvariant()) { throw "Checksum mismatch: $($asset.file)" }
    if ((Get-Item -LiteralPath $path).Length -ne [long]$asset.size) { throw "Size mismatch: $($asset.file)" }
}

Write-Output "Release validation passed: $($manifest.version)"
