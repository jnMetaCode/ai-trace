# AI-Trace Go SDK

Official Go SDK for the AI-Trace platform - Enterprise AI decision auditing and tamper-proof attestation.

## Installation

```bash
go get github.com/ai-trace/sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    aitrace "github.com/ai-trace/sdk-go"
)

func main() {
    // Create client
    client := aitrace.NewClient("your-api-key",
        aitrace.WithBaseURL("https://api.aitrace.cc"),
        aitrace.WithUpstreamAPIKey("sk-your-openai-key"), // Pass-through, never stored
    )

    ctx := context.Background()

    // Create chat completion with attestation
    resp, err := client.Chat.Create(ctx, aitrace.ChatRequest{
        Model: "gpt-4",
        Messages: []aitrace.Message{
            {Role: "user", Content: "What is 2+2?"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Response: %s\n", resp.Choices[0].Message.Content)

    // Commit certificate
    cert, err := client.Certs.Commit(ctx, resp.TraceID, aitrace.EvidenceLevelL2)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Certificate ID: %s\n", cert.CertID)
    fmt.Printf("Root Hash: %s\n", cert.RootHash)
}
```

## Features

### Chat Completions (OpenAI Compatible)

```go
resp, err := client.Chat.Create(ctx, aitrace.ChatRequest{
    Model: "gpt-4",
    Messages: []aitrace.Message{
        {Role: "system", Content: "You are a helpful assistant."},
        {Role: "user", Content: "Hello!"},
    },
    Temperature: 0.7,
    MaxTokens:   100,
    TraceID:     "custom-trace-id",  // Optional
    SessionID:   "session-123",       // Optional
    BusinessID:  "business-456",      // Optional
})
```

### Event Management

```go
// Ingest custom events
events := []aitrace.Event{
    aitrace.InputEvent("trace-123", "User prompt", "gpt-4"),
    aitrace.OutputEvent("trace-123", "AI response", 50),
}
resp, err := client.Events.Ingest(ctx, events)

// Search events
searchResp, err := client.Events.Search(ctx, aitrace.EventSearchRequest{
    TraceID:   "trace-123",
    EventType: aitrace.EventTypeOutput,
    Page:      1,
    PageSize:  20,
})

// Get all events for a trace
events, err := client.Events.GetByTrace(ctx, "trace-123")
```

### Certificate Management

```go
// Commit certificate with different evidence levels
cert, err := client.Certs.Commit(ctx, traceID, aitrace.EvidenceLevelL2)

// Evidence levels:
// - L1: Basic (Merkle tree + timestamp)
// - L2: WORM storage (legal compliance)
// - L3: Blockchain anchor (maximum tamper-proof)

// Verify certificate
result, err := client.Certs.VerifyByCertID(ctx, "cert-123")
if result.Valid {
    fmt.Println("Certificate is valid!")
}

// Generate minimal disclosure proof
proof, err := client.Certs.Prove(ctx, "cert-123", aitrace.ProveRequest{
    DiscloseEvents: []int{0, 2},  // Only reveal events at index 0 and 2
    DiscloseFields: []string{"prompt", "response"},
})
```

### Convenience Methods

```go
// Create and immediately commit
resp, cert, err := client.Chat.CreateAndCommit(ctx, req, aitrace.EvidenceLevelL2)

// Build events programmatically
event := aitrace.NewEventBuilder("trace-123", aitrace.EventTypeInput).
    WithSequence(1).
    AddPayloadField("prompt", "Hello").
    AddPayloadField("model_id", "gpt-4").
    Build()
```

## Client Options

```go
client := aitrace.NewClient("api-key",
    aitrace.WithBaseURL("https://custom.example.com"),
    aitrace.WithUpstreamAPIKey("sk-upstream-key"),
    aitrace.WithUpstreamBaseURL("https://upstream.example.com"),
    aitrace.WithTimeout(30 * time.Second),
    aitrace.WithHTTPClient(customHTTPClient),
)
```

## Error Handling

```go
resp, err := client.Chat.Create(ctx, req)
if err != nil {
    if apiErr, ok := err.(*aitrace.APIError); ok {
        fmt.Printf("API Error: %s (code: %s)\n", apiErr.Message, apiErr.Code)
    } else {
        fmt.Printf("Error: %v\n", err)
    }
}
```

## License

Apache-2.0
