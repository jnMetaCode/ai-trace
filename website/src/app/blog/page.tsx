import Link from 'next/link'

export default function BlogPage() {
  const posts = [
    {
      title: "Introducing AI-Trace: Tamper-Proof Audit for AI Decisions",
      date: "January 2025",
      excerpt: "We're excited to announce the open-source release of AI-Trace, an enterprise-grade audit system for AI applications.",
      slug: "#",
      coming: false
    },
    {
      title: "Understanding Evidence Levels: L1, L2, and L3",
      date: "Coming Soon",
      excerpt: "A deep dive into the three evidence levels in AI-Trace and when to use each one.",
      slug: "#",
      coming: true
    },
    {
      title: "How Merkle Trees Enable Tamper-Proof AI Audit",
      date: "Coming Soon",
      excerpt: "Learn how AI-Trace uses Merkle trees to create cryptographically verifiable audit trails.",
      slug: "#",
      coming: true
    },
    {
      title: "Minimal Disclosure Proofs: Proving Without Revealing",
      date: "Coming Soon",
      excerpt: "How to prove specific facts about AI interactions without revealing sensitive information.",
      slug: "#",
      coming: true
    }
  ]

  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-900 to-gray-800">
      {/* Navigation */}
      <nav className="fixed top-0 w-full bg-gray-900/80 backdrop-blur-sm border-b border-gray-800 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16 items-center">
            <Link href="/" className="flex items-center space-x-2">
              <img src="/logo.svg" alt="AI-Trace" className="w-8 h-8" />
              <span className="text-white font-semibold text-xl">AI-Trace</span>
            </Link>
            <div className="flex items-center space-x-8">
              <Link href="/docs" className="text-gray-300 hover:text-white transition">Docs</Link>
              <Link href="/blog" className="text-white font-medium">Blog</Link>
              <Link href="https://github.com/ai-trace/ai-trace" className="text-gray-300 hover:text-white transition">GitHub</Link>
            </div>
          </div>
        </div>
      </nav>

      <div className="pt-24 pb-20 px-4">
        <div className="max-w-4xl mx-auto">
          <h1 className="text-4xl font-bold text-white mb-4">Blog</h1>
          <p className="text-gray-400 mb-12">
            News, updates, and deep dives into AI audit and compliance.
          </p>

          {/* Posts */}
          <div className="space-y-8">
            {posts.map((post, index) => (
              <article
                key={index}
                className={`bg-gray-800 rounded-xl border border-gray-700 p-6 ${
                  post.coming ? 'opacity-60' : 'hover:border-blue-500/50'
                } transition`}
              >
                <div className="flex items-center gap-4 mb-3">
                  <span className={`text-sm ${post.coming ? 'text-gray-500' : 'text-blue-400'}`}>
                    {post.date}
                  </span>
                  {post.coming && (
                    <span className="bg-gray-700 text-gray-400 text-xs px-2 py-1 rounded">
                      Coming Soon
                    </span>
                  )}
                </div>
                <h2 className="text-xl font-semibold text-white mb-2">
                  {post.coming ? (
                    post.title
                  ) : (
                    <Link href={post.slug} className="hover:text-blue-400 transition">
                      {post.title}
                    </Link>
                  )}
                </h2>
                <p className="text-gray-400">
                  {post.excerpt}
                </p>
                {!post.coming && (
                  <Link
                    href={post.slug}
                    className="inline-flex items-center text-blue-400 hover:text-blue-300 mt-4 transition"
                  >
                    Read more
                    <svg className="w-4 h-4 ml-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                    </svg>
                  </Link>
                )}
              </article>
            ))}
          </div>

          {/* Newsletter */}
          <div className="mt-16 bg-gray-800 rounded-xl border border-gray-700 p-8 text-center">
            <h2 className="text-2xl font-bold text-white mb-2">Stay Updated</h2>
            <p className="text-gray-400 mb-6">
              Subscribe to get notified about new posts and AI-Trace updates.
            </p>
            <div className="flex flex-col sm:flex-row gap-4 justify-center max-w-md mx-auto">
              <input
                type="email"
                placeholder="your@email.com"
                className="flex-1 bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 text-white focus:border-blue-500 focus:outline-none"
              />
              <button className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg font-medium transition">
                Subscribe
              </button>
            </div>
            <p className="text-gray-500 text-sm mt-4">
              No spam, unsubscribe anytime.
            </p>
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
