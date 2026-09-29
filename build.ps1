<#
.SYNOPSIS
    Builds the Atlas CLI for Windows.

.DESCRIPTION
    Builds Atlas with:
    - Trimmed file paths
    - Stripped debug information
    - Version metadata embedded at build time

    The Go build cache is cleaned before compilation to help
    prevent Windows Defender from locking intermediate build files.
#>

$ErrorActionPreference = "Stop"

# ─────────────────────────────────────────────────────────────
# Atlas Build Script
# ─────────────────────────────────────────────────────────────

function Write-AtlasHeader {
    Write-Host ""
    Write-Host "  Atlas" -ForegroundColor Cyan -NoNewline
    Write-Host "  -  Windows build" -ForegroundColor DarkGray
    Write-Host ""
}

function Write-Step {
    param(
        [string]$Text
    )

    Write-Host "  " -NoNewline
    Write-Host "[>] " -NoNewline -ForegroundColor Cyan
    Write-Host $Text -ForegroundColor Gray
}

function Write-Status {
    param(
        [ValidateSet("Success", "Error", "Warning", "Info")]
        [string]$Type,

        [string]$Text
    )

    switch ($Type) {
        "Success" {
            $symbol = "[+]"
            $color = "Green"
        }

        "Error" {
            $symbol = "[X]"
            $color = "Red"
        }

        "Warning" {
            $symbol = "[!]"
            $color = "Yellow"
        }

        "Info" {
            $symbol = "[i]"
            $color = "Cyan"
        }
    }

    Write-Host "  " -NoNewline
    Write-Host "$symbol " -NoNewline -ForegroundColor $color
    Write-Host $Text -ForegroundColor Gray
}

function Write-InfoLine {
    param(
        [string]$Label,
        [string]$Value
    )

    Write-Host "  " -NoNewline
    Write-Host "$Label " -ForegroundColor DarkGray -NoNewline
    Write-Host $Value -ForegroundColor White
}

function Write-Failure {
    param(
        [string]$Message
    )

    Write-Host ""
    Write-Host "  +-- Build failed ------------------------------------" -ForegroundColor Red
    Write-Host "  |" -ForegroundColor Red
    Write-Host "  |  $Message" -ForegroundColor Gray
    Write-Host "  |" -ForegroundColor Red
    Write-Host "  +----------------------------------------------------" -ForegroundColor Red
    Write-Host ""

    exit 1
}

function Write-SuccessPanel {
    param(
        [string]$Version,
        [string]$Path,
        [double]$Size
    )

    Write-Host ""
    Write-Host "  +-- Build complete ----------------------------------" -ForegroundColor Green
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  |  Atlas " -ForegroundColor Green -NoNewline
    Write-Host "v$Version" -ForegroundColor White
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  |  Binary" -ForegroundColor DarkGray
    Write-Host "  |  $Path" -ForegroundColor White
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  |  Size" -ForegroundColor DarkGray
    Write-Host ("  |  {0:N2} MB" -f $Size) -ForegroundColor White
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  +----------------------------------------------------" -ForegroundColor Green
    Write-Host ""
}

# ─────────────────────────────────────────────────────────────
# Start
# ─────────────────────────────────────────────────────────────

Write-AtlasHeader

# ─────────────────────────────────────────────────────────────
# Determine version
# ─────────────────────────────────────────────────────────────

Write-Step "Determining build version..."

$version = "dev"

try {
    $version = (
        git describe --tags --always --dirty 2>$null
    ) -replace '^\s+|\s+$', ''

    if (-not $version) {
        $version = "dev"
    }
}
catch {
    $version = "dev"
}

Write-Status "Success" "Version resolved"

Write-Host ""
Write-InfoLine "Version" $version
Write-InfoLine "Target" "Windows / amd64"
Write-InfoLine "Output" "atlas.exe"
Write-Host ""

# ─────────────────────────────────────────────────────────────
# Clean build cache
# ─────────────────────────────────────────────────────────────

Write-Step "Cleaning Go build cache..."

try {
    go clean -cache

    if ($LASTEXITCODE -ne 0) {
        Write-Failure "Failed to clean the Go build cache."
    }
}
catch {
    Write-Failure "Failed to clean the Go build cache: $($_.Exception.Message)"
}

Write-Status "Success" "Build cache cleaned"

# ─────────────────────────────────────────────────────────────
# Build
# ─────────────────────────────────────────────────────────────

Write-Step "Building atlas.exe..."

$ldflags = "-s -w -X github.com/Yashh56/atlas/internal/version.Version=$version"

try {
    go build `
        -trimpath `
        -ldflags="$ldflags" `
        -o atlas.exe `
        ./cmd/atlas

    if ($LASTEXITCODE -ne 0) {
        Write-Failure "Go compilation failed."
    }
}
catch {
    Write-Failure "Build failed: $($_.Exception.Message)"
}

# ─────────────────────────────────────────────────────────────
# Verify output
# ─────────────────────────────────────────────────────────────

if (-not (Test-Path "atlas.exe")) {
    Write-Failure "Build completed without producing atlas.exe."
}

$size = (Get-Item "atlas.exe").Length / 1MB

Write-Status "Success" "Build completed"

# ─────────────────────────────────────────────────────────────
# Complete
# ─────────────────────────────────────────────────────────────

$binaryPath = (Resolve-Path "atlas.exe").Path

Write-SuccessPanel `
    -Version $version `
    -Path $binaryPath `
    -Size $size