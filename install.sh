#!/bin/sh
set -e

# Atlas Installer
#
# Install latest:
#   curl -fsSL https://raw.githubusercontent.com/Yashh56/atlas/main/install.sh | sh
#
# Install specific version:
#   curl -fsSL https://raw.githubusercontent.com/Yashh56/atlas/main/install.sh | VERSION=v0.1.0 sh

REPO="Yashh56/atlas"
PROJECT_NAME="atlas"
GITHUB_API="https://api.github.com/repos/${REPO}"

# ─────────────────────────────────────────────────────────────
# Terminal colors
# ─────────────────────────────────────────────────────────────

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    CYAN="$(printf '\033[36m')"
    GREEN="$(printf '\033[32m')"
    RED="$(printf '\033[31m')"
    YELLOW="$(printf '\033[33m')"
    GRAY="$(printf '\033[90m')"
    WHITE="$(printf '\033[37m')"
    RESET="$(printf '\033[0m')"
else
    CYAN=""
    GREEN=""
    RED=""
    YELLOW=""
    GRAY=""
    WHITE=""
    RESET=""
fi

# ─────────────────────────────────────────────────────────────
# UI helpers
# ─────────────────────────────────────────────────────────────

header() {
    printf "\n"
    printf "  ${CYAN}Atlas${RESET}"
    printf "  ${GRAY}—  Autonomous deployment pipeline${RESET}\n"
    printf "\n"
}

step() {
    printf "  ${CYAN}◆${RESET} %s\n" "$1"
}

success() {
    printf "  ${GREEN}✓${RESET} %s\n" "$1"
}

info() {
    printf "  ${CYAN}→${RESET} %s\n" "$1"
}

warning() {
    printf "  ${YELLOW}!${RESET} %s\n" "$1"
}

failure() {
    printf "\n"
    printf "  ${RED}┌─ Installation failed ──────────────────────────────${RESET}\n"
    printf "  ${RED}│${RESET}\n"
    printf "  ${RED}│${RESET}  %s\n" "$1"
    printf "  ${RED}│${RESET}\n"
    printf "  ${RED}└────────────────────────────────────────────────────${RESET}\n"
    printf "\n"
    exit 1
}

detail() {
    printf "  ${GRAY}%-10s${RESET} %s\n" "$1" "$2"
}

success_panel() {
    VERSION="$1"
    INSTALL_PATH="$2"

    printf "\n"
    printf "  ${GREEN}┌─ Installation complete ────────────────────────────${RESET}\n"
    printf "  ${GREEN}│${RESET}\n"
    printf "  ${GREEN}│${RESET}  Atlas ${WHITE}v${VERSION}${RESET}\n"
    printf "  ${GREEN}│${RESET}\n"
    printf "  ${GRAY}│${RESET}  Installed to\n"
    printf "  ${GREEN}│${RESET}  ${WHITE}${INSTALL_PATH}${RESET}\n"
    printf "  ${GREEN}│${RESET}\n"
    printf "  ${GRAY}│${RESET}  Run:\n"
    printf "  ${GREEN}│${RESET}\n"
    printf "  ${GREEN}│${RESET}  ${CYAN}atlas --help${RESET}\n"
    printf "  ${GREEN}│${RESET}\n"
    printf "  ${GREEN}└────────────────────────────────────────────────────${RESET}\n"
    printf "\n"
}

# ─────────────────────────────────────────────────────────────
# Start
# ─────────────────────────────────────────────────────────────

header

# ─────────────────────────────────────────────────────────────
# Required commands
# ─────────────────────────────────────────────────────────────

for command in curl tar install sed awk head uname mktemp basename; do
    if ! command -v "$command" >/dev/null 2>&1; then
        failure "Required command '$command' was not found."
    fi
done

# ─────────────────────────────────────────────────────────────
# Detect operating system
# ─────────────────────────────────────────────────────────────

step "Detecting platform..."

OS="$(uname -s)"

case "$OS" in
    Linux*)
        OS_NAME="Linux"
        ;;
    Darwin*)
        OS_NAME="Darwin"
        ;;
    *)
        failure "Unsupported operating system: $OS"
        ;;
esac

# ─────────────────────────────────────────────────────────────
# Detect architecture
# ─────────────────────────────────────────────────────────────

ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64)
        ARCH_NAME="x86_64"
        ;;
    arm64|aarch64)
        ARCH_NAME="arm64"
        ;;
    *)
        failure "Unsupported architecture: $ARCH"
        ;;
esac

success "Platform detected"

# ─────────────────────────────────────────────────────────────
# Determine version
# ─────────────────────────────────────────────────────────────

if [ -n "${VERSION:-}" ]; then
    VERSION="${VERSION#v}"

    if [ -z "$VERSION" ]; then
        failure "Invalid Atlas version."
    fi

    step "Using Atlas v${VERSION}..."
