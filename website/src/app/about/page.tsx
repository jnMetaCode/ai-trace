import Link from 'next/link'

export default function AboutPage() {
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
              <Link href="/faq" className="text-gray-300 hover:text-white transition">FAQ</Link>
              <Link href="https://github.com/ai-trace/ai-trace" className="text-gray-300 hover:text-white transition">GitHub</Link>
            </div>
          </div>
        </div>
      </nav>

      <div className="pt-24 pb-20 px-4">
        <div className="max-w-4xl mx-auto">
          <h1 className="text-4xl font-bold text-white mb-8 text-center">About AI-Trace</h1>

          {/* Mission */}
          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">Our Mission</h2>
            <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
              <p className="text-gray-300 leading-relaxed">
                AI-Trace was created to solve a critical challenge in the age of AI: <strong className="text-white">accountability</strong>.
                As AI systems make more decisions that affect people&apos;s lives, there&apos;s a growing need for transparent,
                verifiable audit trails that can withstand regulatory scrutiny and legal challenges.
              </p>
              <p className="text-gray-300 leading-relaxed mt-4">
                We believe that AI governance shouldn&apos;t be an afterthought. By providing tamper-proof evidence collection
                from day one, organizations can build trust with users, regulators, and stakeholders.
              </p>
            </div>
          </section>

          {/* Why Open Source */}
          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">Why Open Source?</h2>
            <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
              <p className="text-gray-300 leading-relaxed">
                Trust in AI audit systems requires transparency. You shouldn&apos;t have to trust our word that we handle
                your data correctly - you should be able to verify it yourself. That&apos;s why AI-Trace is 100% open source
                under the Apache 2.0 license.
              </p>
              <ul className="mt-4 space-y-2 text-gray-300">
                <li className="flex items-start">
                  <span className="text-green-400 mr-2">✓</span>
                  <span>Audit every line of code on GitHub</span>
                </li>
                <li className="flex items-start">
                  <span className="text-green-400 mr-2">✓</span>
                  <span>Self-host for complete data sovereignty</span>
                </li>
                <li className="flex items-start">
                  <span className="text-green-400 mr-2">✓</span>
                  <span>Community-driven development and security reviews</span>
                </li>
                <li className="flex items-start">
                  <span className="text-green-400 mr-2">✓</span>
                  <span>No vendor lock-in - your data, your infrastructure</span>
                </li>
              </ul>
            </div>
          </section>

          {/* Technology */}
          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">Technology</h2>
            <div className="grid md:grid-cols-2 gap-6">
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                <h3 className="text-lg font-semibold text-white mb-2">Merkle Trees</h3>
                <p className="text-gray-400 text-sm">
                  Every AI interaction is hashed and linked into a Merkle tree, enabling efficient
                  verification and tamper detection.
                </p>
              </div>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                <h3 className="text-lg font-semibold text-white mb-2">Ed25519 Signatures</h3>
                <p className="text-gray-400 text-sm">
                  All certificates are digitally signed using Ed25519, enabling independent
                  verification without trusting any server.
                </p>
              </div>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                <h3 className="text-lg font-semibold text-white mb-2">WORM Storage</h3>
                <p className="text-gray-400 text-sm">
                  Write-Once-Read-Many storage ensures evidence cannot be modified or deleted
                  after creation.
                </p>
              </div>
              <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
                <h3 className="text-lg font-semibold text-white mb-2">Blockchain Anchoring</h3>
                <p className="text-gray-400 text-sm">
                  Optional anchoring to public blockchains provides the highest level of
                  immutability and public verifiability.
                </p>
              </div>
            </div>
          </section>

          {/* Team */}
          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">The Team</h2>
            <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
              <p className="text-gray-300 leading-relaxed">
                AI-Trace is built by a team passionate about AI safety, cryptography, and open source software.
                We come from backgrounds in enterprise software, blockchain, and regulatory compliance.
              </p>
              <p className="text-gray-300 leading-relaxed mt-4">
                We&apos;re always looking for contributors! Whether you&apos;re interested in code, documentation,
                or community building, check out our
                <Link href="https://github.com/ai-trace/ai-trace" className="text-blue-400 hover:underline ml-1">
                  GitHub repository
                </Link>.
              </p>
            </div>
          </section>

          {/* CTA */}
          <section className="text-center">
            <div className="bg-gradient-to-r from-blue-600 to-purple-600 rounded-xl p-8">
              <h2 className="text-2xl font-bold text-white mb-4">Ready to Get Started?</h2>
              <p className="text-blue-100 mb-6">
                Deploy AI-Trace in minutes and start building auditable AI applications.
              </p>
              <div className="flex flex-col sm:flex-row gap-4 justify-center">
                <Link href="/docs" className="bg-white text-blue-600 hover:bg-gray-100 px-6 py-3 rounded-lg font-medium transition">
                  Read the Docs
                </Link>
                <Link href="/contact" className="bg-transparent border-2 border-white text-white hover:bg-white/10 px-6 py-3 rounded-lg font-medium transition">
                  Contact Us
                </Link>
              </div>
            </div>
          </section>
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
