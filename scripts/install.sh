#!/bin/bash
# ClawIDE Installation Script
# Usage: curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="davydany/ClawIDE"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="clawide"

# Detect OS and architecture
detect_system() {
  OS=$(uname -s)
  ARCH=$(uname -m)

  case "$OS" in
    Linux*)     OS_TYPE="linux" ;;
    Darwin*)    OS_TYPE="darwin" ;;
    MINGW*)     OS_TYPE="windows" ;;
    MSYS*)      OS_TYPE="windows" ;;
    *)          OS_TYPE="UNKNOWN" ;;
  esac

  case "$ARCH" in
    x86_64)     ARCH_TYPE="amd64" ;;
    aarch64)    ARCH_TYPE="arm64" ;;
    arm64)      ARCH_TYPE="arm64" ;;  # macOS M1/M2
    *)          ARCH_TYPE="UNKNOWN" ;;
  esac

  echo -e "${BLUE}Detected system: ${OS_TYPE} ${ARCH_TYPE}${NC}"

  if [ "$OS_TYPE" = "UNKNOWN" ] || [ "$ARCH_TYPE" = "UNKNOWN" ]; then
    echo -e "${RED}Error: Unsupported system (OS: $OS, ARCH: $ARCH)${NC}"
    echo "Please install from source: https://github.com/$REPO"
    exit 1
  fi
}

# Get the latest release version
get_latest_version() {
  LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
  VERSION=$(echo "$LATEST_RELEASE" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/' | head -1)

  if [ -z "$VERSION" ]; then
    echo -e "${RED}Error: Could not fetch latest version from GitHub${NC}"
    exit 1
  fi

  echo "$VERSION"
}

# Build download URL
build_download_url() {
  VERSION=$1

  case "$OS_TYPE" in
    linux)
      FILENAME="clawide-v${VERSION}-linux-${ARCH_TYPE}.tar.gz"
      ;;
    darwin)
      FILENAME="clawide-v${VERSION}-darwin-${ARCH_TYPE}.tar.gz"
      ;;
    windows)
      FILENAME="clawide-v${VERSION}-windows-${ARCH_TYPE}.zip"
      ;;
  esac

  DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/$FILENAME"
  echo "$DOWNLOAD_URL"
}

# Show installation plan
show_plan() {
  VERSION=$1
  DOWNLOAD_URL=$2

  echo ""
  echo -e "${YELLOW}Installation Plan:${NC}"
  echo "  Repository:  $REPO"
  echo "  Version:     v$VERSION"
  echo "  Download:    $DOWNLOAD_URL"
  echo "  Install to:  $INSTALL_DIR/$BINARY_NAME"
  echo ""
  echo -e "${BLUE}Before continuing, you can inspect the script:${NC}"
  echo "  • View on GitHub: https://github.com/$REPO/blob/master/scripts/install.sh"
  echo "  • View current script in terminal"
  echo ""
}

# Download and install
install_binary() {
  VERSION=$1
  DOWNLOAD_URL=$2

  # Create temporary directory
  TEMP_DIR=$(mktemp -d)
  trap "rm -rf $TEMP_DIR" EXIT

  echo -e "${BLUE}Downloading ClawIDE v$VERSION...${NC}"

  if ! curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_DIR/$FILENAME" < /dev/null; then
    echo -e "${RED}Error: Failed to download binary${NC}"
    exit 1
  fi

  echo -e "${BLUE}Extracting binary...${NC}"

  if [[ "$FILENAME" == *.tar.gz ]]; then
    tar -xzf "$TEMP_DIR/$FILENAME" -C "$TEMP_DIR"
  elif [[ "$FILENAME" == *.zip ]]; then
    unzip -q "$TEMP_DIR/$FILENAME" -d "$TEMP_DIR"
  fi

  # Find the binary (could be in subdirectory)
  BINARY_PATH=$(find "$TEMP_DIR" -type f -name "clawide" -o -name "clawide.exe" | head -1)

  if [ -z "$BINARY_PATH" ]; then
    echo -e "${RED}Error: Binary not found in archive${NC}"
    exit 1
  fi

  echo -e "${BLUE}Installing to $INSTALL_DIR...${NC}"

  # Check if we need sudo
  if [ ! -w "$INSTALL_DIR" ]; then
    sudo mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
  else
    mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
  fi

  echo -e "${GREEN}✓ Installation successful!${NC}"
  echo ""
  echo -e "${BLUE}Getting started:${NC}"
  echo "  1. Start ClawIDE:        $BINARY_NAME"
  echo "  2. Open browser:         http://localhost:9800"
  echo "  3. Configure if needed:  ~/.clawide/config.json"
  echo ""
  echo -e "${BLUE}Documentation:${NC}"
  echo "  https://github.com/$REPO/blob/master/README.md"
}

# Main installation flow
main() {
  echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║    ClawIDE Installation Script        ║${NC}"
  echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
  echo ""

  # Detect system
  detect_system

  # Get latest version
  echo -e "${BLUE}Fetching latest version...${NC}"
  VERSION=$(get_latest_version)
  echo -e "${GREEN}Latest version: v$VERSION${NC}"

  # Build download URL
  DOWNLOAD_URL=$(build_download_url "$VERSION")

  # Show installation plan
  show_plan "$VERSION" "$DOWNLOAD_URL"

  # Install binary
  install_binary "$VERSION" "$DOWNLOAD_URL"

  echo -e "${GREEN}Done! Enjoy using ClawIDE.${NC}"
}

# Run main installation
main
