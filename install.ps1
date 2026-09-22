$ErrorActionPreference = "Stop"
$Repo = "Yashh56/atlas"

Write-Host "Installing Atlas..." -ForegroundColor Cyan

# Fetch latest release from GitHub API
$releaseUrl = "https://api.github.com/repos/$Repo/releases/latest"
try {
    $release = Invoke-RestMethod -Uri $releaseUrl -UseBasicParsing
    $version = $release.tag_name.TrimStart('v')
} catch {
    Write-Host "Failed to fetch the latest version. Please check your internet connection." -ForegroundColor Red
    exit 1
}

# Determine architecture
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "x86_64" }
$fileName = "atlas_${version}_Windows_${arch}.zip"
$downloadUrl = "https://github.com/$Repo/releases/download/v${version}/$fileName"

Write-Host "Downloading v$version ($arch)..."
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

Write-Host "Atlas (v${version}) was successfully installed!" -ForegroundColor Green
Write-Host "Restart your terminal or run 'atlas --help' to get started."
