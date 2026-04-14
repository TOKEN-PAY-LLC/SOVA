# SOVA Release and Adoption Guide

## Purpose

This guide covers two practical tasks:

1. how to manually clean up and re-publish the public GitHub release state;
2. how to make SOVA attractive for third-party developers and product teams.

---

## Part 1. Manual GitHub Release Cleanup

### Goal

Keep the public repository aligned with the current baseline:

- public version: `v1.0.0`
- branding: `SOVA Proxy`, `SOVA Protocol`, `SOVA VPN`
- no legacy public messaging around external proxy-protocol compatibility

### Option A. GitHub Web UI

1. Open the repository releases page.
2. Find release `v1.0.1`.
3. Open the release.
4. If it should not remain public, choose `Delete`.
5. If the tag should also be removed, delete the Git tag from the repository tags page or from your local/git workflow.
6. Create a new release only after binaries, docs, screenshots, and release notes are aligned with `v1.0.0`.

### Option B. GitHub CLI

If you use GitHub CLI locally, the usual flow is:

```bash
gh release delete v1.0.1
git tag -d v1.0.1
git push origin :refs/tags/v1.0.1
```

Then publish the baseline release again:

```bash
git tag v1.0.0
git push origin v1.0.0
gh release create v1.0.0 --title "SOVA Protocol v1.0.0" --notes-file RELEASE_NOTES.md
```

Adjust the commands if the tag already exists and should be reused instead of recreated.

### Pre-release Checklist

Before publishing any new release:

- run `go build ./...`
- verify UI shows `v1.0.0`
- verify dashboard shows `v1.0.0`
- verify `README.md`, `RELEASE_NOTES.md`, `ROADMAP.md`, `CONTRIBUTING.md` match the same baseline
- verify API exports only native SOVA profile/share-link formats
- verify local proxy path is native SOVA Proxy behavior
- verify WebSocket relay uses native SOVA framing

### Recommended Release Assets

Attach:

- client binary for Windows
- server binary for Windows
- Linux binaries if available
- checksums file
- short changelog focused on native SOVA baseline
- one architecture diagram
- one screenshot of the CLI and one of the dashboard

---

## Part 2. What Third-Party Developers Need

If you want real adoption, third-party teams need more than a repository. They need a stable integration surface.

### Minimum adoption package

Provide these items first:

1. **Protocol specification**
   - handshake sequence
   - frame types
   - error handling
   - keepalive behavior
   - PSK rules
   - WebSocket transport behavior

2. **Configuration specification**
   - `SOVA profile` JSON schema
   - `sova://` share-link schema
   - required vs optional fields
   - defaults for `psk`, `sni_list`, `fragment_size`, `fragment_jitter`

3. **Reference SDK**
   - Go SDK first
   - then lightweight Rust/Kotlin/Swift examples if you want mobile adoption

4. **Conformance tests**
   - sample handshake vectors
   - encrypted frame test vectors
   - interoperability tests against the reference server

5. **Example integrations**
   - browser/app via local `SOVA Proxy`
   - native relay client using `common/protocol.go`
   - import flow for `sova://` links

---

## Part 3. Where to Promote SOVA

### Developer-facing channels

Good places to start:

- GitHub Discussions in your own repository
- GitHub Issues with label `integration`
- Reddit communities around Go, networking, privacy, VPN, censorship-resistance
- Hacker News launch/update posts when the protocol spec and SDK are ready
- Telegram/Discord communities focused on privacy tech and network tooling
- independent VPN and proxy client maintainers looking for new transports

### People to approach

Focus on maintainers who can actually integrate a protocol:

- developers of privacy-focused clients
- maintainers of proxy/VPN control panels
- open-source desktop VPN client authors
- mobile privacy app developers
- self-hosting communities building network gateways

### What to say in outreach

Lead with these messages:

- SOVA has a native encrypted relay protocol
- SOVA provides a local application-facing proxy entry point
- SOVA includes DPI-resistance features out of the box
- SOVA has a stable share-link and profile format
- SOVA can be integrated incrementally: proxy mode first, native mode second

Do not lead with hype alone. Lead with:

- protocol spec
- sample code
- test vectors
- simple importable configs
- clear licensing

---

## Part 4. Best Strategy for Fast Adoption

### Phase 1. Make SOVA easy to try

Ship:

- one-page protocol spec
- one-page config spec
- one Go client example
- one server deployment example
- one `sova://` importer example

### Phase 2. Make SOVA easy to trust

Ship:

- versioned protocol documentation
- interoperability tests
- benchmark results
- threat model notes
- security disclosure policy

### Phase 3. Make SOVA easy to integrate

Ship:

- SDK package
- reference client library API
- compatibility test harness
- CI recipe for external implementers

---

## Part 5. Recommended Messaging for the Project

Use consistent public wording:

- `SOVA Proxy` — local ingress used by apps and browsers
- `SOVA Protocol` — encrypted transport and relay framing
- `SOVA VPN` — end-user product positioning

Avoid confusing public messaging that makes SOVA look like only a wrapper around somebody else's protocol stack.

---

## Part 6. Security Notes for Adoption

If you publish example configs and SDKs:

- do not hardcode private production secrets
- treat the default PSK as a bootstrap/demo value only
- document how operators should rotate PSKs
- document how to pin or validate server identity in production
- keep example SNI lists clearly marked as examples

---

## Part 7. Suggested Next Deliverables

If the goal is serious external adoption, the next best repo additions are:

1. `PROTOCOL.md`
2. `CONFIG_SCHEMA.md`
3. `SDK_GUIDE.md`
4. `examples/` with native client samples
5. automated interoperability tests

These five items will do more for adoption than marketing alone.
