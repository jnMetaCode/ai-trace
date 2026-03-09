# AI-Trace Python SDK

Python SDK for AI-Trace - Tamper-proof attestation for AI decisions.

## Installation

```bash
pip install ai-trace
```

With OpenAI integration:
```bash
pip install ai-trace[openai]
```

## Quick Start

```python
from ai_trace import AITrace

# Initialize client
client = AITrace(
    server_url="http://localhost:8006",
    api_key="your-api-key"
)

# Create a trace
trace = client.traces.create(name="Customer Support Chat")

# Add events
client.events.add(
    trace_id=trace.id,
    event_type="input",
    payload={"prompt": "Hello, how can I help?"}
)

client.events.add(
    trace_id=trace.id,
    event_type="output",
    payload={"response": "I'm here to assist you!"}
)

# Commit to certificate
cert = client.certs.commit(
    trace_id=trace.id,
    evidence_level="L2"  # L1, L2, or L3
)

print(f"Certificate ID: {cert.id}")
print(f"Root Hash: {cert.root_hash}")

# Verify certificate
result = client.certs.verify(cert_id=cert.id)
print(f"Valid: {result.valid}")
```

## OpenAI Integration

Automatic tracing for OpenAI API calls:

```python
from ai_trace.integrations import TracedOpenAI

# Drop-in replacement for OpenAI client
client = TracedOpenAI(
    openai_api_key="sk-...",
    ai_trace_url="http://localhost:8006",
    ai_trace_key="your-key",
)

# All API calls are automatically traced
response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)

# Access the trace
print(f"Trace ID: {client.current_trace_id}")

# Commit the trace
cert = client.commit_trace(evidence_level="L2")
```

## Evidence Levels

- **L1 - Local Signature**: Fast, free, cryptographic proof
- **L2 - WORM + TSA**: Durable storage with timestamp authority
- **L3 - Blockchain**: Immutable anchor on Ethereum/Polygon

## API Reference

### Traces

```python
# Create a trace
trace = client.traces.create(
    name="My Trace",
    tenant_id="default",
    user_id="user-123",
    metadata={"key": "value"}
)

# Get a trace
trace = client.traces.get(trace_id="trace-id")

# List traces
traces = client.traces.list(limit=20, offset=0)
```

### Events

```python
# Add an event
event = client.events.add(
    trace_id="trace-id",
    event_type="input",  # input, output, custom
    payload={"data": "value"}
)

# Get an event
event = client.events.get(event_id="event-id")

# List events
events = client.events.list(trace_id="trace-id")
```

### Certificates

```python
# Commit trace to certificate
cert = client.certs.commit(
    trace_id="trace-id",
    evidence_level="L2",
    chain_type="ethereum"  # For L3 only
)

# Verify certificate
result = client.certs.verify(cert_id="cert-id", full_verification=True)

# Generate proof
proof = client.certs.prove(
    cert_id="cert-id",
    event_indices=[0, 1, 2],
    disclosed_fields=["model", "timestamp"]
)
```

## Configuration

Environment variables:
- `AI_TRACE_SERVER`: Server URL (default: http://localhost:8006)
- `AI_TRACE_API_KEY`: API key
- `AI_TRACE_TENANT_ID`: Tenant ID (default: default)

## Development

```bash
# Install dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Type checking
mypy ai_trace

# Linting
ruff check ai_trace
```

## License

MIT License - see [LICENSE](LICENSE) for details.
