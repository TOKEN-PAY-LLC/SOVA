# Security Policy

## Reporting Security Vulnerabilities

If you discover a security vulnerability in SOVA Protocol, please email **security@sova.io** with the following information:

1. **Description**: A clear description of the vulnerability
2. **Location**: Which component(s) are affected (server, client, crypto, transport, etc.)
3. **Steps to Reproduce**: How to reproduce or trigger the vulnerability
4. **Impact**: The potential impact and severity
5. **Suggested Fix** (optional): Any ideas on how to fix it

**Do not open public GitHub issues for security vulnerabilities.** We will acknowledge your report within 48 hours and work on a fix.

## Security Considerations

### Master Key Security
- The SOVA server generates and securely stores a master key at initialization
- This key is used to derive session keys for each client connection
- **The master key should be:**
  - Backed up in a secure, encrypted location
  - Rotated periodically (recommended: every 90 days)
  - Never transmitted over the network
  - Only accessible to the server process

### Key Rotation
To rotate the master key:
```bash
sova keygen --rotate
```
This will create a new master key and invalidate all existing session keys, requiring clients to re-authenticate.

### Cryptographic Algorithms
- **Symmetric**: AES-256-GCM, ChaCha20-Poly1305
- **Asymmetric**: Ed25519 (Schnorr signatures)
- **Post-Quantum**: Kyber, Dilithium (via Cloudflare's `circl`)
- **Hashing**: SHA-256, SHA-3

### ZKP Authentication
SOVA uses Zero-Knowledge Proof for authentication:
- Passwords are never transmitted in plain text
- Authentication proofs are non-replayable (nonce-based)
- Each session derives unique session keys

### Obfuscation
- Dynamic packet morphing to avoid pattern matching
- Timing jitter to prevent fingerprinting
- SNI rotation from a curated list
- Custom TLS handshake variations

## Compliance and Audits

SOVA undergoes regular security audits:
- Monthly automated vulnerability scanning
- Quarterly manual code reviews
- Annual third-party security audit

## Responsible Disclosure

We appreciate security researchers who responsibly disclose vulnerabilities. Coordinated disclosure timeline:
- **Day 0**: Vulnerability report received
- **Day 3**: Initial triage and acknowledgment
- **Day 14**: Fix development and testing
- **Day 21**: Public disclosure and patch release

## Security Best Practices for Users

1. **Keep SOVA Updated**: Always run the latest version
   ```bash
   sova install --update
   ```

2. **Monitor Connections**: Check active proxy connections
   ```bash
   sova status
   ```

3. **Use Strong Passwords**: Credentials should be:
   - At least 16 characters
   - Mix of uppercase, lowercase, numbers, and symbols
   - Unique for each user

4. **Firewall Rules**: Restrict API access to trusted networks
   ```bash
   # Allow only localhost and internal network
   iptables -A INPUT -p tcp -d localhost -j ACCEPT
   iptables -A INPUT -p tcp -s 192.168.1.0/24 -j ACCEPT
   iptables -P INPUT DROP
   ```

5. **TLS Certificates**: SOVA uses self-signed certificates for Web Mirror Mode
   - Verify certificate fingerprints via out-of-band channels
   - Certificate pinning is automatic in the client

## Bug Bounty

We offer a bug bounty program for disclosed vulnerabilities:
- **Critical (RCE, Key Leakage)**: $5,000
- **High (Authentication Bypass)**: $2,000
- **Medium (Information Disclosure)**: $500
- **Low (DoS, Minor Logic Flaw)**: $100

Eligibility: First reporter of an unpatched vulnerability, following responsible disclosure.

## Security Contact

📧 security@sova.io  
🔗 https://github.com/IvanChernykh/SOVA  
📋 Security advisories: https://github.com/IvanChernykh/SOVA/security/advisories
