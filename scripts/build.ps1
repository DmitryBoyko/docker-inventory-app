param(
  [string]$Version = "dev",
  [switch]$Cross,
  [switch]$SkipUI
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$commit = "none"
try { $commit = (git rev-parse --short HEAD 2>$null) } catch {}
if (-not $commit) { $commit = "none" }

$ldflags = "-s -w -X main.version=$Version -X main.commit=$commit"

if (-not $SkipUI) {
  & "$PSScriptRoot\sync-ui.ps1"
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
  Write-Host "Built bin\$exe"
}
