#!/usr/bin/env python3
"""
Basic AI-Trace usage example.

This example demonstrates:
1. Creating a trace
2. Adding events
3. Committing to a certificate
4. Verifying the certificate
"""

from ai_trace import AITrace, EvidenceLevel

def main():
    # Initialize the client
    client = AITrace(
        server_url="http://localhost:8006",
        api_key="test-api-key-12345",
        tenant_id="default"
    )

    print("=" * 50)
    print("AI-Trace Basic Example")
    print("=" * 50)

    # Step 1: Create a trace
    print("\n1. Creating a trace...")
    trace = client.traces.create(
        name="Customer Support Chat",
        metadata={
            "department": "support",
            "channel": "web"
        }
    )
    print(f"   Trace ID: {trace.id}")

    # Step 2: Add input event
    print("\n2. Adding input event...")
    input_event = client.events.add(
        trace_id=trace.id,
        event_type="input",
        payload={
            "model": "gpt-4",
            "messages": [
                {"role": "system", "content": "You are a helpful assistant."},
                {"role": "user", "content": "What is AI-Trace?"}
            ],
            "parameters": {
                "temperature": 0.7,
                "max_tokens": 1000
            }
        }
    )
    print(f"   Event ID: {input_event.id}")
    print(f"   Hash: {input_event.hash[:20]}...")

    # Step 3: Add output event
    print("\n3. Adding output event...")
    output_event = client.events.add(
        trace_id=trace.id,
        event_type="output",
        payload={
            "response": {
                "content": "AI-Trace is an open-source platform for creating tamper-proof attestations of AI decisions.",
                "model": "gpt-4",
                "usage": {
                    "prompt_tokens": 50,
                    "completion_tokens": 30,
                    "total_tokens": 80
                }
            }
        }
    )
    print(f"   Event ID: {output_event.id}")
    print(f"   Hash: {output_event.hash[:20]}...")

    # Step 4: Commit to certificate
    print("\n4. Committing to certificate...")
    cert = client.certs.commit(
        trace_id=trace.id,
        evidence_level=EvidenceLevel.L1
    )
    print(f"   Certificate ID: {cert.id}")
    print(f"   Evidence Level: {cert.evidence_level.value}")
    print(f"   Root Hash: {cert.root_hash[:20]}...")
    print(f"   Event Count: {cert.event_count}")

    # Step 5: Verify certificate
    print("\n5. Verifying certificate...")
    result = client.certs.verify(cert_id=cert.id)
    print(f"   Valid: {result.valid}")
    print(f"   Hash Valid: {result.hash_valid}")
    print(f"   Signature Valid: {result.signature_valid}")
    print(f"   Timestamp Valid: {result.timestamp_valid}")

    print("\n" + "=" * 50)
    print("Example completed successfully!")
    print("=" * 50)

    # Cleanup
    client.close()


if __name__ == "__main__":
    main()
