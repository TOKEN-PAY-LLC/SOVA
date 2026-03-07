# Release Notes

## v1.0.0 — March 2026

```
    ,___,
    {o,o}    SOVA Protocol v1.0.0
    /)  )    Production Release
    -"  "-
```

### New in v1.0.0

#### Traffic Acceleration
- **Gzip compression** — automatic traffic compression with intelligent threshold
- **Connection pooling** — reuse of idle connections to reduce handshake overhead
- **Route optimizer** — selects fastest route based on latency/bandwidth/loss scoring
- **Accelerated read/write** — framed protocol with length-prefixed compressed packets

#### Stealth Engine
- **Traffic mimicry** — profiles for Chrome HTTPS, YouTube streaming, Cloud API calls
- **Adaptive jitter** — Box-Muller normal distribution for realistic timing
- **Intelligent padding** — pads packets to standard HTTP sizes (64, 128, 256, 512, 1024, 1460 bytes)
- **Decoy traffic** — background keep-alive packets to prevent idle detection
- **TLS fingerprint masking** — cipher suite and extension lists matching popular browsers

#### Web Dashboard
- Purple-themed dashboard at `http://localhost:8080`
- Real-time server stats, active connections, logs
- REST API endpoints for monitoring

#### SOCKS5 Proxy
- Built-in SOCKS5 proxy server for universal app compatibility
- Connection stats tracking (bytes up/down)

#### DNS-over-SOVA
- Encrypted DNS resolver with local caching
- Fallback to system resolver on failure
- Cache stats and management

#### Installer Upgrade
- **Animated owl** ASCII art during installation
- Platform auto-detection (Linux/macOS/Windows, amd64/arm64/armv7/386)
- Download with fallback to build-from-source
- Automatic systemd (Linux) / Windows Service setup

#### Code Quality
- Removed dead placeholder PQ functions from `auth.go` (real PQ crypto in `crypto.go`)
- Fixed circl API usage (Kyber1024 via `Scheme()`, Dilithium via `mode5`)
- Removed unused imports across codebase
- 44+ unit tests covering all major components

### Security Improvements
- Real post-quantum crypto (Kyber1024 KEM + Dilithium mode5 signatures)
- Rate limiting middleware on all API endpoints
- Input validation on all REST handlers
- Documented `InsecureSkipVerify` usage with PQ key verification

---

### Installation

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/IvanChernykh/SOVA/main/install.sh | bash

# Windows (Admin PowerShell)
powershell -ExecutionPolicy Bypass -Command "iwr -useb https://raw.githubusercontent.com/IvanChernykh/SOVA/main/install.ps1 -OutFile install.ps1; .\install.ps1"

# Build from source
git clone https://github.com/IvanChernykh/SOVA.git && cd SOVA
go mod tidy && make build-all
```

### Build Artifacts

```
dist/
├── sova-server-linux-amd64
├── sova-server-linux-arm64
├── sova-server-windows-amd64.exe
├── sova-server-macos-arm64
├── sova-linux-amd64
├── sova-linux-arm64
├── sova-windows-amd64.exe
├── sova-macos-arm64
├── install.sh
└── install.ps1
```

### Dependencies (build only)

| Module | Version |
|---|---|
| Go | 1.21+ |
| cloudflare/circl | v1.3.7 |
| quic-go/quic-go | v0.40.1 |
| gorilla/websocket | v1.5.1 |
| golang.org/x/crypto | v0.20.0 |

Runtime: **none** (static binaries).

### Testing

```bash
go test -v ./common/           # 44+ tests
go test -bench=. ./common/     # benchmarks
```

### Support

SOVA is **100% free**. No paid plans, no premium features.

- **Issues**: https://github.com/IvanChernykh/SOVA/issues
- **Discussions**: https://github.com/IvanChernykh/SOVA/discussions
- **Security**: see [SECURITY.md](SECURITY.md)

### License

MIT License.

### Acknowledgments

- Cloudflare — `circl` post-quantum library
- quic-go maintainers
- Go community

---

**Thank you for using SOVA!** 🦉
