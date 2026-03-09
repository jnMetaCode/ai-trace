# AI-Trace API Reference

Complete REST API documentation for AI-Trace.

## Base URL

```
http://localhost:8006/api/v1
```

## Authentication

Include your API key in the request header:

```
X-API-Key: your-api-key
```

For multi-tenant deployments:
```
X-Tenant-ID: your-tenant-id
```

## Response Format

All responses follow this format:

```json
{
  "data": { ... },       // Response data (on success)
  "error": "message",    // Error message (on failure)
  "code": "ERROR_CODE"   // Error code (on failure)
}
```

HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `429` - Rate Limited
- `500` - Server Error

---

## Gateway API

### Chat Completions (OpenAI Compatible)

Create a chat completion through the AI-Trace gateway.

```http
POST /api/v1/chat/completions
```

**Headers:**
| Header | Required | Description |
|--------|----------|-------------|
| X-API-Key | Yes | AI-Trace API key |
| X-Upstream-API-Key | Yes | Provider API key (OpenAI/Claude) |
| X-Tenant-ID | No | Tenant identifier |
| X-Provider | No | Provider: openai, anthropic, ollama |

**Request Body:**
```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "temperature": 0.7,
  "max_tokens": 1000,
  "stream": false
}
```

**Response:**
```json
{
  "id": "chatcmpl-abc123",
  "trace_id": "trc_xyz789",
  "object": "chat.completion",
  "created": 1705312800,
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
  }
}
```

---

## Traces API

### Create Trace

Create a new trace to track AI decisions.

```http
POST /api/v1/traces
```

**Request Body:**
```json
{
  "name": "Customer Support Chat",
  "tenant_id": "default",
  "user_id": "user-123",
  "session_id": "sess-456",
  "metadata": {
    "department": "support",
    "priority": "high"
  }
}
```

**Response:**
```json
{
  "trace_id": "trc_abc123def456",
  "tenant_id": "default",
  "name": "Customer Support Chat",
  "user_id": "user-123",
  "session_id": "sess-456",
  "created_at": "2024-01-15T10:30:00Z",
  "status": "active",
  "event_count": 0
}
```

### Get Trace

```http
GET /api/v1/traces/{trace_id}
```

**Response:**
```json
{
  "trace_id": "trc_abc123def456",
  "tenant_id": "default",
  "name": "Customer Support Chat",
  "created_at": "2024-01-15T10:30:00Z",
  "status": "completed",
  "event_count": 5,
  "metadata": { ... }
}
```

### List Traces

```http
GET /api/v1/traces?limit=20&offset=0&tenant_id=default
```

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | integer | 20 | Max results (1-100) |
| offset | integer | 0 | Pagination offset |
| tenant_id | string | - | Filter by tenant |
| user_id | string | - | Filter by user |
| status | string | - | Filter by status |

**Response:**
```json
{
  "items": [
    { "trace_id": "trc_001", ... },
    { "trace_id": "trc_002", ... }
  ],
  "total": 156,
  "limit": 20,
  "offset": 0,
  "has_more": true
}
```

---

## Events API

### Ingest Event

Add an event to a trace.

```http
POST /api/v1/events/ingest
```

**Request Body:**
```json
{
  "trace_id": "trc_abc123def456",
  "event_type": "input",
  "payload": {
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "What is AI-Trace?"}
    ]
  },
  "metadata": {
    "source": "api",
    "version": "1.0"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Event Types:**
- `input` - User input/prompt
- `output` - AI response
- `custom` - Application-defined events

**Response:**
```json
{
  "event_id": "evt_xyz789",
  "trace_id": "trc_abc123def456",
  "event_type": "input",
  "sequence": 0,
  "timestamp": "2024-01-15T10:30:00Z",
  "hash": "sha256:3f2a1b..."
}
```

### Get Event

```http
GET /api/v1/events/{event_id}
```

### Search Events

```http
GET /api/v1/events/search?trace_id=trc_xxx&limit=100
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| trace_id | string | Filter by trace |
| event_type | string | Filter by type |
| limit | integer | Max results |
| offset | integer | Pagination offset |

---

## Certificates API

### Commit Certificate

Create a certificate from a trace.

```http
POST /api/v1/certs/commit
```

**Request Body:**
```json
{
  "trace_id": "trc_abc123def456",
  "evidence_level": "compliance",
  "metadata": {
    "purpose": "compliance-audit"
  }
}
```

**Evidence Levels:**
| Level | Description | Use Case |
|-------|-------------|----------|
| `internal` | Ed25519 signature | Internal audit, development |
| `compliance` | WORM storage + TSA timestamp | SOC2, GDPR, HIPAA compliance |
| `legal` | Blockchain anchor | Legal disputes, court evidence |

