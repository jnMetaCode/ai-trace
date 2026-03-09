# AI-Trace Examples

This directory contains example code demonstrating how to use AI-Trace.

## Prerequisites

Make sure AI-Trace server is running:

```bash
# Using Docker
docker compose up -d

# Or locally
ai-trace serve
```

Verify it's running:
```bash
curl http://localhost:8006/health
```

## Examples

### Python Examples

```bash
# Install the SDK
pip install ai-trace

# Basic usage
python examples/python/basic_usage.py

# OpenAI integration (requires openai package)
pip install ai-trace[openai]
export OPENAI_API_KEY=sk-...
python examples/python/openai_integration.py

# Compliance audit example
python examples/python/compliance_audit.py
```

### Curl Examples

```bash
# Basic workflow using curl
chmod +x examples/curl/basic_workflow.sh
./examples/curl/basic_workflow.sh
```

### Go Examples

```bash
# Run the Go example
cd examples/go
go run main.go
```

## Example Descriptions

| Example | Language | Description |
|---------|----------|-------------|
| `basic_usage.py` | Python | Complete workflow: trace → events → cert → verify |
| `openai_integration.py` | Python | Automatic tracing for OpenAI API calls |
| `compliance_audit.py` | Python | HIPAA-compliant healthcare AI example |
| `basic_workflow.sh` | Bash/curl | Same workflow using curl commands |
| `main.go` | Go | Go client implementation example |

## Common Patterns

### 1. Basic Tracing Pattern

```python
from ai_trace import AITrace

client = AITrace(server_url="http://localhost:8006")

# Create trace
trace = client.traces.create(name="My Trace")

# Add events
client.events.add(trace_id=trace.id, event_type="input", payload={...})
client.events.add(trace_id=trace.id, event_type="output", payload={...})

# Commit certificate
cert = client.certs.commit(trace_id=trace.id, evidence_level="L1")

# Verify
result = client.certs.verify(cert_id=cert.id)
```

### 2. Gateway Proxy Pattern

```bash
# Use AI-Trace as a proxy for OpenAI
curl -X POST http://localhost:8006/api/v1/chat/completions \
  -H "X-API-Key: your-ai-trace-key" \
  -H "X-Upstream-API-Key: sk-your-openai-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4", "messages": [...]}'
```

### 3. Minimal Disclosure Pattern

```python
# Prove specific events without revealing everything
proof = client.certs.prove(
    cert_id=cert.id,
    event_indices=[0, 2],  # Only events 0 and 2
    disclosed_fields=["model", "timestamp"]  # Only these fields
)

# Third party can verify the proof
# without seeing the full trace
```

### 4. Compliance Pattern

```python
# L2 for regulatory compliance
cert = client.certs.commit(
    trace_id=trace.id,
    evidence_level="L2",  # WORM + TSA
    metadata={
        "compliance_framework": "HIPAA",
        "retention_years": 7
    }
)
```

## Environment Variables

```bash
# AI-Trace server
export AI_TRACE_SERVER=http://localhost:8006
export AI_TRACE_API_KEY=your-key
export AI_TRACE_TENANT_ID=default

# For OpenAI integration
export OPENAI_API_KEY=sk-...
```

## Troubleshooting

### Server not running
```bash
# Check health
curl http://localhost:8006/health

# Start server
docker compose up -d
# or
ai-trace serve
```

### Authentication errors
```bash
# Make sure API key is set
curl -H "X-API-Key: test-api-key-12345" http://localhost:8006/api/v1/traces
```

### Connection refused
```bash
# Check if server is listening
netstat -an | grep 8006

# Check Docker logs
docker compose logs ai-trace
```

## Need Help?

- [Documentation](https://docs.aitrace.cc)
- [GitHub Issues](https://github.com/jnMetaCode/ai-trace/issues)
- [API Reference](../docs/API.md)
