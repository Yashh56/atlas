<#
.SYNOPSIS
    Builds the Atlas CLI for Windows with DWARF debug info stripped and version metadata linked.
    This prevents false positives from Windows Defender during local compiles.
#>

$ErrorActionPreference = "Stop"

$version = "dev"
try {
    $version = (git describe --tags --always --dirty 2>$null) -replace '^\s+|\s+$',''
    if (-not $version) { $version = "dev" }
} catch {
    $version = "dev"
}

Write-Host "-> Cleaning intermediate Go build cache (preventing Defender lockup)..." -ForegroundColor Cyan
go clean -cache

$ldflags = "-s -w -X github.com/Yashh56/atlas/internal/version.Version=$version"
Write-Host "-> Building atlas.exe (version: $version, trimmed path, DWARF stripped)..." -ForegroundColor Cyan
go build -trimpath -ldflags="$ldflags" -o atlas.exe ./cmd/atlas

if ($LastExitCode -eq 0) {
    $size = (Get-Item "atlas.exe").Length / 1MB
    Write-Host ("+ Successfully built atlas.exe ({0:N2} MB)" -f $size) -ForegroundColor Green
} else {
    Write-Host "- Build failed!" -ForegroundColor Red
    exit $LastExitCode
}
