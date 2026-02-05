#!/bin/bash
set -e

# Install script for timbers
# Usage: curl -fsSL https://raw.githubusercontent.com/gorewood/timbers/main/install.sh | bash
#
# Environment variables:
#   INSTALL_DIR - Installation directory (default: ~/.local/bin)
#   VERSION     - Specific version to install (default: latest)

REPO_OWNER="rbergman"
REPO_NAME="timbers"
BINARY_NAME="timbers"

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    NC=''
fi

error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

warn() {
    echo -e "${YELLOW}Warning: $1${NC}" >&2
}

info() {
    echo -e "${GREEN}$1${NC}"
}

# Detect OS
detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux)
            echo "linux"
            ;;
        darwin)
            echo "darwin"
            ;;
        mingw*|msys*|cygwin*)
            echo "windows"
            ;;
        *)
            error "Unsupported operating system: $os"
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            error "Unsupported architecture: $arch"
            ;;
    esac
}

# Get latest version from GitHub API
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" 2>/dev/null | grep '"tag_name"' | cut -d'"' -f4)
    if [ -z "$version" ]; then
        error "Could not fetch latest version. Check your internet connection and try again."
    fi
    echo "$version"
}

# Main installation
main() {
    local os arch version archive_name url install_dir temp_dir ext

    os=$(detect_os)
    arch=$(detect_arch)

    # Determine archive extension
    if [ "$os" = "windows" ]; then
        ext="zip"
    else
        ext="tar.gz"
    fi

    # Get version (from env or latest)
    if [ -n "$VERSION" ]; then
        version="$VERSION"
        # Ensure version starts with 'v'
        if [[ ! "$version" =~ ^v ]]; then
            version="v$version"
        fi
    else
        echo "Fetching latest release..."
        version=$(get_latest_version)
    fi

    info "Installing ${BINARY_NAME} ${version} for ${os}/${arch}..."

    # Build download URL
    archive_name="${BINARY_NAME}_${os}_${arch}.${ext}"
    url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${archive_name}"

    # Determine install location
    install_dir="${INSTALL_DIR:-$HOME/.local/bin}"
    mkdir -p "$install_dir"

    # Create temp directory
    temp_dir=$(mktemp -d)
    trap "rm -rf '$temp_dir'" EXIT
    cd "$temp_dir"

    # Download
    echo "Downloading ${url}..."
    if ! curl -fsSL "$url" -o "$archive_name"; then
        error "Failed to download ${archive_name}. Check if version ${version} exists."
    fi

    # Verify checksum if available
    checksums_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/checksums.txt"
    if curl -fsSL "$checksums_url" -o checksums.txt 2>/dev/null; then
        if command -v sha256sum &> /dev/null; then
            echo "Verifying checksum..."
            if ! grep "$archive_name" checksums.txt | sha256sum -c - > /dev/null 2>&1; then
                error "Checksum verification failed"
            fi
            info "Checksum verified"
        elif command -v shasum &> /dev/null; then
            echo "Verifying checksum..."
            if ! grep "$archive_name" checksums.txt | shasum -a 256 -c - > /dev/null 2>&1; then
                error "Checksum verification failed"
            fi
            info "Checksum verified"
        else
            warn "Neither sha256sum nor shasum found, skipping checksum verification"
        fi
    fi

    # Extract
    echo "Extracting..."
    if [ "$ext" = "tar.gz" ]; then
        tar -xzf "$archive_name"
    elif [ "$ext" = "zip" ]; then
        unzip -q "$archive_name"
    fi

    # Install
    chmod +x "$BINARY_NAME"
    mv "$BINARY_NAME" "$install_dir/"

    info "${BINARY_NAME} installed to ${install_dir}/${BINARY_NAME}"

    # Check if install dir is in PATH
    if [[ ":$PATH:" != *":$install_dir:"* ]]; then
        echo ""
        warn "$install_dir is not in your PATH"
        echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo "  export PATH=\"\$PATH:$install_dir\""
    fi

    # Verify installation
    echo ""
    if command -v "$BINARY_NAME" &> /dev/null; then
        "$BINARY_NAME" --version
    elif [ -x "$install_dir/$BINARY_NAME" ]; then
        "$install_dir/$BINARY_NAME" --version
    fi
}

main "$@"
