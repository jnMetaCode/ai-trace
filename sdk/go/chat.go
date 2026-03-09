package aitrace

import (
	"context"
	"encoding/json"
	"fmt"
)

// ChatService handles chat completion operations.
type ChatService struct {
	client      *Client
	lastTraceID string
}

// Create creates a chat completion with AI-Trace attestation.
//
// This is an OpenAI-compatible endpoint that proxies requests to the
// upstream provider while recording all events for attestation.
//
// Example:
//
//	resp, err := client.Chat.Create(ctx, aitrace.ChatRequest{
//	    Model: "gpt-4",
//	    Messages: []aitrace.Message{
//	        {Role: "user", Content: "Hello!"},
//	    },
//	    TraceID: "my-trace-123",  // Optional, auto-generated if not provided
//	})
func (s *ChatService) Create(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Input validation
	if req.Model == "" {
		return nil, &APIError{Code: "invalid_request", Message: "model is required", StatusCode: 400}
	}
	if len(req.Messages) == 0 {
		return nil, &APIError{Code: "invalid_request", Message: "at least one message is required", StatusCode: 400}
	}

	// Build headers for trace context
	headers := make(map[string]string)
	if req.TraceID != "" {
		headers["X-Trace-ID"] = req.TraceID
	}
	if req.SessionID != "" {
		headers["X-Session-ID"] = req.SessionID
	}
	if req.BusinessID != "" {
		headers["X-Business-ID"] = req.BusinessID
	}

	// Set default temperature if not specified
	if req.Temperature == 0 {
		req.Temperature = 1.0
	}

	// Make request
	respBody, err := s.client.post(ctx, "/api/v1/chat/completions", req, headers)
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp ChatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Store trace ID for later use
	if resp.TraceID != "" {
		s.lastTraceID = resp.TraceID
	} else if req.TraceID != "" {
		s.lastTraceID = req.TraceID
	}

	return &resp, nil
}

// CreateWithCallback creates a chat completion and executes a callback with the trace ID.
//
// This is useful when you want to immediately process the trace ID for attestation.
//
// Example:
//
//	resp, err := client.Chat.CreateWithCallback(ctx, req, func(traceID string) {
//	    log.Printf("Trace ID: %s", traceID)
//	    // Store trace ID for later attestation
//	})
func (s *ChatService) CreateWithCallback(ctx context.Context, req ChatRequest, callback func(traceID string)) (*ChatResponse, error) {
	resp, err := s.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	if callback != nil && s.lastTraceID != "" {
		callback(s.lastTraceID)
	}

	return resp, nil
}

// LastTraceID returns the trace ID from the last request.
func (s *ChatService) LastTraceID() string {
	return s.lastTraceID
}

// CreateAndCommit creates a chat completion and immediately commits a certificate.
//
// This is a convenience method that combines Create and Certs.Commit.
//
// Example:
//
//	resp, cert, err := client.Chat.CreateAndCommit(ctx, req, "L2")
func (s *ChatService) CreateAndCommit(ctx context.Context, req ChatRequest, evidenceLevel string) (*ChatResponse, *Certificate, error) {
	resp, err := s.Create(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	if s.lastTraceID == "" {
		return resp, nil, fmt.Errorf("no trace ID available for commitment")
	}

	cert, err := s.client.Certs.Commit(ctx, s.lastTraceID, evidenceLevel)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to commit certificate: %w", err)
	}

	return resp, cert, nil
}

// StreamCreate creates a streaming chat completion.
// Note: Streaming is not yet implemented in this SDK version.
func (s *ChatService) StreamCreate(ctx context.Context, req ChatRequest) (<-chan *ChatStreamResponse, <-chan error) {
	respChan := make(chan *ChatStreamResponse)
	errChan := make(chan error, 1)

	go func() {
		defer close(respChan)
		defer close(errChan)

		// TODO: Implement streaming
		errChan <- fmt.Errorf("streaming not yet implemented")
	}()

	return respChan, errChan
}

// ChatStreamResponse represents a streaming chat response chunk.
type ChatStreamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice represents a streaming choice.
type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        MessageDelta `json:"delta"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

// MessageDelta represents a delta message in streaming.
type MessageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}
