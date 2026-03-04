#!/usr/bin/env bash

# install.sh - armoctl installer for Linux and macOS
# Downloads the correct armoctl binary from CloudFront and installs it.

set -e

DIST_URL="https://package-distribution.armosec.io/armoctl"
INSTALL_DIR="/usr/local/bin"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: install.sh [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --version VERSION   Install a specific version (e.g., v0.0.42). Defaults to latest."
            echo "  --dir DIRECTORY     Install to a specific directory. Defaults to /usr/local/bin."
            echo "  -h, --help          Show this help message."
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)
        OS="linux"
        ;;
    darwin)
        OS="darwin"
        ;;
    *)
        echo "Error: Unsupported operating system: $OS"
        exit 1
        ;;
esac

# Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        GOARCH="amd64"
        ;;
    aarch64|arm64)
        GOARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "Detected OS: $OS"
echo "Detected Architecture: $ARCH ($GOARCH)"

# Construct download URL
if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
    URL="${DIST_URL}/armoctl_latest_${OS}_${GOARCH}"
    echo "Downloading latest armoctl..."
else
    case "$VERSION" in
        v*) V_DIR="$VERSION" ;;
        *)  V_DIR="v$VERSION" ;;
    esac
    URL="${DIST_URL}/releases/${V_DIR}/armoctl_${OS}_${GOARCH}"
    echo "Downloading armoctl ${VERSION}..."
fi

# Check for curl
if ! command -v curl >/dev/null 2>&1; then
    echo "Error: curl is required but not installed."
    exit 1
fi

# Determine if sudo is needed for the install directory
SUDO=""
if [ ! -w "$INSTALL_DIR" ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    else
        echo "Warning: Cannot write to $INSTALL_DIR and sudo is not available."
        INSTALL_DIR="."
        echo "Installing to current directory instead."
    fi
fi

# Create install directory if needed
if [ ! -d "$INSTALL_DIR" ]; then
    $SUDO mkdir -p "$INSTALL_DIR"
fi

# Download
TMP_FILE=$(mktemp)
trap 'rm -f "$TMP_FILE"' EXIT

echo "Downloading from $URL..."
if ! curl -L -f -o "$TMP_FILE" "$URL"; then
    echo "Error: Failed to download armoctl from $URL."
    echo "Check that the version exists and your OS/architecture is supported."
    exit 1
fi

# Install
$SUDO mv "$TMP_FILE" "${INSTALL_DIR}/armoctl"
$SUDO chmod +x "${INSTALL_DIR}/armoctl"
trap - EXIT

echo ""
echo "armoctl installed successfully to ${INSTALL_DIR}/armoctl"

# Verify
if command -v "${INSTALL_DIR}/armoctl" >/dev/null 2>&1; then
    "${INSTALL_DIR}/armoctl" version 2>/dev/null || true
fi
