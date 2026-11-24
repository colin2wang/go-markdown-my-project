#Requires -Version 5.1
<#
.SYNOPSIS
    Generate application icons for Wails v3 from frontend/public/icon.svg.

.DESCRIPTION
    Two-step pipeline:
      1. tools/genicon (self-made) rasterizes the SVG into build/appicon.png.
      2. `wails3 generate icons` converts that PNG into the platform icons
         (build/windows/icon.ico, build/darwin/icon.icns), and
         `wails3 generate syso` embeds icon + manifest + version info into the
         Windows resource file (explicit in v3; `wails build` did this implicitly).

.PARAMETER Svg
    Input SVG file. Default: frontend\public\icon.svg

.EXAMPLE
    .\generate-icon.ps1
#>

[CmdletBinding()]
param(
    [string]$Svg = 'frontend\public\icon.svg'
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path

function Write-Step($msg) { Write-Host "[STEP] $msg" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "[ OK ] $msg" -ForegroundColor Green }
function Write-Fail($msg) { Write-Host "[FAIL] $msg" -ForegroundColor Red }

try {
    Push-Location $root

    foreach ($cmd in @('go', 'wails3')) {
        if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
            Write-Fail "Command not found: $cmd"
            exit 1
        }
    }

    $svgPath = Join-Path $root $Svg
    if (-not (Test-Path $svgPath)) {
        Write-Fail "SVG source not found: $svgPath"
        exit 1
    }

    # 1) SVG source -> appicon.png (genicon keeps only this responsibility)
    Write-Step "Convert $Svg -> build\appicon.png (go run ./tools/genicon)"
    go run ./tools/genicon -in $Svg -png build\appicon.png
    if ($LASTEXITCODE -ne 0) { Write-Fail 'PNG conversion failed'; exit 1 }

    # 2) v3 CLI: one PNG -> per-platform icons
    # (wails3 generate icons does not create the output directories itself)
    New-Item -ItemType Directory -Force -Path 'build\darwin' | Out-Null
    Write-Step 'Generate platform icons (wails3 generate icons)'
    wails3 generate icons -input build/appicon.png -windowsfilename build/windows/icon.ico -macfilename build/darwin/icon.icns
    if ($LASTEXITCODE -ne 0) { Write-Fail 'Icon generation failed'; exit 1 }

    # 3) Windows .syso (icon + manifest + version info from info.json)
    Write-Step 'Generate Windows syso (wails3 generate syso)'
    wails3 generate syso -manifest build/windows/wails.exe.manifest -icon build/windows/icon.ico -info build/windows/info.json -arch amd64 -out rsrc_windows_amd64.syso
    if ($LASTEXITCODE -ne 0) { Write-Fail 'syso generation failed'; exit 1 }

    Write-Step 'Generated files'
    foreach ($f in @('build\appicon.png', 'build\windows\icon.ico', 'build\darwin\icon.icns')) {
        $full = Join-Path $root $f
        if (Test-Path $full) {
            $len = (Get-Item $full).Length
            Write-Ok ("{0} ({1} bytes)" -f $f, $len)
        } else {
            Write-Warn "Missing output: $f"
        }
    }

    Write-Host "`nNote: run .\build.ps1 afterwards so the build picks up the new icons" -ForegroundColor DarkGray
} finally {
    Pop-Location
}
