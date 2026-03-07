# CONTRIBUTING.md

## Development Setup

### Prerequisites
- Go 1.21 or higher
- Make
- Git
- Standard C compiler (for circl library)

### Clone and Setup
```bash
git clone https://github.com/IvanChernykh/SOVA.git
cd SOVA
go mod download
go mod tidy
```

### Building

#### Build for current platform
```bash
make build
```

#### Build all platforms
```bash
make build-all
```

Individual builds:
```bash
make build-server          # Build server only
make build-client          # Build client only
make build-linux-amd64     # Build for Linux AMD64
make build-windows-amd64   # Build for Windows AMD64
make build-macos-arm64     # Build for macOS ARM64
```

### Testing

```bash
# Run all tests
make test

# Run specific test suites
make test-crypto
make test-ai
make test-integration

# Run benchmarks
make bench
```

### Code Quality

```bash
# Format code
make fmt

# Lint code
make lint
```

### Creating a Release

1. Update version in code if needed
2. Create a new branch `release/vX.Y.Z`
3. Commit changes
4. Create annotated tag:
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z: Description"
   ```
5. Push tag:
   ```bash
   git push origin vX.Y.Z
   ```
6. GitHub Actions will automatically build binaries and create a release

### Project Structure

```
c:\Users\user\Desktop\SOVA\
├── server/              # Server implementation
│   ├── main.go         # Server entry point
│   ├── api.go          # REST API endpoints
│   ├── config.go       # Configuration
│   └── middleware.go   # HTTP middleware
├── client/             # Client implementation
│   └── main.go         # Client CLI
├── common/             # Shared libraries
│   ├── crypto.go       # Cryptography (AES, PQ)
│   ├── crypto_test.go  # Crypto tests
│   ├── transport.go    # Transport modes
│   ├── auth.go         # Authentication
│   ├── ai.go          # AI adapter
│   ├── ui.go          # Terminal UI
│   └── ...            # Other modules
├── plugin/            # Plugin support
│   └── xray_plugin.go # Xray integration
├── go.mod             # Go module file
├── Makefile           # Build automation
├── LICENSE            # MIT License
├── SECURITY.md        # Security policy
└── README.md          # Documentation
```

## Code Style

- Follow Go standard code conventions
- Use `gofmt` for formatting
- Add tests for new features
- Document public functions and types

## Key Modules

### Cryptography (`common/crypto.go`)
- AES-256-GCM for session encryption
- Post-quantum Kyber1024 for KEM
- Dilithium5 for signatures
- Uses Cloudflare's `circl` library

### AI Adapter (`common/ai.go`)
- Network monitoring
- Adaptive transport switching
- Event-based decision making

### Transports (`common/*_transport.go`)
- Web Mirror Mode: Custom TLS handshake
- QUIC: Cloud-resistant UDP protocol
- WebSocket: CDN-compatible tunneling

## Security Considerations

- Master key is server-only and generated at init
- Session keys derived per-connection using HMAC-SHA256
- All traffic encrypted with AES-GCM
- Zero-Knowledge Proof for authentication
- Post-quantum ready with Kyber/Dilithium

## Issue Reporting

Report security issues to: security@sova.io
Report bugs on GitHub issues with:
- Go version
- OS and architecture
- Error logs or stack trace
- Steps to reproduce

## Pull Requests

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit PR with description
6. Ensure CI/CD passes

## License

SOVA Protocol is licensed under the MIT License - see [LICENSE](LICENSE) file.
