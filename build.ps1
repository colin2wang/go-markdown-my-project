#Requires -Version 5.1
<#
.SYNOPSIS
    project-docs-gui one-click build script: install frontend deps (pnpm), let
    `wails build` regenerate the Go<->frontend bindings, build the frontend and compile the
    Go backend, then copy the CLI and config files next to the binary in build/bin/.
.PARAMETER SkipInstall
    Skip pnpm install (use when dependencies already exist)
.PARAMETER SkipFrontend
    Skip the frontend build performed inside `wails build` (adds -s, reuses existing frontend/dist)
.PARAMETER Package
    Additionally generate Windows installer using NSIS (NSIS must be installed locally)
.PARAMETER Clean
    Clean build/bin directory before build
.EXAMPLE
    .\build.ps1
    .\build.ps1 -Clean
    .\build.ps1 -SkipInstall -SkipFrontend
#>

[CmdletBinding()]
param(
    [switch]$SkipInstall,
    [switch]$SkipFrontend,
    [switch]$Package,
    [switch]$Clean
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$exeName = 'project-docs-gui.exe'

# Print step info with cyan color
function Write-Step($msg) { Write-Host "`n[STEP] $msg" -ForegroundColor Cyan }
# Print success message with green color
function Write-Ok($msg)   { Write-Host "[ OK ] $msg" -ForegroundColor Green }
# Print warning message with yellow color
function Write-Warn($msg) { Write-Host "[WARN] $msg" -ForegroundColor Yellow }
# Print failure message with red color
function Write-Fail($msg) { Write-Host "[FAIL] $msg" -ForegroundColor Red }

# Check if required command exists; exit with hint if missing
function Assert-Command($cmd, $hint) {
    $found = Get-Command $cmd -ErrorAction SilentlyContinue
    if (-not $found) {
        Write-Fail "Command not found: $cmd. $hint"
        exit 1
    }
    return $found
}

# Run an external command, fail with a clear message on non-zero exit code
function Invoke-Checked($failMsg) {
    if ($LASTEXITCODE -ne 0) { Write-Fail $failMsg; exit 1 }
}

try {
    Push-Location $root

    Write-Step 'Check build environment'
    Assert-Command 'go' 'Please install Go 1.21+: https://go.dev/dl/' | Out-Null
    Assert-Command 'pnpm' 'Please install pnpm: npm i -g pnpm' | Out-Null
    Write-Ok ('go   : ' + (go version))
    Write-Ok ('pnpm : ' + (pnpm -v))

    # Wails CLI: responsible for frontend-backend binding generation and app packaging
    if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
        Write-Warn 'wails CLI not detected, installing github.com/wailsapp/wails/v2/cmd/wails@latest ...'
        go install github.com/wailsapp/wails/v2/cmd/wails@latest
        Invoke-Checked 'Failed to install wails CLI'
        $goBin = & go env GOPATH
        $env:Path = "$goBin\bin;$env:Path"
    } else {
        wails version
    }

    # Clean wails output folder if -Clean switch is set
    $binDir = Join-Path $root 'build\bin'
    if ($Clean -and (Test-Path $binDir)) {
        Write-Step 'Clean build/bin directory'
        Remove-Item -Recurse -Force $binDir
        Write-Ok 'build/bin directory cleaned'
    }

    # Install frontend dependencies unless skipped
    if (-not $SkipInstall) {
        Write-Step 'Install frontend dependencies (pnpm install)'
        Push-Location (Join-Path $root 'frontend')
        try {
            # pnpm disables dependency build scripts by default; esbuild needs postinstall.
            # Project policy (pnpm 12): the build-script whitelist lives in
            # frontend/pnpm-workspace.yaml (allowBuilds), so a plain install is preferred.
            pnpm install --frozen-lockfile
            if ($LASTEXITCODE -ne 0) {
                Write-Warn 'Frozen-lockfile install failed, fallback to plain install'
                pnpm install
                Invoke-Checked 'pnpm install failed'
            }
        } finally { Pop-Location }
        Write-Ok 'Frontend dependencies ready'
    }

    Write-Step 'Compile desktop application (wails build)'
    # `wails build` regenerates the Go<->frontend bindings, then builds the frontend,
    # so the embedded dist reflects the latest backend methods.
    # -SkipFrontend reuses an already-built dist (skips the internal frontend build).
    $buildArgs = @('build', '-ldflags', '-s -w')
    if ($SkipFrontend) { $buildArgs += '-s' }
    if ($Package) {
        if (Get-Command makensis -ErrorAction SilentlyContinue) {
            Write-Ok 'NSIS detected, installer will be generated'
            $buildArgs += '-nsis'
        } else {
            Write-Warn 'NSIS(makensis) not detected, only executable will be built. Install NSIS and add -Package to generate installer.'
        }
    }
    wails @buildArgs
    Invoke-Checked 'Go backend compilation failed'

    # Copy assets/ and sample project configs next to the wails output executable
    # (GUI reads assets/langs.yml and config/projects relative to the exe's working directory)
    Write-Step 'Copy assets and sample configs to build/bin/'
    if (-not (Test-Path (Join-Path $root "build\bin\$exeName"))) {
        Write-Fail "wails build output not found: build\bin\$exeName"
        exit 1
    }
    Copy-Item (Join-Path $root 'assets') (Join-Path $root 'build\bin\assets') -Recurse -Force
    Copy-Item (Join-Path $root 'config') (Join-Path $root 'build\bin\config') -Recurse -Force
    Write-Ok 'assets/, config/ -> build/bin'

    Write-Step 'Build completed'
    Get-ChildItem -Path $binDir -Recurse -File |
        Select-Object @{n = 'File'; e = { $_.FullName.Substring($binDir.Length + 1) } },
                      @{n = 'SizeMB'; e = { [math]::Round($_.Length / 1MB, 2) } } |
        Format-Table -AutoSize
    Write-Host "`nRun dev debug mode: wails dev" -ForegroundColor DarkGray
} finally {
    Pop-Location
}
