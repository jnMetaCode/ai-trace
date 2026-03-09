# AI-Trace

<p align="center">
  <img src="docs/images/logo.png" alt="AI-Trace Logo" width="200">
</p>

<p align="center">
  <strong>Enterprise AI Decision Audit & Compliance Platform</strong>
</p>

<p align="center">
  <a href="https://github.com/jnMetaCode/ai-trace/releases"><img src="https://img.shields.io/github/v/release/ai-trace/server" alt="Release"></a>
  <a href="https://github.com/jnMetaCode/ai-trace/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/jnMetaCode/ai-trace"><img src="https://goreportcard.com/badge/github.com/jnMetaCode/ai-trace" alt="Go Report Card"></a>
  <a href="https://github.com/jnMetaCode/ai-trace/actions"><img src="https://github.com/jnMetaCode/ai-trace/workflows/CI/badge.svg" alt="CI"></a>
</p>

<p align="center">
  <a href="https://aitrace.cc">Website</a> |
  <a href="https://docs.aitrace.cc">Documentation</a> |
  <a href="#quick-start">Quick Start</a> |
  <a href="./README_CN.md">中文</a>
</p>

---

## What is AI-Trace?

AI-Trace provides **zero-intrusion** AI call tracing, evidence preservation, and audit capabilities for enterprises. It acts as a transparent proxy between your application and LLM providers, automatically capturing every AI decision for compliance and accountability.

### Key Features

- **Zero Code Changes** - Just replace your API endpoint, no SDK integration needed
- **API Key Passthrough** - Your keys go directly to upstream providers, never stored
- **Multi-Level Evidence** - Internal / Compliance (WORM) / Legal (Blockchain)
- **Merkle Tree Proofs** - Cryptographically verifiable decision trails
- **Minimal Disclosure** - Selective disclosure proofs for third-party verification
- **Federation Support** - Decentralized verification across multiple nodes

### Supported LLM Providers

| Provider | Status | Endpoint |
|----------|--------|----------|
| OpenAI | ✅ Full Support | `/api/v1/chat/completions` |
| Claude | ✅ Full Support | `/api/v1/chat/completions` |
| Ollama | ✅ Full Support | `/api/v1/chat/completions` |
| Azure OpenAI | 🚧 Coming Soon | - |

---

## Quick Start

### Option 1: Simple Mode (Single-File, Recommended for Getting Started)

```bash
# Clone the repository
git clone https://github.com/jnMetaCode/ai-trace.git
cd ai-trace/server

# Start with Simple Mode - SQLite only, no external dependencies
docker compose -f docker-compose.simple.yml up -d

# Verify it's running
curl http://localhost:8006/health
```

### Option 2: Standard Mode (Production)

```bash
# Clone the repository
git clone https://github.com/jnMetaCode/ai-trace.git
cd ai-trace/server

# Copy configuration
cp config.yaml.example config.yaml
cp .env.example .env

# Start with Docker Compose (PostgreSQL + Redis + MinIO)
docker compose up -d

# Verify it's running
curl http://localhost:8006/health
```

### Option 3: Build from Source

```bash
# Prerequisites: Go 1.21+, PostgreSQL 15+, Redis 7+, MinIO

# Build
go build -o ai-trace-server ./cmd/ai-trace-server

# Run
./ai-trace-server
```

### Your First Traced API Call

```bash
# Replace your OpenAI endpoint with AI-Trace
curl -X POST http://localhost:8006/api/v1/chat/completions \
  -H "X-API-Key: test-api-key-12345" \
  -H "X-Upstream-API-Key: sk-your-openai-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

Your API call is now traced! Get the certificate:

```bash
# Commit a certificate for the trace
curl -X POST http://localhost:8006/api/v1/certs/commit \
  -H "X-API-Key: test-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"trace_id": "trc_xxx", "evidence_level": "compliance"}'
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Your Application                        │
│                 (No code changes needed)                     │
└─────────────────────────┬───────────────────────────────────┘
                          │ HTTP Request
                          │ (OpenAI-compatible)
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                     AI-Trace Server                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ Gateway  │──│  Events  │──│  Merkle  │──│   Cert   │    │
│  │  Proxy   │  │  Store   │  │   Tree   │  │  Engine  │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
└─────────────────────────┬───────────────────────────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
    ┌──────────┐    ┌──────────┐    ┌──────────┐
    │ OpenAI   │    │ Claude   │    │ Ollama   │
    │   API    │    │   API    │    │  Local   │
    └──────────┘    └──────────┘    └──────────┘
