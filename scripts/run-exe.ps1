<#
.SYNOPSIS
  Run the Windows EXE (full rebuild if missing or -Build).

.DESCRIPTION
  Rebuilds always include npm UI + embed. There is no SkipUI flag.

.PARAMETER Listen
  Bind address (default 127.0.0.1:8080).

.PARAMETER AuthToken
  Bearer token; required if Listen is not loopback.

.PARAMETER DockerHost
  Override Docker Engine endpoint (e.g. npipe:////./pipe/docker_engine).

.PARAMETER Build
  Force full rebuild (UI + EXE) before run.

.PARAMETER OpenBrowser
  Open the UI in the default browser after start.

.EXAMPLE
  .\scripts\run-exe.ps1
  .\scripts\run-exe.ps1 -Build -OpenBrowser
#>
param(
  [string]$Listen = "127.0.0.1:8080",
  [string]$AuthToken = "",
  [string]$DockerHost = "",
  [switch]$Build,
  [switch]$OpenBrowser
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$exe = Join-Path $root "bin\docker-visualizer.exe"

if ($Build -or -not (Test-Path $exe)) {
  Write-Host "==> Full rebuild (UI + EXE)..." -ForegroundColor Cyan
  & "$PSScriptRoot\build-exe.ps1"
}

if (-not (Test-Path $exe)) {
  throw "Binary not found: $exe"
}

$exeArgs = @("--listen", $Listen)
if ($AuthToken) {
  $exeArgs += @("--auth-token", $AuthToken)
}
if ($DockerHost) {
  $exeArgs += @("--docker-host", $DockerHost)
}

$url = if ($Listen -match "^(127\.0\.0\.1|localhost|\[::1\]):(\d+)$") {
  "http://127.0.0.1:$($Matches[2])"
} elseif ($Listen -match ":(\d+)$") {
  "http://127.0.0.1:$($Matches[1])"
} else {
  "http://127.0.0.1:8080"
}

Write-Host "==> Starting $exe" -ForegroundColor Cyan
Write-Host "    $($exeArgs -join ' ')"
Write-Host "    UI: $url  (Ctrl+C to stop)"
Write-Host ""

if ($OpenBrowser) {
  Start-Process $url
}

& $exe @exeArgs