else
    step "Checking latest release..."

    RELEASE_JSON="$(
        curl -fsSL \
            -H "Accept: application/vnd.github+json" \
            "${GITHUB_API}/releases/latest"
    )" || failure "Unable to fetch the latest Atlas release."

    LATEST_TAG="$(
        printf '%s\n' "$RELEASE_JSON" |
            sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' |
            head -n 1
    )"

    if [ -z "$LATEST_TAG" ]; then
        failure "Failed to determine the latest Atlas release."
    fi

    VERSION="${LATEST_TAG#v}"

    success "Latest release found"
fi

# ─────────────────────────────────────────────────────────────
# Release information
# ─────────────────────────────────────────────────────────────

printf "\n"

detail "Version" "v${VERSION}"
detail "Platform" "${OS_NAME} / ${ARCH_NAME}"

printf "\n"

# ─────────────────────────────────────────────────────────────
# Fetch release metadata
# ─────────────────────────────────────────────────────────────

step "Finding release asset..."

RELEASE_JSON="$(
    curl -fsSL \
        -H "Accept: application/vnd.github+json" \
        "${GITHUB_API}/releases/tags/v${VERSION}"
)" || failure "Unable to fetch Atlas v${VERSION}."

DOWNLOAD_URL="$(
    printf '%s\n' "$RELEASE_JSON" |
        grep '"browser_download_url"' |
        grep "${OS_NAME}" |
        grep "${ARCH_NAME}" |
        grep '\.tar\.gz"' |
        sed -n 's/.*"browser_download_url": *"\([^"]*\)".*/\1/p' |
        head -n 1
)"

if [ -z "$DOWNLOAD_URL" ]; then
    failure "No ${OS_NAME} ${ARCH_NAME} release asset was found for v${VERSION}."
fi

FILE_NAME="$(basename "$DOWNLOAD_URL")"

success "Release asset found"

# ─────────────────────────────────────────────────────────────
# Temporary directory
# ─────────────────────────────────────────────────────────────

TMP_DIR="$(mktemp -d)"

cleanup() {
    rm -rf "$TMP_DIR"
}

trap cleanup EXIT INT TERM

ARCHIVE_PATH="${TMP_DIR}/${FILE_NAME}"
CHECKSUM_PATH="${TMP_DIR}/checksums.txt"

# ─────────────────────────────────────────────────────────────
# Download archive
# ─────────────────────────────────────────────────────────────

step "Downloading Atlas v${VERSION}..."

if ! curl -fsSL \
    "$DOWNLOAD_URL" \
    -o "$ARCHIVE_PATH"; then

    failure "Failed to download Atlas."
fi

success "Download complete"

# ─────────────────────────────────────────────────────────────
# Download checksums
# ─────────────────────────────────────────────────────────────

step "Downloading checksums..."

CHECKSUM_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"

if ! curl -fsSL \
    "$CHECKSUM_URL" \
    -o "$CHECKSUM_PATH"; then

    failure "Failed to download release checksums."
fi

success "Checksums downloaded"

# ─────────────────────────────────────────────────────────────
# Verify checksum
# ─────────────────────────────────────────────────────────────

step "Verifying download..."

EXPECTED_CHECKSUM="$(
    awk -v file="$FILE_NAME" '
        $2 == file {
            print $1
            exit
        }
    ' "$CHECKSUM_PATH"
)"

if [ -z "$EXPECTED_CHECKSUM" ]; then
    failure "No checksum was found for ${FILE_NAME}."
fi

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL_CHECKSUM="$(sha256sum "$ARCHIVE_PATH" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL_CHECKSUM="$(shasum -a 256 "$ARCHIVE_PATH" | awk '{print $1}')"
else
    failure "Neither sha256sum nor shasum is available."
fi

if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
    failure "Checksum verification failed."
fi

success "Download verified"

# ─────────────────────────────────────────────────────────────
# Extract
# ─────────────────────────────────────────────────────────────

step "Preparing Atlas..."

if ! tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"; then
    failure "Failed to extract the Atlas archive."
fi

BINARY_PATH="${TMP_DIR}/${PROJECT_NAME}"

if [ ! -f "$BINARY_PATH" ]; then
    failure "Atlas binary was not found in the downloaded archive."
fi

success "Archive extracted"

# ─────────────────────────────────────────────────────────────
# Determine installation directory
# ─────────────────────────────────────────────────────────────

INSTALL_DIR="/usr/local/bin"

if [ -w "$INSTALL_DIR" ]; then
    SUDO=""
else
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        failure "Cannot write to ${INSTALL_DIR} and sudo is not available."
    fi
fi

# ─────────────────────────────────────────────────────────────
# Install
# ─────────────────────────────────────────────────────────────

step "Installing Atlas..."

if ! $SUDO install \
    -m 755 \
    "$BINARY_PATH" \
    "${INSTALL_DIR}/${PROJECT_NAME}"; then

    failure "Could not install Atlas to ${INSTALL_DIR}."
fi

success "Atlas installed"

# ─────────────────────────────────────────────────────────────
# Complete
# ─────────────────────────────────────────────────────────────

success_panel \
    "$VERSION" \
    "${INSTALL_DIR}/${PROJECT_NAME}"