# AI-Trace Architecture Documentation

## System Overview

AI-Trace is an enterprise-grade platform for AI decision auditing and tamper-proof attestation. The system provides cryptographic guarantees for AI inference records, enabling regulatory compliance, dispute resolution, and audit trails.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Layer                              │
├──────────┬──────────┬──────────┬──────────┬────────────────────┤
│ Python   │ Go SDK   │ Java SDK │ JS SDK   │ REST API           │
│ SDK      │          │          │          │                    │
└──────────┴──────────┴──────────┴──────────┴────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       API Gateway                                │
│  • Authentication    • Rate Limiting    • Request Routing       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Core Services                                │
├────────────────┬────────────────┬───────────────────────────────┤
│ Event Service  │ Cert Service   │ Proxy Service                 │
│ • Ingestion    │ • Commitment   │ • OpenAI Proxy                │
│ • Streaming    │ • Verification │ • Anthropic Proxy             │
│ • Search       │ • Proof Gen    │ • Request Logging             │
└────────────────┴────────────────┴───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Cryptographic Layer                            │
├────────────────┬────────────────┬───────────────────────────────┤
│ Merkle Tree    │ DAG (Causal    │ Zero-Knowledge                │
│ • Event Hash   │  Event Graph)  │  Proofs                       │
│ • Root Compute │ • Parallel     │ • Selective                   │
│ • Proof Gen    │   Events       │   Disclosure                  │
└────────────────┴────────────────┴───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Attestation Layer                              │
├────────────────┬────────────────┬───────────────────────────────┤
│ L1: Timestamp  │ L2: WORM       │ L3: Blockchain                │
│ • RFC 3161 TSA │   Storage      │ • Ethereum                    │
│ • DigiCert     │ • S3 Glacier   │ • Polygon                     │
│ • Trusted Time │ • Azure WORM   │ • Smart Contract              │
└────────────────┴────────────────┴───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Storage Layer                                 │
├────────────────┬────────────────┬───────────────────────────────┤
│ PostgreSQL     │ ClickHouse     │ Object Storage                │
│ • Events       │ • Analytics    │ • Media Files                 │
│ • Certs        │ • Metrics      │ • Encrypted Content           │
│ • Users        │ • Time Series  │ • Fingerprints                │
└────────────────┴────────────────┴───────────────────────────────┘
```

## Core Components

### 1. Event Service

The Event Service handles the ingestion and management of AI inference events.

**Key Features:**
- Real-time streaming event ingestion
- Event hash chain computation
- Payload encryption (AES-256-GCM)
- Search and retrieval

**Event Structure:**
```go
type Event struct {
    EventID         string    // Unique identifier
    TraceID         string    // Groups related events
    EventType       string    // llm.input, llm.output, etc.
    Timestamp       time.Time // RFC 3339
    Sequence        int       // Order within trace
    Payload         map       // Event data
    EventHash       string    // SHA-256 of event
    PrevEventHash   string    // Hash chain link
    PrevEventHashes []string  // DAG support
    PayloadHash     string    // Hash of payload
}
```

### 2. Certificate Service

Generates and verifies tamper-proof attestation certificates.

**Evidence Levels:**

| Level | Storage | Verification | Use Case |
|-------|---------|--------------|----------|
| L1 | Database + TSA | Timestamp proof | Basic audit |
| L2 | WORM Storage | Immutable storage | Legal compliance |
| L3 | Blockchain | Smart contract | Dispute resolution |

**Certificate Structure:**
```go
type Certificate struct {
    CertID        string
    TraceID       string
    RootHash      string      // Merkle root
    EventCount    int
    EvidenceLevel string
    CreatedAt     time.Time
    TimeProof     *TimeProof  // RFC 3161
    AnchorProof   *AnchorProof // Blockchain tx
}
```

### 3. DAG (Directed Acyclic Graph)

Supports parallel event tracking for multi-agent and concurrent AI operations.

```
        [Input Event]
             │
      ┌──────┴──────┐
      ▼             ▼
[Tool Call A]  [Tool Call B]    ← Parallel Events
      │             │
      └──────┬──────┘
             ▼
      [Merge Event]              ← Convergence Point
             │
             ▼
      [Output Event]
```

**Key Operations:**
- Topological sort for causal ordering
- Cycle detection
- Parallel event grouping
- Merge point validation

### 4. Multimodal Fingerprinting

Supports attestation of AI-generated media content.

**Image Fingerprinting:**
- pHash (DCT-based perceptual hash)
- aHash (Average hash)
- dHash (Difference hash)
- Color histogram

**Audio Fingerprinting:**
- Chromaprint-like algorithm
- FFT spectral analysis
- Chroma feature extraction

**Video Fingerprinting:**
- Key frame extraction
- Scene change detection
- Frame sequence hashing

### 5. Zero-Knowledge Proofs

Enables privacy-preserving verification using gnark (Groth16).

**Supported Circuits:**
- Hash Preimage Proof
- Content Ownership Proof
- Fingerprint Verification Proof
- Merkle Inclusion Proof
- Selective Disclosure Proof

### 6. Smart Contract Integration

Ethereum/Polygon smart contracts for L3 attestation.

**AITraceRegistry Contract:**
```solidity
function createAttestation(
    bytes32 certId,
    bytes32 merkleRoot,
    bytes32 fingerprintHash,
    uint256 eventCount,
    uint8 evidenceLevel
) external returns (bytes32);

