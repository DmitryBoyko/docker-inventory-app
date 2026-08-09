<#
.SYNOPSIS
  Build Windows EXE with freshly built embedded UI.

.DESCRIPTION
  Always runs full UI sync (npm + embed) then Go build. No way to skip UI —
  prevents shipping a binary with a stale SPA.

.PARAMETER Version
  Version string stamped into the binary (default: "dev").

.PARAMETER OpenFolder
  Open bin\ in Explorer when done.

.EXAMPLE
  .\scripts\build-exe.ps1
  .\scripts\build-exe.ps1 -Version 1.0.0 -OpenFolder
#>
param(
  [string]$Version = "dev",
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
Assert-Command node

Write-Host "    $(go version)"
Write-Host "    node $(node -v)"

Write-Host "==> Full build (UI + EXE)..." -ForegroundColor Cyan
& "$PSScriptRoot\build.ps1" -Version $Version
if ($LASTEXITCODE -ne 0) { throw "build.ps1 failed with exit code $LASTEXITCODE" }

$exe = Join-Path $root "bin\docker-visualizer.exe"
if (-not (Test-Path $exe)) {
  throw "Expected output missing: $exe"
}

$distName = "docker-visualizer-windows-amd64.exe"
$distPath = Join-Path $root "bin\$distName"
Copy-Item -Force $exe $distPath

# Stamp so operators can see what UI went into the binary
$stamp = Join-Path $root "bin\BUILD_UI.txt"
$index = Join-Path $root "internal\uiembed\dist\index.html"
@(
  "builtAt=$(Get-Date -Format o)"
  "version=$Version"
  "exeSha256=$((Get-FileHash $exe -Algorithm SHA256).Hash)"
  "embedIndex=$index"
  "embedIndexTime=$((Get-Item $index).LastWriteTimeUtc.ToString('o'))"
) | Set-Content -Path $stamp -Encoding utf8

$hash = (Get-FileHash $exe -Algorithm SHA256).Hash
$sizeMB = [math]::Round((Get-Item $exe).Length / 1MB, 2)

Write-Host ""
Write-Host "OK  $exe ($sizeMB MB) — UI freshly embedded" -ForegroundColor Green
Write-Host "    SHA256: $hash"
Write-Host "    Also:   bin\$distName"
Write-Host "    Stamp:  bin\BUILD_UI.txt"
Write-Host ""
Write-Host "Run:" -ForegroundColor Yellow
Write-Host "  .\scripts\run-exe.ps1"
Write-Host "  # or"
Write-Host "  .\bin\docker-visualizer.exe"
Write-Host ""
Write-Host "Then open http://127.0.0.1:8080 (hard-refresh Ctrl+F5 after rebuild)"
Write-Host "Docker Desktop must be running (default named pipe)."

if ($OpenFolder) {
  Invoke-Item (Join-Path $root "bin")
}
