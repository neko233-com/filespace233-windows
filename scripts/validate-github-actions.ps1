$ErrorActionPreference = 'Stop'
$actionlint = Get-Command actionlint -ErrorAction SilentlyContinue
if ($null -eq $actionlint) { throw 'actionlint is required. Install it before validating workflows.' }
& $actionlint.Source
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Output 'GitHub Actions validation passed.'
