#!/bin/bash
# ClawIDE Installation Script
# Usage: curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
REPO="davydany/ClawIDE"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="clawide"

echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║    ClawIDE Installation Script        ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""

# Detect system
OS=$(uname -s)
ARCH=$(uname -m)

case "$OS" in
  Linux*)     OS_TYPE="linux" ;;
  Darwin*)    OS_TYPE="darwin" ;;
  *)          echo -e "${RED}Error: Unsupported OS ($OS)${NC}"; exit 1 ;;
esac

case "$ARCH" in
  x86_64)     ARCH_TYPE="amd64" ;;
  aarch64)    ARCH_TYPE="arm64" ;;
  arm64)      ARCH_TYPE="arm64" ;;
  *)          echo -e "${RED}Error: Unsupported architecture ($ARCH)${NC}"; exit 1 ;;
esac

echo -e "${BLUE}Detected system: ${OS_TYPE} ${ARCH_TYPE}${NC}"

# Fetch latest release info
echo -e "${BLUE}Fetching latest version...${NC}"
API_RESPONSE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
VERSION=$(echo "$API_RESPONSE" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/' | head -1)

if [ -z "$VERSION" ]; then
  echo -e "${RED}Error: Could not fetch latest version from GitHub${NC}"
  exit 1
fi

echo -e "${GREEN}Latest version: v$VERSION${NC}"

# Build filename and download URL
FILENAME="clawide-v${VERSION}-${OS_TYPE}-${ARCH_TYPE}.tar.gz"
if [ "$OS_TYPE" = "windows" ]; then
  FILENAME="clawide-v${VERSION}-${OS_TYPE}-${ARCH_TYPE}.zip"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/${FILENAME}"

echo ""
echo -e "${YELLOW}Installation Plan:${NC}"
echo "  Repository:  $REPO"
echo "  Version:     v$VERSION"
echo "  Download:    $DOWNLOAD_URL"
echo "  Install to:  $INSTALL_DIR/$BINARY_NAME"
echo ""

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Download
echo -e "${BLUE}Downloading ClawIDE v$VERSION...${NC}"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_DIR/$FILENAME"; then
  echo -e "${RED}Error: Failed to download binary${NC}"
  echo -e "${RED}URL: $DOWNLOAD_URL${NC}"
  exit 1
fi

echo -e "${BLUE}Extracting binary...${NC}"
if [[ "$FILENAME" == *.tar.gz ]]; then
  tar -xzf "$TEMP_DIR/$FILENAME" -C "$TEMP_DIR"
elif [[ "$FILENAME" == *.zip ]]; then
  unzip -q "$TEMP_DIR/$FILENAME" -d "$TEMP_DIR"
fi

# Find the binary
BINARY_PATH=$(find "$TEMP_DIR" -type f \( -name "clawide" -o -name "clawide.exe" \) | head -1)

if [ -z "$BINARY_PATH" ]; then
  echo -e "${RED}Error: Binary not found in archive${NC}"
  exit 1
fi

echo -e "${BLUE}Installing to $INSTALL_DIR...${NC}"

# Install (use sudo if needed)
if [ ! -w "$INSTALL_DIR" ]; then
  if [ -t 0 ]; then
    # Terminal is available, can use sudo interactively
    sudo mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
  else
    # No terminal (piped script), can't use sudo interactively
    echo -e "${RED}Error: $INSTALL_DIR is not writable and running in non-interactive mode${NC}"
    echo -e "${RED}Please run with sudo or set INSTALL_DIR to a writable directory:${NC}"
    echo "  sudo curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/install.sh | bash"
    echo "  or"
    echo "  INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/install.sh | bash"
    exit 1
  fi
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
