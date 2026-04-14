# Release Notes

## v1.0.0 — March 2026 (Autonomous Baseline)

This repository remains on **v1.0.0** while the native SOVA stack is being hardened. The current state is positioned as the stable autonomous baseline, not as a finalized feature jump beyond that baseline.

---

## What defines the v1.0.0 baseline

### Native SOVA runtime

- **SOVA Proxy** — local HTTP CONNECT / plain HTTP ingress for browsers and applications
- **SOVA Protocol** — encrypted frame-based relay with native handshake and ACK flow
- **SOVA Relay Server** — direct target dialing and bidirectional relay on the server side
- **SOVA WebSocket Relay** — WebSocket transport carrying the same native SOVA protocol

### Security and stealth

- AES-256-GCM transport framing
- TLS camouflage with SNI spoofing
- ClientHello fragmentation and jitter for DPI resistance
- Random padding and traffic shaping support
- Kyber1024 + Dilithium mode5 primitives
- Zero-Knowledge authentication path

### Product surface

- Refreshed terminal banner and owl-themed SOVA UI
- Updated dashboard branding for `v1.0.0`
- Native SOVA profile export
- Native `sova://` share-link generation
- Upstream chaining moved to **SOVA gateway** semantics

---

## Key cleanup in this revision

- Removed active legacy ingress handling from the local proxy path
- Switched WebSocket relay handling to native SOVA framing end-to-end
- Removed external client-config export paths from the server API
- Reverted public versioning and UI output back to **v1.0.0**
- Refreshed terminology across code and UI to emphasize:
  - `SOVA Proxy`
  - `SOVA Protocol`
  - `SOVA VPN`

---

## Operational highlights

- `go build ./...` passes for the repository baseline
- Local proxy path is aligned with SOVA-native behavior
- Web dashboard and server health output are aligned with `v1.0.0`
- Client menu/help output no longer presents legacy protocol branding

---

## Notes for the next public release

Before promoting a new public version above `v1.0.0`, the recommended next milestones are:

- publish a formal SOVA protocol specification;
- stabilize third-party SDK / integration guidance;
- finish UI polish and onboarding flow;
- expand interoperability tests for native SOVA clients.

---

## Support

- **Issues**: https://github.com/IvanChernykh/SOVA/issues
- **Discussions**: https://github.com/IvanChernykh/SOVA/discussions
- **Security**: see [SECURITY.md](SECURITY.md)

---

**SOVA stays on v1.0.0 until the next release is genuinely stronger.**
