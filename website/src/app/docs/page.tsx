'use client'

import Link from 'next/link'
import { useState } from 'react'

type TabId = 'quickstart' | 'integration' | 'api' | 'sdk' | 'verify'

export default function DocsPage() {
  const [activeTab, setActiveTab] = useState<TabId>('quickstart')

  const tabs: { id: TabId; label: string }[] = [
    { id: 'quickstart', label: 'Quick Start' },
    { id: 'integration', label: 'Multi-Model Integration' },
    { id: 'api', label: 'API Reference' },
    { id: 'sdk', label: 'Python SDK' },
    { id: 'verify', label: 'Verification' },
  ]

  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-900 to-gray-800">
      {/* Navigation */}
      <nav className="fixed top-0 w-full bg-gray-900/80 backdrop-blur-sm border-b border-gray-800 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16 items-center">
            <Link href="/" className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-blue-500 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-sm">AT</span>
              </div>
              <span className="text-white font-semibold text-xl">AI-Trace</span>
            </Link>
            <div className="flex items-center space-x-8">
              <Link href="/#features" className="text-gray-300 hover:text-white transition">Features</Link>
              <Link href="/#pricing" className="text-gray-300 hover:text-white transition">Pricing</Link>
              <Link href="https://github.com/ai-trace/ai-trace" className="text-gray-300 hover:text-white transition">GitHub</Link>
            </div>
          </div>
        </div>
      </nav>

      <div className="pt-24 pb-20 px-4">
        <div className="max-w-5xl mx-auto">
          <h1 className="text-4xl font-bold text-white mb-4">Documentation</h1>
          <p className="text-gray-400 mb-8">Everything you need to integrate AI-Trace into your application</p>

          {/* Tabs */}
          <div className="flex space-x-1 bg-gray-800 p-1 rounded-lg mb-8 overflow-x-auto">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`px-4 py-2 rounded-md text-sm font-medium transition whitespace-nowrap ${
                  activeTab === tab.id
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Quick Start Tab */}
          {activeTab === 'quickstart' && (
            <div className="space-y-8">
              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">1. Deploy AI-Trace</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <div className="mb-6">
                    <h4 className="text-white font-medium mb-2 flex items-center gap-2">
                      <span className="bg-green-500/20 text-green-400 text-xs px-2 py-1 rounded">Recommended</span>
                      Simple Mode (SQLite)
                    </h4>
                    <p className="text-gray-400 mb-3">Zero dependencies - perfect for getting started:</p>
                    <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                      <code className="text-green-400">{`# Clone and start in simple mode
git clone https://github.com/ai-trace/ai-trace.git
cd ai-trace/server

# Single container, SQLite storage
docker compose -f docker-compose.simple.yml up -d

# Verify it's running
curl http://localhost:8006/health`}</code>
                    </pre>
                  </div>
                  <div>
                    <h4 className="text-white font-medium mb-2">Standard Mode (PostgreSQL + Redis + MinIO)</h4>
                    <p className="text-gray-400 mb-3">For production deployments:</p>
                    <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                      <code className="text-green-400">{`# Full stack deployment
docker compose up -d

# Services: API (8006), PostgreSQL (5432), Redis (6379), MinIO (9000)`}</code>
                    </pre>
                  </div>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">2. Get Your API Key</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <p className="text-gray-400 mb-4">Create an API key via the console or CLI:</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`# Via Console
# Open http://localhost:3000 → Settings → API Keys → Create

# Via API (with admin token)
curl -X POST http://localhost:8006/api/v1/api-keys \\
  -H "Authorization: Bearer <admin-token>" \\
  -H "Content-Type: application/json" \\
  -d '{"name": "my-app", "scopes": ["trace:write", "cert:write"]}'`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">3. Integrate with Your App</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <p className="text-gray-400 mb-4">Point your OpenAI SDK to AI-Trace with API Key passthrough:</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`from openai import OpenAI

# API Key Passthrough: Your key goes directly to upstream, never stored
client = OpenAI(
    api_key="your-ai-trace-key",           # AI-Trace API Key
    base_url="http://localhost:8006/api/v1",
    default_headers={
        "X-Upstream-API-Key": "sk-your-openai-key"  # Passed to OpenAI
    }
)

# Use exactly as before
response = client.chat.completions.create(
    model="gpt-4",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "What is 2+2?"}
    ]
)

