'use client'

import Link from 'next/link'
import { useState } from 'react'

interface FAQItem {
  question: string
  answer: string
  category: string
}

const faqItems: FAQItem[] = [
  // Security & Privacy
  {
    category: "Security & Privacy",
    question: "Will my OpenAI/Claude API key be leaked?",
    answer: `No. Your API keys are NEVER stored in our database. They are only used in memory to forward requests to upstream providers (OpenAI, Claude, etc.) and are immediately discarded after the request completes.

You can verify this by:
1. Reviewing our open-source code on GitHub
2. Using Proxy Mode where your keys never touch our servers
3. Self-hosting the entire solution`
  },
  {
    category: "Security & Privacy",
    question: "What data does AI-Trace store?",
    answer: `We store cryptographic hashes and metadata, NOT your actual content:

**Stored (Hashed):**
- Event IDs and Trace IDs
- Event types (INPUT/MODEL/OUTPUT)
- SHA-256 hashes of prompts and responses
- Timestamps and sequence numbers
- Token usage statistics
- Merkle tree structure for verification

**NEVER Stored:**
- Your API keys
- Original prompt content
- AI response content
- System prompts
- Any recoverable plaintext`
  },
  {
    category: "Security & Privacy",
    question: "Can you read my prompts or AI responses?",
    answer: `No. We only store SHA-256 hashes of your content. Cryptographic hashes are one-way functions - it's mathematically impossible to recover the original content from a hash.

For example, if your prompt is "What is the capital of France?", we store something like "a8f5f167f44f4964e6c998dee827110c" - which cannot be reversed to reveal your original text.`
  },
  {
    category: "Security & Privacy",
    question: "How can I verify you're not storing my data?",
    answer: `Multiple ways:

1. **Code Audit**: Our entire codebase is open source. Review it on GitHub.

2. **Proxy Mode**: Use X-Upstream-Base-URL header to route API calls through your own proxy. Your keys and data never touch our servers.

3. **Self-Hosted**: Deploy AI-Trace on your own infrastructure with complete data sovereignty.

4. **Network Inspection**: Use tools like Wireshark to verify what data leaves your network.`
  },

  // Deployment
  {
    category: "Deployment",
    question: "What is Trust Mode?",
    answer: `Trust Mode is the quickest way to get started. Simply change your API base URL to AI-Trace.

Your API keys pass through our gateway to reach OpenAI/Claude, but are never stored. This mode requires trusting that our servers handle your keys securely.

Best for: Quick prototypes, testing, startups who prioritize speed.`
  },
  {
    category: "Deployment",
    question: "What is Proxy Mode?",
    answer: `In Proxy Mode, you set up your own proxy server to call OpenAI/Claude. AI-Trace only receives the hashed metadata for audit purposes - your API keys NEVER touch our servers.

Use the X-Upstream-Base-URL header to specify your proxy:
\`\`\`
X-Upstream-Base-URL: https://your-proxy.company.com/v1
\`\`\`

Best for: Enterprises, teams handling sensitive data.`
  },
  {
    category: "Deployment",
    question: "What is Self-Hosted Mode?",
    answer: `Self-Hosted mode means deploying AI-Trace entirely on your own infrastructure. You have complete control over:

- All data storage
- Network configuration
- Security policies
- Access controls

We provide Docker images and Kubernetes Helm charts for easy deployment. This mode offers maximum security and data sovereignty.

Best for: Regulated industries (healthcare, finance), government, organizations with strict compliance requirements.`
  },

  // Technical
  {
    category: "Technical",
    question: "How does the certificate signing work?",
    answer: `We use Ed25519 digital signatures for all certificates:

1. When you commit a trace, we build a Merkle tree from all event hashes
2. The certificate (containing cert_id, root_hash, evidence_level, timestamp) is signed with our Ed25519 private key
3. The public key is included in the certificate for independent verification

Anyone can verify a certificate's authenticity using the public key, without trusting our servers.`
  },
  {
    category: "Technical",
    question: "What are the evidence levels?",
    answer: `**internal** (formerly L1) - Internal Audit
- Ed25519 digital signature
- Instant generation, no external dependencies
- Best for: Development, testing, team reviews

**compliance** (formerly L2) - Compliance Evidence
- All internal features + WORM storage + TSA timestamp
- Write-Once-Read-Many storage with Object Lock
- Best for: SOC2, GDPR, financial regulations, HIPAA

**legal** (formerly L3) - Legal Evidence
- All compliance features + blockchain anchoring
- Immutable, publicly verifiable on Ethereum
- Best for: Legal disputes, contracts, court evidence`
  },
  {
    category: "Technical",
    question: "What is Minimal Disclosure Proof?",
    answer: `Minimal Disclosure allows you to prove specific facts about an AI interaction without revealing everything.

For example, you can prove:
- "An AI generated output X at timestamp T"
- Without revealing the input prompt

This uses Merkle proofs - you only disclose the specific events you choose, while the Merkle root proves they're part of a valid certificate chain.`
  },

  // Integration
  {
    category: "Integration",
    question: "How do I integrate AI-Trace?",
    answer: `Just change your base URL - one line of code:

\`\`\`python
from openai import OpenAI

client = OpenAI(
    api_key="sk-...",
    base_url="https://gateway.aitrace.cc/api/v1"  # Add this line
)
\`\`\`

That's it! All your requests are now automatically traced and certifiable.`
  },
  {
    category: "Integration",
    question: "Which AI providers do you support?",
    answer: `Currently supported:
- **OpenAI** - Full support (GPT-4, GPT-3.5, etc.)
- **Anthropic Claude** - Full support (Claude 3, Claude 2, etc.)
- **Ollama** - Local model support (Llama, Mistral, Qwen, etc.)

More providers coming soon. Request support for your provider on GitHub.`
  },
  {
    category: "Integration",
    question: "Can I use my own proxy to avoid IP bans?",
    answer: `Yes! Use the X-Upstream-Base-URL header:

\`\`\`
X-Upstream-Base-URL: https://your-proxy.com/v1
\`\`\`

This is especially useful if:
- You're in a region where OpenAI/Claude access is restricted
- You want to use a company VPN or proxy
- You need to route through specific network paths`
  },

  // Compliance
  {
    category: "Compliance",
    question: "Is AI-Trace compliant with GDPR?",
    answer: `Yes. AI-Trace is designed with privacy-by-design principles:

- We don't store personal data or PII
- Only cryptographic hashes are retained
- Self-hosted option for complete data sovereignty
- Data minimization by default

For GDPR-sensitive deployments, we recommend Self-Hosted mode.`
  },
  {
    category: "Compliance",
    question: "Can AI-Trace help with AI regulations (EU AI Act)?",
    answer: `Yes. AI-Trace provides the audit trail infrastructure needed for:

- **Transparency**: Document all AI decision inputs/outputs
- **Traceability**: Complete chain from input to output
- **Accountability**: Tamper-proof certificates with timestamps
- **Human Oversight**: Enable review of AI decisions

We're actively tracking regulatory requirements and updating our platform accordingly.`
  }
]

