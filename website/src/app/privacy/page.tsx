import Link from 'next/link'

export default function PrivacyPage() {
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
              <Link href="/docs" className="text-gray-300 hover:text-white transition">Docs</Link>
              <Link href="https://github.com/ai-trace/ai-trace" className="text-gray-300 hover:text-white transition">GitHub</Link>
            </div>
          </div>
        </div>
      </nav>

      <div className="pt-24 pb-20 px-4">
        <div className="max-w-3xl mx-auto">
          <h1 className="text-4xl font-bold text-white mb-4">Privacy Policy</h1>
          <p className="text-gray-400 mb-8">Last updated: January 2025</p>

          <div className="prose prose-invert max-w-none">
            {/* Introduction */}
            <section className="mb-8">
              <h2 className="text-2xl font-semibold text-white mb-4">Introduction</h2>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 text-gray-300">
                <p>
                  AI-Trace (&quot;we&quot;, &quot;our&quot;, or &quot;us&quot;) is committed to protecting your privacy.
                  This Privacy Policy explains how we collect, use, and safeguard information when you use our
                  open-source AI audit platform.
                </p>
                <p className="mt-4">
                  <strong className="text-white">Key Principle:</strong> AI-Trace is designed with privacy-by-design.
                  We store cryptographic hashes, not your actual data. Your API keys and content are never stored
                  in our systems.
                </p>
              </div>
            </section>

            {/* What We Collect */}
            <section className="mb-8">
              <h2 className="text-2xl font-semibold text-white mb-4">What We Collect</h2>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                <h3 className="text-lg font-semibold text-green-400 mb-3">Data We Store (Hashed)</h3>
                <ul className="text-gray-300 space-y-2">
                  <li>• Event IDs and Trace IDs</li>
                  <li>• Event types (INPUT, MODEL, OUTPUT, TOOL_CALL)</li>
                  <li>• SHA-256 cryptographic hashes of prompts and responses</li>
                  <li>• Timestamps and sequence numbers</li>
                  <li>• Token usage statistics</li>
                  <li>• Merkle tree structures for verification</li>
                </ul>

                <h3 className="text-lg font-semibold text-red-400 mb-3 mt-6">Data We NEVER Store</h3>
                <ul className="text-gray-300 space-y-2">
                  <li>• Your API keys (OpenAI, Claude, etc.)</li>
                  <li>• Original prompt content</li>
                  <li>• AI response content</li>
                  <li>• System prompts</li>
                  <li>• Personal identifiable information (PII)</li>
                  <li>• Any recoverable plaintext</li>
                </ul>
              </div>
            </section>

            {/* API Key Handling */}
            <section className="mb-8">
              <h2 className="text-2xl font-semibold text-white mb-4">API Key Handling</h2>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 text-gray-300">
                <p>
                  When you use AI-Trace as a gateway to LLM providers (OpenAI, Claude, etc.), your API keys
                  are handled with the following principles:
                </p>
                <ul className="mt-4 space-y-2">
                  <li className="flex items-start">
                    <span className="text-green-400 mr-2">✓</span>
                    <span>API keys are passed through in-memory only</span>
                  </li>
                  <li className="flex items-start">
                    <span className="text-green-400 mr-2">✓</span>
                    <span>Keys are forwarded directly to upstream providers</span>
                  </li>
                  <li className="flex items-start">
                    <span className="text-green-400 mr-2">✓</span>
                    <span>Keys are never written to disk, database, or logs</span>
                  </li>
                  <li className="flex items-start">
                    <span className="text-green-400 mr-2">✓</span>
                    <span>Keys are discarded immediately after request completion</span>
                  </li>
                </ul>
              </div>
            </section>

            {/* Self-Hosted */}
            <section className="mb-8">
              <h2 className="text-2xl font-semibold text-white mb-4">Self-Hosted Deployments</h2>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 text-gray-300">
                <p>
                  When you self-host AI-Trace, you have complete control over all data. The software runs
                  entirely on your infrastructure, and no data is sent to us. This is the recommended
                  deployment mode for organizations with strict privacy requirements.
                </p>
              </div>
            </section>

            {/* Website Analytics */}
            <section className="mb-8">
              <h2 className="text-2xl font-semibold text-white mb-4">Website Analytics</h2>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 text-gray-300">
                <p>
                  Our website (aitrace.cc) may collect basic analytics to improve user experience:
                </p>
                <ul className="mt-4 space-y-2">
                  <li>• Page views and navigation patterns</li>
                  <li>• Browser type and operating system</li>
                  <li>• Referring website</li>
                  <li>• General geographic location (country level)</li>
                </ul>
                <p className="mt-4">
                  We use privacy-respecting analytics and do not track individual users across sites.
                </p>
              </div>
            </section>

            {/* Data Security */}
            <section className="mb-8">
              <h2 className="text-2xl font-semibold text-white mb-4">Data Security</h2>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 text-gray-300">
                <p>We implement industry-standard security measures:</p>
                <ul className="mt-4 space-y-2">
                  <li>• TLS encryption for all data in transit</li>
                  <li>• Encryption at rest for stored data</li>
                  <li>• Regular security audits</li>
                  <li>• Access controls and authentication</li>
                  <li>• Open source code for community security review</li>
                </ul>
              </div>
            </section>

            {/* Your Rights */}
            <section className="mb-8">
              <h2 className="text-2xl font-semibold text-white mb-4">Your Rights</h2>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 text-gray-300">
                <p>You have the right to:</p>
                <ul className="mt-4 space-y-2">
                  <li>• Access the data we hold about you</li>
                  <li>• Request deletion of your data</li>
                  <li>• Export your certificates and audit trails</li>
                  <li>• Opt out of analytics tracking</li>
                </ul>
                <p className="mt-4">
                  For self-hosted deployments, you have full control over all data and can manage it
                  according to your own policies.
                </p>
              </div>
            </section>

            {/* Contact */}
            <section className="mb-8">
              <h2 className="text-2xl font-semibold text-white mb-4">Contact Us</h2>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 text-gray-300">
                <p>
                  If you have questions about this Privacy Policy or our data practices, please contact us:
                </p>
                <ul className="mt-4 space-y-2">
                  <li>• Email: <a href="mailto:privacy@aitrace.cc" className="text-blue-400 hover:underline">privacy@aitrace.cc</a></li>
                  <li>• GitHub: <a href="https://github.com/ai-trace/ai-trace/issues" className="text-blue-400 hover:underline">Open an issue</a></li>
                </ul>
              </div>
            </section>

            {/* Changes */}
            <section>
              <h2 className="text-2xl font-semibold text-white mb-4">Changes to This Policy</h2>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6 text-gray-300">
                <p>
                  We may update this Privacy Policy from time to time. We will notify users of any material
                  changes by posting the new Privacy Policy on this page and updating the &quot;Last updated&quot; date.
                </p>
              </div>
            </section>
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-8 px-4">
        <div className="max-w-4xl mx-auto text-center text-gray-500 text-sm">
          © 2025 AI-Trace. Open source under Apache 2.0.
        </div>
      </footer>
    </div>
  )
}
