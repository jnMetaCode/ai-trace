package aitrace_test

import (
	"context"
	"fmt"
	"log"

	aitrace "github.com/ai-trace/sdk-go"
)

func Example() {
	// Create AI-Trace client
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
	fmt.Printf("Trace ID: %s\n", resp.TraceID)

	// Commit certificate for L2 (WORM storage)
	cert, err := client.Certs.Commit(ctx, resp.TraceID, aitrace.EvidenceLevelL2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Certificate ID: %s\n", cert.CertID)
	fmt.Printf("Root Hash: %s\n", cert.RootHash)

	// Verify certificate
	result, err := client.Certs.VerifyByCertID(ctx, cert.CertID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Valid: %v\n", result.Valid)
}

func ExampleChatService_Create() {
	client := aitrace.NewClient("your-api-key")
	ctx := context.Background()

	// Simple chat completion
	resp, err := client.Chat.Create(ctx, aitrace.ChatRequest{
		Model: "gpt-4",
		Messages: []aitrace.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Choices[0].Message.Content)
}

func ExampleEventsService_Ingest() {
	client := aitrace.NewClient("your-api-key")
	ctx := context.Background()

	// Manually ingest events
	events := []aitrace.Event{
		aitrace.InputEvent("my-trace-123", "What is the weather?", "gpt-4"),
		aitrace.OutputEvent("my-trace-123", "I cannot check the weather.", 10),
	}

	resp, err := client.Events.Ingest(ctx, events)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Ingested %d events\n", resp.Ingested)

	// Search events
	searchResp, err := client.Events.Search(ctx, aitrace.EventSearchRequest{
		TraceID: "my-trace-123",
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, event := range searchResp.Events {
		fmt.Printf("Event: %s - %s\n", event.EventID, event.EventType)
	}
}

func ExampleCertsService_Prove() {
	client := aitrace.NewClient("your-api-key")
	ctx := context.Background()

	// Generate minimal disclosure proof
	// Only reveal events at index 0 and 2
	proof, err := client.Certs.Prove(ctx, "cert-123", aitrace.ProveRequest{
		DiscloseEvents: []int{0, 2},
		DiscloseFields: []string{"prompt", "response"},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Proof root hash: %s\n", proof.RootHash)
	fmt.Printf("Disclosed events: %d\n", len(proof.DiscloseEvents))
}

func ExampleEventBuilder() {
	// Build custom events programmatically
	event := aitrace.NewEventBuilder("trace-123", aitrace.EventTypeInput).
		WithSequence(1).
		AddPayloadField("prompt", "Hello, world!").
		AddPayloadField("model_id", "gpt-4").
		AddPayloadField("temperature", 0.7).
		Build()

	fmt.Printf("Event ID: %s\n", event.EventID)
	fmt.Printf("Event Type: %s\n", event.EventType)
}

func ExampleChatService_CreateAndCommit() {
	client := aitrace.NewClient("your-api-key")
	ctx := context.Background()

	// Create chat and immediately commit certificate
	resp, cert, err := client.Chat.CreateAndCommit(ctx, aitrace.ChatRequest{
		Model: "gpt-4",
		Messages: []aitrace.Message{
			{Role: "user", Content: "Explain quantum computing"},
		},
	}, aitrace.EvidenceLevelL2)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response: %s\n", resp.Choices[0].Message.Content)
	fmt.Printf("Certificate: %s\n", cert.CertID)
}
