#!/bin/bash
# ClawIDE Uninstall Script
# Usage: curl -fsSL https://raw.githubusercontent.com/davydany/ClawIDE/refs/heads/master/scripts/uninstall.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
CLAWIDE_DATA_DIR="${CLAWIDE_DATA_DIR:-$HOME/.clawide}"
BINARY_NAME="clawide"

echo -e "${RED}+========================================+${NC}"
echo -e "${RED}|    ClawIDE Uninstall Script            |${NC}"
echo -e "${RED}+========================================+${NC}"
echo ""

# Check what exists
BINARY_PATH="$INSTALL_DIR/$BINARY_NAME"
FOUND_BINARY=false
FOUND_DATA=false

if [ -f "$BINARY_PATH" ]; then
  FOUND_BINARY=true
fi

if [ -d "$CLAWIDE_DATA_DIR" ]; then
  FOUND_DATA=true
fi

if [ "$FOUND_BINARY" = false ] && [ "$FOUND_DATA" = false ]; then
  echo -e "${YELLOW}Nothing to uninstall.${NC}"
  echo "  Binary not found at: $BINARY_PATH"
  echo "  Config directory not found at: $CLAWIDE_DATA_DIR"
  exit 0
fi

# Warn if ClawIDE is currently running
if pgrep -x "$BINARY_NAME" > /dev/null 2>&1; then
  echo -e "${RED}Warning: ClawIDE appears to be running.${NC}"
  echo -e "${RED}Please stop it before uninstalling.${NC}"
  exit 1
fi

# Show what will be removed
echo -e "${YELLOW}The following will be removed:${NC}"
echo ""

if [ "$FOUND_BINARY" = true ]; then
  echo "  Binary:  $BINARY_PATH"
fi

if [ "$FOUND_DATA" = true ]; then
  echo "  Config:  $CLAWIDE_DATA_DIR/"
fi

echo ""

# Ask for confirmation
echo -e -n "${YELLOW}Proceed with uninstall? [y/N] ${NC}"
read -r CONFIRM < /dev/tty

if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
  echo -e "${BLUE}Uninstall cancelled.${NC}"
  exit 0
fi

echo ""

# Remove binary
if [ "$FOUND_BINARY" = true ]; then
  rm -f "$BINARY_PATH"
  echo -e "${GREEN}Removed binary: $BINARY_PATH${NC}"
fi

# Remove config directory
if [ "$FOUND_DATA" = true ]; then
  rm -rf "$CLAWIDE_DATA_DIR"
  echo -e "${GREEN}Removed config directory: $CLAWIDE_DATA_DIR${NC}"
fi

echo ""
echo -e "${GREEN}ClawIDE has been uninstalled.${NC}"
echo ""
echo -e "${YELLOW}Note: PATH entries in your shell config (~/.bashrc, ~/.zshrc) were not modified.${NC}"
echo -e "${YELLOW}You can remove the ~/.local/bin PATH line manually if no longer needed.${NC}"
