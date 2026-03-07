# Release Notes - v1.0.0

## 🚀 SOVA Protocol v1.0.0 - Production Ready

**Release Date:** December 2025

### Overview
SOVA Protocol v1.0.0 is the first production release of an open, DPI-resistant, AI-adaptive protocol for secure data transmission. This release includes full end-to-end encryption, Zero-Knowledge Proof authentication, and post-quantum cryptography support.

### ✨ New Features

#### Core Protocol
- **Dynamic Traffic Masking**: Masquerade traffic under legitimate HTTPS, QUIC, and WebSocket protocols
- **AI-Adaptive Transport**: Intelligent switching between three transport modes based on network conditions
- **Post-Quantum Ready**: Integrated Kyber1024 KEM and Dilithium5 signatures via Cloudflare's circl
- **Zero-Knowledge Proof**: Non-interactive ZKP authentication without password transmission

#### Security
- **Master Key Architecture**: Server generates and securely stores a master key; all session keys derived per-connection
- **Encryption**: AES-256-GCM + ChaCha20-Poly1305 for symmetric encryption
- **Digital Signatures**: Ed25519 Schnorr signatures + post-quantum Dilithium5
- **Security Policy**: See SECURITY.md for vulnerability disclosure and best practices

#### Client & Server
- **Cross-Platform**: Windows, macOS, Linux (AMD64, ARM64) clients and servers
- **CLI Interface**: Rich terminal UI with SOVA owl logo
- **REST API**: Full-featured HTTP API for configuration, monitoring, and proxy export
  - `/api/register` - User registration
  - `/api/config` - Configuration export
  - `/api/status` - Server statistics
  - `/api/export` - Client-specific configs (Xray, Sing-Box)
  - `/api/proxy` - Ready-made proxy links

#### Autonomous Installation
- **Zero Dependency**: Pre-compiled static binaries; no Go, Python, or runtime required
- **Single Command Install**: 
  ```bash
  curl -sSL https://github.com/IvanChernykh/SOVA/releases/download/v1.0.0/install.sh | bash
  ```
- **Auto-Platform Detection**: Installation script detects OS and architecture
- **Service Management**: Automatic daemon/service setup on installation

#### Features
- **Transport Modes**:
  1. Web Mirror Mode - Custom TLS handshake with fingerprint variation
  2. QUIC Mode - UDP-based with jitter and adaptive congestion control
  3. WebSocket Mode - CDN-compatible tunneling with IP rotation

- **Development Tools**:
  - `Makefile` for cross-platform compilation
  - GitHub Actions for automated CI/CD
  - Comprehensive unit tests (crypto, auth, transport)
  - Performance benchmarks included

### 📋 What's Included

```
dist/
├── sova-server-linux-amd64          # Linux server
├── sova-server-linux-arm64          # Linux ARM server
├── sova-server-windows-amd64.exe    # Windows server
├── sova-server-windows-arm64.exe    # Windows ARM server
├── sova-server-macos-amd64          # macOS server
├── sova-server-macos-arm64          # macOS ARM server
├── sova-linux-amd64                 # Linux client
├── sova-linux-arm64                 # Linux ARM client
├── sova-windows-amd64.exe           # Windows client
├── sova-windows-arm64.exe           # Windows ARM client
├── sova-macos-amd64                 # macOS client
├── sova-macos-arm64                 # macOS ARM client
├── install.sh                        # Unix installer
└── install.ps1                       # Windows installer
```

### 🔧 Dependencies

**Runtime**: None (static binaries)

**Build Dependencies** (for developers):
- Go 1.21+
- Cloudflare circl (github.com/cloudflare/circl v1.3.7)
- quic-go (github.com/quic-go/quic-go v0.40.1)
- gorilla/websocket (github.com/gorilla/websocket v1.5.1)
- golang.org/x/crypto v0.20.0

### 📚 Documentation

- **README.md**: Complete protocol documentation
- **SECURITY.md**: Security policy, vulnerability disclosure, and best practices
- **CONTRIBUTING.md**: Development guide for contributors
- **API Documentation**: Inline in server/api.go

### 🧪 Testing

Run the test suite:
```bash
make test                   # All tests
make test-crypto           # Cryptography only
make test-ai               # AI adapter only
make bench                  # Performance benchmarks
```

Test coverage includes:
- AES-GCM encryption/decryption
- Post-quantum encapsulation/decapsulation
- Post-quantum signatures
- Session key derivation (HMAC-SHA256)
- AI adaptation strategies
- API endpoints

### 🔐 Security Highlights

1. **Zero-Knowledge Authentication**: Passwords proven without transmission
2. **Server-Only Master Key**: Private key never leaves server
3. **Per-Connection Session Keys**: Each client gets unique derived key
4. **Post-Quantum Ready**: Kyber1024 + Dilithium5 included
5. **DPI Resistance**: Multiple transport modes with dynamic switching
6. **Regular Audits**: Security reviews publish to /SECURITY_AUDIT/
7. **Bug Bounty**: Incentivized responsible disclosure

### 🚀 Getting Started

#### Install Server
```bash
curl -sSL https://github.com/IvanChernykh/SOVA/releases/download/v1.0.0/install.sh | bash
sova install  # Initialize and generate keys
```

#### Install Client
```bash
# On client machine
curl -sSL https://github.com/IvanChernykh/SOVA/releases/download/v1.0.0/install.sh | bash
sova connect <json_config>
```

#### Usage
```bash
# Server status
sova status

# List proxies
sova proxy

# Get configuration
sova config <user_id>

# Connect
sova connect <base64_config>
```

### 🛠️ Building from Source

```bash
git clone https://github.com/IvanChernykh/SOVA.git
cd SOVA
make install-deps
make build-all
```

### 📝 Known Limitations

- Mobile platforms (Android/iOS) require forked client build (gomobile)
- TUN/TAP mode planned for v1.1.0
- Plugin API expansion planned for v1.2.0
- Advanced ML-based adaptation in post-v1.0

### 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup
- Code style guidelines
- Testing procedures
- Pull request process

### 📞 Support

- **Issues**: https://github.com/IvanChernykh/SOVA/issues
- **Security**: security@sova.io
- **Documentation**: https://github.com/IvanChernykh/SOVA#readme

### 📄 License

SOVA Protocol is licensed under the MIT License.

### 🙏 Acknowledgments

- Cloudflare for the `circl` post-quantum library
- QUIC-go maintainers for QUIC transport
- Go community for excellent tooling

---

**Thank you for using SOVA Protocol!** 🦉
