#!/usr/bin/env python3
"""
Compliance audit example.

This example demonstrates creating audit-ready certificates
for regulatory compliance use cases (GDPR, HIPAA, SOC2, etc.)
"""

from datetime import datetime
from ai_trace import AITrace, EvidenceLevel

def main():
    client = AITrace(
        server_url="http://localhost:8006",
        api_key="test-api-key-12345"
    )

    print("=" * 60)
    print("AI-Trace Compliance Audit Example")
    print("=" * 60)

    # Simulate a healthcare AI interaction
    print("\n[Scenario: Healthcare AI Assistant]")
    print("-" * 40)

    # Create trace with compliance metadata
    trace = client.traces.create(
        name="Patient Symptom Assessment",
        metadata={
            "compliance_framework": "HIPAA",
            "data_classification": "PHI",
            "department": "radiology",
            "requesting_physician": "dr.smith@hospital.org",
            "audit_required": True
        }
    )
    print(f"Trace created: {trace.id}")

    # Record patient query (anonymized)
    client.events.add(
        trace_id=trace.id,
        event_type="input",
        payload={
            "query_type": "symptom_assessment",
            "patient_id_hash": "sha256:abc123...",  # Hashed for privacy
            "symptoms": ["chest pain", "shortness of breath"],
            "metadata": {
                "age_range": "50-60",
                "gender": "M",
                "urgency": "high"
            }
        },
        metadata={
            "data_minimization": True,
            "anonymized": True
        }
    )
    print("Input event recorded (anonymized)")

    # Record AI recommendation
    client.events.add(
        trace_id=trace.id,
        event_type="output",
        payload={
            "recommendation": "Recommend immediate ECG and cardiac enzyme panel",
            "confidence": 0.92,
            "risk_factors": ["age", "symptom_combination"],
            "model": "medical-assessment-v2",
            "disclaimer": "AI-assisted recommendation - physician review required"
        },
        metadata={
            "model_version": "2.1.0",
            "decision_timestamp": datetime.utcnow().isoformat()
        }
    )
    print("Output event recorded")

    # Record physician review
    client.events.add(
        trace_id=trace.id,
        event_type="custom",
        payload={
            "event_name": "physician_review",
            "reviewer": "dr.smith",
            "action": "approved",
            "notes": "Concur with AI recommendation. Ordered tests.",
            "review_timestamp": datetime.utcnow().isoformat()
        }
    )
    print("Physician review recorded")

    # Commit with L2 (WORM + TSA) for regulatory compliance
    print("\nCommitting to L2 certificate (WORM + TSA)...")
    cert = client.certs.commit(
        trace_id=trace.id,
        evidence_level=EvidenceLevel.L2,
        metadata={
            "retention_period_years": 7,
            "compliance_framework": "HIPAA",
            "audit_trail": True
        }
    )

    print("\n" + "=" * 60)
    print("CERTIFICATE DETAILS")
    print("=" * 60)
    print(f"Certificate ID:    {cert.id}")
    print(f"Evidence Level:    {cert.evidence_level.value}")
    print(f"Root Hash:         {cert.root_hash}")
    print(f"Event Count:       {cert.event_count}")
    print(f"Created:           {cert.created_at}")
    if cert.tsa_timestamp:
        print(f"TSA Timestamp:     {cert.tsa_timestamp[:50]}...")
    if cert.worm_location:
        print(f"WORM Location:     {cert.worm_location}")

    # Verify for audit
    print("\n" + "-" * 60)
    print("VERIFICATION FOR AUDIT")
    print("-" * 60)

    result = client.certs.verify(cert_id=cert.id, full_verification=True)

    print(f"Overall Valid:     {'PASS' if result.valid else 'FAIL'}")
    print(f"Hash Integrity:    {'PASS' if result.hash_valid else 'FAIL'}")
    print(f"Signature:         {'PASS' if result.signature_valid else 'FAIL'}")
    print(f"Timestamp:         {'PASS' if result.timestamp_valid else 'FAIL'}")
    if result.anchor_verified is not None:
        print(f"Anchor:            {'PASS' if result.anchor_verified else 'FAIL'}")

    # Generate minimal disclosure proof
    print("\n" + "-" * 60)
    print("MINIMAL DISCLOSURE PROOF")
    print("-" * 60)
    print("(For third-party auditor - reveals only necessary information)")

    proof = client.certs.prove(
        cert_id=cert.id,
        event_indices=[1],  # Only the AI output
        disclosed_fields=["recommendation", "confidence", "model"]
    )

    print(f"Proof ID:          {proof.proof_id}")
    print(f"Events Disclosed:  {len(proof.disclosed_events)}")
    print(f"Root Hash:         {proof.root_hash}")
    print(f"Verifiable:        {proof.verifiable}")

    print("\n" + "=" * 60)
    print("Compliance audit example completed")
    print("=" * 60)

    client.close()


if __name__ == "__main__":
    main()
