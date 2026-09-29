$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$Repo = "Yashh56/atlas"
$ApiUrl = "https://api.github.com/repos/$Repo/releases/latest"

# -------------------------------------------------------------
# Atlas Installer
# -------------------------------------------------------------

function Write-AtlasHeader {
    Write-Host ""
    Write-Host "  Atlas" -ForegroundColor Cyan -NoNewline
    Write-Host "  -  Autonomous deployment pipeline" -ForegroundColor DarkGray
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
    Write-Host "  +-- Installation failed -----------------------------" -ForegroundColor Red
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
        [string]$InstallPath
    )

    Write-Host ""
    Write-Host "  +-- Installation complete ---------------------------" -ForegroundColor Green
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  |  Atlas " -ForegroundColor Green -NoNewline
    Write-Host "v$Version" -ForegroundColor White
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  |  Installed to" -ForegroundColor DarkGray
    Write-Host "  |  $InstallPath" -ForegroundColor White
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  |  Restart your terminal, then run:" -ForegroundColor DarkGray
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  |  atlas --help" -ForegroundColor Cyan
    Write-Host "  |" -ForegroundColor Green
    Write-Host "  +----------------------------------------------------" -ForegroundColor Green
    Write-Host ""
}

# -------------------------------------------------------------
# Start
# -------------------------------------------------------------

Write-AtlasHeader

# -------------------------------------------------------------
# Detect architecture
# -------------------------------------------------------------

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    "arm64"
} else {
    "x86_64"
}

# -------------------------------------------------------------
# Fetch latest release
# -------------------------------------------------------------

Write-Step "Checking latest release..."

try {
    $release = Invoke-RestMethod `
        -Uri $ApiUrl `
        -Method Get `
        -UseBasicParsing

    $version = $release.tag_name.TrimStart("v")
} catch {
    Write-Failure "Unable to fetch the latest Atlas release: $($_.Exception.Message)"
}

$matchPattern = "Windows.*${arch}.*\.zip$"

$asset = $release.assets |
    Where-Object {
        $_.name -match $matchPattern
    } |
    Select-Object -First 1

if (-not $asset) {
    Write-Failure "No Windows $arch release asset was found for v$version."
}

Write-Status "Success" "Latest release found"

Write-Host ""
Write-InfoLine "Version" "v$version"
Write-InfoLine "Platform" "Windows / $arch"
Write-Host ""

# -------------------------------------------------------------
# Download
# -------------------------------------------------------------

$fileName = $asset.name
$downloadUrl = $asset.browser_download_url
$tempZip = Join-Path $env:TEMP $fileName

Write-Step "Downloading Atlas v$version..."

try {
    Invoke-WebRequest `
        -Uri $downloadUrl `
        -OutFile $tempZip `
        -UseBasicParsing
} catch {
    Write-Failure "Download failed: $($_.Exception.Message)"
}

if (-not (Test-Path $tempZip)) {
    Write-Failure "Download completed without creating the expected archive."
}

Write-Status "Success" "Download complete"

# -------------------------------------------------------------
# Install
# -------------------------------------------------------------

$installDir = Join-Path $env:LOCALAPPDATA "atlas\bin"

Write-Step "Installing Atlas..."

try {
    if (-not (Test-Path $installDir)) {
        New-Item `
            -ItemType Directory `
            -Path $installDir `
            -Force | Out-Null
    }

    Expand-Archive `
        -Path $tempZip `
        -DestinationPath $installDir `
        -Force

    Remove-Item `
        -Path $tempZip `
        -Force
} catch {
    Write-Failure "Could not install Atlas: $($_.Exception.Message)"
}

Write-Status "Success" "Atlas installed"

# -------------------------------------------------------------
# Update User PATH
# -------------------------------------------------------------

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")

if ([string]::IsNullOrWhiteSpace($userPath)) {
    $pathEntries = @()
} else {
    $pathEntries = $userPath -split ";" |
        Where-Object {
            -not [string]::IsNullOrWhiteSpace($_)
        }
}

$pathExists = $pathEntries |
    Where-Object {
        $_.TrimEnd("\") -ieq $installDir.TrimEnd("\")
    }

if (-not $pathExists) {

    Write-Step "Updating user PATH..."

    try {
        $pathEntries += $installDir

        [Environment]::SetEnvironmentVariable(
            "Path",
            ($pathEntries -join ";"),
            "User"
        )

        Write-Status "Success" "PATH updated"
    } catch {
        Write-Failure "Could not update your user PATH: $($_.Exception.Message)"
    }

} else {
    Write-Status "Info" "Atlas is already in PATH"
}

# -------------------------------------------------------------
# Update Current PowerShell Session
# -------------------------------------------------------------

$currentPathEntries = $env:Path -split ";" |
    Where-Object {
        -not [string]::IsNullOrWhiteSpace($_)
    }

$currentPathExists = $currentPathEntries |
    Where-Object {
        $_.TrimEnd("\") -ieq $installDir.TrimEnd("\")
    }

if (-not $currentPathExists) {
    $env:Path = (($currentPathEntries + $installDir) -join ";")
}

# -------------------------------------------------------------
# Complete
# -------------------------------------------------------------

Write-SuccessPanel `
    -Version $version `
    -InstallPath $installDir