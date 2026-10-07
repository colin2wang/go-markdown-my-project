# build.ps1 - one-click build (Go backend + pnpm frontend)
# Output goes to dist/, including the executable and config files (assets, config).
# Usage:  .\build.ps1              # normal build
#         .\build.ps1 -Clean       # wipe dist first
#         .\build.ps1 -SkipFrontend  # skip frontend build (when frontend/dist exists)
param(
    [switch]$Clean,
    [switch]$SkipFrontend
)

$ErrorActionPreference = "Stop"
$Root = $PSScriptRoot
$Dist = Join-Path $Root "dist"

Write-Host "==> Project Docs GUI build" -ForegroundColor Cyan

# 0) check tools
foreach ($tool in @("go", "node", "pnpm")) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        Write-Error "tool not found: $tool (install it and add to PATH)"
    }
}

# 1) clean
if ($Clean -and (Test-Path $Dist)) {
    Write-Host "==> cleaning dist/"
    Remove-Item $Dist -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $Dist | Out-Null

# 2) frontend: pnpm install + build (output at frontend/dist, embedded by Wails)
if (-not $SkipFrontend) {
    Write-Host "==> frontend build (pnpm)" -ForegroundColor Cyan
    Push-Location (Join-Path $Root "frontend")
    try {
        pnpm install --frozen-lockfile
        if ($LASTEXITCODE -ne 0) { Write-Error "pnpm install failed" }
        pnpm run build
        if ($LASTEXITCODE -ne 0) { Write-Error "frontend build failed" }
    } finally {
        Pop-Location
    }
} else {
    Write-Host "==> skipping frontend build"
}

# 3) backend: wails build (fallback: plain go build), copy binary to dist/
Write-Host "==> backend build (wails)" -ForegroundColor Cyan
$exeName = "project-docs-gui.exe"
$wails = Get-Command wails -ErrorAction SilentlyContinue
if ($wails) {
    Push-Location $Root
    try {
        wails build
        if ($LASTEXITCODE -ne 0) { Write-Error "wails build failed" }
    } finally {
        Pop-Location
    }
    Copy-Item (Join-Path $Root "build\bin\$exeName") $Dist -Force
} else {
    Write-Host "    wails CLI not found, fallback to go build (no desktop shell)" -ForegroundColor Yellow
    Push-Location $Root
    try {
        go build -o (Join-Path $Dist $exeName) .
        if ($LASTEXITCODE -ne 0) { Write-Error "go build failed" }
    } finally {
        Pop-Location
    }
}

# 4) CLI headless build
Write-Host "==> CLI build" -ForegroundColor Cyan
Push-Location $Root
try {
    go build -o (Join-Path $Dist "project-docs.exe") ./cmd/project-docs
    if ($LASTEXITCODE -ne 0) { Write-Error "CLI build failed" }
} finally {
    Pop-Location
}

# 5) config files: copy config/ wholesale to dist/config (projects + global yml)
Write-Host "==> copying config files to dist/" -ForegroundColor Cyan
Copy-Item (Join-Path $Root "assets") (Join-Path $Dist "assets") -Recurse -Force
Copy-Item (Join-Path $Root "config") (Join-Path $Dist "config") -Recurse -Force

# 6) summary
Write-Host ""
Write-Host "==> build finished, dist/ contents:" -ForegroundColor Green
Get-ChildItem $Dist -Recurse -File | ForEach-Object {
    $rel = $_.FullName.Substring($Dist.Length + 1)
    Write-Host ("    {0}  ({1:N0} bytes)" -f $rel, $_.Length)
}
