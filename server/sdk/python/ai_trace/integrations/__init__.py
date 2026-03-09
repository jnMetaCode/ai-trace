"""
AI-Trace integrations for popular AI providers.

These integrations provide automatic tracing for AI API calls.
"""

from ai_trace.integrations.openai import TracedOpenAI

__all__ = ["TracedOpenAI"]
