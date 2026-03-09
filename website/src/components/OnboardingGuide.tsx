'use client'

import { useState } from 'react'

interface Step {
  id: number
  title: string
  description: string
  content: React.ReactNode
  duration: string
}

export default function OnboardingGuide() {
  const [currentStep, setCurrentStep] = useState(0)
  const [completedSteps, setCompletedSteps] = useState<number[]>([])
  const [copiedCode, setCopiedCode] = useState<string | null>(null)

  const copyToClipboard = async (code: string, id: string) => {
    await navigator.clipboard.writeText(code)
    setCopiedCode(id)
    setTimeout(() => setCopiedCode(null), 2000)
  }

  const markComplete = (stepId: number) => {
    if (!completedSteps.includes(stepId)) {
      setCompletedSteps([...completedSteps, stepId])
    }
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1)
    }
  }

  const CodeBlock = ({ code, id, language = 'bash' }: { code: string; id: string; language?: string }) => (
    <div className="relative bg-gray-950 rounded-lg border border-gray-800 overflow-hidden mt-4">
      <div className="flex items-center justify-between px-4 py-2 bg-gray-900 border-b border-gray-800">
        <span className="text-gray-500 text-xs">{language}</span>
        <button
          onClick={() => copyToClipboard(code, id)}
          className="text-gray-400 hover:text-white text-sm flex items-center gap-1 transition"
        >
          {copiedCode === id ? (
            <>
              <svg className="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              Copied!
            </>
          ) : (
            <>
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
              Copy
            </>
          )}
        </button>
      </div>
      <pre className="p-4 text-sm overflow-x-auto">
        <code className="text-gray-300">{code}</code>
      </pre>
    </div>
  )

  const steps: Step[] = [
    {
      id: 0,
      title: 'Deploy AI-Trace',
      description: 'Get AI-Trace running in under a minute',
      duration: '1 min',
      content: (
        <div className="space-y-4">
          <p className="text-gray-400">
            The simplest way to start is with our single-container SQLite mode.
            No database setup, no dependencies - just Docker.
          </p>

          <div className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-4">
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-blue-400 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
              <div className="text-sm text-blue-200">
                <strong>Simple Mode</strong> uses SQLite and is perfect for development,
                testing, or small deployments (&lt; 10K events/day).
              </div>
            </div>
          </div>

          <h4 className="text-white font-medium mt-6">Option A: Using Docker Compose (Recommended)</h4>
          <CodeBlock
            id="deploy-compose"
            code={`# Clone the repository
git clone https://github.com/ai-trace/ai-trace.git
cd ai-trace/server

# Start AI-Trace in simple mode
docker compose -f docker-compose.simple.yml up -d

# Check if it's running
curl http://localhost:8006/health`}
          />

          <h4 className="text-white font-medium mt-6">Option B: Using Docker directly</h4>
          <CodeBlock
            id="deploy-docker"
            code={`docker run -d \\
  --name ai-trace \\
  -p 8006:8006 \\
  -e AI_TRACE_MODE=simple \\
  -e AI_TRACE_DEFAULT_API_KEY=my-secret-key \\
  -v ai-trace-data:/data \\
  ghcr.io/ai-trace/ai-trace:latest`}
          />

          <div className="mt-6 p-4 bg-green-500/10 border border-green-500/30 rounded-lg">
            <div className="flex items-center gap-2 text-green-400">
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
              </svg>
              <span className="font-medium">Success indicator:</span>
            </div>
            <p className="text-gray-400 text-sm mt-2">
              You should see <code className="text-green-400 bg-gray-800 px-1 rounded">{`{"status":"healthy"}`}</code> from the health check.
            </p>
          </div>
        </div>
      )
    },
    {
      id: 1,
      title: 'Make Your First Traced Call',
      description: 'Route an AI request through AI-Trace',
      duration: '2 min',
      content: (
        <div className="space-y-4">
          <p className="text-gray-400">
            AI-Trace acts as a transparent proxy. Simply change your API base URL
            to route requests through AI-Trace. Your original API key is passed through
            and never stored.
          </p>

          <h4 className="text-white font-medium mt-6">Using curl</h4>
          <CodeBlock
            id="curl-call"
            code={`curl -X POST http://localhost:8006/api/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: my-secret-key" \\
  -H "X-Upstream-API-Key: sk-your-openai-key" \\
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "What is 2+2?"}]
  }'`}
          />

          <h4 className="text-white font-medium mt-6">Using Python</h4>
          <CodeBlock
            id="python-call"
            language="python"
            code={`from openai import OpenAI

client = OpenAI(
    api_key="sk-your-openai-key",
    base_url="http://localhost:8006/api/v1",  # Route through AI-Trace
    default_headers={"X-API-Key": "my-secret-key"}
)

response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "What is 2+2?"}]
)

print(response.choices[0].message.content)

# Check the response headers for trace info
# X-Trace-Id: trc_abc123...
# X-AI-Trace-Events: 3
# X-AI-Trace-Hint: Generate certificate: POST /api/v1/certs/commit...`}
          />

          <div className="mt-6 p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-lg">
            <div className="flex items-start gap-2">
              <svg className="w-5 h-5 text-yellow-400 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              <div>
                <span className="text-yellow-400 font-medium">Note the response headers!</span>
                <p className="text-gray-400 text-sm mt-1">
                  AI-Trace adds helpful headers like <code className="bg-gray-800 px-1 rounded">X-Trace-Id</code> and
                  <code className="bg-gray-800 px-1 rounded ml-1">X-AI-Trace-Hint</code> to help you understand what happened.
                </p>
              </div>
            </div>
          </div>
        </div>
      )
    },
    {
      id: 2,
      title: 'Generate a Certificate',
      description: 'Create tamper-proof evidence for your trace',
      duration: '1 min',
      content: (
        <div className="space-y-4">
          <p className="text-gray-400">
            After your AI call completes, you can generate a certificate that proves
            exactly what happened. This certificate contains a Merkle root hash of all
            events and is digitally signed.
          </p>

          <div className="bg-gray-800 rounded-lg p-4 border border-gray-700 mt-4">
            <h4 className="text-white font-medium mb-3">Evidence Levels</h4>
            <div className="space-y-3">
              <div className="flex items-start gap-3">
                <span className="bg-blue-500/20 text-blue-400 px-2 py-1 rounded text-xs font-mono">internal</span>
                <div className="text-sm">
                  <span className="text-gray-300">Internal Audit</span>
                  <span className="text-gray-500 ml-2">- Ed25519 signature, instant generation</span>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <span className="bg-purple-500/20 text-purple-400 px-2 py-1 rounded text-xs font-mono">compliance</span>
                <div className="text-sm">
                  <span className="text-gray-300">Compliance Evidence</span>
                  <span className="text-gray-500 ml-2">- + WORM storage + TSA timestamp</span>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <span className="bg-green-500/20 text-green-400 px-2 py-1 rounded text-xs font-mono">legal</span>
                <div className="text-sm">
                  <span className="text-gray-300">Legal Evidence</span>
                  <span className="text-gray-500 ml-2">- + Blockchain anchoring (court-ready)</span>
                </div>
              </div>
            </div>
          </div>

          <h4 className="text-white font-medium mt-6">Generate certificate from trace ID</h4>
          <CodeBlock
            id="gen-cert"
            code={`# Replace with your actual trace ID from the X-Trace-Id header
curl -X POST http://localhost:8006/api/v1/certs/commit \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: my-secret-key" \\
  -d '{
    "trace_id": "trc_abc123...",
    "evidence_level": "internal"
  }'`}
          />

          <h4 className="text-white font-medium mt-6">Response</h4>
          <CodeBlock
            id="cert-response"
            language="json"
            code={`{
  "cert_id": "cert_xyz789...",
  "trace_id": "trc_abc123...",
  "merkle_root": "a1b2c3d4e5f6...",
  "evidence_level": "internal",
  "event_count": 3,
  "signature": "ed25519:...",
  "created_at": "2025-01-15T10:30:00Z"
}`}
          />

          <div className="mt-4 p-4 bg-green-500/10 border border-green-500/30 rounded-lg">
            <p className="text-green-400 text-sm">
              Save your <code className="bg-gray-800 px-1 rounded">cert_id</code> - you&apos;ll need it to verify the certificate later!
            </p>
          </div>
        </div>
      )
    },
    {
      id: 3,
      title: 'Verify the Certificate',
      description: 'Independently verify the integrity of your trace',
      duration: '1 min',
      content: (
        <div className="space-y-4">
          <p className="text-gray-400">
            Anyone with the certificate ID can verify its integrity. The verification
            rebuilds the Merkle tree and checks the digital signature - no trust required.
          </p>

          <h4 className="text-white font-medium mt-6">Using the API</h4>
          <CodeBlock
            id="verify-api"
            code={`curl http://localhost:8006/api/v1/certs/cert_xyz789.../verify \\
  -H "X-API-Key: my-secret-key"`}
          />

          <h4 className="text-white font-medium mt-6">Verification response</h4>
          <CodeBlock
            id="verify-response"
            language="json"
            code={`{
  "valid": true,
  "cert_id": "cert_xyz789...",
  "merkle_root_match": true,
  "signature_valid": true,
  "event_count": 3,
  "verified_at": "2025-01-15T10:35:00Z"
}`}
          />

          <h4 className="text-white font-medium mt-6">Using the CLI (offline verification)</h4>
          <CodeBlock
            id="verify-cli"
            code={`# Download the certificate bundle
curl -o cert.json \\
  http://localhost:8006/api/v1/certs/cert_xyz789.../bundle

# Verify offline with the CLI tool
ai-trace verify cert.json --offline`}
          />

          <div className="mt-6 p-4 bg-blue-500/10 border border-blue-500/30 rounded-lg">
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-blue-400 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
              <div className="text-sm text-blue-200">
                <strong>Why offline verification matters:</strong> You don&apos;t need to trust
                AI-Trace to verify certificates. The CLI tool uses only cryptographic proof -
                Ed25519 signatures and Merkle tree verification.
              </div>
            </div>
          </div>
        </div>
      )
    },
    {
      id: 4,
      title: 'Next Steps',
      description: 'Explore advanced features',
      duration: '5 min',
      content: (
        <div className="space-y-6">
          <p className="text-gray-400">
            Congratulations! You&apos;ve successfully set up AI-Trace, made a traced call,
            generated a certificate, and verified it. Here are some next steps:
          </p>

          <div className="grid md:grid-cols-2 gap-4">
            <a href="/docs" className="block p-4 bg-gray-800 rounded-lg border border-gray-700 hover:border-blue-500/50 transition group">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center">
                  <svg className="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                </div>
                <h4 className="text-white font-medium group-hover:text-blue-400 transition">Full Documentation</h4>
              </div>
              <p className="text-gray-500 text-sm">Complete API reference and guides</p>
            </a>

            <a href="/docs#sdk" className="block p-4 bg-gray-800 rounded-lg border border-gray-700 hover:border-purple-500/50 transition group">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-10 h-10 bg-purple-500/20 rounded-lg flex items-center justify-center">
                  <svg className="w-5 h-5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                  </svg>
                </div>
                <h4 className="text-white font-medium group-hover:text-purple-400 transition">Python SDK</h4>
              </div>
              <p className="text-gray-500 text-sm">Type-safe SDK with auto-completion</p>
            </a>

            <a href="/docs#auto-cert" className="block p-4 bg-gray-800 rounded-lg border border-gray-700 hover:border-green-500/50 transition group">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-10 h-10 bg-green-500/20 rounded-lg flex items-center justify-center">
                  <svg className="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                  </svg>
                </div>
                <h4 className="text-white font-medium group-hover:text-green-400 transition">Auto-Certification</h4>
              </div>
              <p className="text-gray-500 text-sm">Automatically generate certificates based on rules</p>
            </a>

            <a href="/docs#production" className="block p-4 bg-gray-800 rounded-lg border border-gray-700 hover:border-yellow-500/50 transition group">
              <div className="flex items-center gap-3 mb-2">
                <div className="w-10 h-10 bg-yellow-500/20 rounded-lg flex items-center justify-center">
                  <svg className="w-5 h-5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                  </svg>
                </div>
                <h4 className="text-white font-medium group-hover:text-yellow-400 transition">Production Deployment</h4>
              </div>
              <p className="text-gray-500 text-sm">PostgreSQL, Redis, MinIO for scale</p>
            </a>
          </div>

          <div className="mt-6 p-6 bg-gradient-to-r from-blue-600/20 to-purple-600/20 rounded-xl border border-blue-500/30">
            <h4 className="text-white font-semibold mb-2">Need Help?</h4>
            <p className="text-gray-400 text-sm mb-4">
              Join our community or reach out for enterprise support.
            </p>
            <div className="flex flex-wrap gap-3">
              <a href="https://github.com/ai-trace/ai-trace/discussions" className="inline-flex items-center gap-2 bg-gray-800 hover:bg-gray-700 text-white px-4 py-2 rounded-lg text-sm transition">
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                  <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
                </svg>
                GitHub Discussions
              </a>
              <a href="/contact" className="inline-flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm transition">
                Contact Sales
              </a>
            </div>
          </div>
        </div>
      )
    }
  ]

  const progress = ((completedSteps.length) / steps.length) * 100

  return (
    <div className="max-w-4xl mx-auto">
      {/* Progress bar */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-2">
          <span className="text-gray-400 text-sm">Getting Started Progress</span>
          <span className="text-gray-400 text-sm">{completedSteps.length} of {steps.length} complete</span>
        </div>
        <div className="h-2 bg-gray-800 rounded-full overflow-hidden">
          <div
            className="h-full bg-gradient-to-r from-blue-500 to-purple-500 transition-all duration-500"
            style={{ width: `${progress}%` }}
          />
        </div>
      </div>

      {/* Step navigation */}
      <div className="flex flex-wrap gap-2 mb-8">
        {steps.map((step, index) => (
          <button
            key={step.id}
            onClick={() => setCurrentStep(index)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm transition ${
              currentStep === index
                ? 'bg-blue-600 text-white'
                : completedSteps.includes(step.id)
                ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                : 'bg-gray-800 text-gray-400 hover:text-white'
            }`}
          >
            {completedSteps.includes(step.id) ? (
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
              </svg>
            ) : (
              <span className="w-5 h-5 rounded-full bg-gray-700 flex items-center justify-center text-xs">
                {index + 1}
              </span>
            )}
            {step.title}
          </button>
        ))}
      </div>

      {/* Current step content */}
      <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-8">
        <div className="flex items-start justify-between mb-6">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <span className="bg-blue-500/20 text-blue-400 px-2 py-1 rounded text-xs font-medium">
                Step {currentStep + 1} of {steps.length}
              </span>
              <span className="text-gray-500 text-xs">~{steps[currentStep].duration}</span>
            </div>
            <h2 className="text-2xl font-bold text-white">{steps[currentStep].title}</h2>
            <p className="text-gray-400 mt-1">{steps[currentStep].description}</p>
          </div>
        </div>

        <div className="prose prose-invert max-w-none">
          {steps[currentStep].content}
        </div>

        {/* Navigation buttons */}
        <div className="flex items-center justify-between mt-8 pt-6 border-t border-gray-700">
          <button
            onClick={() => currentStep > 0 && setCurrentStep(currentStep - 1)}
            disabled={currentStep === 0}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg transition ${
              currentStep === 0
                ? 'text-gray-600 cursor-not-allowed'
                : 'text-gray-400 hover:text-white'
            }`}
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            Previous
          </button>

          <button
            onClick={() => markComplete(steps[currentStep].id)}
            className={`flex items-center gap-2 px-6 py-3 rounded-lg font-medium transition ${
              completedSteps.includes(steps[currentStep].id)
                ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                : 'bg-blue-600 hover:bg-blue-700 text-white'
            }`}
          >
            {completedSteps.includes(steps[currentStep].id) ? (
              <>
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                </svg>
                Completed
              </>
            ) : currentStep === steps.length - 1 ? (
              'Finish Tutorial'
            ) : (
              <>
                Mark as Complete
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
