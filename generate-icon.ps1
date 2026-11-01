#Requires -Version 5.1
<#
.SYNOPSIS
    Convert assets/icon.svg into the application icons (PNG + ICO) via tools/genicon.

.DESCRIPTION
    The SVG is the single source of truth for the app icon. This script runs the
    Go converter (tools/genicon) which rasterizes a restricted SVG subset with the
    standard library only, then writes:
      - build/appicon.png        (used by `wails build` for the app bundle icon)
      - build/windows/icon.ico   (used by the Windows executable / taskbar)

.PARAMETER Svg
    Input SVG file. Default: assets\icon.svg
.PARAMETER Size
    PNG edge length in pixels. Default: 512
.PARAMETER Sizes
    Comma separated ICO entry sizes. Default: 16,24,32,48,64,128,256
.PARAMETER PngOnly
    Only generate the PNG.
.PARAMETER IcoOnly
    Only generate the ICO.

.EXAMPLE
    .\generate-icon.ps1
    .\generate-icon.ps1 -Size 256
    .\generate-icon.ps1 -IcoOnly
#>

[CmdletBinding()]
param(
    [string]$Svg = 'frontend\public\icon.svg',
    [int]$Size = 512,
    [string]$Sizes = '16,24,32,48,64,128,256',
    [switch]$PngOnly,
    [switch]$IcoOnly
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path

function Write-Step($msg) { Write-Host "[STEP] $msg" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "[ OK ] $msg" -ForegroundColor Green }
function Write-Fail($msg) { Write-Host "[FAIL] $msg" -ForegroundColor Red }

try {
    Push-Location $root

    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Fail 'Command not found: go. Please install Go 1.21+: https://go.dev/dl/'
        exit 1
    }

    $svgPath = Join-Path $root $Svg
    if (-not (Test-Path $svgPath)) {
        Write-Fail "SVG source not found: $svgPath"
        exit 1
    }

    $pngOut = ''
    $icoOut = ''
    if (-not $IcoOnly) { $pngOut = 'build\appicon.png' }
    if (-not $PngOnly) { $icoOut = 'build\windows\icon.ico' }

    Write-Step "Convert $Svg -> png/ico (go run ./tools/genicon)"
    & go run ./tools/genicon -in $Svg -png $pngOut -ico $icoOut -size $Size -sizes $Sizes
    if ($LASTEXITCODE -ne 0) {
        Write-Fail 'Icon conversion failed'
        exit 1
    }

    Write-Step 'Generated files'
    foreach ($f in @($pngOut, $icoOut)) {
        if ([string]::IsNullOrWhiteSpace($f)) { continue }
        $full = Join-Path $root $f
        if (Test-Path $full) {
            $len = (Get-Item $full).Length
            Write-Ok ("{0} ({1} bytes)" -f $f, $len)
        } else {
            Write-Fail "Missing output: $f"
        }
    }

    Write-Host "`nNote: run .\build.ps1 afterwards so wails picks up the new build/appicon.png" -ForegroundColor DarkGray
} finally {
    Pop-Location
}
