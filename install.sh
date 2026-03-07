#!/bin/bash

# SOVA Protocol v1.0.0 - Installer for Linux/macOS
# No dependencies required except bash and curl/wget
# Usage: curl -fsSL https://raw.githubusercontent.com/IvanChernykh/SOVA/main/install.sh | bash

set -euo pipefail

VERSION="1.0.0"
REPO_URL="https://github.com/IvanChernykh/SOVA"
BASE_URL="${REPO_URL}/releases/download/v${VERSION}"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="${HOME}/.sova"
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
    local f1="         ___________\n        /   /   \\   \\\\\n       |   | O   O |  |\n       |   |   V   |  |\n        \\   \\_____/   /\n      // \\___________/ \\\\\\\\\n     //   |||||||||||   \\\\\\\\\n    ||    |||||||||||    ||\n           ||   ||\n          _||___||_"
    local f2="         ___________\n        /   /   \\   \\\\\n       |   | *   * |  |\n       |   |   V   |  |\n        \\   \\_____/   /\n      // \\___________/ \\\\\\\\\n     //   |||||||||||   \\\\\\\\\n    ||    |||||||||||    ||\n           ||   ||\n          _||___||_"
    local f3="         ___________\n        /   /   \\   \\\\\n       |   | O   O |  |\n       |   |   V   |  |\n        \\   \\_____/   /\n     /  \\___________/  \\\\\n    /    |||||||||||    \\\\\n   /     |||||||||||     \\\\\n           ||   ||\n          _||___||_"
    local frames=("$f1" "$f2" "$f1" "$f3" "$f1")
    for frame in "${frames[@]}"; do
        echo -en "\033[2J\033[H"
        echo -e "${PURPLE}${frame}${RESET}"
        sleep 0.2
    done
}

print_banner() {
    if [ -t 1 ]; then
        animate_owl 2>/dev/null || true
    fi
    echo -e "${PURPLE}${BOLD}"
    echo "  ╔════════════════════════════════════════════════════╗"
    echo "  ║         ___________                               ║"
    echo "  ║        /   /   \\   \\                              ║"
    echo "  ║       |   | O   O |  |   S O V A  Protocol       ║"
    echo "  ║       |   |   V   |  |   v${VERSION}                    ║"
    echo "  ║        \\   \\_____/   /                            ║"
    echo "  ║      // \\___________/ \\\\                          ║"
    echo "  ║                                                   ║"
    echo "  ║   AI-Powered  |  Post-Quantum  |  Free & Open    ║"
    echo "  ╚════════════════════════════════════════════════════╝"
    echo -e "${RESET}"
    echo -e "${CYAN}  github.com/IvanChernykh/SOVA${RESET}"
    echo ""
}

log_info()  { echo -e "${CYAN}  ▸ $1${RESET}"; }
log_ok()    { echo -e "${GREEN}  ✓ $1${RESET}"; }
log_warn()  { echo -e "${YELLOW}  ⚠ $1${RESET}"; }
log_error() { echo -e "${RED}  ✗ $1${RESET}"; }

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

download_from_release() {
    local archive="sova-${OS}-${ARCH}-v${VERSION}.tar.gz"
    local url="${BASE_URL}/${archive}"
    local tmp_dir
    tmp_dir=$(mktemp -d)
    local archive_path="${tmp_dir}/${archive}"

    log_info "Downloading ${archive}..."

    if command -v curl &>/dev/null; then
        curl -fsSL -o "$archive_path" "$url" 2>/dev/null || {
            log_warn "Download failed, will try build from source"
            rm -rf "$tmp_dir"
            return 1
        }
    elif command -v wget &>/dev/null; then
        wget -q -O "$archive_path" "$url" 2>/dev/null || {
            log_warn "Download failed, will try build from source"
            rm -rf "$tmp_dir"
            return 1
        }
    else
        log_error "curl or wget required"
        rm -rf "$tmp_dir"
        return 1
    fi

    log_info "Extracting binaries..."
    tar -xzf "$archive_path" -C "$tmp_dir" 2>/dev/null || {
        log_warn "Extract failed"
        rm -rf "$tmp_dir"
        return 1
    }

    local server_bin
    server_bin=$(find "$tmp_dir" -name "sova-server*" -type f | head -1)
    local client_bin
    client_bin=$(find "$tmp_dir" -name "sova-*" -not -name "sova-server*" -not -name "*.tar.gz" -type f | head -1)

    if [ -n "$server_bin" ]; then
        $SUDO cp "$server_bin" "${INSTALL_DIR}/sova-server"
        $SUDO chmod +x "${INSTALL_DIR}/sova-server"
        log_ok "Installed sova-server"
    fi
    if [ -n "$client_bin" ]; then
        $SUDO cp "$client_bin" "${INSTALL_DIR}/sova"
        $SUDO chmod +x "${INSTALL_DIR}/sova"
        log_ok "Installed sova"
    fi

    rm -rf "$tmp_dir"

    if [ -n "$server_bin" ] && [ -n "$client_bin" ]; then
        return 0
    fi
    log_warn "Some binaries missing from archive"
    return 1
}

