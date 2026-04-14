# Contributing to SOVA Protocol

## Development Setup

### Prerequisites
- Go 1.21+
- Git
- Make (optional)

### Clone and Build

```bash
git clone https://github.com/IvanChernykh/SOVA.git
cd SOVA
go mod tidy
go build ./...
```

Optional Make targets:

```bash
make build
make build-server
make build-client
make build-all
```

### Testing

```bash
go test -v ./common/
go test -bench=. ./common/
```

### Formatting

```bash
go fmt ./...
```

---

## Architecture Focus

Contributions should strengthen the native SOVA stack:

- **SOVA Proxy** — local ingress for applications and browsers
- **SOVA Protocol** — encrypted relay framing and handshake
- **SOVA Relay** — target dialing, ACK flow, transport orchestration
- **Stealth / AI** — DPI resistance, adaptation, traffic shaping
- **Dashboard / CLI** — onboarding, status, UX, observability

---

## Project Structure

```text
SOVA/
├── server/              # Relay server, dashboard, API, middleware
├── client/              # Interactive CLI and local SOVA Proxy control
├── common/              # Shared protocol, crypto, config, UI, transports
├── plugin/              # Reserved area for future integration adapters
├── singbox-patch/       # Reference patch materials and build notes
├── install.sh
├── install.ps1
├── Makefile
└── *.md
```

Key files to understand first:

- `common/sova_proxy.go`
- `common/protocol.go`
- `common/dpi.go`
- `client/main.go`
- `server/relay.go`
- `server/api.go`

---

## Contribution Guidelines

- Follow standard Go conventions and keep code `gofmt`-clean.
- Prefer native SOVA abstractions over adding third-party protocol dependencies.
- Add or update tests when changing protocol, crypto, relay, or config logic.
- Keep imports grouped and minimal.
- Preserve product terminology: `SOVA Proxy`, `SOVA Protocol`, `SOVA VPN`.

---

## Good Areas for Contribution

- Protocol correctness and fuzzing
- Transport resilience and performance
- Dashboard / CLI polish
- Config ergonomics and validation
- Documentation for native SOVA integrations
- SDK and share-link specification work

---

## Reporting Issues

Report bugs via **GitHub Issues**: https://github.com/IvanChernykh/SOVA/issues

Useful information:

- Go version
- OS and architecture
- SOVA version
- logs / stack trace
- exact reproduction steps

Security issues should go through [GitHub Security Advisories](https://github.com/IvanChernykh/SOVA/security/advisories/new).

---

## Pull Requests

1. Fork the repository.
2. Create a feature branch.
3. Make the change with tests where appropriate.
4. Run `go build ./...` and relevant tests.
5. Submit a focused PR with a clear description.

---

## License

MIT License — see [LICENSE](LICENSE).