print(response.choices[0].message.content)
# Response includes X-Trace-Id header for certificate generation`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">4. Generate Certificate</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <p className="text-gray-400 mb-4">Create a tamper-proof certificate for any trace:</p>
                  <div className="bg-gray-900 rounded-lg p-4 mb-4">
                    <h4 className="text-white font-medium mb-2">Evidence Levels</h4>
                    <div className="grid grid-cols-3 gap-3 text-sm">
                      <div className="bg-blue-500/10 border border-blue-500/30 rounded p-2">
                        <span className="text-blue-400 font-mono">internal</span>
                        <p className="text-gray-500 text-xs mt-1">Ed25519 signature</p>
                      </div>
                      <div className="bg-purple-500/10 border border-purple-500/30 rounded p-2">
                        <span className="text-purple-400 font-mono">compliance</span>
                        <p className="text-gray-500 text-xs mt-1">+ WORM + TSA</p>
                      </div>
                      <div className="bg-green-500/10 border border-green-500/30 rounded p-2">
                        <span className="text-green-400 font-mono">legal</span>
                        <p className="text-gray-500 text-xs mt-1">+ Blockchain</p>
                      </div>
                    </div>
                  </div>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`# Get the trace_id from X-Trace-Id response header
curl -X POST http://localhost:8006/api/v1/certs/commit \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: your-api-key" \\
  -d '{
    "trace_id": "trc_abc123",
    "evidence_level": "internal"
  }'

# Response:
{
  "cert_id": "cert_xyz789",
  "merkle_root": "a1b2c3...",
  "created_at": "2025-01-15T10:30:00Z",
  "evidence_level": "internal",
  "event_count": 3
}`}</code>
                  </pre>
                  <p className="text-gray-500 text-sm mt-4">
                    Tip: Check the <code className="text-blue-400">X-AI-Trace-Hint</code> response header for the exact certificate command!
                  </p>
                </div>
              </section>
            </div>
          )}

          {/* Multi-Model Integration Tab */}
          {activeTab === 'integration' && (
            <div className="space-y-8">
              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">API Key Passthrough</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <p className="text-gray-400 mb-4">
                    AI-Trace uses <strong className="text-white">API Key Passthrough</strong> - your upstream API keys are
                    passed directly to LLM providers and <strong className="text-blue-400">never stored</strong> on our servers.
                  </p>
                  <div className="bg-gray-950 rounded-lg p-4 mb-4">
                    <pre className="text-gray-300 text-sm">{`Your App → AI-Trace (8006) → LLM Provider
           ↓                    ↓
    X-API-Key: ai-trace-key    Authorization: Bearer sk-xxx
    X-Upstream-API-Key: sk-xxx  (your key passed through)`}</pre>
                  </div>
                  <p className="text-gray-500 text-sm">
                    Headers: <code className="text-blue-400">X-Upstream-API-Key</code> for the upstream LLM key,
                    <code className="text-blue-400 ml-1">X-Upstream-Base-URL</code> for custom endpoints.
                  </p>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">OpenAI Integration</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`from openai import OpenAI

client = OpenAI(
    api_key="your-ai-trace-key",
    base_url="http://localhost:8006/api/v1",
    default_headers={
        "X-Upstream-API-Key": "sk-your-openai-key"
    }
)

response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">DeepSeek Integration</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`from openai import OpenAI

# DeepSeek uses OpenAI-compatible API
client = OpenAI(
    api_key="your-ai-trace-key",
    base_url="http://localhost:8006/api/v1",
    default_headers={
        "X-Upstream-API-Key": "sk-your-deepseek-key"
    }
)

response = client.chat.completions.create(
    model="deepseek-chat",  # or deepseek-coder
    messages=[{"role": "user", "content": "你好！"}]
)`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Claude (Anthropic) Integration</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`from openai import OpenAI

# Claude via AI-Trace (uses OpenAI-compatible wrapper)
client = OpenAI(
    api_key="your-ai-trace-key",
    base_url="http://localhost:8006/api/v1",
    default_headers={
        "X-Upstream-API-Key": "your-anthropic-key"
    }
)

response = client.chat.completions.create(
    model="claude-3-opus-20240229",
    messages=[{"role": "user", "content": "Hello Claude!"}]
)`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Ollama (Local) Integration</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`from openai import OpenAI

# Ollama - no upstream key needed (local model)
client = OpenAI(
    api_key="your-ai-trace-key",
    base_url="http://localhost:8006/api/v1"
)

response = client.chat.completions.create(
    model="llama3.2",  # or qwen2, mistral, etc.
    messages=[{"role": "user", "content": "Hello!"}]
)`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">cURL Examples</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`# OpenAI via AI-Trace
curl -X POST http://localhost:8006/api/v1/chat/completions \\
  -H "X-API-Key: your-ai-trace-key" \\
  -H "X-Upstream-API-Key: sk-your-openai-key" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"Hi"}]}'

