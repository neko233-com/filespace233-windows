param(
    [Parameter(Mandatory = $false)][string]$PackagePath
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($PackagePath)) {
    $PackagePath = Get-ChildItem -Path (Join-Path (Get-Location) 'AppPackages') -Recurse -File -Filter '*.msix' |
        Where-Object { $_.Name -match '_x64(?:[_.]|$)' -and $_.Name -notmatch 'bundle|upload|Microsoft.WindowsAppRuntime' } |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1 -ExpandProperty FullName
}
if ([string]::IsNullOrWhiteSpace($PackagePath) -or -not (Test-Path -LiteralPath $PackagePath)) {
    throw 'No MSIX package was found.'
}

$makeAppx = Get-ChildItem 'C:\Program Files (x86)\Windows Kits\10\bin' -Recurse -Filter makeappx.exe -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -match '\\x64\\makeappx\.exe$' } |
    Sort-Object FullName -Descending |
    Select-Object -First 1 -ExpandProperty FullName
if ([string]::IsNullOrWhiteSpace($makeAppx)) { throw 'Windows SDK makeappx.exe was not found.' }

$unpackDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("filespace-msix-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $unpackDirectory | Out-Null
try {
    & $makeAppx unpack /p $PackagePath /d $unpackDirectory /o
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    $manifestPath = Join-Path $unpackDirectory 'AppxManifest.xml'
    if (-not (Test-Path -LiteralPath $manifestPath)) { throw 'The unpacked MSIX has no AppxManifest.xml.' }
    [xml]$manifest = Get-Content -Raw -LiteralPath $manifestPath
    $identity = $manifest.Package.Identity
    if ([string]::IsNullOrWhiteSpace($identity.Name) -or [string]::IsNullOrWhiteSpace($identity.Version)) {
        throw 'The package identity is missing Name or Version.'
    }
    foreach ($asset in @('Assets\StoreLogo.png', 'Assets\Square150x150Logo.png', 'Assets\Square44x44Logo.png')) {
        if (-not (Test-Path -LiteralPath (Join-Path $unpackDirectory $asset))) { throw "Required package asset is missing: $asset" }
    }
    Write-Output "MSIX validation passed: $PackagePath ($($identity.Name) $($identity.Version))"
}
finally {
    if (Test-Path -LiteralPath $unpackDirectory) { Remove-Item -LiteralPath $unpackDirectory -Recurse -Force }
}
