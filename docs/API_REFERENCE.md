# AI-Trace API Reference

## Overview

AI-Trace provides a RESTful API for enterprise AI decision auditing and tamper-proof attestation. All API endpoints use JSON for request and response bodies.

## Base URL

```
https://api.aitrace.cc
```

## Authentication

All API requests require an API key in the header:

```
X-API-Key: your-api-key
```

For upstream AI provider pass-through:

```
X-Upstream-API-Key: sk-your-openai-key
X-Upstream-Base-URL: https://api.openai.com  (optional)
```

## Response Format

All responses follow a consistent JSON structure:

### Success Response

```json
{
  "data": { ... },
  "meta": {
    "request_id": "req-123",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

### Error Response

```json
{
  "error": {
    "code": "invalid_request",
    "message": "Detailed error message"
  }
}
```

---

## Chat Completions

### Create Chat Completion

Create a chat completion with AI-Trace attestation.

**Endpoint:** `POST /api/v1/chat/completions`

**Headers:**
| Header | Required | Description |
|--------|----------|-------------|
| X-API-Key | Yes | AI-Trace API key |
| X-Upstream-API-Key | Yes | Upstream AI provider API key |
| X-Trace-ID | No | Custom trace ID (auto-generated if not provided) |
| X-Session-ID | No | Session identifier for multi-turn conversations |
| X-Business-ID | No | Business context identifier |

**Request Body:**

```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "temperature": 0.7,
  "max_tokens": 100,
  "top_p": 1.0,
  "n": 1,
  "stream": false,
  "stop": null
}
```

**Response:**

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1704067200,
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 10,
    "total_tokens": 30
  },
  "trace_id": "trace-abc123"
}
```

---

## Events

### Ingest Events

Ingest a batch of events for attestation.

**Endpoint:** `POST /api/v1/events/ingest`

**Request Body:**

```json
{
  "events": [
    {
      "event_id": "evt-123",
      "trace_id": "trace-abc123",
      "event_type": "llm.input",
      "timestamp": "2024-01-01T00:00:00Z",
      "sequence": 1,
      "payload": {
        "prompt": "User input text",
        "model_id": "gpt-4"
      },
      "prev_event_hash": "optional-previous-hash"
    }
  ]
}
```

**Response:**

```json
{
  "ingested": 1,
  "event_ids": ["evt-123"]
}
```

### Search Events

Search for events with filters.

**Endpoint:** `GET /api/v1/events/search`

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| trace_id | string | Filter by trace ID |
| event_type | string | Filter by event type |
| start_time | string | ISO 8601 start time |
| end_time | string | ISO 8601 end time |
| page | int | Page number (default: 1) |
| page_size | int | Items per page (default: 20, max: 100) |

**Response:**

```json
{
  "events": [...],
  "total": 100,
  "page": 1,
  "page_size": 20,
  "total_pages": 5
}
```

### Get Event

Get a single event by ID.

**Endpoint:** `GET /api/v1/events/{event_id}`

**Response:**

```json
{
  "event_id": "evt-123",
  "trace_id": "trace-abc123",
  "event_type": "llm.input",
  "timestamp": "2024-01-01T00:00:00Z",
  "sequence": 1,
  "payload": {...},
  "event_hash": "sha256:...",
  "prev_event_hash": "sha256:...",
  "payload_hash": "sha256:..."
}
```

---

## Certificates

### Commit Certificate

Generate an attestation certificate for a trace.

**Endpoint:** `POST /api/v1/certs/commit`

**Request Body:**

```json
{
  "trace_id": "trace-abc123",
  "evidence_level": "L2"
}
```

**Evidence Levels:**
| Level | Description |
|-------|-------------|
| L1 | Basic: Merkle tree + timestamp |
| L2 | WORM storage for legal compliance |
| L3 | Blockchain anchor for maximum security |

**Response:**

```json
{
  "cert_id": "cert-xyz789",
  "trace_id": "trace-abc123",
  "root_hash": "sha256:abc123...",
  "event_count": 5,
  "evidence_level": "L2",
  "created_at": "2024-01-01T00:00:00Z",
  "time_proof": {
    "timestamp": "2024-01-01T00:00:00Z",
    "timestamp_id": "ts-123",
    "tsa_name": "DigiCert TSA",
    "tsa_hash": "sha256:..."
  },
  "anchor_proof": null
}
```

### Verify Certificate

Verify a certificate's integrity.

**Endpoint:** `POST /api/v1/certs/verify`

**Request Body:**

```json
{
  "cert_id": "cert-xyz789",
  "root_hash": "sha256:abc123..."
}
```

*Note: Provide either `cert_id` or `root_hash`*

**Response:**

```json
{
  "valid": true,
  "checks": {
    "merkle_root": true,
    "timestamp": true,
    "event_hashes": true,
    "causal_chain": true
  },
  "certificate": {...}
}
```

### Search Certificates

Search for certificates.

**Endpoint:** `GET /api/v1/certs/search`

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | int | Page number |
| page_size | int | Items per page |

**Response:**

```json
{
  "certificates": [...],
  "total": 50,
  "page": 1,
  "page_size": 20,
  "total_pages": 3
}
```

### Get Certificate

Get a certificate by ID.

**Endpoint:** `GET /api/v1/certs/{cert_id}`

### Generate Proof

Generate a minimal disclosure proof.

**Endpoint:** `POST /api/v1/certs/{cert_id}/prove`

**Request Body:**

```json
{
  "disclose_events": [0, 2, 4],
  "disclose_fields": ["prompt", "response"]
}
```

**Response:**

```json
{
  "cert_id": "cert-xyz789",
  "root_hash": "sha256:abc123...",
  "disclosed_events": [...],
  "merkle_proofs": [
    {
      "event_index": 0,
      "siblings": ["sha256:...", "sha256:..."],
      "direction": [0, 1, 0]
    }
  ],
  "metadata": {...}
}
```

---

## Event Types

| Type | Description |
|------|-------------|
| llm.input | User input/prompt |
| llm.output | Model output/response |
| llm.chunk | Streaming chunk |
| llm.tool_call | Tool/function call |
| llm.tool_result | Tool/function result |
| llm.error | Error event |

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| invalid_request | 400 | Request validation failed |
| unauthorized | 401 | Invalid or missing API key |
| forbidden | 403 | Access denied |
| not_found | 404 | Resource not found |
| rate_limited | 429 | Too many requests |
| internal_error | 500 | Server error |

---

## Rate Limits

| Tier | Requests/min | Events/batch |
|------|--------------|--------------|
| Free | 60 | 100 |
| Pro | 600 | 1000 |
| Enterprise | Unlimited | 10000 |

---

## SDKs

Official SDKs are available for:

- **Python**: `pip install ai-trace`
- **Go**: `go get github.com/ai-trace/sdk-go`
- **Java**: Maven/Gradle dependency
- **JavaScript**: `npm install @ai-trace/sdk`

See the SDK documentation for language-specific usage examples.
