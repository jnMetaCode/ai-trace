# Quick Start Guide

Get AI-Trace running in under 5 minutes.

## Prerequisites

- Docker and Docker Compose (recommended)
- OR: Go 1.21+, PostgreSQL 15+, Redis 7+

## Option 1: Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/jnMetaCode/ai-trace.git
cd ai-trace/server

# Copy example configuration
cp config.yaml.example config.yaml
cp .env.example .env

# Start all services
docker compose up -d

# Verify it's running
curl http://localhost:8006/health
```

## Option 2: Using Pre-built Binary

```bash
# Download latest release
curl -LO https://github.com/jnMetaCode/ai-trace/releases/latest/download/ai-trace-linux-amd64
chmod +x ai-trace-linux-amd64

# Initialize configuration
./ai-trace-linux-amd64 init

# Start the server
./ai-trace-linux-amd64 serve
```

## Option 3: Build from Source

```bash
# Clone and build
git clone https://github.com/jnMetaCode/ai-trace.git
cd ai-trace/server
go build -o ai-trace ./cmd/ai-trace

# Initialize and run
./ai-trace init
./ai-trace serve
```

## Your First Traced API Call

### Step 1: Make an AI API Call Through AI-Trace

```bash
# Use AI-Trace as a proxy for your OpenAI calls
curl -X POST http://localhost:8006/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-api-key-12345" \
  -H "X-Upstream-API-Key: sk-your-openai-key" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "What is 2 + 2?"}
    ]
  }'
```

Response includes the trace ID:
```json
{
  "id": "chatcmpl-...",
  "trace_id": "trc_abc123def456",
  "choices": [...]
}
```

### Step 2: Commit to Certificate

```bash
# Create a tamper-proof certificate
curl -X POST http://localhost:8006/api/v1/certs/commit \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-api-key-12345" \
  -d '{
    "trace_id": "trc_abc123def456",
    "evidence_level": "L1"
  }'
```

Response:
```json
{
  "cert_id": "cert_xyz789",
  "root_hash": "sha256:3f2a1b...",
  "evidence_level": "L1",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Step 3: Verify Certificate

```bash
# Verify the certificate integrity
curl -X POST http://localhost:8006/api/v1/certs/verify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-api-key-12345" \
  -d '{
    "cert_id": "cert_xyz789"
  }'
```

Response:
```json
{
  "valid": true,
  "hash_valid": true,
  "signature_valid": true,
  "timestamp_valid": true
}
```

## Using the CLI

```bash
# Create a trace
ai-trace trace create --name "Customer Support"

# Add events
ai-trace trace add-event trc_xxx --type input --data '{"prompt": "Hello"}'

# Commit to certificate
ai-trace cert commit trc_xxx --level L2

# Verify certificate
ai-trace cert verify cert_yyy

# Export certificate for offline verification
ai-trace cert export cert_yyy -o cert_yyy.json
```

## Using Python SDK

```bash
pip install ai-trace
```

```python
from ai_trace import AITrace

client = AITrace(
    server_url="http://localhost:8006",
    api_key="test-api-key-12345"
)

# Create trace
trace = client.traces.create(name="My First Trace")

# Add events
client.events.add(
    trace_id=trace.id,
    event_type="input",
    payload={"prompt": "What is AI-Trace?"}
)

client.events.add(
    trace_id=trace.id,
    event_type="output",
    payload={"response": "AI-Trace provides tamper-proof attestation..."}
)

# Commit certificate
cert = client.certs.commit(trace_id=trace.id, evidence_level="L2")
print(f"Certificate: {cert.id}")

# Verify
result = client.certs.verify(cert_id=cert.id)
print(f"Valid: {result.valid}")
```

## Evidence Levels

| Level | Description | Cost | Use Case |
|-------|-------------|------|----------|
| **L1** | Local Ed25519 signature | Free | Internal audit, development |
| **L2** | WORM storage + TSA | Low | Regulatory compliance |
| **L3** | Blockchain anchor | Medium | Legal evidence, disputes |

## Next Steps

- [Installation Guide](./INSTALLATION.md) - Detailed setup instructions
- [API Reference](./API.md) - Complete API documentation
- [Architecture](./ARCHITECTURE.md) - How AI-Trace works
- [Configuration](./CONFIGURATION.md) - All configuration options
- [Python SDK](../sdk/python/README.md) - Python client library

## Need Help?

- GitHub Issues: https://github.com/jnMetaCode/ai-trace/issues
- Documentation: https://docs.aitrace.cc
- Community Discord: https://discord.gg/ai-trace
