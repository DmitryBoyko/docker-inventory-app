<#
.SYNOPSIS
  Build Docker Visualizer binary (always rebuilds and embeds the UI).

.DESCRIPTION
  Every invocation bumps SemVer patch (VERSION), syncs npm UI into embed, then
  compiles the Go binary with -X main.version=<semver>.

.PARAMETER Version
  Optional explicit SemVer (skips auto bump and writes VERSION to this value).

.PARAMETER NoBump
  Keep VERSION as-is (no patch increment).

.PARAMETER Cross
  Also build linux/darwin amd64+arm64 artifacts under bin\.
#>
param(
  [string]$Version = "",
  [switch]$NoBump,
  [switch]$Cross
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if ($Version) {
  $ver = (& "$PSScriptRoot\bump-version.ps1" -Set $Version | Select-Object -Last 1).ToString().Trim()
} elseif ($NoBump) {
  $ver = (Get-Content (Join-Path $root "VERSION") -Raw).Trim()
  & "$PSScriptRoot\bump-version.ps1" -Set $ver | Out-Null
} else {
  $ver = (& "$PSScriptRoot\bump-version.ps1" | Select-Object -Last 1).ToString().Trim()
}

$commit = "none"
try { $commit = (git rev-parse --short HEAD 2>$null) } catch {}
if (-not $commit) { $commit = "none" }

$ldflags = "-s -w -X main.version=$ver -X main.commit=$commit"
Write-Host "==> Building version $ver (commit $commit)" -ForegroundColor Cyan

Write-Host "==> Sync UI (npm build → embed)..." -ForegroundColor Cyan
& "$PSScriptRoot\sync-ui.ps1"
if ($LASTEXITCODE -ne 0) { throw "sync-ui.ps1 failed with exit code $LASTEXITCODE" }

$embedIndex = Join-Path $root "internal\uiembed\dist\index.html"
if (-not (Test-Path $embedIndex)) {
  throw "UI embed missing after sync: $embedIndex"
}

New-Item -ItemType Directory -Force -Path (Join-Path $root "bin") | Out-Null

function Build-One($goos, $goarch, $out) {
  $env:CGO_ENABLED = "0"
  $env:GOOS = $goos
  $env:GOARCH = $goarch
  Write-Host "Building $out"
  go build -trimpath -ldflags $ldflags -o (Join-Path $root "bin\$out") ./cmd/docker-visualizer
}

if ($Cross) {
  Build-One windows amd64 "docker-visualizer-windows-amd64.exe"
  Build-One linux amd64 "docker-visualizer-linux-amd64"
  Build-One linux arm64 "docker-visualizer-linux-arm64"
  Build-One darwin amd64 "docker-visualizer-darwin-amd64"
  Build-One darwin arm64 "docker-visualizer-darwin-arm64"
  Get-ChildItem (Join-Path $root "bin\docker-visualizer-*") | ForEach-Object {
    Get-FileHash $_.FullName -Algorithm SHA256
  } | Format-Table -AutoSize
} else {
  $exe = if ($IsWindows -or $env:OS -match "Windows") { "docker-visualizer.exe" } else { "docker-visualizer" }
  Remove-Item Env:GOOS -ErrorAction SilentlyContinue
  Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
  $env:CGO_ENABLED = "0"
  go build -trimpath -ldflags $ldflags -o (Join-Path $root "bin\$exe") ./cmd/docker-visualizer
  Write-Host "Built bin\$exe v$ver (UI embedded)"
}