> Note: Legacy names `L1`, `L2`, `L3` are still supported for backwards compatibility.

**Response:**
```json
{
  "cert_id": "cert_xyz789",
  "trace_id": "trc_abc123def456",
  "evidence_level": "compliance",
  "root_hash": "sha256:9c7b8a...",
  "event_count": 5,
  "signature": "ed25519:...",
  "created_at": "2024-01-15T10:35:00Z",
  "tsa_timestamp": "...",
  "worm_location": "s3://bucket/cert_xyz789"
}
```

### Verify Certificate

```http
POST /api/v1/certs/verify
```

**Request Body:**
```json
{
  "cert_id": "cert_xyz789",
  "full_verification": true
}
```

**Response:**
```json
{
  "valid": true,
  "cert_id": "cert_xyz789",
  "evidence_level": "compliance",
  "hash_valid": true,
  "signature_valid": true,
  "timestamp_valid": true,
  "anchor_verified": true,
  "verified_at": "2024-01-15T10:40:00Z",
  "details": {
    "merkle_root": "sha256:9c7b8a...",
    "event_count": 5
  }
}
```

### Get Certificate

```http
GET /api/v1/certs/{cert_id}?include_events=true
```

### List Certificates

```http
GET /api/v1/certs/search?evidence_level=compliance&limit=20
```

### Generate Proof

Generate a minimal disclosure proof.

```http
POST /api/v1/certs/{cert_id}/prove
```

**Request Body:**
```json
{
  "event_indices": [0, 2, 4],
  "disclosed_fields": ["model", "timestamp"]
}
```

**Response:**
```json
{
  "proof_id": "prf_abc123",
  "cert_id": "cert_xyz789",
  "root_hash": "sha256:9c7b8a...",
  "created_at": "2024-01-15T10:45:00Z",
  "event_indices": [0, 2, 4],
  "disclosed_events": [
    {"model": "gpt-4", "timestamp": "..."},
    {"model": "gpt-4", "timestamp": "..."},
    {"model": "gpt-4", "timestamp": "..."}
  ],
  "merkle_proofs": [
    {
      "leaf_hash": "sha256:...",
      "siblings": ["sha256:...", "sha256:..."],
      "path": [0, 1],
      "root_hash": "sha256:9c7b8a..."
    }
  ]
}
```

---

## Reports API

### Generate Report

```http
POST /api/v1/reports/generate
```

**Request Body:**
```json
{
  "trace_ids": ["trc_001", "trc_002"],
  "report_type": "audit",
  "format": "pdf",
  "options": {
    "include_events": true,
    "include_proofs": true
  }
}
```

**Response:**
```json
{
  "report_id": "rpt_abc123",
  "status": "completed",
  "download_url": "/api/v1/reports/rpt_abc123/download",
  "created_at": "2024-01-15T11:00:00Z",
  "expires_at": "2024-01-16T11:00:00Z"
}
```

---

## Health API

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": 86400,
  "services": {
    "database": "connected",
    "redis": "connected",
    "minio": "connected"
  }
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| `INVALID_REQUEST` | Request validation failed |
| `UNAUTHORIZED` | Authentication required |
| `FORBIDDEN` | Insufficient permissions |
| `NOT_FOUND` | Resource not found |
| `CONFLICT` | Resource already exists |
| `RATE_LIMITED` | Too many requests |
| `INTERNAL_ERROR` | Server error |
| `UPSTREAM_ERROR` | Provider error |

---

## Rate Limits

| Endpoint | Limit |
|----------|-------|
| Gateway | 100 req/min |
| Events | 1000 req/min |
| Certs | 50 req/min |
| Reports | 10 req/min |

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705312860
```

---

## Webhooks (Coming Soon)

```json
{
  "webhook_url": "https://your-app.com/webhook",
  "events": ["cert.created", "cert.verified"],
  "secret": "your-webhook-secret"
}
```

---

## SDK Examples

### Python

```python
from ai_trace import AITrace

client = AITrace(server_url="http://localhost:8006", api_key="key")
trace = client.traces.create(name="Test")
cert = client.certs.commit(trace_id=trace.id)
```

### cURL

```bash
curl -X POST http://localhost:8006/api/v1/traces \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"name": "Test"}'
```

### JavaScript

```javascript
const response = await fetch('http://localhost:8006/api/v1/traces', {
  method: 'POST',
  headers: {
    'X-API-Key': 'your-key',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ name: 'Test' })
});
```
