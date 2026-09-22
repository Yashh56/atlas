#!/bin/sh

set -e

# Atlas Installer Script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Yashh56/atlas/master/install.sh | sh
#
# Install a specific version:
#   curl -fsSL https://raw.githubusercontent.com/Yashh56/atlas/master/install.sh | VERSION=v0.1.0 sh

REPO="Yashh56/atlas"
PROJECT_NAME="atlas"
GITHUB_API="https://api.github.com/repos/${REPO}"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

info() {
    printf "%b%s%b\n" "$BLUE" "$1" "$NC"
}

success() {
    printf "%b%s%b\n" "$GREEN" "$1" "$NC"
}

error() {
    printf "%b%s%b\n" "$RED" "$1" "$NC" >&2
}

info "Installing Atlas..."

# Check required commands
for command in curl tar; do
    if ! command -v "$command" >/dev/null 2>&1; then
        error "Required command '$command' was not found."
        exit 1
    fi
done

# Detect OS
OS="$(uname -s)"

case "$OS" in
    Linux*)
        OS_NAME="Linux"
        ;;
    Darwin*)
        OS_NAME="Darwin"
        ;;
    *)
        error "Unsupported operating system: $OS"
        exit 1
        ;;
esac

# Detect architecture
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64)
        ARCH_NAME="x86_64"
        ;;
    arm64|aarch64)
        ARCH_NAME="arm64"
        ;;
    *)
        error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Determine version
if [ -z "${VERSION:-}" ]; then
    info "Fetching latest Atlas release..."

    LATEST_TAG="$(
        curl -fsSL \
            -H "Accept: application/vnd.github+json" \
            "${GITHUB_API}/releases/latest" |
        sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' |
        head -n 1
    )"

    if [ -z "$LATEST_TAG" ]; then
        error "Failed to determine the latest Atlas release."
        exit 1
    fi

    VERSION="$LATEST_TAG"
fi

# Remove optional v prefix
VERSION="${VERSION#v}"

# Validate version
if [ -z "$VERSION" ]; then
    error "Invalid Atlas version."
    exit 1
fi

# Find matching asset from the release API response
info "Looking for ${OS_NAME} ${ARCH_NAME} release asset..."

RELEASE_JSON="$(curl -fsSL -H "Accept: application/vnd.github+json" "${GITHUB_API}/releases/tags/v${VERSION}")"

DOWNLOAD_URL="$(
    echo "$RELEASE_JSON" |
    sed -n 's/.*"browser_download_url": *"\([^"]*'"${OS_NAME}"'[^"]*'"${ARCH_NAME}"'[^"]*\.tar\.gz\)".*/\1/p' |
    head -n 1
)"

if [ -z "$DOWNLOAD_URL" ]; then
    error "Could not find a ${OS_NAME} ${ARCH_NAME} tar.gz asset for v${VERSION}."
    exit 1
fi

FILE_NAME="$(basename "$DOWNLOAD_URL")"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"

info "Version: v${VERSION}"
info "Platform: ${OS_NAME}/${ARCH_NAME}"

# Create temporary directory
TMP_DIR="$(mktemp -d)"

cleanup() {
    rm -rf "$TMP_DIR"
}

trap cleanup EXIT INT TERM

ARCHIVE_PATH="${TMP_DIR}/${FILE_NAME}"
CHECKSUM_PATH="${TMP_DIR}/checksums.txt"

# Download archive
info "Downloading ${FILE_NAME}..."

if ! curl -fsSL "$DOWNLOAD_URL" -o "$ARCHIVE_PATH"; then
    error "Failed to download Atlas."
    error "URL: ${DOWNLOAD_URL}"
    exit 1
fi

# Download checksums
info "Downloading checksums..."

if ! curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM_PATH"; then
    error "Failed to download checksums."
    exit 1
fi

# Verify checksum
info "Verifying checksum..."

EXPECTED_CHECKSUM="$(
    sed -n "s/^.*  ${FILE_NAME}$/\1/p" "$CHECKSUM_PATH"
)"

if [ -z "$EXPECTED_CHECKSUM" ]; then
    error "Could not find checksum for ${FILE_NAME}."
    exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL_CHECKSUM="$(sha256sum "$ARCHIVE_PATH" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL_CHECKSUM="$(shasum -a 256 "$ARCHIVE_PATH" | awk '{print $1}')"
else
    error "Neither sha256sum nor shasum is available."
    exit 1
fi

if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
    error "Checksum verification failed."
    error "Expected: ${EXPECTED_CHECKSUM}"
    error "Actual:   ${ACTUAL_CHECKSUM}"
    exit 1
fi

success "Checksum verified."

# Extract
info "Extracting..."

tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"

BINARY_PATH="${TMP_DIR}/${PROJECT_NAME}"

if [ ! -f "$BINARY_PATH" ]; then
    error "Atlas binary was not found in the downloaded archive."
    exit 1
fi

# Determine installation directory
INSTALL_DIR="/usr/local/bin"

if [ -w "$INSTALL_DIR" ]; then
    SUDO=""
else
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        error "Cannot write to ${INSTALL_DIR} and sudo is not available."
        error "Please install Atlas manually or run this script with appropriate permissions."
        exit 1
    fi
fi

# Install binary
info "Installing Atlas to ${INSTALL_DIR}..."

$SUDO install -m 755 "$BINARY_PATH" "${INSTALL_DIR}/${PROJECT_NAME}"

success "Atlas v${VERSION} was successfully installed!"
echo ""
echo "Run:"
echo "  atlas --help"
