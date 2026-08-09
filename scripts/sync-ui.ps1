$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Push-Location (Join-Path $root "web")
try {
  npm ci
  if ($LASTEXITCODE -ne 0) { throw "npm ci failed ($LASTEXITCODE)" }
  npm run build
  if ($LASTEXITCODE -ne 0) { throw "npm run build failed ($LASTEXITCODE)" }
} finally {
  Pop-Location
}

$webIndex = Join-Path $root "web\dist\index.html"
if (-not (Test-Path $webIndex)) {
  throw "Vite build did not produce web\dist\index.html"
}

$dest = Join-Path $root "internal\uiembed\dist"
if (Test-Path $dest) {
  Remove-Item -Recurse -Force $dest
}
New-Item -ItemType Directory -Path $dest | Out-Null
Copy-Item -Path (Join-Path $root "web\dist\*") -Destination $dest -Recurse -Force

$embedIndex = Join-Path $dest "index.html"
if (-not (Test-Path $embedIndex)) {
  throw "Failed to sync UI into internal\uiembed\dist"
}

Write-Host "Synced web/dist -> internal/uiembed/dist ($(Get-Date -Format o))"