# DeepSeek via AI-Trace
curl -X POST http://localhost:8006/api/v1/chat/completions \\
  -H "X-API-Key: your-ai-trace-key" \\
  -H "X-Upstream-API-Key: sk-your-deepseek-key" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"Hi"}]}'

# Custom upstream URL (e.g., Azure OpenAI)
curl -X POST http://localhost:8006/api/v1/chat/completions \\
  -H "X-API-Key: your-ai-trace-key" \\
  -H "X-Upstream-API-Key: your-azure-key" \\
  -H "X-Upstream-Base-URL: https://your-resource.openai.azure.com" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"Hi"}]}'`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Supported Models</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-gray-700">
                        <th className="text-left text-white p-3">Provider</th>
                        <th className="text-left text-white p-3">Models</th>
                        <th className="text-left text-white p-3">Status</th>
                      </tr>
                    </thead>
                    <tbody className="text-gray-300">
                      <tr className="border-b border-gray-700">
                        <td className="p-3 font-medium">OpenAI</td>
                        <td className="p-3">gpt-4, gpt-4-turbo, gpt-3.5-turbo</td>
                        <td className="p-3"><span className="text-green-400">✓ Full Support</span></td>
                      </tr>
                      <tr className="border-b border-gray-700">
                        <td className="p-3 font-medium">DeepSeek</td>
                        <td className="p-3">deepseek-chat, deepseek-coder</td>
                        <td className="p-3"><span className="text-green-400">✓ Full Support</span></td>
                      </tr>
                      <tr className="border-b border-gray-700">
                        <td className="p-3 font-medium">Anthropic</td>
                        <td className="p-3">claude-3-opus, claude-3-sonnet, claude-3-haiku</td>
                        <td className="p-3"><span className="text-green-400">✓ Full Support</span></td>
                      </tr>
                      <tr className="border-b border-gray-700">
                        <td className="p-3 font-medium">Ollama</td>
                        <td className="p-3">llama3, qwen2, mistral, codellama, phi</td>
                        <td className="p-3"><span className="text-green-400">✓ Full Support</span></td>
                      </tr>
                      <tr>
                        <td className="p-3 font-medium">Azure OpenAI</td>
                        <td className="p-3">All Azure-hosted models</td>
                        <td className="p-3"><span className="text-yellow-400">◐ Coming Soon</span></td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </section>
            </div>
          )}

          {/* API Reference Tab */}
          {activeTab === 'api' && (
            <div className="space-y-8">
              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Authentication</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <p className="text-gray-400 mb-4">All API requests require authentication via API key:</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`# Header authentication
X-API-Key: your-api-key

# Or Bearer token
Authorization: Bearer your-api-key`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Endpoints</h2>

                {/* Chat Completions */}
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 mb-4">
                  <div className="flex items-center gap-3 mb-4">
                    <span className="bg-green-600 text-white text-xs px-2 py-1 rounded">POST</span>
                    <code className="text-white">/api/v1/chat/completions</code>
                  </div>
                  <p className="text-gray-400 mb-4">OpenAI-compatible chat completion endpoint. Proxies to upstream LLM and records all events.</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto text-sm">
                    <code className="text-gray-300">{`# Request
{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "temperature": 0.7
}

# Response (OpenAI compatible)
{
  "id": "chatcmpl-xxx",
  "choices": [...],
  "usage": {...}
}

# Response Headers:
X-Trace-Id: trc_abc123
X-AI-Trace-Summary: Traced[abc123] | 3 events | hash:a1b2c3d4
X-AI-Trace-Events: 3
X-AI-Trace-Hash: a1b2c3d4
X-AI-Trace-Hint: Generate certificate: POST /api/v1/certs/commit...`}</code>
                  </pre>
                </div>

                {/* Events Search */}
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 mb-4">
                  <div className="flex items-center gap-3 mb-4">
                    <span className="bg-blue-600 text-white text-xs px-2 py-1 rounded">GET</span>
                    <code className="text-white">/api/v1/events</code>
                  </div>
                  <p className="text-gray-400 mb-4">Search and list captured events.</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto text-sm">
                    <code className="text-gray-300">{`# Query Parameters
?trace_id=trc_xxx     # Filter by trace
?event_type=INPUT     # Filter by type (INPUT|MODEL|OUTPUT|TOOL_CALL)
?start_time=2024-01-01T00:00:00Z
?end_time=2024-01-02T00:00:00Z
?limit=100
?offset=0

# Response
{
  "events": [
    {
      "event_id": "evt_xxx",
      "trace_id": "trc_xxx",
      "event_type": "INPUT",
      "payload_hash": "sha256:...",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 150,
  "has_more": true
}`}</code>
                  </pre>
                </div>

                {/* Cert Commit */}
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 mb-4">
                  <div className="flex items-center gap-3 mb-4">
                    <span className="bg-green-600 text-white text-xs px-2 py-1 rounded">POST</span>
                    <code className="text-white">/api/v1/certs/commit</code>
                  </div>
                  <p className="text-gray-400 mb-4">Generate a tamper-proof certificate for a trace.</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto text-sm">
                    <code className="text-gray-300">{`# Request
{
  "trace_id": "trc_xxx",
  "evidence_level": "internal",  // internal | compliance | legal
  "metadata": {                  // optional
    "purpose": "audit",
    "requester": "compliance-team"
  }
}

# Response
{
  "cert_id": "cert_xxx",
  "trace_id": "trc_xxx",
  "merkle_root": "a1b2c3d4...",
  "evidence_level": "internal",
  "event_count": 5,
  "created_at": "2025-01-15T10:30:00Z"
}

# Evidence Levels:
# internal   - Ed25519 signature (instant)
# compliance - + WORM storage + TSA timestamp
# legal      - + Blockchain anchoring`}</code>
                  </pre>
                </div>

                {/* Cert Verify */}
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 mb-4">
                  <div className="flex items-center gap-3 mb-4">
                    <span className="bg-green-600 text-white text-xs px-2 py-1 rounded">POST</span>
                    <code className="text-white">/api/v1/certs/verify</code>
                  </div>
                  <p className="text-gray-400 mb-4">Verify a certificate&apos;s integrity.</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto text-sm">
                    <code className="text-gray-300">{`# Request
{
  "cert_id": "cert_xxx"
}
# Or upload certificate file

# Response
{
  "valid": true,
  "cert_id": "cert_xxx",
  "merkle_root_match": true,
  "signature_valid": true,
  "timestamp_valid": true,
  "verified_at": "2024-01-15T10:35:00Z"
}`}</code>
                  </pre>
                </div>

                {/* Cert Download */}
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <div className="flex items-center gap-3 mb-4">
                    <span className="bg-blue-600 text-white text-xs px-2 py-1 rounded">GET</span>
                    <code className="text-white">/api/v1/certs/:cert_id/download</code>
                  </div>
                  <p className="text-gray-400 mb-4">Download certificate as JSON file for offline verification.</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto text-sm">
                    <code className="text-gray-300">{`# Response: application/json file download
{
  "version": "1.0",
  "cert_id": "cert_xxx",
  "trace_id": "trc_xxx",
  "merkle_root": "a1b2c3d4...",
  "merkle_tree": {...},
  "events": [...],
  "signature": "...",
  "evidence_level": "internal",
  "created_at": "2024-01-15T10:30:00Z"
}`}</code>
                  </pre>
                </div>
              </section>
            </div>
          )}

          {/* Python SDK Tab */}
          {activeTab === 'sdk' && (
            <div className="space-y-8">
              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Installation</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-green-400">{`pip install ai-trace`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Basic Usage</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`from ai_trace import AITraceClient

# Initialize client
client = AITraceClient(
    base_url="http://localhost:8006",
    api_key="your-api-key"
)

# Search events by trace
events = client.events.search(trace_id="trc_xxx")
for event in events:
    print(f"{event.event_type}: {event.created_at}")

# Create certificate
cert = client.certs.commit(
    trace_id="trc_xxx",
    evidence_level="internal",  # or "compliance", "legal"
    metadata={"purpose": "audit"}
)
print(f"Certificate created: {cert.cert_id}")

# Verify certificate
result = client.certs.verify(cert_id=cert.cert_id)
print(f"Valid: {result.valid}")`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">With OpenAI Integration</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`from openai import OpenAI
from ai_trace import AITraceClient

# Setup clients
openai_client = OpenAI(
    api_key="sk-...",
    base_url="http://localhost:8006/api/v1"  # AI-Trace gateway
)
trace_client = AITraceClient(
    base_url="http://localhost:8006",
    api_key="your-trace-api-key"
)

# Make AI request (automatically traced)
response = openai_client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Explain quantum computing"}]
)

# Get trace ID from response header
trace_id = response.headers.get("X-Trace-Id")

# Generate certificate for compliance
cert = trace_client.certs.commit(
    trace_id=trace_id,
    evidence_level="compliance",  # With WORM + TSA timestamp
    metadata={
        "department": "research",
        "project": "quantum-explainer"
    }
)

# Download certificate for records
cert_file = trace_client.certs.download(cert.cert_id)
with open(f"audit/{cert.cert_id}.json", "wb") as f:
    f.write(cert_file)`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Batch Operations</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`# List all traces in a time range
traces = client.traces.list(
    start_time="2024-01-01T00:00:00Z",
    end_time="2024-01-31T23:59:59Z",
    limit=1000
)

# Bulk generate certificates
for trace in traces:
    if not trace.has_certificate:
        cert = client.certs.commit(
            trace_id=trace.trace_id,
            evidence_level="internal"
        )
        print(f"Created cert {cert.cert_id} for {trace.trace_id}")`}</code>
                  </pre>
                </div>
              </section>
            </div>
          )}

          {/* Verification Tab */}
          {activeTab === 'verify' && (
            <div className="space-y-8">
              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Why Verification Matters</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <p className="text-gray-400">
                    AI-Trace certificates are designed for <strong className="text-white">independent verification</strong>.
                    Anyone with the certificate file can verify its integrity without needing access to the AI-Trace server.
                    This is crucial for regulatory compliance, legal proceedings, and third-party audits.
                  </p>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">CLI Verifier</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <p className="text-gray-400 mb-4">Install the open-source offline verifier:</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto mb-4">
                    <code className="text-green-400">{`# Install via Go
go install github.com/ai-trace/ai-trace/cmd/ai-trace-verify@latest

# Or download binary from releases
curl -L https://github.com/ai-trace/ai-trace/releases/latest/download/ai-trace-verify-linux-amd64 -o ai-trace-verify
chmod +x ai-trace-verify`}</code>
                  </pre>
                  <p className="text-gray-400 mb-4">Verify a certificate:</p>
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`# Basic verification
ai-trace-verify --cert certificate.json

# Output:
Certificate: cert_xyz789
Trace ID:    trc_abc123
Events:      5
Evidence:    internal (Ed25519)

Verification Results:
  ✓ Merkle root matches
  ✓ All event hashes valid
  ✓ Signature verified
  ✓ Certificate structure valid

Status: VALID`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">Verification via API</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <pre className="bg-gray-950 rounded-lg p-4 overflow-x-auto">
                    <code className="text-gray-300">{`# Verify by cert_id
curl -X POST http://localhost:8006/api/v1/certs/verify \\
  -H "Content-Type: application/json" \\
  -d '{"cert_id": "cert_xyz789"}'

# Verify by uploading certificate file
curl -X POST http://localhost:8006/api/v1/certs/verify \\
  -H "Content-Type: application/json" \\
  -d @certificate.json`}</code>
                  </pre>
                </div>
              </section>

              <section>
                <h2 className="text-2xl font-semibold text-white mb-4">What Gets Verified</h2>
                <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-gray-700">
                        <th className="text-left text-white p-3">Check</th>
                        <th className="text-left text-white p-3">Description</th>
                      </tr>
                    </thead>
                    <tbody className="text-gray-300">
                      <tr className="border-b border-gray-700">
                        <td className="p-3 font-medium">Merkle Root</td>
                        <td className="p-3">Recomputes tree from events, compares with stored root</td>
                      </tr>
                      <tr className="border-b border-gray-700">
                        <td className="p-3 font-medium">Event Hashes</td>
                        <td className="p-3">Each event payload hash matches its recorded hash</td>
                      </tr>
                      <tr className="border-b border-gray-700">
                        <td className="p-3 font-medium">Signature</td>
                        <td className="p-3">Certificate signed by valid AI-Trace key</td>
                      </tr>
                      <tr className="border-b border-gray-700">
                        <td className="p-3 font-medium">Timestamp (compliance+)</td>
                        <td className="p-3">TSA timestamp is valid and within tolerance</td>
                      </tr>
                      <tr>
                        <td className="p-3 font-medium">Blockchain (legal)</td>
                        <td className="p-3">On-chain anchor matches certificate hash</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </section>
            </div>
          )}

          {/* Bottom Links */}
          <div className="mt-12 pt-8 border-t border-gray-800">
            <h3 className="text-lg font-semibold text-white mb-4">More Resources</h3>
            <div className="grid md:grid-cols-3 gap-4">
              <Link href="https://github.com/ai-trace/ai-trace" className="block bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-blue-500 transition">
                <h4 className="text-white font-medium mb-1">GitHub Repository</h4>
                <p className="text-gray-400 text-sm">Source code, issues, discussions</p>
              </Link>
              <Link href="https://github.com/ai-trace/ai-trace/tree/main/docs" className="block bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-blue-500 transition">
                <h4 className="text-white font-medium mb-1">Full Documentation</h4>
                <p className="text-gray-400 text-sm">Architecture, deployment, advanced topics</p>
              </Link>
              <Link href="/contact" className="block bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-blue-500 transition">
                <h4 className="text-white font-medium mb-1">Get Help</h4>
                <p className="text-gray-400 text-sm">Contact us for support</p>
              </Link>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-8 px-4">
        <div className="max-w-5xl mx-auto text-center text-gray-500 text-sm">
          © 2025 AI-Trace. Open source under Apache 2.0.
        </div>
      </footer>
    </div>
  )
}