```

---

## Evidence Levels

| Level | Description | Trust Model | Use Case |
|-------|-------------|-------------|----------|
| **internal** | Local Ed25519 signature | Self-signed | Internal audit, development |
| **compliance** | WORM storage + TSA timestamp | Tamper-proof storage | SOC2, GDPR, HIPAA |
| **legal** | Blockchain anchor | Decentralized consensus | Legal disputes, contracts |

---

## API Documentation

Interactive API documentation is available at:

```
http://localhost:8006/swagger/index.html
```

### Core Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/chat/completions` | POST | OpenAI-compatible proxy |
| `/api/v1/events/ingest` | POST | Ingest trace events |
| `/api/v1/events/search` | GET | Search events |
| `/api/v1/certs/commit` | POST | Generate certificate |
| `/api/v1/certs/verify` | POST | Verify certificate |
| `/api/v1/certs/{id}/prove` | POST | Generate minimal disclosure proof |
| `/api/v1/reports/generate` | POST | Generate audit report |

---

## Federation

AI-Trace supports federated verification across multiple independent nodes:

```bash
# Start a 3-node federation
cd deploy/federation
./start-federation.sh start

# Nodes will automatically discover and verify each other
# Certificates require confirmation from 2+ nodes
```

Learn more in [Federation Guide](./deploy/federation/README.md).

---

## Configuration

```yaml
# config.yaml
server:
  port: 8006
  mode: release

features:
  blockchain_anchor: false  # Requires -tags blockchain build
  federated_nodes: true
  metrics: true
  reports: true

anchor:
  federated:
    enabled: true
    nodes:
      - http://node2:8006
      - http://node3:8006
    min_confirmations: 2
```

See [Configuration Guide](./docs/CONFIGURATION.md) for all options.

---

## SDKs

Official SDKs for easy integration:

- **Python**: `pip install ai-trace` - [Documentation](./sdk/python/)
- **JavaScript**: `npm install @ai-trace/sdk` - [Documentation](./sdk/javascript/)

### Python Example

```python
from ai_trace import AITraceClient

client = AITraceClient(
    api_key="your-ai-trace-key",
    upstream_api_key="sk-your-openai-key"  # Passed through, not stored
)

# Use like OpenAI
response = client.chat.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)

# Get the certificate
cert = client.certs.commit(trace_id=response.trace_id, evidence_level="compliance")
print(f"Certificate: {cert.cert_id}")
```

---

## Deployment

### Docker

```bash
docker run -d \
  -p 8006:8006 \
  -e DB_HOST=postgres \
  -e REDIS_HOST=redis \
  ghcr.io/ai-trace/server:latest
```

### Kubernetes

```bash
kubectl apply -f deploy/k8s/
```

### Build Options

```bash
# Standard build (no blockchain/metrics dependencies)
go build ./cmd/ai-trace-server

# With Prometheus metrics
go build -tags metrics ./cmd/ai-trace-server

# With blockchain support
go build -tags blockchain ./cmd/ai-trace-server

# Full build
go build -tags "blockchain metrics" ./cmd/ai-trace-server
```

See [Deployment Guide](./docs/DEPLOYMENT.md) for production setup.

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

```bash
# Setup development environment
git clone https://github.com/jnMetaCode/ai-trace.git
cd ai-trace/server
cp config.yaml.example config.yaml
cp .env.example .env
docker compose up -d
make run
```

---

## Security

- **Vulnerability Reports**: security@aitrace.cc
- **Security Policy**: [SECURITY.md](./SECURITY.md)

Your API keys are **never stored**. They are passed through to upstream providers in real-time.

---

## License

Apache License 2.0 - see [LICENSE](./LICENSE)

---

## Acknowledgements

Built with:
- [Gin](https://github.com/gin-gonic/gin) - HTTP framework
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver
- [MinIO](https://min.io/) - Object storage
- [go-ethereum](https://github.com/ethereum/go-ethereum) - Blockchain integration

---

<p align="center">
  <sub>Built with ❤️ for AI accountability</sub>
</p>