find_source_dir() {
    # 1. Check if script is in the repo
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd)" || true
    if [ -n "$script_dir" ] && [ -f "${script_dir}/go.mod" ]; then
        echo "$script_dir"
        return 0
    fi
    # 2. Check common locations
    for dir in "$HOME/SOVA" "$HOME/Desktop/SOVA" "$HOME/Documents/SOVA" "/opt/SOVA"; do
        if [ -f "${dir}/go.mod" ]; then
            echo "$dir"
            return 0
        fi
    done
    return 1
}

build_from_source() {
    if ! command -v go &>/dev/null; then
        log_error "Go is not installed. Install Go 1.21+ or download pre-built binaries."
        exit 1
    fi

    local go_ver
    go_ver=$(go version | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
    log_info "Building from source with Go ${go_ver}..."

    local src_dir=""
    local cloned=false

    src_dir=$(find_source_dir) || true

    if [ -z "$src_dir" ]; then
        # 3. Clone the repo
        if command -v git &>/dev/null; then
            log_info "Cloning SOVA repository..."
            src_dir=$(mktemp -d)
            if git clone --depth 1 --branch "v${VERSION}" "${REPO_URL}.git" "$src_dir" 2>/dev/null; then
                cloned=true
                log_ok "Repository cloned"
            elif git clone --depth 1 "${REPO_URL}.git" "$src_dir" 2>/dev/null; then
                cloned=true
                log_ok "Repository cloned (latest)"
            else
                log_error "Failed to clone repository"
                exit 1
            fi
        else
            log_error "Cannot find SOVA source code and git is not available."
            log_error "Either clone the repo manually or install git:"
            log_error "  git clone ${REPO_URL}.git"
            log_error "  cd SOVA && sudo ./install.sh"
            exit 1
        fi
    else
        log_info "Found source at ${src_dir}"
    fi

    cd "$src_dir"
    go mod download

    log_info "Building server..."
    $SUDO go build -ldflags "-s -w -X main.Version=v${VERSION}" -o "${INSTALL_DIR}/sova-server" ./server/
    log_ok "Built sova-server"

    log_info "Building client..."
    $SUDO go build -ldflags "-s -w -X main.Version=v${VERSION}" -o "${INSTALL_DIR}/sova" ./client/
    log_ok "Built sova client"

    if [ "$cloned" = true ] && [ -n "$src_dir" ]; then
        rm -rf "$src_dir"
    fi
}

setup_directories() {
    log_info "Creating directories..."
    mkdir -p "$CONFIG_DIR" "$CONFIG_DIR/profiles" "$CONFIG_DIR/logs"
    $SUDO mkdir -p "$DATA_DIR" "$LOG_DIR"
    log_ok "Directories created"
}

generate_config() {
    local config_file="${CONFIG_DIR}/config.json"
    if [ -f "$config_file" ]; then
        log_warn "Config already exists at ${config_file}, skipping"
        return
    fi

    log_info "Generating default configuration..."
    cat > "$config_file" << 'EOF'
{
  "mode": "local",
  "listen_addr": "127.0.0.1",
  "listen_port": 1080,
  "server_addr": "",
  "server_port": 443,
  "encryption": {
    "algorithm": "aes-256-gcm",
    "pq_enabled": true,
    "zkp_enabled": true
  },
  "stealth": {
    "enabled": true,
    "profile": "chrome",
    "jitter_ms": 50,
    "padding_enabled": true,
    "decoy_enabled": false,
    "tls_fingerprint": "chrome"
  },
  "api": {
    "enabled": true,
    "port": 8080,
    "host": "127.0.0.1",
    "auth_key": ""
  },
  "dns": {
    "enabled": false,
    "port": 5353,
    "upstream": "8.8.8.8:53"
  },
  "log_level": "info",
  "log_file": "",
  "features": {
    "compression": true,
    "connection_pool": true,
    "smart_routing": true,
    "mesh_network": false,
    "offline_first": false,
    "ai_adapter": true,
    "dashboard": true,
    "auto_proxy": false
  },
  "transport": {
    "mode": "auto",
    "sni_list": ["www.google.com", "cdn.cloudflare.com", "aws.amazon.com"],
    "cdn_list": ["cdn.cloudflare.com", "fastly.net"],
    "fallback": true
  }
}
EOF
    log_ok "Configuration at ${config_file}"
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
if ! download_from_release; then
    log_info "Falling back to build from source..."
    build_from_source
fi

generate_config
setup_systemd

echo ""
echo -e "${PURPLE}${BOLD}  ╔════════════════════════════════════════════════════╗${RESET}"
echo -e "${PURPLE}${BOLD}  ║  SOVA Protocol v${VERSION} installed successfully!       ║${RESET}"
echo -e "${PURPLE}${BOLD}  ╚════════════════════════════════════════════════════╝${RESET}"
echo ""
log_info "Client:     sova                     (SOCKS5 tunnel)"
log_info "Server:     sova-server              (relay server)"
log_info "API:        http://127.0.0.1:8080/api/"
log_info "Config:     ${CONFIG_DIR}/config.json"
log_info "Proxy:      SOCKS5 127.0.0.1:1080"
echo ""
log_info "Quick start:"
echo "  sova                               # Start tunnel"
echo "  sova connect server.example.com    # Remote server"
echo "  sova help                          # All commands"
echo ""