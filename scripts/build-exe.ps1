<#
.SYNOPSIS
  Build Windows EXE with embedded UI (Docker Visualizer).

.DESCRIPTION
  Requires Go and Node.js on PATH. Syncs web UI into the embed package, then
  produces bin\docker-visualizer.exe (and optionally a named amd64 artifact).

.PARAMETER Version
  Version string stamped into the binary (default: "dev").

.PARAMETER SkipUI
  Skip npm build / embed sync (reuse last synced UI).

.PARAMETER OpenFolder
  Open bin\ in Explorer when done.

.EXAMPLE
  .\scripts\build-exe.ps1
  .\scripts\build-exe.ps1 -Version 1.0.0 -OpenFolder
#>
param(
  [string]$Version = "dev",
  [switch]$SkipUI,
  [switch]$OpenFolder
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

function Assert-Command([string]$Name) {
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Required tool not found on PATH: $Name"
  }
}

Write-Host "==> Checking tools..." -ForegroundColor Cyan
Assert-Command go
Assert-Command npm
if (-not $SkipUI) {
  Assert-Command node
}

$goVer = (go version)
Write-Host "    $goVer"
Write-Host "    node $((node -v 2>$null))"

Write-Host "==> Building Windows EXE..." -ForegroundColor Cyan
& "$PSScriptRoot\build.ps1" -Version $Version -SkipUI:$SkipUI
if ($LASTEXITCODE -ne 0) { throw "build.ps1 failed with exit code $LASTEXITCODE" }

$exe = Join-Path $root "bin\docker-visualizer.exe"
if (-not (Test-Path $exe)) {
  throw "Expected output missing: $exe"
}

# Stable named copy for distribution
$distName = "docker-visualizer-windows-amd64.exe"
$distPath = Join-Path $root "bin\$distName"
Copy-Item -Force $exe $distPath

$hash = (Get-FileHash $exe -Algorithm SHA256).Hash
$sizeMB = [math]::Round((Get-Item $exe).Length / 1MB, 2)

Write-Host ""
Write-Host "OK  $exe ($sizeMB MB)" -ForegroundColor Green
Write-Host "    SHA256: $hash"
Write-Host "    Also:   bin\$distName"
Write-Host ""
Write-Host "Run:" -ForegroundColor Yellow
Write-Host "  .\scripts\run-exe.ps1"
Write-Host "  # or"
Write-Host "  .\bin\docker-visualizer.exe"
Write-Host ""
Write-Host "Then open http://127.0.0.1:8080"
Write-Host "Docker Desktop must be running (default named pipe)."
Write-Host "Help: .\bin\docker-visualizer.exe -h"

if ($OpenFolder) {
  Invoke-Item (Join-Path $root "bin")
}
