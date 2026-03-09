#!/usr/bin/env python3
"""
OpenAI integration example with automatic tracing.

This example demonstrates using TracedOpenAI as a drop-in
replacement for the OpenAI client with automatic tracing.

Requirements:
    pip install ai-trace[openai]
"""

import os
from ai_trace.integrations import TracedOpenAI

def main():
    # Get API keys from environment
    openai_key = os.environ.get("OPENAI_API_KEY")
    if not openai_key:
        print("Error: OPENAI_API_KEY environment variable not set")
        print("Set it with: export OPENAI_API_KEY=sk-...")
        return

    print("=" * 50)
    print("AI-Trace OpenAI Integration Example")
    print("=" * 50)

    # Create a traced OpenAI client
    # This is a drop-in replacement for openai.OpenAI()
    client = TracedOpenAI(
        openai_api_key=openai_key,
        ai_trace_url="http://localhost:8006",
        ai_trace_key="test-api-key-12345",
        auto_commit=True,  # Automatically commit on context exit
        evidence_level="L1"
    )

    print("\n1. Making a traced API call...")

    # Use just like regular OpenAI client
    response = client.chat.completions.create(
        model="gpt-3.5-turbo",
        messages=[
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "What is 2 + 2?"}
        ],
        temperature=0.7,
        max_tokens=100
    )

    print(f"   Response: {response.choices[0].message.content}")
    print(f"   Trace ID: {client.current_trace_id}")

    # Make another call in the same trace
    print("\n2. Making another call in the same trace...")

    response2 = client.chat.completions.create(
        model="gpt-3.5-turbo",
        messages=[
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "What is 3 * 4?"}
        ]
    )

    print(f"   Response: {response2.choices[0].message.content}")
    print(f"   Same Trace ID: {client.current_trace_id}")

    # Manually commit the trace
    print("\n3. Committing trace to certificate...")
    cert = client.commit_trace(evidence_level="L1")
    print(f"   Certificate ID: {cert.id}")
    print(f"   Root Hash: {cert.root_hash[:20]}...")

    print("\n" + "=" * 50)
    print("Example completed successfully!")
    print("=" * 50)


def streaming_example():
    """Example with streaming responses."""
    openai_key = os.environ.get("OPENAI_API_KEY")
    if not openai_key:
        return

    print("\n" + "=" * 50)
    print("Streaming Example")
    print("=" * 50)

    client = TracedOpenAI(
        openai_api_key=openai_key,
        ai_trace_url="http://localhost:8006",
        ai_trace_key="test-api-key-12345",
    )

    client.start_trace(name="Streaming Demo")

    print("\nStreaming response:")
    stream = client.chat.completions.create(
        model="gpt-3.5-turbo",
        messages=[{"role": "user", "content": "Count from 1 to 5"}],
        stream=True
    )

    for chunk in stream:
        if chunk.choices[0].delta.content:
            print(chunk.choices[0].delta.content, end="", flush=True)
    print()

    # Commit the trace
    cert = client.commit_trace()
    print(f"\nCertificate: {cert.id}")


if __name__ == "__main__":
    main()
    # streaming_example()  # Uncomment to run streaming example
