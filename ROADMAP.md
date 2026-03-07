# SOVA Protocol Roadmap

## Vision

SOVA — самый быстрый, умный и невидимый протокол для свободного интернета. Бесплатный, с открытым кодом, управляемый сообществом.

---

## v1.0.0 ✅ [Released — March 2026]

- [x] **Flying purple owl animation** — 3-frame flight, sparkle trail, 256-color ANSI
- [x] **18 REST API endpoints** — CORS, API key auth, profiles, logs, stats, encryption, stealth
- [x] **15 toggleable modules** — PQ crypto, stealth, AI, mesh, DNS, compression, etc.
- [x] **Configuration profiles** — save/load/switch via CLI and API
- [x] Traffic acceleration (compression, pooling, route optimizer)
- [x] Stealth engine (traffic mimicry, jitter, padding, decoy, TLS fingerprint)
- [x] Web dashboard (purple theme, real-time stats)
- [x] SOCKS5 proxy server (verified HTTP 200 through tunnel)
- [x] DNS-over-SOVA resolver with configurable upstream
- [x] Post-quantum crypto: Kyber1024 KEM + Dilithium mode5 (circl v1.3.7)
- [x] Zero-Knowledge Proof authentication (Ed25519)
- [x] AES-256-GCM + ChaCha20-Poly1305 encryption
- [x] 3 transport modes: Web Mirror, QUIC, WebSocket + Auto mode
- [x] AI-adaptive transport switching
- [x] Offline-first architecture + mesh networking
- [x] Animated owl installers (Linux/macOS/Windows)
- [x] Cross-platform builds (amd64, arm64)
- [x] 58+ unit tests + benchmarks
- [x] Full documentation (README, RELEASE_NOTES, SECURITY, CONTRIBUTING)
- [x] CI/CD GitHub Actions workflow (auto-build + release on tag push)

---

## v1.1.0 🚀 [Q2 2026]

### Performance
- [ ] TUN/TAP mode for full VPN capability
- [ ] Split tunneling
- [ ] IPv6 support
- [ ] HTTP/3 multiplexing

### Client
- [ ] System tray integration (Windows/macOS)
- [ ] Bandwidth monitoring graph
- [ ] Auto-reconnect with exponential backoff
- [ ] Profile sync across devices

---

## v2.0.0 📦 [Q3 2026]

### Plugin System
- [ ] Public plugin API
- [ ] Full Xray/V2Ray integration
- [ ] Sing-Box native support
- [ ] Custom transport modules

### Mobile
- [ ] Android app
- [ ] iOS app
- [ ] Config sync across devices

---

## v3.0.0 🌐 [Q4 2026]

### Decentralization
- [ ] P2P node infrastructure
- [ ] Multi-hop routing
- [ ] Load balancing
- [ ] Advanced ML-based DPI evasion
- [ ] Hardware security module (HSM) support

### New Transport Modes
- [ ] RTP/VoIP masquerade
- [ ] Game packet morphing
- [ ] IoT protocol tunneling

---

## Version Policy

- **Major** (v2.0.0): Breaking changes, architecture redesign
- **Minor** (v1.1.0): New features, backward-compatible
- **Patch** (v1.0.1): Bug fixes, security patches

Security fixes are released immediately.

---

## Contributing

We welcome contributions! Priority areas:
1. **Testing** — platform-specific, edge cases
2. **Performance** — optimization patches
3. **Features** — new transports, strategies
4. **Security** — vulnerability research

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## Feedback

- **GitHub Issues**: https://github.com/IvanChernykh/SOVA/issues (label: `roadmap`)
- **Discussions**: https://github.com/IvanChernykh/SOVA/discussions

---

*Last updated: March 2026*
