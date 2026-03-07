#!/bin/bash

# SOVA Protocol v1.0 - Autonomous Installer for Linux/macOS
# No dependencies required except bash and curl/wget
# Usage: curl -fsSL https://raw.githubusercontent.com/IvanChernykh/SOVA/main/install.sh | bash

set -euo pipefail

VERSION="1.0.0"
BASE_URL="https://github.com/IvanChernykh/SOVA/releases/download/v${VERSION}"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sova"
DATA_DIR="/var/lib/sova"
LOG_DIR="/var/log/sova"

# Colors
PURPLE='\033[95m'
CYAN='\033[36m'
GREEN='\033[92m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'
BOLD='\033[1m'

animate_owl() {
    local frames=(
        "    ,___,\n    {o,o}\n    /)  )\n    -\"  \"-"
        "    ,___,\n    {O,O}\n    /)  )\n    -\"  \"-"
        "    ,___,\n    {o,o}\n    /)  )\n    -\"  \"-"
        "    ,___,\n    {-,-}\n    /)  )\n    -\"  \"-"
        "    ,___,\n    {o,o}\n    /)  )\n    -\"  \"-"
    )
    for frame in "${frames[@]}"; do
        echo -en "\033[2J\033[H"
        echo -e "${PURPLE}"
        echo -e "$frame"
        echo -e "${RESET}"
        sleep 0.15
    done
}

print_banner() {
    if [ -t 1 ]; then
        animate_owl
    fi
    echo -e "${PURPLE}${BOLD}"
    echo "    ╔══════════════════════════════════════╗"
    echo "    ║            ,___,                     ║"
    echo "    ║            {o,o}    S O V A          ║"
    echo "    ║            /)  )    Protocol v${VERSION}   ║"
    echo '    ║            -"  "-                    ║'
    echo "    ║                                      ║"
    echo "    ║  Autonomous AI-Powered Anti-DPI      ║"
    echo "    ║  Post-Quantum  |  100% Free & Open   ║"
    echo "    ╚══════════════════════════════════════╝"
    echo -e "${RESET}"
    echo -e "${CYAN}  github.com/IvanChernykh/SOVA${RESET}"
    echo ""
}

log_info()  { echo -e "${CYAN}  \u25b8 $1${RESET}"; }
log_ok()    { echo -e "${GREEN}  \u2713 $1${RESET}"; }
log_warn()  { echo -e "${YELLOW}  \u26a0 $1${RESET}"; }
log_error() { echo -e "${RED}  \u2717 $1${RESET}"; }

detect_platform() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)          ARCH="amd64" ;;
        aarch64|arm64)   ARCH="arm64" ;;
        armv7l|armhf)    ARCH="armv7" ;;
        i386|i686)       ARCH="386" ;;
        *) log_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case $OS in
        linux|darwin|freebsd) ;;
        *) log_error "Unsupported OS: $OS"; exit 1 ;;
    esac

    log_info "Platform: ${OS}/${ARCH}"
}

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        if command -v sudo &>/dev/null; then
            SUDO="sudo"
            log_warn "Running with sudo"
        else
            log_error "This script requires root privileges. Run with sudo."
            exit 1
        fi
    else
        SUDO=""
    fi
}

download_binary() {
    local component=$1
    local url="${BASE_URL}/sova-${component}-${OS}-${ARCH}"
    local dest="${INSTALL_DIR}/sova-${component}"

    log_info "Downloading sova-${component}..."

    if command -v curl &>/dev/null; then
        $SUDO curl -fsSL -o "$dest" "$url" 2>/dev/null || {
            log_warn "Download failed from release URL, building from source if Go available"
            return 1
        }
    elif command -v wget &>/dev/null; then
        $SUDO wget -q -O "$dest" "$url" 2>/dev/null || {
            log_warn "Download failed, trying build from source"
            return 1
        }
    else
        log_error "curl or wget required"
        return 1
    fi

    $SUDO chmod +x "$dest"
    log_ok "Installed sova-${component} to ${dest}"
    return 0
}

build_from_source() {
    if ! command -v go &>/dev/null; then
        log_error "Go is not installed. Install Go 1.21+ or download pre-built binaries."
        exit 1
    fi

    GO_VERSION=$(go version | grep -oP '\d+\.\d+')
    log_info "Building from source with Go ${GO_VERSION}..."

    # Find the source directory
    SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
    if [ -f "${SCRIPT_DIR}/go.mod" ]; then
        cd "$SCRIPT_DIR"
    else
        log_error "Cannot find SOVA source code. Clone the repo first."
        exit 1
    fi

    go mod download
    log_info "Building server..."
    go build -ldflags "-s -w -X main.version=${VERSION}" -o "${INSTALL_DIR}/sova-server" ./server/
    log_ok "Built sova-server"

    log_info "Building client..."
    go build -ldflags "-s -w -X main.version=${VERSION}" -o "${INSTALL_DIR}/sova" ./client/
    log_ok "Built sova client"
}

setup_directories() {
    log_info "Creating directories..."
    $SUDO mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
    $SUDO chmod 750 "$CONFIG_DIR" "$DATA_DIR"
    log_ok "Directories created"
}

generate_config() {
    if [ -f "${CONFIG_DIR}/config.json" ]; then
        log_warn "Config already exists at ${CONFIG_DIR}/config.json, skipping"
        return
    fi

    log_info "Generating default configuration..."
    $SUDO tee "${CONFIG_DIR}/config.json" > /dev/null << 'EOF'
{
  "port": 443,
  "api": {
    "enabled": true,
    "port": 8080
  },
  "security": {
    "enable_pq": true,
    "allowed_users": [],
    "rate_limit": 100
  },
  "transports": ["web_mirror", "cloud_carrier", "shadow_websocket"],
  "sni_list": ["sova.example.com", "cdn.cloudflare.com", "aws.amazon.com"]
}
EOF
    log_ok "Configuration generated at ${CONFIG_DIR}/config.json"
}

setup_systemd() {
    if [ "$OS" != "linux" ] || ! command -v systemctl &>/dev/null; then
        return
    fi

    log_info "Setting up systemd service..."
    $SUDO tee /etc/systemd/system/sova.service > /dev/null << EOF
[Unit]
Description=SOVA Protocol Server
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/sova-server
Restart=always
RestartSec=5
WorkingDirectory=${DATA_DIR}
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

    $SUDO systemctl daemon-reload
    log_ok "Systemd service installed"
    log_info "Start with: sudo systemctl start sova"
    log_info "Enable on boot: sudo systemctl enable sova"
}

# Main
print_banner
detect_platform
check_root
setup_directories

# Try download first, fall back to build
if ! download_binary "server" || ! download_binary "client"; then
    log_info "Falling back to build from source..."
    build_from_source
fi

generate_config
setup_systemd

echo ""
log_ok "SOVA Protocol v${VERSION} installed successfully!"
echo ""
log_info "Dashboard:  http://localhost:8080"
log_info "Server:     sova-server"
log_info "Client:     sova connect <config>"
log_info "Config:     ${CONFIG_DIR}/config.json"
echo ""