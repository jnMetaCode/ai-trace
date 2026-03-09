'use client'

import Link from 'next/link'
import { useState } from 'react'

export default function ContactPage() {
  const [submitted, setSubmitted] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    company: '',
    interest: 'consulting',
    message: ''
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    // In production, this would send to an API
    console.log('Form submitted:', formData)
    setSubmitted(true)
  }

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
              <Link href="https://github.com/ai-trace/ai-trace" className="text-gray-300 hover:text-white transition">GitHub</Link>
            </div>
          </div>
        </div>
      </nav>

      <div className="pt-24 pb-20 px-4">
        <div className="max-w-2xl mx-auto">
          <h1 className="text-4xl font-bold text-white mb-4 text-center">Get in Touch</h1>
          <p className="text-gray-400 text-center mb-12">
            Interested in consulting, support, or the upcoming cloud version? Let us know.
          </p>

          {submitted ? (
            <div className="bg-green-900/30 border border-green-500/50 rounded-xl p-8 text-center">
              <div className="text-4xl mb-4">✓</div>
              <h2 className="text-2xl font-semibold text-white mb-2">Thank you!</h2>
              <p className="text-gray-400 mb-6">
                We&apos;ve received your message and will get back to you soon.
              </p>
              <Link href="/" className="text-blue-400 hover:underline">
                Back to Home
              </Link>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="bg-gray-800 rounded-xl border border-gray-700 p-8">
              <div className="grid md:grid-cols-2 gap-6 mb-6">
                <div>
                  <label className="block text-gray-300 text-sm mb-2">Name *</label>
                  <input
                    type="text"
                    required
                    value={formData.name}
                    onChange={(e) => setFormData({...formData, name: e.target.value})}
                    className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 text-white focus:border-blue-500 focus:outline-none"
                    placeholder="Your name"
                  />
                </div>
                <div>
                  <label className="block text-gray-300 text-sm mb-2">Email *</label>
                  <input
                    type="email"
                    required
                    value={formData.email}
                    onChange={(e) => setFormData({...formData, email: e.target.value})}
                    className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 text-white focus:border-blue-500 focus:outline-none"
                    placeholder="you@company.com"
                  />
                </div>
              </div>

              <div className="mb-6">
                <label className="block text-gray-300 text-sm mb-2">Company</label>
                <input
                  type="text"
                  value={formData.company}
                  onChange={(e) => setFormData({...formData, company: e.target.value})}
                  className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 text-white focus:border-blue-500 focus:outline-none"
                  placeholder="Your company name"
                />
              </div>

              <div className="mb-6">
                <label className="block text-gray-300 text-sm mb-2">I&apos;m interested in *</label>
                <select
                  required
                  value={formData.interest}
                  onChange={(e) => setFormData({...formData, interest: e.target.value})}
                  className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 text-white focus:border-blue-500 focus:outline-none"
                >
                  <option value="consulting">Consulting & Custom Integration</option>
                  <option value="support">Priority Support</option>
                  <option value="cloud-waitlist">Cloud SaaS Waitlist</option>
                  <option value="partnership">Partnership / Reseller</option>
                  <option value="other">Other</option>
                </select>
              </div>

              <div className="mb-8">
                <label className="block text-gray-300 text-sm mb-2">Message *</label>
                <textarea
                  required
                  rows={5}
                  value={formData.message}
                  onChange={(e) => setFormData({...formData, message: e.target.value})}
                  className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 text-white focus:border-blue-500 focus:outline-none resize-none"
                  placeholder="Tell us about your needs..."
                />
              </div>

              <button
                type="submit"
                className="w-full bg-blue-600 hover:bg-blue-700 text-white py-4 rounded-lg font-medium transition"
              >
                Send Message
              </button>
            </form>
          )}

          {/* Alternative Contact */}
          <div className="mt-12 text-center">
            <p className="text-gray-500 mb-4">Or reach us directly:</p>
            <div className="flex flex-col sm:flex-row justify-center gap-4">
              <Link href="https://github.com/ai-trace/ai-trace/discussions" className="text-blue-400 hover:underline">
                GitHub Discussions
              </Link>
              <span className="text-gray-600 hidden sm:inline">•</span>
              <Link href="mailto:hello@aitrace.cc" className="text-blue-400 hover:underline">
                hello@aitrace.cc
              </Link>
            </div>
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
