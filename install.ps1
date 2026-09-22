$ErrorActionPreference = "Stop"
$Repo = "Yashh56/atlas"

Write-Host "Installing Atlas..." -ForegroundColor Cyan

# Fetch latest release from GitHub API
$releaseUrl = "https://api.github.com/repos/$Repo/releases/latest"
try {
    $release = Invoke-RestMethod -Uri $releaseUrl
    $version = $release.tag_name.TrimStart('v')
} catch {
    Write-Host "Failed to fetch the latest version. Please check your internet connection." -ForegroundColor Red
    exit 1
}

# Determine architecture
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "x86_64" }

# Find the matching asset from the release (handles any naming convention)
$matchPattern = "Windows.*${arch}.*\.zip$"
$asset = $release.assets | Where-Object { $_.name -match $matchPattern } | Select-Object -First 1

if (-not $asset) {
    Write-Host "Could not find a Windows $arch release asset for v$version." -ForegroundColor Red
    Write-Host "Available assets:" -ForegroundColor Yellow
    $release.assets | ForEach-Object { Write-Host "  - $($_.name)" }
    exit 1
}

$fileName = $asset.name
$downloadUrl = $asset.browser_download_url

Write-Host "Downloading v$version ($arch)..."
Write-Host "  Asset: $fileName"
$tempZip = Join-Path $env:TEMP $fileName

Invoke-WebRequest -Uri $downloadUrl -OutFile $tempZip -UseBasicParsing

Write-Host "Extracting..."
$installDir = Join-Path $env:LOCALAPPDATA "atlas\bin"
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Expand-Archive -Path $tempZip -DestinationPath $installDir -Force
Remove-Item $tempZip

# Add to PATH if not already present
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$installDir*") {
    Write-Host "Adding $installDir to your PATH..."
    [Environment]::SetEnvironmentVariable("PATH", "$userPath;$installDir", "User")
    $env:PATH = "$env:PATH;$installDir"
}

Write-Host ""
Write-Host "Atlas (v${version}) was successfully installed!" -ForegroundColor Green
Write-Host "Restart your terminal, then run 'atlas --help' to get started."
