'use client'

import Link from 'next/link'
import { useState, useEffect, useRef } from 'react'

/* ------------------------------------------------------------------ */
/*  Fade-in-on-scroll hook                                            */
/* ------------------------------------------------------------------ */
function useFadeIn() {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const obs = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          el.classList.add('opacity-100', 'translate-y-0')
          el.classList.remove('opacity-0', 'translate-y-5')
          obs.unobserve(el)
        }
      },
      { threshold: 0.1 }
    )
    obs.observe(el)
    return () => obs.disconnect()
  }, [])
  return ref
}

function Section({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  const ref = useFadeIn()
  return (
    <div ref={ref} className={`opacity-0 translate-y-5 transition-all duration-700 ease-out ${className}`}>
      {children}
    </div>
  )
}

/* ------------------------------------------------------------------ */
/*  Shared SVG icons                                                  */
/* ------------------------------------------------------------------ */
const CheckIcon = () => (
  <svg className="w-5 h-5 mr-1.5 text-green-400 shrink-0" fill="currentColor" viewBox="0 0 20 20">
    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
  </svg>
)

const GitHubIcon = () => (
  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
    <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
  </svg>
)

const ArrowIcon = () => (
  <svg className="w-5 h-5 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
  </svg>
)

/* ------------------------------------------------------------------ */
/*  Page                                                              */
/* ------------------------------------------------------------------ */
export default function Home() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [scrolled, setScrolled] = useState(false)

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 20)
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      {/* ============================================================ */}
      {/*  NAV                                                         */}
      {/* ============================================================ */}
      <nav className={`fixed top-0 w-full z-50 transition-all duration-300 ${scrolled ? 'bg-gray-900/80 backdrop-blur-md border-b border-gray-800' : 'bg-transparent'}`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16 items-center">
            <div className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-blue-500 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-sm">AT</span>
              </div>
              <span className="text-white font-semibold text-xl">AI-Trace</span>
            </div>
            <div className="hidden md:flex items-center space-x-8">
              <Link href="#features" className="text-gray-300 hover:text-white transition text-sm">Features</Link>
              <Link href="/docs" className="text-gray-300 hover:text-white transition text-sm">Docs</Link>
              <Link href="#pricing" className="text-gray-300 hover:text-white transition text-sm">Pricing</Link>
              <Link href="https://github.com/ai-trace/ai-trace" className="text-gray-300 hover:text-white transition text-sm flex items-center gap-1.5">
                <GitHubIcon /> GitHub
              </Link>
              <Link href="/get-started" className="bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded-lg transition-all duration-150 hover:scale-[1.02] text-sm font-medium flex items-center">
                Get Started <ArrowIcon />
              </Link>
            </div>
            <button
              className="md:hidden text-gray-300 hover:text-white"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              aria-label="Toggle menu"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                {mobileMenuOpen ? (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                ) : (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                )}
              </svg>
            </button>
          </div>
        </div>
        {mobileMenuOpen && (
          <div className="md:hidden bg-gray-900/95 backdrop-blur-md border-t border-gray-800">
            <div className="px-4 py-4 space-y-3">
              <Link href="#features" className="block text-gray-300 hover:text-white transition" onClick={() => setMobileMenuOpen(false)}>Features</Link>
              <Link href="/docs" className="block text-gray-300 hover:text-white transition" onClick={() => setMobileMenuOpen(false)}>Docs</Link>
              <Link href="#pricing" className="block text-gray-300 hover:text-white transition" onClick={() => setMobileMenuOpen(false)}>Pricing</Link>
              <Link href="https://github.com/ai-trace/ai-trace" className="block text-gray-300 hover:text-white transition">GitHub</Link>
              <Link href="/get-started" className="block bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition text-center" onClick={() => setMobileMenuOpen(false)}>
                Get Started
              </Link>
            </div>
          </div>
        )}
      </nav>

      {/* ============================================================ */}
      {/*  HERO                                                        */}
      {/* ============================================================ */}
      <section className="min-h-screen flex flex-col items-center justify-center px-4 pt-16 pb-12">
        <div className="max-w-5xl mx-auto text-center">
          {/* Badge */}
          <div className="inline-flex items-center px-4 py-1.5 rounded-full bg-blue-500/10 border border-blue-500/20 text-blue-400 text-sm mb-8">
            <span className="w-2 h-2 bg-green-400 rounded-full mr-2 animate-pulse"></span>
            Open Source &nbsp;&middot;&nbsp; EU AI Act Ready
          </div>

          {/* Headline */}
          <h1 className="text-5xl md:text-6xl lg:text-7xl font-bold text-white mb-6 leading-tight tracking-tight">
            One Line to Make<br />
            <span className="bg-gradient-to-r from-blue-400 to-indigo-400 bg-clip-text text-transparent">
              Every AI Decision Auditable
            </span>
          </h1>

          {/* Subheadline */}
          <p className="text-lg md:text-xl text-gray-400 mb-10 max-w-3xl mx-auto leading-relaxed">
            AI-Trace is an open-source proxy that captures, hashes, and certifies
            every LLM interaction — giving you tamper-proof evidence for compliance,
            debugging, and trust.
          </p>

          {/* CTA Buttons */}
          <div className="flex flex-col sm:flex-row gap-4 justify-center mb-10">
            <Link href="/get-started" className="bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded-lg text-lg font-medium transition-all duration-150 hover:scale-[1.02] flex items-center justify-center">
              Quick Start <ArrowIcon />
            </Link>
            <Link href="https://github.com/ai-trace/ai-trace" className="border border-gray-600 hover:border-gray-500 bg-gray-800/50 hover:bg-gray-800 text-white px-8 py-4 rounded-lg text-lg font-medium transition-all duration-150 hover:scale-[1.02] flex items-center justify-center gap-2">
              <GitHubIcon /> View on GitHub
            </Link>
          </div>

          {/* Trust badges */}
          <div className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-gray-400 text-sm mb-16">
            {['OpenAI Compatible', 'Self-Hosted', 'Privacy First', '< 5 min setup'].map((t) => (
              <span key={t} className="flex items-center">
                <CheckIcon /> {t}
              </span>
            ))}
          </div>

          {/* Terminal widget */}
          <div className="max-w-3xl mx-auto">
            <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden shadow-2xl shadow-blue-500/5">
              <div className="flex items-center gap-2 px-4 py-3 bg-gray-900 border-b border-gray-800">
                <div className="w-3 h-3 rounded-full bg-red-500/80"></div>
                <div className="w-3 h-3 rounded-full bg-yellow-500/80"></div>
                <div className="w-3 h-3 rounded-full bg-green-500/80"></div>
                <span className="ml-3 text-gray-500 text-xs font-mono">main.py</span>
              </div>
              <pre className="p-6 text-sm md:text-base overflow-x-auto text-left">
                <code className="font-mono">
                  <span className="text-blue-400">from</span> <span className="text-gray-300">openai</span> <span className="text-blue-400">import</span> <span className="text-gray-300">OpenAI</span>{'\n'}
                  {'\n'}
                  <span className="text-gray-300">client = OpenAI(</span>{'\n'}
                  <span className="text-gray-300">    api_key=</span><span className="text-amber-300">&quot;sk-...&quot;</span><span className="text-gray-300">,</span>{'\n'}
                  <span className="text-green-400 font-semibold">    base_url=&quot;https://your-domain/api/v1&quot;  # &larr; only change</span>{'\n'}
                  <span className="text-gray-300">)</span>{'\n'}
                  {'\n'}
                  <span className="text-gray-500"># That&apos;s it. Every request is now traced &amp; certified.</span>{'\n'}
                  <span className="text-gray-300">response = client.chat.completions.create(</span>{'\n'}
                  <span className="text-gray-300">    model=</span><span className="text-amber-300">&quot;gpt-4&quot;</span><span className="text-gray-300">,</span>{'\n'}
                  <span className="text-gray-300">    messages=[&#123;</span><span className="text-amber-300">&quot;role&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;user&quot;</span><span className="text-gray-300">, </span><span className="text-amber-300">&quot;content&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;Summarize this contract&quot;</span><span className="text-gray-300">&#125;]</span>{'\n'}
                  <span className="text-gray-300">)</span>
                </code>
              </pre>
            </div>
          </div>
        </div>
      </section>

      {/* ============================================================ */}
      {/*  PROBLEM — Why This Matters                                  */}
      {/* ============================================================ */}
      <section className="py-24 px-4">
        <Section>
          <div className="max-w-7xl mx-auto">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">Why This Matters</h2>
            </div>

            <div className="grid md:grid-cols-3 gap-8">
              {[
                {
                  icon: '🔲',
                  title: 'AI Decisions Are Black Boxes',
                  desc: 'Your LLM made a recommendation that cost $2M. Can you prove what prompt was sent, what context was retrieved, and what the model returned? Today, you can\'t.',
                },
                {
                  icon: '⚠️',
                  title: 'Logs Can Be Tampered With',
                  desc: 'Traditional logging has no integrity guarantees. Anyone with database access can alter records after the fact. Regulators know this.',
                },
                {
                  icon: '⚖️',
                  title: 'Compliance Is Coming — Fast',
                  desc: 'The EU AI Act requires "traceability" and "record-keeping" for high-risk AI systems. Fines up to 35M EUR or 7% of global turnover. Enforcement begins 2025.',
                },
              ].map((card, i) => (
                <div
                  key={i}
                  className="bg-gray-800/50 border border-gray-700 rounded-xl p-8 hover:border-blue-500/50 transition-colors duration-200"
                >
                  <div className="w-12 h-12 bg-blue-500/10 border border-blue-500/20 rounded-xl flex items-center justify-center text-2xl mb-5">
                    {card.icon}
                  </div>
                  <h3 className="text-xl font-semibold text-white mb-3">{card.title}</h3>
                  <p className="text-gray-400 leading-relaxed">{card.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </Section>
      </section>

      {/* ============================================================ */}
      {/*  HOW IT WORKS                                                */}
      {/* ============================================================ */}
      <section className="py-24 px-4 bg-gray-900/50">
        <Section>
          <div className="max-w-7xl mx-auto">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">How It Works</h2>
              <p className="text-lg text-gray-400">Three steps. Zero code rewrite.</p>
            </div>

            {/* Steps */}
            <div className="grid md:grid-cols-3 gap-8 mb-20">
              {[
                {
                  step: '1',
                  label: 'Proxy',
                  desc: 'Point your OpenAI SDK at AI-Trace. It forwards requests to the LLM provider and captures the full interaction.',
                },
                {
                  step: '2',
                  label: 'Capture',
                  desc: 'Every event (INPUT, MODEL_CONFIG, RETRIEVAL, OUTPUT) is hashed with SHA-256 and recorded with a sequence number and timestamp.',
                },
                {
                  step: '3',
                  label: 'Certify',
                  desc: 'Events are bound into a Merkle tree. The root is signed with Ed25519. Optionally anchored to a blockchain for legal-grade evidence.',
                },
              ].map((item, i) => (
                <div key={i} className="relative text-center">
                  <div className="w-14 h-14 bg-blue-600 rounded-full flex items-center justify-center text-white font-bold text-xl mx-auto mb-5 shadow-lg shadow-blue-500/20">
                    {item.step}
                  </div>
                  <h3 className="text-xl font-semibold text-white mb-2">{item.label}</h3>
                  <p className="text-gray-400 leading-relaxed max-w-sm mx-auto">{item.desc}</p>
                  {/* Arrow connector on desktop */}
                  {i < 2 && (
                    <div className="hidden md:block absolute top-7 -right-4 text-gray-600 text-2xl">
                      &rarr;
                    </div>
                  )}
                </div>
              ))}
            </div>

            {/* Architecture diagram */}
            <div className="max-w-3xl mx-auto">
              <div className="flex flex-col md:flex-row items-center justify-center gap-4 md:gap-0">
                {/* Your App */}
                <div className="bg-gray-800 border border-gray-700 rounded-xl px-6 py-4 text-center min-w-[140px]">
                  <div className="text-sm text-gray-400 mb-1">Your App</div>
                  <div className="text-xs text-gray-500 font-mono">OpenAI SDK</div>
                </div>
                <div className="text-gray-500 text-2xl rotate-90 md:rotate-0 mx-2">&rarr;</div>
                {/* AI-Trace Proxy */}
                <div className="bg-blue-600/20 border-2 border-blue-500/50 rounded-xl px-6 py-4 text-center min-w-[160px] relative">
                  <div className="text-sm text-blue-400 font-semibold mb-1">AI-Trace Proxy</div>
                  <div className="text-xs text-gray-400 font-mono">Capture + Hash</div>
                  {/* Down arrow */}
                  <div className="absolute -bottom-8 left-1/2 -translate-x-1/2 text-gray-500 text-2xl">&darr;</div>
                </div>
                <div className="text-gray-500 text-2xl rotate-90 md:rotate-0 mx-2">&rarr;</div>
                {/* LLM Provider */}
                <div className="bg-gray-800 border border-gray-700 rounded-xl px-6 py-4 text-center min-w-[140px]">
                  <div className="text-sm text-gray-400 mb-1">OpenAI / Claude</div>
                  <div className="text-xs text-gray-500 font-mono">LLM Provider</div>
                </div>
              </div>
              {/* Evidence Store + Merkle */}
              <div className="flex flex-col items-center mt-12 gap-4">
                <div className="bg-gray-800 border border-gray-700 rounded-xl px-6 py-4 text-center min-w-[160px]">
                  <div className="text-sm text-gray-400 mb-1">Evidence Store</div>
                  <div className="text-xs text-gray-500 font-mono">PostgreSQL</div>
                </div>
                <div className="text-gray-500 text-2xl">&darr;</div>
                <div className="bg-gray-800 border border-indigo-500/30 rounded-xl px-6 py-4 text-center min-w-[160px]">
                  <div className="text-sm text-indigo-400 mb-1">Merkle + Sign</div>
                  <div className="text-xs text-gray-500 font-mono">Ed25519 / Blockchain</div>
                </div>
              </div>
            </div>
          </div>
        </Section>
      </section>

      {/* ============================================================ */}
      {/*  FEATURES GRID                                               */}
      {/* ============================================================ */}
      <section id="features" className="py-24 px-4">
        <Section>
          <div className="max-w-7xl mx-auto">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">
                Everything You Need for AI Accountability
              </h2>
            </div>

            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {[
                {
                  icon: '🔗',
                  color: 'blue',
                  title: 'Full-Chain Capture',
                  desc: 'Records every step of the AI pipeline: user input, system prompt, RAG retrieval context, model configuration, and final output. Nothing is missed.',
                },
                {
                  icon: '🛡️',
                  color: 'indigo',
                  title: 'Tamper-Proof Certificates',
                  desc: 'Events are bound using a Merkle tree. Change one byte of one event, and the root hash breaks. Certificates are signed with Ed25519.',
                },
                {
                  icon: '📊',
                  color: 'purple',
                  title: 'Three Evidence Levels',
                  desc: 'Choose your assurance: Internal (hash-only, free), Compliance (signed certificate), or Legal (blockchain-anchored, court-admissible).',
                },
                {
                  icon: '🔐',
                  color: 'green',
                  title: 'Zero-Knowledge Proofs',
                  desc: 'Prove properties about an AI interaction ("model was GPT-4", "response was under 500 tokens") without revealing the actual content. Privacy-preserving compliance.',
                },
                {
                  icon: '🔌',
                  color: 'amber',
                  title: 'OpenAI-Compatible API',
                  desc: 'Drop-in replacement. Change base_url and you\'re done. Works with any SDK or tool that speaks the OpenAI API. Python, Node, curl — all supported.',
                },
                {
                  icon: '✅',
                  color: 'teal',
                  title: 'Open Source Verifier',
                  desc: 'Verify any AI-Trace certificate offline with our open-source CLI tool. No need to trust us — verify yourself. npx aitrace-verify cert.json',
                },
              ].map((feature, i) => (
                <div
                  key={i}
                  className="bg-gray-800/50 border border-gray-700 rounded-xl p-6 hover:border-blue-500/50 transition-colors duration-200 group"
                >
                  <div className="text-3xl mb-4">{feature.icon}</div>
                  <h3 className="text-lg font-semibold text-white mb-2">{feature.title}</h3>
                  <p className="text-gray-400 text-sm leading-relaxed">{feature.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </Section>
      </section>

      {/* ============================================================ */}
      {/*  CODE EXAMPLE — Before / After                               */}
      {/* ============================================================ */}
      <section className="py-24 px-4 bg-gray-900/50">
        <Section>
          <div className="max-w-7xl mx-auto">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">See the Difference</h2>
            </div>

            <div className="grid lg:grid-cols-2 gap-6 mb-8">
              {/* Before */}
              <div>
                <div className="text-sm font-medium text-gray-400 mb-3 flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-red-400"></span>
                  Before (Standard OpenAI)
                </div>
                <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
                  <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-800">
                    <div className="w-3 h-3 rounded-full bg-red-500/80"></div>
                    <div className="w-3 h-3 rounded-full bg-yellow-500/80"></div>
                    <div className="w-3 h-3 rounded-full bg-green-500/80"></div>
                  </div>
                  <pre className="p-5 text-xs md:text-sm overflow-x-auto">
                    <code className="font-mono">
<span className="text-blue-400">from</span> <span className="text-gray-300">openai</span> <span className="text-blue-400">import</span> <span className="text-gray-300">OpenAI</span>{'\n'}
{'\n'}
<span className="text-gray-300">client = OpenAI(api_key=</span><span className="text-amber-300">&quot;sk-...&quot;</span><span className="text-gray-300">)</span>{'\n'}
{'\n'}
<span className="text-gray-300">response = client.chat.completions.create(</span>{'\n'}
<span className="text-gray-300">    model=</span><span className="text-amber-300">&quot;gpt-4&quot;</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-gray-300">    messages=[</span>{'\n'}
<span className="text-gray-300">        &#123;</span><span className="text-amber-300">&quot;role&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;system&quot;</span><span className="text-gray-300">, </span><span className="text-amber-300">&quot;content&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;You are a loan officer.&quot;</span><span className="text-gray-300">&#125;,</span>{'\n'}
<span className="text-gray-300">        &#123;</span><span className="text-amber-300">&quot;role&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;user&quot;</span><span className="text-gray-300">, </span><span className="text-amber-300">&quot;content&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;Should I approve this loan?&quot;</span><span className="text-gray-300">&#125;</span>{'\n'}
<span className="text-gray-300">    ]</span>{'\n'}
<span className="text-gray-300">)</span>{'\n'}
{'\n'}
<span className="text-gray-300">print(response.choices[0].message.content)</span>{'\n'}
<span className="text-gray-500"># &quot;Based on the applicant&apos;s credit score...&quot;</span>{'\n'}
{'\n'}
<span className="text-red-400"># No record of what happened</span>{'\n'}
<span className="text-red-400"># No proof it wasn&apos;t modified</span>{'\n'}
<span className="text-red-400"># Not auditable</span>
                    </code>
                  </pre>
                </div>
              </div>

              {/* After */}
              <div>
                <div className="text-sm font-medium text-gray-400 mb-3 flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-green-400"></span>
                  After (With AI-Trace)
                </div>
                <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
                  <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-800">
                    <div className="w-3 h-3 rounded-full bg-red-500/80"></div>
                    <div className="w-3 h-3 rounded-full bg-yellow-500/80"></div>
                    <div className="w-3 h-3 rounded-full bg-green-500/80"></div>
                  </div>
                  <pre className="p-5 text-xs md:text-sm overflow-x-auto">
                    <code className="font-mono">
<span className="text-blue-400">from</span> <span className="text-gray-300">openai</span> <span className="text-blue-400">import</span> <span className="text-gray-300">OpenAI</span>{'\n'}
{'\n'}
<span className="text-gray-300">client = OpenAI(</span>{'\n'}
<span className="text-gray-300">    api_key=</span><span className="text-amber-300">&quot;sk-...&quot;</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-green-400 font-semibold">    base_url=&quot;https://aitrace.cc/api/v1&quot;  # &larr; add this</span>{'\n'}
<span className="text-gray-300">)</span>{'\n'}
{'\n'}
<span className="text-gray-300">response = client.chat.completions.create(</span>{'\n'}
<span className="text-gray-300">    model=</span><span className="text-amber-300">&quot;gpt-4&quot;</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-gray-300">    messages=[</span>{'\n'}
<span className="text-gray-300">        &#123;</span><span className="text-amber-300">&quot;role&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;system&quot;</span><span className="text-gray-300">, </span><span className="text-amber-300">&quot;content&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;You are a loan officer.&quot;</span><span className="text-gray-300">&#125;,</span>{'\n'}
<span className="text-gray-300">        &#123;</span><span className="text-amber-300">&quot;role&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;user&quot;</span><span className="text-gray-300">, </span><span className="text-amber-300">&quot;content&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;Should I approve this loan?&quot;</span><span className="text-gray-300">&#125;</span>{'\n'}
<span className="text-gray-300">    ]</span>{'\n'}
<span className="text-gray-300">)</span>{'\n'}
{'\n'}
<span className="text-gray-300">print(response.choices[0].message.content)</span>{'\n'}
<span className="text-gray-500"># Same response — but now with a certificate:</span>{'\n'}
{'\n'}
<span className="text-green-400"># trace_id: &quot;tr_8f3a...c421&quot;</span>{'\n'}
<span className="text-green-400"># merkle_root: &quot;a1b2c3...&quot;</span>{'\n'}
<span className="text-green-400"># signature: &quot;Ed25519:...&quot;</span>{'\n'}
<span className="text-green-400"># verify: npx aitrace-verify tr_8f3a...c421</span>
                    </code>
                  </pre>
                </div>
              </div>
            </div>

            {/* Certificate Preview */}
            <div className="max-w-3xl mx-auto">
              <div className="text-sm font-medium text-gray-400 mb-3 text-center">Certificate Output</div>
              <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
                <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-800">
                  <div className="w-3 h-3 rounded-full bg-red-500/80"></div>
                  <div className="w-3 h-3 rounded-full bg-yellow-500/80"></div>
                  <div className="w-3 h-3 rounded-full bg-green-500/80"></div>
                  <span className="ml-3 text-gray-500 text-xs font-mono">certificate.json</span>
                </div>
                <pre className="p-5 text-xs md:text-sm overflow-x-auto">
                  <code className="font-mono">
<span className="text-gray-300">&#123;</span>{'\n'}
<span className="text-blue-400">  &quot;certificate_id&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;cert_2024_8f3a1b&quot;</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-blue-400">  &quot;trace_id&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;tr_8f3a...c421&quot;</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-blue-400">  &quot;evidence_level&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;compliance&quot;</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-blue-400">  &quot;events&quot;</span><span className="text-gray-300">: </span><span className="text-purple-400">4</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-blue-400">  &quot;merkle_root&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;a1b2c3d4e5f6...&quot;</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-blue-400">  &quot;signature&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;Ed25519:7f8e9d...&quot;</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-blue-400">  &quot;issued_at&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;2024-12-15T10:30:00Z&quot;</span><span className="text-gray-300">,</span>{'\n'}
<span className="text-blue-400">  &quot;verify_url&quot;</span><span className="text-gray-300">: </span><span className="text-amber-300">&quot;https://aitrace.cc/verify/cert_2024_8f3a1b&quot;</span>{'\n'}
<span className="text-gray-300">&#125;</span>
                  </code>
                </pre>
              </div>
            </div>
          </div>
        </Section>
      </section>

      {/* ============================================================ */}
      {/*  COMPARISON TABLE                                            */}
      {/* ============================================================ */}
      <section className="py-24 px-4">
        <Section>
          <div className="max-w-7xl mx-auto">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">How AI-Trace Compares</h2>
            </div>

            <div className="overflow-x-auto rounded-xl border border-gray-800">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-800">
                    <th className="text-left text-gray-400 font-medium p-4">Feature</th>
                    <th className="text-center p-4 bg-blue-500/10 text-blue-400 font-semibold min-w-[120px]">AI-Trace</th>
                    <th className="text-center text-gray-400 font-medium p-4 min-w-[110px]">LangSmith</th>
                    <th className="text-center text-gray-400 font-medium p-4 min-w-[110px]">Arize AI</th>
                    <th className="text-center text-gray-400 font-medium p-4 min-w-[120px]">Weights &amp; Biases</th>
                  </tr>
                </thead>
                <tbody className="text-gray-300">
                  {[
                    { feature: 'Full prompt/response capture', ai: 'Yes', ls: 'Yes', ar: 'Partial', wb: 'Yes' },
                    { feature: 'Tamper-proof (Merkle tree)', ai: 'Yes', ls: 'No', ar: 'No', wb: 'No' },
                    { feature: 'Cryptographic signatures', ai: 'Ed25519', ls: 'No', ar: 'No', wb: 'No' },
                    { feature: 'Blockchain anchoring', ai: 'Optional', ls: 'No', ar: 'No', wb: 'No' },
                    { feature: 'Zero-knowledge proofs', ai: 'Yes', ls: 'No', ar: 'No', wb: 'No' },
                    { feature: 'OpenAI-compatible proxy', ai: 'Yes', ls: 'SDK req.', ar: 'SDK req.', wb: 'SDK req.' },
                    { feature: 'Self-hostable', ai: 'Yes', ls: 'No', ar: 'No', wb: 'No' },
                    { feature: 'Open source', ai: 'Apache 2.0', ls: 'Partial', ar: 'No', wb: 'No' },
                    { feature: 'EU AI Act traceability', ai: 'Designed for it', ls: 'No', ar: 'No', wb: 'No' },
                    { feature: 'Pricing starts at', ai: 'Free', ls: 'Free*', ar: 'Free*', wb: 'Free*' },
                  ].map((row, i) => (
                    <tr key={i} className="border-b border-gray-800/50 hover:bg-gray-800/30">
                      <td className="p-4 text-gray-300 font-medium">{row.feature}</td>
                      <td className="p-4 text-center bg-blue-500/5">
                        {row.ai === 'Yes' || row.ai === 'Ed25519' || row.ai === 'Optional' || row.ai === 'Apache 2.0' || row.ai === 'Free' || row.ai === 'Designed for it' ? (
                          <span className="text-green-400 font-medium">{row.ai}</span>
                        ) : (
                          <span>{row.ai}</span>
                        )}
                      </td>
                      <td className="p-4 text-center">
                        <span className={row.ls === 'Yes' ? 'text-gray-300' : 'text-gray-500'}>{row.ls}</span>
                      </td>
                      <td className="p-4 text-center">
                        <span className={row.ar === 'Partial' || row.ar === 'Yes' ? 'text-gray-300' : 'text-gray-500'}>{row.ar}</span>
                      </td>
                      <td className="p-4 text-center">
                        <span className={row.wb === 'Yes' ? 'text-gray-300' : 'text-gray-500'}>{row.wb}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="text-gray-500 text-xs mt-3 text-center">* Free tiers have limited usage</p>
          </div>
        </Section>
      </section>

      {/* ============================================================ */}
      {/*  PRICING                                                     */}
      {/* ============================================================ */}
      <section id="pricing" className="py-24 px-4 bg-gray-900/50">
        <Section>
          <div className="max-w-7xl mx-auto">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">Simple, Transparent Pricing</h2>
              <p className="text-lg text-gray-400">Start free. No credit card required.</p>
            </div>

            <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
              {/* Open Source */}
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-8 flex flex-col">
                <div className="text-blue-400 text-sm font-medium uppercase tracking-wide mb-3">Open Source</div>
                <div className="text-4xl font-bold text-white mb-1">Free</div>
                <p className="text-gray-500 text-sm mb-6">forever</p>
                <ul className="space-y-3 text-gray-300 text-sm mb-8 flex-1">
                  <li className="flex items-start"><CheckIcon /> Self-hosted</li>
                  <li className="flex items-start"><CheckIcon /> Unlimited events</li>
                  <li className="flex items-start"><CheckIcon /> Internal + Compliance levels</li>
                  <li className="flex items-start"><CheckIcon /> Community support</li>
                  <li className="flex items-start"><CheckIcon /> Docker deploy</li>
                </ul>
                <Link href="/get-started" className="block w-full text-center bg-blue-600 hover:bg-blue-500 text-white py-3 rounded-lg transition-all duration-150 hover:scale-[1.02] font-medium">
                  Get Started
                </Link>
              </div>

              {/* Developer */}
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-8 flex flex-col relative overflow-hidden">
                <div className="absolute top-4 right-4 bg-gray-700 text-gray-300 text-xs px-2.5 py-1 rounded-full font-medium">
                  Coming Soon
                </div>
                <div className="text-gray-400 text-sm font-medium uppercase tracking-wide mb-3">Developer</div>
                <div className="text-4xl font-bold text-white mb-1">Free</div>
                <p className="text-gray-500 text-sm mb-6">hosted</p>
                <ul className="space-y-3 text-gray-300 text-sm mb-8 flex-1">
                  <li className="flex items-start"><CheckIcon /> 1K traces/mo</li>
                  <li className="flex items-start"><CheckIcon /> 1 project</li>
                  <li className="flex items-start"><CheckIcon /> Compliance certificates</li>
                  <li className="flex items-start"><CheckIcon /> Community support</li>
                </ul>
                <Link href="/contact?interest=developer-waitlist" className="block w-full text-center border border-gray-600 hover:border-gray-500 text-gray-300 hover:text-white py-3 rounded-lg transition font-medium">
                  Join Waitlist
                </Link>
              </div>

              {/* Team — Most Popular */}
              <div className="bg-gray-800/50 border-2 border-blue-500/50 rounded-xl p-8 flex flex-col relative overflow-hidden">
                <div className="absolute top-4 right-4 bg-blue-500 text-white text-xs px-2.5 py-1 rounded-full font-medium">
                  Most Popular
                </div>
                <div className="absolute top-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-blue-500 to-transparent"></div>
                <div className="text-blue-400 text-sm font-medium uppercase tracking-wide mb-3">Team</div>
                <div className="text-4xl font-bold text-white mb-1">$49<span className="text-lg text-gray-400 font-normal">/mo</span></div>
                <p className="text-gray-500 text-sm mb-6">per team</p>
                <ul className="space-y-3 text-gray-300 text-sm mb-8 flex-1">
                  <li className="flex items-start"><CheckIcon /> 50K traces/mo</li>
                  <li className="flex items-start"><CheckIcon /> 5 projects</li>
                  <li className="flex items-start"><CheckIcon /> Legal-grade certificates</li>
                  <li className="flex items-start"><CheckIcon /> Slack support</li>
                  <li className="flex items-start"><CheckIcon /> Team dashboard</li>
                </ul>
                <Link href="/contact?interest=team-waitlist" className="block w-full text-center border border-blue-500 hover:bg-blue-500/10 text-blue-400 hover:text-blue-300 py-3 rounded-lg transition font-medium">
                  Join Waitlist
                </Link>
              </div>

              {/* Business */}
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-8 flex flex-col relative overflow-hidden">
                <div className="absolute top-4 right-4 bg-gray-700 text-gray-300 text-xs px-2.5 py-1 rounded-full font-medium">
                  Coming Soon
                </div>
                <div className="text-gray-400 text-sm font-medium uppercase tracking-wide mb-3">Business</div>
                <div className="text-4xl font-bold text-white mb-1">$199<span className="text-lg text-gray-400 font-normal">/mo</span></div>
                <p className="text-gray-500 text-sm mb-6">per team</p>
                <ul className="space-y-3 text-gray-300 text-sm mb-8 flex-1">
                  <li className="flex items-start"><CheckIcon /> 500K traces/mo</li>
                  <li className="flex items-start"><CheckIcon /> Unlimited projects</li>
                  <li className="flex items-start"><CheckIcon /> Blockchain anchoring</li>
                  <li className="flex items-start"><CheckIcon /> Priority support + SSO</li>
                  <li className="flex items-start"><CheckIcon /> Audit exports</li>
                </ul>
                <Link href="/contact?interest=business-waitlist" className="block w-full text-center border border-gray-600 hover:border-gray-500 text-gray-300 hover:text-white py-3 rounded-lg transition font-medium">
                  Join Waitlist
                </Link>
              </div>
            </div>
          </div>
        </Section>
      </section>

      {/* ============================================================ */}
      {/*  CTA BANNER                                                  */}
      {/* ============================================================ */}
      <section className="py-24 px-4">
        <Section>
          <div className="max-w-4xl mx-auto">
            <div className="bg-gradient-to-r from-blue-600 to-indigo-700 rounded-2xl p-12 md:p-16 text-center relative overflow-hidden">
              {/* Decorative elements */}
              <div className="absolute top-0 left-0 w-full h-full opacity-10">
                <div className="absolute -top-24 -right-24 w-64 h-64 rounded-full bg-white/20"></div>
                <div className="absolute -bottom-24 -left-24 w-64 h-64 rounded-full bg-white/10"></div>
              </div>
              <div className="relative z-10">
                <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">
                  Ready to Make Your AI Auditable?
                </h2>
                <p className="text-blue-100 text-lg mb-8 max-w-2xl mx-auto">
                  Deploy in under 5 minutes. Open source, self-hosted, free forever.
                </p>
                <div className="flex flex-col sm:flex-row gap-4 justify-center">
                  <Link href="/get-started" className="bg-white text-blue-600 hover:bg-gray-100 px-8 py-4 rounded-lg text-lg font-medium transition-all duration-150 hover:scale-[1.02]">
                    Get Started Free
                  </Link>
                  <Link href="/docs" className="border-2 border-white/80 text-white hover:bg-white/10 px-8 py-4 rounded-lg text-lg font-medium transition-all duration-150 hover:scale-[1.02]">
                    Read the Docs
                  </Link>
                </div>
              </div>
            </div>
          </div>
        </Section>
      </section>

      {/* ============================================================ */}
      {/*  FOOTER                                                      */}
      {/* ============================================================ */}
      <footer className="border-t border-gray-800 py-16 px-4">
        <div className="max-w-7xl mx-auto">
          <div className="grid md:grid-cols-4 gap-10">
            {/* Brand */}
            <div>
              <div className="flex items-center space-x-2 mb-4">
                <div className="w-8 h-8 bg-blue-500 rounded-lg flex items-center justify-center">
                  <span className="text-white font-bold text-sm">AT</span>
                </div>
                <span className="text-white font-semibold text-lg">AI-Trace</span>
              </div>
              <p className="text-gray-500 text-sm mb-6">Tamper-proof audit trails for AI.</p>
              {/* Social icons */}
              <div className="flex items-center gap-4">
                <Link href="https://github.com/ai-trace/ai-trace" className="text-gray-500 hover:text-white transition" aria-label="GitHub">
                  <GitHubIcon />
                </Link>
                <Link href="#" className="text-gray-500 hover:text-white transition" aria-label="Twitter / X">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
                  </svg>
                </Link>
              </div>
            </div>

            {/* Product */}
            <div>
              <h4 className="text-white font-semibold mb-4">Product</h4>
              <ul className="space-y-3 text-gray-400 text-sm">
                <li><Link href="#features" className="hover:text-white transition">Features</Link></li>
                <li><Link href="#pricing" className="hover:text-white transition">Pricing</Link></li>
                <li><Link href="/docs" className="hover:text-white transition">Docs</Link></li>
                <li><Link href="#" className="hover:text-white transition">Changelog</Link></li>
              </ul>
            </div>

            {/* Resources */}
            <div>
              <h4 className="text-white font-semibold mb-4">Resources</h4>
              <ul className="space-y-3 text-gray-400 text-sm">
                <li><Link href="https://github.com/ai-trace/ai-trace" className="hover:text-white transition">GitHub</Link></li>
                <li><Link href="/blog" className="hover:text-white transition">Blog</Link></li>
                <li><Link href="/docs" className="hover:text-white transition">API Reference</Link></li>
                <li><Link href="#" className="hover:text-white transition">Status</Link></li>
              </ul>
            </div>

            {/* Company */}
            <div>
              <h4 className="text-white font-semibold mb-4">Company</h4>
              <ul className="space-y-3 text-gray-400 text-sm">
                <li><Link href="/about" className="hover:text-white transition">About</Link></li>
                <li><Link href="/contact" className="hover:text-white transition">Contact</Link></li>
                <li><Link href="/privacy" className="hover:text-white transition">Privacy</Link></li>
                <li><Link href="#" className="hover:text-white transition">Terms</Link></li>
              </ul>
            </div>
          </div>

          <div className="border-t border-gray-800 mt-12 pt-8 flex flex-col md:flex-row justify-between items-center text-gray-500 text-sm gap-4">
            <span>&copy; 2025 AI-Trace. Open source under Apache 2.0.</span>
            <span>Made for the EU AI Act era.</span>
          </div>
        </div>
      </footer>
    </div>
  )
}
