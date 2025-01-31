#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print gommit banner
echo -e "${BLUE}"
echo "  ________                          .__.__  __   "
echo " /  _____/  ____   _____   _____   |__|__|/  |_ "
echo "/   \  ___ /  _ \ /     \ /     \  |  |  \   __\\"
echo "\    \_\  (  <_> )  Y Y  \  Y Y  \ |  |  ||  |  "
echo " \______  /\____/|__|_|  /__|_|  / |__|__||__|  "
echo "        \/             \/      \/               "
echo -e "${NC}"

# Detect operating system
OS="$(uname -s)"
ARCH="$(uname -m)"

# Convert architecture to Go format
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Convert OS to Go format
case $OS in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    *)
        echo -e "${RED}Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

# GitHub repository information
REPO="edhuardotierrez/gommit"
BINARY_NAME="gommit"

echo -e "${BLUE}• Detecting latest version...${NC}"

# Get the latest release version
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}Error: Could not detect latest version${NC}"
    exit 1
fi

echo -e "${GREEN}• Latest version: $LATEST_VERSION${NC}"

# Construct the download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}_${OS}_${ARCH}"

# Create temporary directory
TMP_DIR=$(mktemp -d)
BINARY_PATH="$TMP_DIR/$BINARY_NAME"

echo -e "${BLUE}• Downloading gommit...${NC}"

# Download the binary
if curl -sL "$DOWNLOAD_URL" -o "$BINARY_PATH"; then
    chmod +x "$BINARY_PATH"
else
    echo -e "${RED}Error: Failed to download gommit${NC}"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Determine install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    # If /usr/local/bin is not writable, try to use sudo
    echo -e "${BLUE}• Installing gommit (requires sudo)...${NC}"
    if ! sudo mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"; then
        echo -e "${RED}Error: Failed to install gommit${NC}"
        rm -rf "$TMP_DIR"
        exit 1
    fi
else
    echo -e "${BLUE}• Installing gommit...${NC}"
    if ! mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"; then
        echo -e "${RED}Error: Failed to install gommit${NC}"
        rm -rf "$TMP_DIR"
        exit 1
    fi
fi

# Clean up
rm -rf "$TMP_DIR"

echo -e "${GREEN}✨ gommit $LATEST_VERSION has been installed successfully!${NC}"
echo -e "${BLUE}• Run 'gommit --help' to get started${NC}" 