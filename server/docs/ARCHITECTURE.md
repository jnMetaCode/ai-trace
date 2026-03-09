# AI-Trace Architecture

This document describes the architecture and design principles of AI-Trace.

## Overview

AI-Trace provides cryptographic proof that an AI system produced a specific output at a specific time. It captures AI decisions, builds verifiable Merkle trees, and anchors proofs to various trust levels.

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Application Layer                             │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐               │
│  │ REST API    │   │ Gateway     │   │ CLI         │               │
│  │ /api/v1/*   │   │ Proxy       │   │ ai-trace    │               │
│  └──────┬──────┘   └──────┬──────┘   └──────┬──────┘               │
└─────────┼─────────────────┼─────────────────┼───────────────────────┘
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Core Services                                │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐               │
│  │ Event       │   │ Certificate │   │ Report      │               │
│  │ Service     │   │ Service     │   │ Service     │               │
│  └──────┬──────┘   └──────┬──────┘   └──────┬──────┘               │
└─────────┼─────────────────┼─────────────────┼───────────────────────┘
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Cryptographic Layer                             │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐               │
│  │ Merkle Tree │   │ Signature   │   │ Anchor      │               │
│  │ (Standard/  │   │ (Ed25519)   │   │ (TSA/Chain) │               │
│  │ Incremental)│   │             │   │             │               │
│  └─────────────┘   └─────────────┘   └─────────────┘               │
└─────────────────────────────────────────────────────────────────────┘
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Storage Layer                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐               │
│  │ PostgreSQL  │   │ Redis       │   │ MinIO/S3    │               │
│  │ (Events,    │   │ (Cache,     │   │ (WORM,      │               │
│  │  Certs)     │   │  Sessions)  │   │  Reports)   │               │
│  └─────────────┘   └─────────────┘   └─────────────┘               │
└─────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Gateway Proxy

The Gateway intercepts AI API calls and creates traces automatically:

```go
// Request flow
App Request → Gateway → Upstream Provider → Gateway → Response + Trace

// What gets captured:
- Request: model, messages, parameters
- Response: choices, usage, timing
- Metadata: timestamps, latency, tokens
```

Key features:
- **Zero intrusion**: Drop-in replacement for OpenAI/Claude endpoints
- **Key passthrough**: API keys go directly to providers, never stored
- **Streaming support**: Full SSE streaming with real-time tracing

### 2. Event Service

Events are the atomic units of a trace:

```go
type Event struct {
    ID        string    // Unique identifier
    TraceID   string    // Parent trace
    Type      string    // input, output, custom
    Sequence  int       // Order in trace
    Timestamp time.Time // When it occurred
    Hash      string    // SHA-256 of content
    Payload   JSON      // The actual data
}
```

Events are:
- Immutable once created
- Sequentially ordered within a trace
- Content-addressable via hash

### 3. Merkle Tree

AI-Trace uses Merkle trees to create verifiable proofs:

```
                    Root Hash
                       │
           ┌───────────┴───────────┐
           │                       │
        Hash(0,1)               Hash(2,3)
           │                       │
     ┌─────┴─────┐           ┌─────┴─────┐
     │           │           │           │
  Event 0    Event 1     Event 2    Event 3
```

**Standard Merkle Tree**: Built once when trace is complete
- Simple implementation
- Efficient for small-medium traces

**Incremental Merkle Tree**: Grows as events are added
- Append-only operations
- Efficient for large/streaming traces
- Checkpoints for persistence

### 4. Certificate Engine

Certificates bundle proofs at different evidence levels:

```go
type Certificate struct {
    ID            string        // cert_xxx
    TraceID       string        // Source trace
    EvidenceLevel string        // internal, compliance, legal
    RootHash      string        // Merkle root
    Signature     string        // Ed25519 signature
    Timestamp     time.Time     // Creation time

    // compliance level specific
    TSAToken      []byte        // RFC 3161 timestamp
    WORMLocation  string        // Immutable storage

    // legal level specific
    TxHash        string        // Blockchain tx
    BlockNumber   uint64        // Block height
}
```

### 5. Anchor Service

Anchoring provides external trust for certificates:

**internal - Local Signature**
```
Certificate Hash → Ed25519 Sign → Signature
```

**compliance - WORM + TSA**
```
Certificate Hash → TSA Server → Timestamp Token
                → MinIO WORM → Immutable Storage
```

**legal - Blockchain**
```
Certificate Hash → Batch Queue → Merkle Root → Blockchain Tx
```

Batch processing for legal level:
- Collects multiple certificates
- Creates batch Merkle root
- Single blockchain transaction
- Individual proofs preserved

## Data Flow

### 1. API Call Tracing

```
1. App sends request to AI-Trace gateway
2. Gateway creates trace record
3. Gateway records input event
4. Gateway forwards to upstream (OpenAI/Claude)
5. Gateway receives response
6. Gateway records output event
7. Gateway returns response + trace_id to app
```

### 2. Certificate Commit

```
1. App requests certificate for trace_id
2. Service fetches all events for trace
3. Service builds/updates Merkle tree
4. Service computes root hash
5. Service creates signature
6. Service applies anchoring (internal/compliance/legal)
7. Service stores certificate
8. Service returns certificate to app
```

### 3. Verification

```
1. Verifier requests certificate verification
2. Service loads certificate
3. Service validates signature
4. Service verifies Merkle structure
5. Service checks anchors (if compliance/legal)
6. Service returns verification result
```

### 4. Minimal Disclosure Proof

```
1. Prover requests proof for specific events
2. Service loads certificate
3. Service generates Merkle proofs for selected events
4. Service creates proof package
5. Verifier can verify subset without seeing full trace
```

## Security Model

### Trust Hierarchy

```
legal (Blockchain)      ← Public consensus, highest trust
      │
compliance (WORM+TSA)   ← Tamper-proof storage, external timestamp
      │
internal (Local)        ← Self-signed, internal audit
```

### Threat Model

| Threat | Mitigation |
|--------|------------|
| Data tampering | Merkle trees, immutable storage |
| Time manipulation | TSA timestamps, blockchain |
| Key compromise | Key rotation, HSM support |
| Selective disclosure | Merkle proofs for subset |
| Denial of existence | Distributed anchoring |

### Cryptographic Choices

- **Hash**: SHA-256 (collision resistant)
- **Signature**: Ed25519 (fast, secure)
- **Merkle**: Binary tree with sorted leaves
- **Timestamp**: RFC 3161 compliant TSA

## Scalability

### Horizontal Scaling

```
Load Balancer
      │
  ┌───┼───┐
  │   │   │
Node1 Node2 Node3
  │   │   │
  └───┼───┘
      │
Shared Storage
(PostgreSQL, Redis, MinIO)
```

### Performance Characteristics

| Operation | Complexity | Typical Latency |
|-----------|------------|-----------------|
| Event ingest | O(1) | <10ms |
| Merkle update | O(log n) | <5ms |
| Certificate commit (internal) | O(n) | <100ms |
| Certificate commit (compliance) | O(n) | <1s |
| Certificate commit (legal) | O(n) | Batched |
| Verification | O(n log n) | <50ms |
| Proof generation | O(k log n) | <20ms |

### Storage Estimates

Per trace (10 events average):
- Events: ~5KB
- Merkle tree: ~1KB
- Certificate: ~2KB
- Total: ~8KB per trace

## Federation

Optional multi-node verification:

```
                Certificate
                    │
    ┌───────────────┼───────────────┐
    │               │               │
  Node 1          Node 2          Node 3
  (verify)        (verify)        (verify)
    │               │               │
    └───────────────┼───────────────┘
                    │
            Confirmation (2/3)
```

Federation provides:
- Geographic distribution
- Independent verification
- Byzantine fault tolerance

## Future Architecture

Planned enhancements:

1. **Zero-Knowledge Proofs**: Verify without revealing content
2. **Trusted Execution**: Hardware-backed attestation
3. **Cross-Chain**: Multiple blockchain anchors
4. **Sharding**: Horizontal data partitioning
5. **Real-time Streaming**: WebSocket event streams

## References

- [Merkle Trees](https://en.wikipedia.org/wiki/Merkle_tree)
- [RFC 3161 - TSA Protocol](https://tools.ietf.org/html/rfc3161)
- [Ed25519 Signatures](https://ed25519.cr.yp.to/)
- [Certificate Transparency](https://certificate.transparency.dev/)
