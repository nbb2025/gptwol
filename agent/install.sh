#!/bin/bash
# GPTWol Agent - One-click install script for Linux
# Usage: curl -sSL https://raw.githubusercontent.com/nbb2025/gptwol/main/agent/install.sh | sudo bash -s -- -a shutdown

set -e

VERSION="latest"
REPO="nbb2025/gptwol"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="gptwol-agent"
ACTION="shutdown"
MAC=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

usage() {
    echo "GPTWol Agent Installer - Sleep-on-LAN listener"
    echo ""
    echo "Usage: $0 [-a ACTION] [-m MAC]"
    echo "  -a ACTION    Action on SOL packet: shutdown, reboot, sleep, hibernate (default: shutdown)"
    echo "  -m MAC       Specific MAC address to monitor (default: auto-detect all)"
    echo "  -h           Show this help"
    echo ""
    echo "Example:"
    echo "  curl -sSL .../install.sh | sudo bash -s -- -a shutdown"
    exit 1
}

# Parse arguments
while getopts "a:m:h" opt; do
    case $opt in
        a) ACTION="$OPTARG" ;;
        m) MAC="$OPTARG" ;;
        h) usage ;;
        *) usage ;;
    esac
done

# Check root
if [ "$EUID" -ne 0 ]; then
    print_error "Please run as root (sudo)"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l)  ARCH="arm" ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
print_info "Detected: ${OS}/${ARCH}"

# Download binary
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/gptwol-agent-${OS}-${ARCH}"
print_info "Downloading from: ${DOWNLOAD_URL}"

if command -v curl &> /dev/null; then
    curl -sSL -o /tmp/${BINARY_NAME} "${DOWNLOAD_URL}"
elif command -v wget &> /dev/null; then
    wget -q -O /tmp/${BINARY_NAME} "${DOWNLOAD_URL}"
else
    print_error "curl or wget required"
    exit 1
fi

chmod +x /tmp/${BINARY_NAME}

# Install
print_info "Installing to ${INSTALL_DIR}/${BINARY_NAME}"
mv /tmp/${BINARY_NAME} ${INSTALL_DIR}/${BINARY_NAME}

# Run install with service
print_info "Installing as system service..."
INSTALL_CMD="${INSTALL_DIR}/${BINARY_NAME} -install -action ${ACTION}"
if [ -n "$MAC" ]; then
    INSTALL_CMD="${INSTALL_CMD} -mac ${MAC}"
fi
eval $INSTALL_CMD

print_info "Installation complete!"
echo ""
echo "  Service: gptwol-agent"
echo "  Action: ${ACTION}"
echo "  Listening on: UDP ports 7, 9"
echo "  Status: systemctl status gptwol-agent"
echo "  Logs: journalctl -u gptwol-agent -f"
echo ""
echo "  To trigger ${ACTION}, send a Sleep-on-LAN packet (reversed MAC)"
echo "  from your gptwol server."
