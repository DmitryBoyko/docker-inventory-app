<#
.SYNOPSIS
  Run the Windows EXE (build it first if missing).

.PARAMETER Listen
  Bind address (default 127.0.0.1:8080).

.PARAMETER AuthToken
  Bearer token; required if Listen is not loopback.

.PARAMETER DockerHost
  Override Docker Engine endpoint (e.g. npipe:////./pipe/docker_engine).

.PARAMETER Build
  Force rebuild with build-exe.ps1 before run.

.PARAMETER SkipUI
  When -Build, skip npm UI sync.

.PARAMETER OpenBrowser
  Open the UI in the default browser after start.

.EXAMPLE
  .\scripts\run-exe.ps1
  .\scripts\run-exe.ps1 -OpenBrowser
  .\scripts\run-exe.ps1 -Listen 0.0.0.0:8080 -AuthToken "secret"
#>
param(
  [string]$Listen = "127.0.0.1:8080",
  [string]$AuthToken = "",
  [string]$DockerHost = "",
  [switch]$Build,
  [switch]$SkipUI,
  [switch]$OpenBrowser
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$exe = Join-Path $root "bin\docker-visualizer.exe"

if ($Build -or -not (Test-Path $exe)) {
  Write-Host "==> EXE not found or -Build set; building..." -ForegroundColor Cyan
  & "$PSScriptRoot\build-exe.ps1" -SkipUI:$SkipUI
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
