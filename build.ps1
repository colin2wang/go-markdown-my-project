#Requires -Version 5.1
<#
.SYNOPSIS
    go-markdown-my-project one-click build script (Wails v3): check the toolchain,
    run `wails3 build` (Taskfile: bindings -> frontend -> go build) and copy the
    config files next to the binary in build/bin/.
.PARAMETER SkipInstall
    Skip frontend dependency install (use when dependencies already exist)
.PARAMETER Clean
    Clean the output directory before build
.EXAMPLE
    .\build.ps1
    .\build.ps1 -Clean
    .\build.ps1 -SkipInstall
#>

[CmdletBinding()]
param(
    [switch]$SkipInstall,
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
    Assert-Command 'go' 'Please install Go 1.26+: https://go.dev/dl/' | Out-Null
    Assert-Command 'pnpm' 'Please install pnpm: npm i -g pnpm' | Out-Null
    Write-Ok ('go   : ' + (go version))
    Write-Ok ('pnpm : ' + (pnpm -v))

    # Wails v3 CLI (independent from the v2 `wails` binary)
    if (-not (Get-Command wails3 -ErrorAction SilentlyContinue)) {
        Write-Warn 'wails3 CLI not detected, installing github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0 ...'
        go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
        Invoke-Checked 'Failed to install wails3 CLI'
        $goBin = & go env GOPATH
        $env:Path = "$goBin\bin;$env:Path"
    } else {
        wails3 version
    }

    Write-Step 'Clean output directory (build/bin)'
    $binDir = Join-Path $root 'build\bin'
    if ($Clean -and (Test-Path $binDir)) {
        Remove-Item -Recurse -Force $binDir
    }
    Write-Ok 'done'

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

    # Build stamp: local wall-clock time, never UTC. [DateTime]::Now uses the
    # machine's local time zone. The value must stay space-free: it travels
    # through PowerShell -> Taskfile -> go and nested quoting gets stripped,
    # which truncates the value at the space. app.FormatBuildVersion() turns the
    # underscore into a space for display as "(build YYYY-MM-DD HH:mm:ss)".
    $buildTime = [DateTime]::Now
    $invariant = [System.Globalization.CultureInfo]::InvariantCulture
    $buildStamp = $buildTime.ToString('yyyy-MM-dd_HH:mm:ss', $invariant)
    $buildLocal = $buildTime.ToString('yyyy-MM-dd HH:mm:ss', $invariant) + ' ' + $buildTime.ToString('zzz', $invariant) + ' (local)'
    Write-Ok "Build version: $buildStamp"
    Write-Ok "Build time   : $buildLocal"

    # `wails3 task` does not accept variable overrides on the command line, so
    # the stamp travels as an environment variable; the Taskfile reads it with
    # a "dev" fallback for direct `wails3 task build` runs.
    Write-Step 'Compile desktop application (wails3 task build)'
    $env:BUILD_TIME = $buildStamp
    wails3 task build
    Invoke-Checked 'Go backend compilation failed'

    # The .syso Windows resource file has served its purpose once the exe is
    # linked (go build embeds it automatically); remove it so it doesn't linger.
    Remove-Item -Path (Join-Path $root 'rsrc_windows_*.syso') -Force -ErrorAction SilentlyContinue

    # Copy sample project configs + language mapping next to the output executable
    # (GUI reads config/langs.yml and config/projects relative to the exe's working directory)
    Write-Step 'Copy configs to build/bin/'
    if (-not (Test-Path (Join-Path $root "build\bin\$exeName"))) {
        Write-Fail "wails3 build output not found: build\bin\$exeName"
        exit 1
    }
    # Copy-Item -Recurse pitfall: if the destination dir already exists, the whole source
    # dir gets nested inside it (config/config). Delete the old dir first so the layout
    # is always build/bin/config/*
    $configDest = Join-Path $root 'build\bin\config'
    if (Test-Path $configDest) { Remove-Item $configDest -Recurse -Force }
    Copy-Item (Join-Path $root 'config') $configDest -Recurse -Force
    Write-Ok 'config/ -> bin'

    Write-Step 'Build completed'
    Get-ChildItem -Path $binDir -Recurse -File |
        Select-Object @{n = 'File'; e = { $_.FullName.Substring($binDir.Length + 1) } },
                      @{n = 'SizeMB'; e = { [math]::Round($_.Length / 1MB, 2) } } |
        Format-Table -AutoSize
    Write-Host "`nRun dev debug mode: wails3 dev" -ForegroundColor DarkGray
} finally {
    Pop-Location
}
