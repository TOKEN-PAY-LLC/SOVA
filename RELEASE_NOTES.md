# Release Notes

## v2.1.0 — March 2026

### Mobile ISP Bypass — WebSocket Relay

Mobile ISPs block non-standard ports (9443, etc.), causing SOVA timeout on cellular data.
v2.1.0 adds native WebSocket relay support, enabling traffic to flow through nginx:443
as standard HTTPS — invisible to ISP DPI.

**Architecture:**
```
Client (Hiddify/sing-box) → cupol.space:443/sova-ws (HTTPS WebSocket)
     → nginx (TLS termination + WS upgrade)
     → xray VMess+WS inbound (127.0.0.1:9444)
     → internet
```

### Changes

- **WebSocket relay** — `relay.go` now supports WS connections via `gorilla/websocket`
- **WSConn adapter** — `net.Conn`-compatible wrapper for WebSocket, enabling reuse of all existing relay logic
- **Server config** — new `websocket` section with `enabled`, `port`, `path` fields
- **Auto-enable** — WS relay starts automatically when transport mode is `websocket` or `auto`
- **Health endpoint** — `/health` on WS relay port returns JSON status
- **CUPOL integration** — CSUB now injects `CUPOL-SOVA` as VMess+WS+TLS outbound on port 443
- **Streisand support** — `vmess://` share link generated for V2RayNG/Streisand clients
- **Dashboard** — version badge updated to v2.1, mobile-safe subtitle added
- **Protocol switch** — SOVA moved from Shadowsocks AEAD to VMess+WS+TLS (sing-box doesn't support transport on SS)

### Why VMess instead of Shadowsocks?

sing-box 1.x does NOT support the `transport` field on shadowsocks outbound/inbound.
VMess+WebSocket is universally supported by xray (server) and sing-box/Hiddify (client),
making it the most reliable choice for mobile ISP bypass.

---

## v1.0.0 — March 2026

```
         ▄▄▄████▄▄▄
       ▄██▀▀    ▀▀██▄
      ███  ◉    ◉  ███     SOVA Protocol v1.0.0
      ███    ▾▾    ███     Production Release
       ▀██▄▄▄▄▄▄██▀
      ╱╱ ▀████████▀ ╲╲
     ╱╱   ║██████║   ╲╲
    ▕▕    ║██████║    ▏▏
           ║║  ║║
          ▄╩╩▄▄╩╩▄
```

### Highlights

- **Flying purple owl animation** — the owl flies across the terminal at startup with wing flapping, sparkle trail, and gradient purple 256-color ANSI
- **18 REST API endpoints** — full management with CORS, API key auth, profiles, logs, stats
- **15 toggleable modules** — PQ crypto, stealth, AI adapter, mesh, DNS, and more
- **Config profiles** — save/load/switch via CLI and API
- **Verified working** — SOCKS5 proxy (HTTP 200 through tunnel), all 18 API endpoints tested

---

### New in v1.0.0

#### Terminal UI
- **Animated flying owl** — 3-frame flight animation (wings up / glide / wings down) across terminal width
- **Sparkle trail** — fading `✦✧⋆˚·∗⊹✶✵⁺` stars behind the owl
- **Landing animation** — owl appears line-by-line, blinks, looks left/right
- **256-color purple gradient** — 8 purple shades from dark to lavender
- **Beautiful banner box** — `╔═══╗` styled box with version and tagline
- **Rich status output** — purple-themed sections, key-value pairs, progress bars

#### Management API (18 endpoints)
- `GET /api/status` — system status with traffic stats
- `GET /api/health` — health check
- `GET /api/config` — full config JSON
- `PUT /api/config` — update full config
- `POST /api/config/set` — set single key
- `POST /api/config/reset` — reset to defaults
- `GET /api/features` — all 15 modules on/off
- `POST /api/feature/` — toggle module
- `GET /api/system` — CPU, RAM, GC, goroutines
- `GET /api/stats` — traffic statistics
- `GET /api/logs` — log entries (with `?limit=N`)
- `GET /api/profiles` — saved config profiles
- `POST /api/profile` — switch profile
- `POST /api/profile/save` — save current as profile
- `POST /api/restart` — schedule restart
- `GET /api/transport` — transport info (mode, SNI, CDN)
- `GET /api/encryption` — encryption details (algorithm, PQ, ZKP)
- `GET /api/stealth` — stealth engine info
- **CORS middleware** — `Access-Control-Allow-Origin: *`
- **API key auth** — `X-API-Key` header or `?api_key=` query param

#### Configuration System
- 14 configurable keys via CLI and API
- 15 toggleable modules: `pq_crypto, zkp, stealth, padding, decoy, ai_adapter, compression, connection_pool, smart_routing, mesh_network, offline_first, dns, api, dashboard, auto_proxy`
- Persistent JSON config at `~/.sova/config.json`
- Config profiles: save, load, switch, list

#### Traffic Acceleration
- **Gzip compression** — automatic traffic compression
- **Connection pooling** — reuse of idle connections
- **Route optimizer** — latency/bandwidth/loss scoring
- **Accelerated read/write** — framed protocol with compressed packets

#### Stealth Engine
- **Traffic mimicry** — Chrome, YouTube, Cloud API profiles
- **Adaptive jitter** — Box-Muller normal distribution
- **Intelligent padding** — standard HTTP packet sizes
- **Decoy traffic** — background keep-alive packets
- **TLS fingerprint masking** — Chrome, Firefox, Safari fingerprints

#### SOCKS5 Proxy
- Built-in SOCKS5 proxy on `127.0.0.1:1080`
- Remote tunnel mode via `sova connect <server>`
- Connection stats tracking (bytes up/down, active/total)

#### DNS-over-SOVA
- Encrypted DNS resolver with local caching
- Configurable upstream and port
- Fallback to system resolver

#### Web Dashboard
- Purple-themed dashboard at `http://localhost:8080`
- Real-time stats, connections, logs

#### Encryption
- **AES-256-GCM** + **ChaCha20-Poly1305**
- **Kyber1024** KEM (post-quantum key exchange)
- **Dilithium mode5** (post-quantum signatures)
- **Zero-Knowledge Proof** auth on Ed25519

#### Code Quality
- Fixed circl API usage (Kyber1024 via `Scheme()`, Dilithium via `mode5`)
- 58+ unit tests covering all major components
- All 4 packages compile cleanly: `common`, `server`, `client`, `plugin`
- `go vet` and `go build` pass with zero warnings

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
├── sova-server-windows-arm64.exe
├── sova-server-macos-amd64
├── sova-server-macos-arm64
├── sova-linux-amd64
├── sova-linux-arm64
├── sova-windows-amd64.exe
├── sova-windows-arm64.exe
├── sova-macos-amd64
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
go test -v ./common/           # 58+ tests
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
