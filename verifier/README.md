<!-- badges -->
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.21-blue?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/ai-trace/ai-trace-verify)](https://github.com/jnMetaCode/ai-trace-verify/releases)

# ai-trace-verify

Open-source CLI tool for independently verifying [AI-Trace](https://github.com/jnMetaCode/ai-trace) certificates offline.

Given a certificate or minimal-disclosure proof file produced by an AI-Trace server, `ai-trace-verify` re-computes Merkle roots, validates hash formats, and checks time/anchor proofs -- all without any network access or server dependency.

## Installation

### Using Go Install

```bash
go install github.com/jnMetaCode/ai-trace-verify/cmd/ai-trace-verify@latest
```

### From Source

```bash
git clone https://github.com/jnMetaCode/ai-trace-verify.git
cd ai-trace-verify
make build
```

### Pre-built Binaries

Download pre-built binaries for your platform from the [Releases](https://github.com/jnMetaCode/ai-trace-verify/releases) page.

## Usage

### Verify a Proof File

```bash
ai-trace-verify --proof proof.json
```

### Verify a Certificate File

```bash
ai-trace-verify --cert certificate.json
```

### Output as JSON

```bash
ai-trace-verify --proof proof.json --json
```

### Verbose Mode

```bash
ai-trace-verify --proof proof.json --verbose
```

### Print Version

```bash
ai-trace-verify version
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--proof` | `-p` | Path to a proof JSON file |
| `--cert` | `-c` | Path to a certificate JSON file |
| `--root-hash` | `-r` | Root hash to verify |
| `--verbose` | `-v` | Enable verbose output |
| `--json` | `-j` | Output results as JSON |

## Example Output

```
Verifying proof file: proof.json

═══════════════════════════════════════════════════════════
                    VERIFICATION RESULT
═══════════════════════════════════════════════════════════

  ✓ VERIFICATION PASSED

Certificate Information:
  Cert ID:    cert_abc123
  Root Hash:  sha256:1234567890abcdef...
  Events:     3

Verification Checks:
  ✓ Schema Version - 0.1
  ✓ Root Hash Format - Valid SHA256 format
  ✓ Merkle Proofs - All 3 proofs verified
  ✓ Time Proof - local @ 2024-01-01T00:00:00Z
  ✓ Anchor Proof - local: anchor_xyz

═══════════════════════════════════════════════════════════
```

### JSON Output

```json
{
  "valid": true,
  "cert_id": "cert_abc123",
  "root_hash": "sha256:1234567890abcdef...",
  "event_count": 3,
  "checks": [
    {"name": "Schema Version", "passed": true, "message": "0.1"},
    {"name": "Root Hash Format", "passed": true, "message": "Valid SHA256 format"},
    {"name": "Merkle Proofs", "passed": true, "message": "All 3 proofs verified"},
    {"name": "Time Proof", "passed": true, "message": "local @ 2024-01-01T00:00:00Z"},
    {"name": "Anchor Proof", "passed": true, "message": "local: anchor_xyz"}
  ]
}
```

## How It Works

`ai-trace-verify` performs the following checks depending on the input type:

**For proof files (minimal-disclosure proofs):**

1. **Schema Version** -- verifies the proof declares a known schema version.
2. **Root Hash Format** -- ensures the root hash uses the `sha256:` prefix convention.
3. **Merkle Proofs** -- recomputes each event's Merkle path from leaf to root and confirms it matches the declared root hash.
4. **Time Proof** -- validates that a timestamp proof is present and well-formed.
5. **Anchor Proof** -- validates that an anchor proof (e.g. blockchain, storage) is present and well-formed.

**For certificate files:**

1. **Certificate ID** -- verifies the certificate has an identifier.
2. **Event Hashes** -- checks that event hashes are present.
3. **Merkle Tree Integrity** -- if the full Merkle tree is included, rebuilds it from the event hashes and confirms the root matches; otherwise validates the root hash format.
4. **Time Proof / Anchor Proof** -- same as above.
5. **Evidence Level** -- reports the declared evidence level from metadata.

All verification is performed locally. No network calls are made.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Verification passed |
| `1` | Verification failed or an error occurred |

## File Formats

See the [AI-Trace specification](https://github.com/jnMetaCode/ai-trace) for details on the proof and certificate JSON schemas.

## License

This project is licensed under the Apache License 2.0 -- see the [LICENSE](LICENSE) file for details.

## Related

- [AI-Trace](https://github.com/jnMetaCode/ai-trace) -- Full AI-Trace platform (server, console, SDKs)