export default function FAQ() {
  const [openIndex, setOpenIndex] = useState<number | null>(null)
  const [activeCategory, setActiveCategory] = useState<string>("all")

  const categories = ["all", ...Array.from(new Set(faqItems.map(item => item.category)))]
  const filteredItems = activeCategory === "all"
    ? faqItems
    : faqItems.filter(item => item.category === activeCategory)

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
            <div className="hidden md:flex items-center space-x-8">
              <Link href="/#features" className="text-gray-300 hover:text-white transition">Features</Link>
              <Link href="/#deployment" className="text-gray-300 hover:text-white transition">Deployment</Link>
              <Link href="/docs" className="text-gray-300 hover:text-white transition">Docs</Link>
              <Link href="/faq" className="text-white font-medium">FAQ</Link>
              <Link href="https://github.com/ai-trace/ai-trace" className="text-gray-300 hover:text-white transition">GitHub</Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Header */}
      <section className="pt-32 pb-12 px-4">
        <div className="max-w-4xl mx-auto text-center">
          <h1 className="text-4xl md:text-5xl font-bold text-white mb-4">
            Frequently Asked Questions
          </h1>
          <p className="text-xl text-gray-400">
            Everything you need to know about AI-Trace security, privacy, and integration.
          </p>
        </div>
      </section>

      {/* Category Filter */}
      <section className="px-4 pb-8">
        <div className="max-w-4xl mx-auto">
          <div className="flex flex-wrap gap-2 justify-center">
            {categories.map((category) => (
              <button
                key={category}
                onClick={() => setActiveCategory(category)}
                className={`px-4 py-2 rounded-full text-sm font-medium transition ${
                  activeCategory === category
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-800 text-gray-300 hover:bg-gray-700'
                }`}
              >
                {category === "all" ? "All Questions" : category}
              </button>
            ))}
          </div>
        </div>
      </section>

      {/* FAQ Items */}
      <section className="px-4 pb-20">
        <div className="max-w-4xl mx-auto space-y-4">
          {filteredItems.map((item, index) => (
            <div
              key={index}
              className="bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden"
            >
              <button
                onClick={() => setOpenIndex(openIndex === index ? null : index)}
                className="w-full px-6 py-5 flex items-center justify-between text-left"
              >
                <div>
                  <span className="text-xs text-blue-400 font-medium">{item.category}</span>
                  <h3 className="text-lg font-medium text-white mt-1">{item.question}</h3>
                </div>
                <svg
                  className={`w-5 h-5 text-gray-400 transition-transform ${
                    openIndex === index ? 'rotate-180' : ''
                  }`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>
              {openIndex === index && (
                <div className="px-6 pb-5">
                  <div className="prose prose-invert prose-sm max-w-none text-gray-300">
                    <pre className="whitespace-pre-wrap font-sans bg-transparent p-0 m-0">
                      {item.answer}
                    </pre>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </section>

      {/* CTA Section */}
      <section className="px-4 pb-20">
        <div className="max-w-4xl mx-auto text-center bg-gray-800 border border-gray-700 rounded-xl p-8">
          <h2 className="text-2xl font-bold text-white mb-4">
            Still have questions?
          </h2>
          <p className="text-gray-400 mb-6">
            We&apos;re here to help. Reach out to our team or check out our documentation.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/docs" className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg transition">
              Read the Docs
            </Link>
            <Link href="/contact" className="bg-gray-700 hover:bg-gray-600 text-white px-6 py-3 rounded-lg transition">
              Contact Us
            </Link>
            <Link href="https://github.com/ai-trace/ai-trace/issues" className="bg-gray-700 hover:bg-gray-600 text-white px-6 py-3 rounded-lg transition">
              GitHub Issues
            </Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-8 px-4">
        <div className="max-w-7xl mx-auto text-center text-gray-500 text-sm">
          <p>© 2025 AI-Trace. Open source under Apache 2.0.</p>
          <div className="mt-4 space-x-6">
            <Link href="/" className="hover:text-white transition">Home</Link>
            <Link href="/docs" className="hover:text-white transition">Docs</Link>
            <Link href="https://github.com/ai-trace/ai-trace" className="hover:text-white transition">GitHub</Link>
            <Link href="/contact" className="hover:text-white transition">Contact</Link>
          </div>
        </div>
      </footer>
    </div>
  )
}
