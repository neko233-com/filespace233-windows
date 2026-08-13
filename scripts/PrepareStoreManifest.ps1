param(
    [Parameter(Mandatory = $false)][string]$IdentityName = $env:FILESPACE_STORE_IDENTITY_NAME,
    [Parameter(Mandatory = $false)][string]$Publisher = $env:FILESPACE_STORE_PUBLISHER,
    [Parameter(Mandatory = $false)][string]$Version = $env:FILESPACE_STORE_VERSION,
    [Parameter(Mandatory = $false)][switch]$RequirePartnerCenterIdentity
)

$manifestPath = Join-Path $PSScriptRoot '..\Package.appxmanifest'
if (-not (Test-Path -LiteralPath $manifestPath)) { throw "Package.appxmanifest was not found." }

if ($RequirePartnerCenterIdentity -and [string]::IsNullOrWhiteSpace($IdentityName)) { throw 'FILESPACE_STORE_IDENTITY_NAME is required for a Store release.' }
if ($RequirePartnerCenterIdentity -and [string]::IsNullOrWhiteSpace($Publisher)) { throw 'FILESPACE_STORE_PUBLISHER is required for a Store release.' }
if ([string]::IsNullOrWhiteSpace($IdentityName)) { $IdentityName = 'Filespace233' }
if ([string]::IsNullOrWhiteSpace($Publisher)) { $Publisher = 'CN=Filespace233' }
if ([string]::IsNullOrWhiteSpace($Version)) { $Version = '1.0.0.0' }

if ($IdentityName -notmatch '^[-.A-Za-z0-9]+$') { throw "IdentityName must match the Microsoft package identity pattern." }
if ($Publisher -notmatch '^CN=[^,]+') { throw "Publisher must be the exact Partner Center publisher string." }
if ($Version -notmatch '^\d+\.\d+\.\d+\.\d+$') { throw "Version must contain four numeric components." }

[xml]$manifest = Get-Content -Raw -LiteralPath $manifestPath
$identity = $manifest.Package.Identity
$identity.Name = $IdentityName
$identity.Publisher = $Publisher
$identity.Version = $Version
$manifest.Save((Resolve-Path -LiteralPath $manifestPath))

Write-Output "Prepared Package.appxmanifest for $IdentityName $Version."
