$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Push-Location (Join-Path $root "web")
try {
  npm ci
  npm run build
} finally {
  Pop-Location
}

$dest = Join-Path $root "internal\uiembed\dist"
if (Test-Path $dest) {
  Remove-Item -Recurse -Force $dest
}
New-Item -ItemType Directory -Path $dest | Out-Null
Copy-Item -Path (Join-Path $root "web\dist\*") -Destination $dest -Recurse -Force
Write-Host "Synced web/dist -> internal/uiembed/dist"
