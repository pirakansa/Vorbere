#!/bin/bash
# vorbere installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/pirakansa/vorbere/main/install.sh | bash
#
# Environment variables:
#   VORBERE_VERSION     - Version to install (default: latest)
#   VORBERE_INSTALL_DIR - Installation directory (default: ~/.local/bin)

set -euo pipefail

REPO="pirakansa/vorbere"
BINARY_NAME="vorbere"
TEMP_DIR=""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    exit 1
}

cleanup() {
    if [ -n "${TEMP_DIR:-}" ]; then
        rm -rf "$TEMP_DIR"
    fi
}

detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux) echo "linux" ;;
        darwin) echo "darwin" ;;
        mingw*|msys*|cygwin*) echo "win" ;;
        *) error "Unsupported OS: $os" ;;
    esac
}

detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7*|armv6*|arm) echo "arm" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
}

ensure_supported_target() {
    local os="$1"
    local arch="$2"

    case "${os}-${arch}" in
        linux-amd64|linux-arm|linux-arm64|darwin-amd64|darwin-arm64|win-amd64) ;;
        win-arm|win-arm64)
            error "Unsupported target: ${os}-${arch} (no release artifact published yet)"
            ;;
        *)
            error "Unsupported target: ${os}-${arch}"
            ;;
    esac
}

get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$version" ]; then
        error "Failed to fetch latest version"
    fi
    echo "$version"
}

main() {
    local os arch version install_dir archive_name url binary_on_disk

    trap cleanup EXIT

    os=$(detect_os)
    arch=$(detect_arch)
    ensure_supported_target "$os" "$arch"

    version="${VORBERE_VERSION:-latest}"
    if [ "$version" = "latest" ]; then
        info "Fetching latest version..."
        version=$(get_latest_version)
    fi
    info "Installing ${BINARY_NAME} ${version}"

    install_dir="${VORBERE_INSTALL_DIR:-$HOME/.local/bin}"
    archive_name="${BINARY_NAME}_${os}-${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"

    info "Downloading from ${url}"

    TEMP_DIR=$(mktemp -d)

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "${TEMP_DIR}/${archive_name}"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "${TEMP_DIR}/${archive_name}"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    tar -xzf "${TEMP_DIR}/${archive_name}" -C "$TEMP_DIR"

    mkdir -p "$install_dir"

    binary_on_disk="$BINARY_NAME"
    if [ "$os" = "win" ]; then
        binary_on_disk="${BINARY_NAME}.exe"
    fi

    if [ ! -f "${TEMP_DIR}/${binary_on_disk}" ]; then
        error "Extracted archive did not contain ${binary_on_disk}"
    fi

    mv "${TEMP_DIR}/${binary_on_disk}" "${install_dir}/"
    if [ "$os" != "win" ]; then
        chmod +x "${install_dir}/${binary_on_disk}"
    fi

    info "Installed ${binary_on_disk} to ${install_dir}"

    if [[ ":$PATH:" != *":${install_dir}:"* ]]; then
        warn "${install_dir} is not in your PATH"
        echo ""
        echo "Add the following line to your shell configuration file:"
        echo "  export PATH=\"\$PATH:${install_dir}\""
        echo ""
    fi

    info "Installation complete! Run '${BINARY_NAME} --help' to verify."
}

main "$@"
