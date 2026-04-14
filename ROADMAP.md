# SOVA Protocol Roadmap

## Vision

SOVA — автономная сеть нового поколения: собственный локальный прокси, собственный зашифрованный протокол, собственная серверная релейная модель и собственный UX-слой без обязательной зависимости от сторонних транспортов и брендов.

---

## v1.0.0 ✅ [Current Baseline — March 2026]

- [x] **SOVA Proxy** — локальный HTTP CONNECT / plain HTTP ingress для приложений и браузеров
- [x] **SOVA Protocol** — нативный фреймовый протокол поверх TLS с AES-256-GCM
- [x] **SOVA WebSocket Relay** — тот же SOVA protocol поверх WebSocket для обхода DPI
- [x] **AI + Stealth stack** — SNI spoofing, fragmentation, jitter, padding, adaptive switching
- [x] **Post-quantum security** — Kyber1024 + Dilithium mode5
- [x] **Zero-Knowledge authentication path**
- [x] **Management API + Dashboard**
- [x] **Config profiles и модульная конфигурация**
- [x] **DNS-over-SOVA**
- [x] **Purple owl terminal UI / dashboard refresh**
- [x] **Native SOVA share link и profile export**

---

## v1.1.0 🚀 [Stabilization Track]

### Core
- [ ] TUN/TAP mode for full-device SOVA VPN routing
- [ ] Split tunneling policy engine
- [ ] IPv6 parity across client and relay
- [ ] Better multi-hop orchestration between SOVA gateways

### UX
- [ ] Native desktop tray integration
- [ ] Session telemetry graphs in dashboard
- [ ] Automatic reconnect with smarter backoff
- [ ] Profile sync and backup workflow

### Developer Adoption
- [ ] Public protocol specification
- [ ] Share-link schema documentation
- [ ] Minimal Go SDK for native SOVA clients
- [ ] Example integrations for browser/system proxy consumers

---

## v2.0.0 📦 [Platform Expansion]

### SDK and Ecosystem
- [ ] Stable SOVA SDK for external developers
- [ ] Native mobile client reference apps
- [ ] Public extension points for transports and policy engines
- [ ] Formal compatibility test suite for third-party implementers

### Performance
- [ ] HTTP/3-style multiplexing improvements
- [ ] Better congestion adaptation for unstable mobile links
- [ ] Advanced route scoring and relay selection
- [ ] Persistent encrypted session resumption

---

## v3.0.0 🌐 [Network Layer]

### Distributed Topology
- [ ] Federated relay mesh
- [ ] Load-aware multi-hop routing
- [ ] Community-operated edge nodes
- [ ] Hardware-backed key isolation support

### AI Networking
- [ ] Real adaptive traffic shaping based on live network conditions
- [ ] Smarter cover traffic profiles
- [ ] Dynamic anti-censorship strategies per region

---

## Version Policy

- **Major** — architectural shifts or protocol-level breaking changes
- **Minor** — new features without breaking the native SOVA baseline
- **Patch** — stability, UX, and security refinements inside the current baseline

At the moment, the repository baseline remains **v1.0.0** until the next release is stable enough to justify a newer public version.

---

## Contributing Priorities

1. **Protocol hardening** — tests, fuzzing, edge cases
2. **Performance** — latency, memory, relay throughput
3. **Developer tooling** — SDKs, examples, integration docs
4. **UX polish** — client flows, dashboard, onboarding

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution details.

---

## Feedback

- **GitHub Issues**: https://github.com/IvanChernykh/SOVA/issues
- **Discussions**: https://github.com/IvanChernykh/SOVA/discussions

---

*Last updated: March 2026*
