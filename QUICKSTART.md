# AI-Trace Quick Start Guide

## One-Line Deployment

```bash
curl -sSL https://raw.githubusercontent.com/ai-trace/ai-trace/main/scripts/deploy.sh | bash
```

Or clone and deploy:

```bash
git clone https://github.com/jnMetaCode/ai-trace.git
cd ai-trace
./scripts/deploy.sh deploy
```

## Deployment Options

### 1. Docker Compose (Recommended for getting started)

```bash
./scripts/deploy.sh deploy
```

Services started:
- **API Server**: http://localhost:8006
- **Swagger UI**: http://localhost:8006/swagger/index.html
- **MinIO Console**: http://localhost:9000

### 2. Kubernetes

```bash
./scripts/deploy.sh k8s production
```

### 3. Verifier Node Only

Run a lightweight verification-only node:

```bash
./scripts/deploy.sh verifier
```

## Integration

### Python

```python
from ai_trace import AITraceClient

client = AITraceClient(
    api_key="YOUR_AI_TRACE_KEY",
    upstream_api_key="sk-YOUR_OPENAI_KEY"  # Passed through, NOT stored
)

# Call AI
response = client.chat.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)

# Generate certificate
cert = client.certs.commit(trace_id="xxx", evidence_level="L2")
```

### JavaScript

```javascript
import { AITraceClient } from 'ai-trace';

const client = new AITraceClient({
  apiKey: 'YOUR_AI_TRACE_KEY',
  upstreamApiKey: 'sk-YOUR_OPENAI_KEY'
});

const response = await client.chat.create({
  model: 'gpt-4',
  messages: [{ role: 'user', content: 'Hello!' }]
});
```

### cURL

```bash
# Chat completion
curl -X POST http://localhost:8006/api/v1/chat/completions \
  -H "X-API-Key: YOUR_AI_TRACE_KEY" \
  -H "X-Upstream-API-Key: sk-YOUR_OPENAI_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# Generate certificate
curl -X POST http://localhost:8006/api/v1/certs/commit \
  -H "X-API-Key: YOUR_AI_TRACE_KEY" \
  -H "Content-Type: application/json" \
  -d '{"trace_id": "trc_xxx", "evidence_level": "compliance"}'

# Verify certificate
curl -X POST http://localhost:8006/api/v1/certs/verify \
  -H "X-API-Key: YOUR_AI_TRACE_KEY" \
  -H "Content-Type: application/json" \
  -d '{"cert_id": "cert_xxx"}'
```

## Deployment Modes

| Mode | Description | API Key Handling |
|------|-------------|------------------|
| **Trust Mode** | Use our gateway | Passed through, not stored |
| **Proxy Mode** | Use your own proxy | Keys never touch our servers |
| **Self-Hosted** | Run everything yourself | Complete data sovereignty |

### Proxy Mode Configuration

```bash
# Your API keys route through YOUR proxy, not ours
curl -X POST http://localhost:8006/api/v1/chat/completions \
  -H "X-API-Key: YOUR_AI_TRACE_KEY" \
  -H "X-Upstream-Base-URL: https://your-proxy.com/v1" \
  -H "X-Upstream-API-Key: sk-YOUR_OPENAI_KEY" \
  ...
```

## Evidence Levels

| Level | Storage | Time Proof | Use Case |
|-------|---------|------------|----------|
| **internal** | Local | Ed25519 Signature | Internal audit |
| **compliance** | WORM (MinIO) | TSA | SOC2/GDPR/HIPAA |
| **legal** | Blockchain | On-chain | Legal disputes |

## Federated Verification Nodes

Run your own verification node that participates in the AI-Trace network:

```bash
# Deploy verifier node
./scripts/deploy.sh verifier

# Or with Docker
docker run -d -p 8081:8080 \
  -e VERIFIER_ONLY=true \
  ai-trace/server:latest
```

Configure federated nodes in `config.yaml`:

```yaml
anchor:
  federated_nodes:
    - https://node1.aitrace.cc
    - https://node2.company.com
    - https://node3.partner.org
  min_confirmations: 2
```

## Health Check

```bash
curl http://localhost:8006/health
```

## Logs

```bash
docker compose logs -f ai-trace-server
```

## Stop Services

```bash
./scripts/deploy.sh stop
```

## Support

- Documentation: https://docs.aitrace.cc
- GitHub Issues: https://github.com/jnMetaCode/ai-trace/issues
- Discord: https://discord.gg/ai-trace
