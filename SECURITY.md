# Security Policy

## Reporting Vulnerabilities

If you discover a security vulnerability in SOVA Protocol, please report it through **GitHub Security Advisories**:

https://github.com/IvanChernykh/SOVA/security/advisories/new

**Do not open public issues for security vulnerabilities.** Use the private advisory feature instead.

Include in your report:
1. **Description** of the vulnerability
2. **Affected component** (server, client, crypto, transport, etc.)
3. **Steps to reproduce**
4. **Potential impact**
5. **Suggested fix** (optional)

We will acknowledge your report within 72 hours and work on a fix.

---

## Cryptographic Architecture

### Symmetric Encryption
- **AES-256-GCM** — primary session encryption
- **ChaCha20-Poly1305** — alternative AEAD cipher (XChaCha20 variant)
- Session keys derived via HMAC-SHA256 from master key + per-connection nonce

### Post-Quantum Cryptography
- **Kyber1024** (KEM) — key encapsulation via Cloudflare `circl`
- **Dilithium mode5** (signatures) — post-quantum digital signatures via `circl`
- Both algorithms are NIST PQC Round 3 finalists

### Authentication
- **Zero-Knowledge Proof** on Ed25519 (Schnorr-like)
- Passwords never transmitted — only proofs
- Nonce-based challenge-response (non-replayable)

### Obfuscation (Stealth Engine)
- Traffic mimicry (Chrome, YouTube, Cloud API profiles)
- Adaptive timing jitter (Box-Muller distribution)
- Intelligent packet padding to standard HTTP sizes
- Decoy traffic generation
- TLS fingerprint masking
- SNI rotation

---

## Master Key Security

The server generates a 256-bit master key at initialization. This key:
- Derives unique session keys per connection
- **Must never** be transmitted over the network
- Should be backed up in a secure location
- Recommended rotation: every 90 days

---

## Security Best Practices

1. **Keep updated** — always run the latest version from [Releases](https://github.com/IvanChernykh/SOVA/releases)
2. **Strong passwords** — at least 16 characters, mixed case + numbers + symbols
3. **Monitor connections** — use the dashboard at `http://localhost:8080`
4. **Firewall** — restrict API port access to trusted networks
5. **TLS certificates** — verify certificate fingerprints out-of-band

---

## Known Security Considerations

| Item | Status | Note |
|---|---|---|
| `InsecureSkipVerify` in transports | Documented | Required for custom handshake; server identity verified via PQ keys |
| Self-signed TLS certificates | By design | Certificate pinning via JSON config |
| Rate limiting | Implemented | Configurable per-IP rate limiter in middleware |
| Input validation | Implemented | All API inputs validated and sanitized |

---

## Responsible Disclosure Timeline

- **Day 0** — Report received
- **Day 3** — Acknowledgment
- **Day 14** — Fix developed and tested
- **Day 21** — Patch release published

---

## Contact

- **Security advisories**: https://github.com/IvanChernykh/SOVA/security/advisories
- **Issues (non-security)**: https://github.com/IvanChernykh/SOVA/issues
- **Repository**: https://github.com/IvanChernykh/SOVA

SOVA is a free, open-source project. No paid support or bug bounties — we rely on the community.
