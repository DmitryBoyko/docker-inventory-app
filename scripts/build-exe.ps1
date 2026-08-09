<#
.SYNOPSIS
  Build Windows EXE with freshly built embedded UI.

.DESCRIPTION
  Always bumps SemVer patch, runs full UI sync (npm + embed), then Go build.

.PARAMETER Version
  Optional exact SemVer (writes VERSION; skips auto bump).

.PARAMETER NoBump
  Keep current VERSION (still refreshes generated UI constant).

.PARAMETER OpenFolder
  Open bin\ in Explorer when done.

.EXAMPLE
  .\scripts\build-exe.ps1
  .\scripts\build-exe.ps1 -Version 1.0.0 -OpenFolder
#>
param(
  [string]$Version = "",
  [switch]$NoBump,
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

Write-Host "==> Full build (bump + UI + EXE)..." -ForegroundColor Cyan
$buildArgs = @{}
if ($Version) { $buildArgs.Version = $Version }
if ($NoBump) { $buildArgs.NoBump = $true }
& "$PSScriptRoot\build.ps1" @buildArgs
if ($LASTEXITCODE -ne 0) { throw "build.ps1 failed with exit code $LASTEXITCODE" }

$ver = (Get-Content (Join-Path $root "VERSION") -Raw).Trim()
$exe = Join-Path $root "bin\docker-visualizer.exe"
if (-not (Test-Path $exe)) {
  throw "Expected output missing: $exe"
}

$distName = "docker-visualizer-windows-amd64.exe"
$distPath = Join-Path $root "bin\$distName"
Copy-Item -Force $exe $distPath

$stamp = Join-Path $root "bin\BUILD_UI.txt"
$index = Join-Path $root "internal\uiembed\dist\index.html"
@(
  "builtAt=$(Get-Date -Format o)"
  "version=$ver"
  "exeSha256=$((Get-FileHash $exe -Algorithm SHA256).Hash)"
  "embedIndex=$index"
  "embedIndexTime=$((Get-Item $index).LastWriteTimeUtc.ToString('o'))"
) | Set-Content -Path $stamp -Encoding utf8

$hash = (Get-FileHash $exe -Algorithm SHA256).Hash
$sizeMB = [math]::Round((Get-Item $exe).Length / 1MB, 2)

Write-Host ""
Write-Host "OK  $exe v$ver ($sizeMB MB) — UI freshly embedded" -ForegroundColor Green
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

if ($OpenFolder) {
  Invoke-Item (Join-Path $root "bin")
}
