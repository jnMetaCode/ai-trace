'use client'

import Link from 'next/link'
import { useState } from 'react'
import OnboardingGuide from '@/components/OnboardingGuide'

export default function GetStartedPage() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-900 to-gray-800">
      {/* Navigation */}
      <nav className="fixed top-0 w-full bg-gray-900/80 backdrop-blur-sm border-b border-gray-800 z-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16 items-center">
            <div className="flex items-center space-x-2">
              <Link href="/" className="flex items-center space-x-2">
                <div className="w-8 h-8 bg-blue-500 rounded-lg flex items-center justify-center">
                  <span className="text-white font-bold text-sm">AT</span>
                </div>
                <span className="text-white font-semibold text-xl">AI-Trace</span>
              </Link>
            </div>
            {/* Desktop Navigation */}
            <div className="hidden md:flex items-center space-x-8">
              <Link href="/#features" className="text-gray-300 hover:text-white transition">Features</Link>
              <Link href="/#deployment" className="text-gray-300 hover:text-white transition">Deployment</Link>
              <Link href="/docs" className="text-gray-300 hover:text-white transition">Docs</Link>
              <Link href="/faq" className="text-gray-300 hover:text-white transition">FAQ</Link>
              <Link href="https://github.com/ai-trace/ai-trace" className="text-gray-300 hover:text-white transition">GitHub</Link>
              <span className="bg-blue-600 text-white px-4 py-2 rounded-lg">
                Get Started
              </span>
            </div>
            {/* Mobile menu button */}
            <button
              className="md:hidden text-gray-300 hover:text-white"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
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
        {/* Mobile Navigation */}
        {mobileMenuOpen && (
          <div className="md:hidden bg-gray-900 border-t border-gray-800">
            <div className="px-4 py-4 space-y-3">
              <Link href="/#features" className="block text-gray-300 hover:text-white transition" onClick={() => setMobileMenuOpen(false)}>Features</Link>
              <Link href="/#deployment" className="block text-gray-300 hover:text-white transition" onClick={() => setMobileMenuOpen(false)}>Deployment</Link>
              <Link href="/docs" className="block text-gray-300 hover:text-white transition" onClick={() => setMobileMenuOpen(false)}>Docs</Link>
              <Link href="/faq" className="block text-gray-300 hover:text-white transition" onClick={() => setMobileMenuOpen(false)}>FAQ</Link>
              <Link href="https://github.com/ai-trace/ai-trace" className="block text-gray-300 hover:text-white transition">GitHub</Link>
            </div>
          </div>
        )}
      </nav>

      {/* Header */}
      <section className="pt-24 pb-8 px-4">
        <div className="max-w-4xl mx-auto text-center">
          <div className="inline-flex items-center px-3 py-1 rounded-full bg-green-500/10 border border-green-500/20 text-green-400 text-sm mb-6">
            <span className="w-2 h-2 bg-green-400 rounded-full mr-2 animate-pulse"></span>
            Interactive Tutorial
          </div>
          <h1 className="text-4xl md:text-5xl font-bold text-white mb-4">
            Get Started with AI-Trace
          </h1>
          <p className="text-xl text-gray-400 max-w-2xl mx-auto">
            Follow this step-by-step guide to set up AI-Trace, make your first traced API call,
            and generate tamper-proof certificates.
          </p>
        </div>
      </section>

      {/* Quick info cards */}
      <section className="py-6 px-4">
        <div className="max-w-4xl mx-auto">
          <div className="grid md:grid-cols-3 gap-4 mb-8">
            <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-4 text-center">
              <div className="text-3xl font-bold text-blue-400">5</div>
              <div className="text-gray-500 text-sm">Minutes to deploy</div>
            </div>
            <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-4 text-center">
              <div className="text-3xl font-bold text-purple-400">1</div>
              <div className="text-gray-500 text-sm">Line code change</div>
            </div>
            <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-4 text-center">
              <div className="text-3xl font-bold text-green-400">0</div>
              <div className="text-gray-500 text-sm">Dependencies needed</div>
            </div>
          </div>
        </div>
      </section>

      {/* Onboarding Guide */}
      <section className="py-8 px-4">
        <OnboardingGuide />
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-8 px-4 mt-12">
        <div className="max-w-4xl mx-auto text-center">
          <p className="text-gray-500 text-sm">
            Need help? Check out our{' '}
            <Link href="/docs" className="text-blue-400 hover:text-blue-300">documentation</Link>
            {' '}or{' '}
            <Link href="https://github.com/ai-trace/ai-trace/discussions" className="text-blue-400 hover:text-blue-300">ask on GitHub</Link>.
          </p>
          <p className="text-gray-600 text-sm mt-4">
            &copy; 2025 AI-Trace. Open source under Apache 2.0.
          </p>
        </div>
      </footer>
    </div>
  )
}
