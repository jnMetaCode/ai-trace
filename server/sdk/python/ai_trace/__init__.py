"""
AI-Trace Python SDK

Tamper-proof attestation for AI decisions.

Example usage:

    from ai_trace import AITrace

    # Initialize client
    client = AITrace(
        server_url="http://localhost:8006",
        api_key="your-api-key"
    )

    # Create a trace
    trace = client.traces.create(name="Customer Support Chat")

    # Add events
    client.events.add(
        trace_id=trace.id,
        event_type="input",
        payload={"prompt": "Hello, how can I help?"}
    )

    # Commit to certificate
    cert = client.certs.commit(trace_id=trace.id, evidence_level="L2")

    # Verify certificate
    result = client.certs.verify(cert_id=cert.id)
    print(f"Valid: {result.valid}")
"""

from ai_trace.client import AITrace
from ai_trace.models import (
    Trace,
    Event,
    Certificate,
    VerificationResult,
    Proof,
    EvidenceLevel,
)
from ai_trace.exceptions import (
    AITraceError,
    AuthenticationError,
    NotFoundError,
    ValidationError,
    ServerError,
)

__version__ = "0.1.0"
__all__ = [
    "AITrace",
    "Trace",
    "Event",
    "Certificate",
    "VerificationResult",
    "Proof",
    "EvidenceLevel",
    "AITraceError",
    "AuthenticationError",
    "NotFoundError",
    "ValidationError",
    "ServerError",
]
