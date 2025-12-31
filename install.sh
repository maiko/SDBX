#!/usr/bin/env bash
# SDBX Installation Script
# Installs the latest release of SDBX from GitHub

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="maiko/SDBX"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="sdbx"

# Functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        *)          echo "unknown" ;;
    esac
}

# Detect Architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64)     echo "amd64" ;;
        aarch64)    echo "arm64" ;;
        arm64)      echo "arm64" ;;
        *)          echo "unknown" ;;
    esac
}

# Check if running as root (for system-wide install)
check_permissions() {
    if [ "$EUID" -ne 0 ] && [ ! -w "$INSTALL_DIR" ]; then
        log_warn "Installation requires root privileges for $INSTALL_DIR"
        log_info "Please run with sudo or install to a user directory"
        return 1
    fi
    return 0
}

# Get latest release version
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        log_error "Failed to fetch latest version from GitHub"
        exit 1
    fi

    echo "$version"
}

# Download and install
install_sdbx() {
    local os arch version archive_name download_url tmp_dir

    # Detect system
    os=$(detect_os)
    arch=$(detect_arch)

    if [ "$os" = "unknown" ] || [ "$arch" = "unknown" ]; then
        log_error "Unsupported operating system or architecture"
        log_info "OS: $(uname -s), Arch: $(uname -m)"
        exit 1
    fi

    log_info "Detected system: $os/$arch"

    # Get latest version
    log_info "Fetching latest release information..."
    version=$(get_latest_version)
    log_success "Latest version: $version"

    # Construct download URL
    archive_name="${BINARY_NAME}_${version#v}_${os}_${arch}.tar.gz"
    download_url="https://github.com/$REPO/releases/download/$version/$archive_name"

    log_info "Downloading $archive_name..."

    # Create temporary directory
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download archive
    if ! curl -fsSL "$download_url" -o "$tmp_dir/$archive_name"; then
        log_error "Failed to download $archive_name"
        log_info "URL: $download_url"
        exit 1
    fi

    log_success "Download complete"

    # Extract archive
    log_info "Extracting archive..."
    if ! tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"; then
        log_error "Failed to extract archive"
        exit 1
    fi

    # Check if binary exists
    if [ ! -f "$tmp_dir/$BINARY_NAME" ]; then
        log_error "Binary not found in archive"
        exit 1
    fi

    # Install binary
    log_info "Installing to $INSTALL_DIR/$BINARY_NAME..."

    # Check permissions
    if ! check_permissions; then
        # Try user local bin
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        log_info "Installing to user directory: $INSTALL_DIR"
    fi

    if ! mv "$tmp_dir/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"; then
        log_error "Failed to install binary"
        exit 1
    fi

    # Make executable
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    log_success "SDBX $version installed successfully!"

    # Check if in PATH
    if ! command -v "$BINARY_NAME" &> /dev/null; then
        log_warn "$INSTALL_DIR is not in your PATH"
        log_info "Add the following to your ~/.bashrc or ~/.zshrc:"
        echo ""
        echo "    export PATH=\"$INSTALL_DIR:\$PATH\""
        echo ""
    else
        log_success "SDBX is ready to use!"
        echo ""
        log_info "Get started with: $BINARY_NAME init"
    fi

    # Show version
    if command -v "$BINARY_NAME" &> /dev/null; then
        echo ""
        "$BINARY_NAME" version
    fi
}

# Main
main() {
    echo ""
    echo "╔═══════════════════════════════════════╗"
    echo "║   SDBX — Seedbox in a Box 📦✨       ║"
    echo "║   Installer                           ║"
    echo "╚═══════════════════════════════════════╝"
    echo ""

    # Check requirements
    for cmd in curl tar; do
        if ! command -v "$cmd" &> /dev/null; then
            log_error "Required command not found: $cmd"
            exit 1
        fi
    done

    install_sdbx
}

main "$@"
