# Contributing to SOVA Protocol

## Development Setup

### Prerequisites
- Go 1.21+
- Git
- Make (optional, for Makefile targets)

### Clone and Build

```bash
git clone https://github.com/IvanChernykh/SOVA.git
cd SOVA
go mod tidy
go build ./server/
go build ./client/
```

Or use Make:

```bash
make build            # server + client for current OS
make build-all        # all platforms (linux/windows/macos, amd64/arm64)
make build-server     # server only
make build-client     # client only
```

### Testing

```bash
go test -v ./common/           # all tests (44+)
go test -bench=. ./common/     # benchmarks
make test                      # with coverage report
```

### Code Formatting

```bash
go fmt ./...
```

---

## Project Structure

```
SOVA/
├── server/
│   ├── main.go          # Server entry point, TLS listener, ZKP auth
│   ├── api.go           # REST API endpoints
│   ├── dashboard.go     # Web dashboard (purple theme)
│   ├── config.go        # Configuration structs
│   └── middleware.go     # Rate limiting, logging
├── client/
│   └── main.go          # Client CLI (connect, status, proxy)
├── common/
│   ├── crypto.go        # AES-GCM, ChaCha20, Kyber1024, Dilithium mode5
│   ├── auth.go          # ZKP authentication, config encoding
│   ├── transport.go     # Transport modes, adaptive switching
│   ├── ai.go            # AI adapter for DPI evasion
│   ├── accelerator.go   # Traffic compression, pooling, routing
│   ├── stealth.go       # Traffic mimicry, jitter, padding, decoy
│   ├── socks5.go        # SOCKS5 proxy server
│   ├── dns.go           # DNS-over-SOVA resolver
│   ├── ui.go            # Terminal UI (purple theme, owl)
│   ├── quic_transport.go
│   ├── websocket_transport.go
│   ├── custom_handshake.go
│   ├── offline_first.go
│   └── *_test.go        # Tests for all modules
├── plugin/
│   └── xray_plugin.go   # Xray/Sing-Box/V2Ray integration
├── install.sh           # Linux/macOS installer
├── install.ps1          # Windows installer
├── Makefile             # Build automation
├── go.mod / go.sum      # Go modules
└── *.md                 # Documentation
```

---

## Key Modules

| Module | File | Description |
|---|---|---|
| Crypto | `crypto.go` | AES-256-GCM, ChaCha20, Kyber1024 KEM, Dilithium mode5 |
| Auth | `auth.go` | Ed25519 ZKP, config encode/decode |
| AI | `ai.go` | Event recording, strategy matching, adaptive switching |
| Accelerator | `accelerator.go` | Gzip compression, connection pool, route optimizer |
| Stealth | `stealth.go` | Traffic mimicry, jitter, padding, decoy generation |
| Transport | `transport.go` | TLS/QUIC/WebSocket with AI-driven switching |
| SOCKS5 | `socks5.go` | Built-in SOCKS5 proxy |
| DNS | `dns.go` | DNS-over-SOVA with caching |

---

## Code Style

- Follow standard Go conventions (`gofmt`)
- Add tests for new features
- Document exported functions and types
- Keep imports organized (stdlib, external, internal)

---

## Issue Reporting

Report bugs via **GitHub Issues**: https://github.com/IvanChernykh/SOVA/issues

Include:
- Go version (`go version`)
- OS and architecture
- Error logs or stack trace
- Steps to reproduce

**Security vulnerabilities**: use [GitHub Security Advisories](https://github.com/IvanChernykh/SOVA/security/advisories/new) (not public issues).

---

## Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make changes and add tests
4. Run `go test -v ./...` to verify
5. Submit PR with clear description
6. Wait for review

---

## License

MIT License — see [LICENSE](LICENSE).
