param(
    [Parameter(Mandatory = $false)][string]$Version = '0.1.0',
    [Parameter(Mandatory = $false)][ValidateSet('win-x64', 'win-arm64')][string]$Runtime = 'win-x64',
    [Parameter(Mandatory = $false)][string]$OutputRoot
)

$ErrorActionPreference = 'Stop'
$project = Resolve-Path (Join-Path $PSScriptRoot '..\Filespace233.csproj')
$root = Resolve-Path (Join-Path $PSScriptRoot '..')
$installerSource = Join-Path $root 'Installer\Filespace.wxs'
$publishDirectory = Join-Path $root "artifacts\publish\$Runtime"
$outputDirectory = if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
    [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\artifacts\release'))
} else {
    [System.IO.Path]::GetFullPath($OutputRoot)
}
$msiName = "Filespace-$Version-$Runtime.msi"
$msiPath = Join-Path $outputDirectory $msiName
$wix = Get-Command wix -ErrorAction SilentlyContinue
if ($null -eq $wix) { throw 'WiX Toolset 7 is required. Install it with: dotnet tool install --global wix --version 7.0.0' }

if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw 'Version must be a three-part semantic version, for example 0.1.0.' }
if (-not (Test-Path -LiteralPath $installerSource)) { throw "Installer source was not found: $installerSource" }

New-Item -ItemType Directory -Force -Path $publishDirectory, $outputDirectory | Out-Null
if (Test-Path -LiteralPath $publishDirectory) { Get-ChildItem -LiteralPath $publishDirectory -Force | Remove-Item -Recurse -Force }

$platform = if ($Runtime -eq 'win-arm64') { 'ARM64' } else { 'x64' }
& dotnet publish $project -c Release -r $Runtime --self-contained true `
    -p:WindowsPackageType=None -p:EnableMsixTooling=false -p:WindowsAppSDKSelfContained=true `
    -p:WindowsAppSdkBootstrapInitialize=false `
    -p:Platform=$platform -p:Version=$Version -p:AssemblyVersion="$Version.0" -p:FileVersion="$Version.0" `
    -o $publishDirectory
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

if (-not (Test-Path -LiteralPath (Join-Path $publishDirectory 'Filespace233.exe'))) { throw 'The unpackaged publish did not produce Filespace233.exe.' }
if (Test-Path -LiteralPath $msiPath) { Remove-Item -LiteralPath $msiPath -Force }
& $wix.Source build $installerSource -arch $platform.ToLowerInvariant() `
    -d ProductVersion=$Version -d SourceDir=$publishDirectory -out $msiPath
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $msiPath).Hash.ToLowerInvariant()
Set-Content -LiteralPath "$msiPath.sha256" -Value "$hash  $msiName" -NoNewline
Write-Output "Built $msiPath"
Write-Output "SHA256 $hash"
