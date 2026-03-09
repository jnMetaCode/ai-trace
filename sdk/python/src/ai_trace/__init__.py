"""
AI-Trace Python SDK
Enterprise AI Audit & Evidence System SDK

Provides full-chain traceability for AI applications with Merkle tree
certification, minimal disclosure proofs, and OpenAI integration.

Usage:
    from ai_trace import AITraceClient

    client = AITraceClient(
        base_url="http://localhost:8080",
        api_key="your-api-key"
    )

    # Search events
    events = client.events.search(trace_id="trc_xxx")

    # Create certificate
    cert = client.certs.commit(trace_id="trc_xxx")

    # Verify certificate
    result = client.certs.verify(cert_id="cert_xxx")
"""

from .client import AITraceClient, AsyncAITraceClient
from .models import (
    Certificate,
    CommitCertResponse,
    EvidenceLevel,
    Event,
    EventType,
    MinimalDisclosureProof,
    SearchEventsResponse,
    VerifyResult,
)
from .wrapper import TracedOpenAI

__version__ = "0.1.0"
__all__ = [
    "AITraceClient",
    "AsyncAITraceClient",
    "Certificate",
    "CommitCertResponse",
    "Event",
    "EventType",
    "EvidenceLevel",
    "MinimalDisclosureProof",
    "SearchEventsResponse",
    "TracedOpenAI",
    "VerifyResult",
]
