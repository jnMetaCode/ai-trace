# AI-Trace Python SDK

Enterprise AI Audit System Python SDK

## Installation

```bash
pip install ai-trace-sdk
```

Or with OpenAI integration:

```bash
pip install ai-trace-sdk[openai]
```

## Quick Start

### Basic Usage

```python
from ai_trace import AITraceClient

# Create client
client = AITraceClient(
    base_url="http://localhost:8006",
    api_key="your-api-key",
    tenant_id="default"
)

# Search events
events = client.events.search(trace_id="trc_xxx")
print(f"Found {events.size} events")

# Get event details
event = client.events.get("evt_xxx")
print(f"Event type: {event.event_type}")

# Create certificate
cert = client.certs.commit(
    trace_id="trc_xxx",
    evidence_level="internal"  # internal, compliance, or legal
)
print(f"Certificate created: {cert.cert_id}")

# Verify certificate
result = client.certs.verify(cert_id=cert.cert_id)
print(f"Valid: {result.valid}")

# Generate minimal disclosure proof
proof = client.certs.generate_proof(
    cert_id=cert.cert_id,
    disclose_events=[0, 2],  # Event indices to disclose
    disclose_fields=["event_type", "timestamp"]
)

# Close client
client.close()
```

### Context Manager

```python
from ai_trace import AITraceClient

with AITraceClient(base_url="http://localhost:8006") as client:
    events = client.events.search(trace_id="trc_xxx")
```

### Async Client

```python
import asyncio
from ai_trace.client import AsyncAITraceClient

async def main():
    async with AsyncAITraceClient(base_url="http://localhost:8006") as client:
        events = await client.events.search(trace_id="trc_xxx")
        print(f"Found {events.size} events")

asyncio.run(main())
```

## OpenAI Integration

The SDK provides a transparent wrapper for the OpenAI client that automatically traces all API calls.

```python
from openai import OpenAI
from ai_trace import TracedOpenAI

# Create OpenAI client
openai_client = OpenAI(api_key="sk-...")

# Wrap with tracing
traced = TracedOpenAI(
    openai_client=openai_client,
    trace_server="http://localhost:8006",
    trace_api_key="your-trace-api-key",
    tenant_id="default",
    user_id="user123",
)

# Use as normal - calls are automatically traced
response = traced.chat.completions.create(
    model="gpt-4",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)

# Get trace ID
print(f"Trace ID: {traced.current_trace_id}")

# Commit certificate for audit
cert = traced.commit_certificate(evidence_level="internal")
print(f"Certificate: {cert.cert_id}")

# Clean up
traced.close()
```

### Auto-commit Mode

```python
traced = TracedOpenAI(
    openai_client=openai_client,
    trace_server="http://localhost:8006",
    auto_commit=True  # Automatically commit after each call
)
```

## Chat API (Proxy Mode)

Use AI-Trace server as an OpenAI-compatible proxy:

```python
from ai_trace import AITraceClient

client = AITraceClient(base_url="http://localhost:8006")

# Chat through AI-Trace proxy
response = client.chat.completions(
    messages=[
        {"role": "user", "content": "Hello!"}
    ],
    model="gpt-3.5-turbo"
)

print(response["choices"][0]["message"]["content"])
```

## API Reference

### AITraceClient

```python
AITraceClient(
    base_url: str = "http://localhost:8006",
    api_key: str = None,
    tenant_id: str = "default",
    timeout: float = 30.0
)
```

### Events API

- `client.events.search(trace_id, event_type, start_time, end_time, page, page_size)`
- `client.events.get(event_id)`
- `client.events.ingest(events)`

### Certificates API

- `client.certs.search(page, page_size)`
- `client.certs.get(cert_id)`
- `client.certs.commit(trace_id, evidence_level)`
- `client.certs.verify(cert_id, root_hash)`
- `client.certs.generate_proof(cert_id, disclose_events, disclose_fields)`

### Chat API

- `client.chat.completions(messages, model, temperature, max_tokens)`

## Development

```bash
# Install dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Format code
black src tests
isort src tests

# Type check
mypy src
```

## License

MIT License