function verifyAttestation(bytes32 certId)
    external view returns (bool valid, bytes32 merkleRoot);
```

**AITraceArbitration Contract:**
- Dispute creation
- Evidence submission
- Voting mechanism
- Resolution execution

## Data Flow

### Chat Completion with Attestation

```
1. Client sends chat request
       │
       ▼
2. Proxy Service intercepts
   • Logs input event
   • Forwards to upstream (OpenAI/Anthropic)
       │
       ▼
3. Upstream response received
   • Logs output event
   • Computes fingerprint
       │
       ▼
4. Event Service processes
   • Computes event hashes
   • Builds hash chain
       │
       ▼
5. Returns response + trace_id
       │
       ▼
6. Client commits certificate
       │
       ▼
7. Certificate Service
   • Builds Merkle tree
   • Obtains timestamp proof
   • (Optional) Blockchain anchor
       │
       ▼
8. Returns certificate
```

### Verification Flow

```
1. Verifier receives certificate
       │
       ▼
2. Fetch events by trace_id
       │
       ▼
3. Recompute Merkle root
       │
       ▼
4. Verify timestamp proof (RFC 3161)
       │
       ▼
5. (If L3) Verify blockchain anchor
       │
       ▼
6. Return verification result
```

## Security Model

### Cryptographic Primitives

| Component | Algorithm | Key Size |
|-----------|-----------|----------|
| Event Hash | SHA-256 | 256-bit |
| Content Encryption | AES-256-GCM | 256-bit |
| Merkle Tree | SHA-256 | 256-bit |
| ZK Proofs | Groth16/BN254 | 254-bit |
| Timestamps | RFC 3161 | - |

### Threat Model

**Protected Against:**
- Post-hoc modification of AI decisions
- Forged inference records
- Timestamp manipulation
- Partial disclosure attacks

**Trust Assumptions:**
- Trusted timestamp authorities
- Blockchain consensus security
- Client-side key management

## Scalability

### Horizontal Scaling

```
┌─────────────────────────────────────────┐
│           Load Balancer                  │
└─────────────────────────────────────────┘
         │           │           │
         ▼           ▼           ▼
    ┌─────────┐ ┌─────────┐ ┌─────────┐
    │ API #1  │ │ API #2  │ │ API #3  │
    └─────────┘ └─────────┘ └─────────┘
         │           │           │
         └───────────┼───────────┘
                     ▼
         ┌───────────────────────┐
         │   Message Queue       │
         │   (Redis/Kafka)       │
         └───────────────────────┘
                     │
         ┌───────────┼───────────┐
         ▼           ▼           ▼
    ┌─────────┐ ┌─────────┐ ┌─────────┐
    │Worker #1│ │Worker #2│ │Worker #3│
    └─────────┘ └─────────┘ └─────────┘
```

### Performance Targets

| Metric | Target |
|--------|--------|
| API Latency (p99) | < 100ms |
| Event Ingestion | 10,000/sec |
| Merkle Computation | < 50ms/1000 events |
| Certificate Generation | < 500ms |

## Deployment

### Docker Compose (Development)

```yaml
services:
  api:
    image: ai-trace/server
    ports:
      - "8080:8080"
  postgres:
    image: postgres:15
  redis:
    image: redis:7
```

### Kubernetes (Production)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ai-trace-api
spec:
  replicas: 3
  ...
```

## Monitoring

### Metrics (Prometheus)

- `aitrace_events_ingested_total`
- `aitrace_certs_committed_total`
- `aitrace_verification_duration_seconds`
- `aitrace_merkle_computation_duration_seconds`

### Logging (Structured JSON)

```json
{
  "level": "info",
  "timestamp": "2024-01-01T00:00:00Z",
  "trace_id": "trace-123",
  "event": "certificate_committed",
  "evidence_level": "L2",
  "event_count": 5
}
```

## References

- [RFC 3161 - Time-Stamp Protocol](https://tools.ietf.org/html/rfc3161)
- [Merkle Tree](https://en.wikipedia.org/wiki/Merkle_tree)
- [gnark ZK Library](https://github.com/ConsenSys/gnark)
- [EIP-712 Typed Structured Data](https://eips.ethereum.org/EIPS/eip-712)
