package aitrace

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// EventsService handles event operations.
type EventsService struct {
	client *Client
}

// Ingest ingests a batch of events.
//
// Example:
//
//	resp, err := client.Events.Ingest(ctx, []aitrace.Event{
//	    {
//	        TraceID:   "trace-123",
//	        EventType: aitrace.EventTypeInput,
//	        Payload:   map[string]interface{}{"prompt": "Hello"},
//	    },
//	})
func (s *EventsService) Ingest(ctx context.Context, events []Event) (*IngestResponse, error) {
	// Input validation
	if len(events) == 0 {
		return nil, &APIError{Code: "invalid_request", Message: "at least one event is required", StatusCode: 400}
	}
	for i, e := range events {
		if e.TraceID == "" {
			return nil, &APIError{Code: "invalid_request", Message: fmt.Sprintf("event[%d]: trace_id is required", i), StatusCode: 400}
		}
		if e.EventType == "" {
			return nil, &APIError{Code: "invalid_request", Message: fmt.Sprintf("event[%d]: event_type is required", i), StatusCode: 400}
		}
	}

	req := IngestRequest{Events: events}

	respBody, err := s.client.post(ctx, "/api/v1/events/ingest", req, nil)
	if err != nil {
		return nil, err
	}

	var resp IngestResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// Search searches for events.
//
// Example:
//
//	resp, err := client.Events.Search(ctx, aitrace.EventSearchRequest{
//	    TraceID:   "trace-123",
//	    EventType: aitrace.EventTypeOutput,
//	    Page:      1,
//	    PageSize:  20,
//	})
func (s *EventsService) Search(ctx context.Context, req EventSearchRequest) (*EventSearchResponse, error) {
	params := make(map[string]string)

	if req.TraceID != "" {
		params["trace_id"] = req.TraceID
	}
	if req.EventType != "" {
		params["event_type"] = req.EventType
	}
	if req.StartTime != "" {
		params["start_time"] = req.StartTime
	}
	if req.EndTime != "" {
		params["end_time"] = req.EndTime
	}
	if req.Page > 0 {
		params["page"] = strconv.Itoa(req.Page)
	}
	if req.PageSize > 0 {
		params["page_size"] = strconv.Itoa(req.PageSize)
	}

	respBody, err := s.client.get(ctx, "/api/v1/events/search", params)
	if err != nil {
		return nil, err
	}

	var resp EventSearchResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// Get retrieves a single event by ID.
func (s *EventsService) Get(ctx context.Context, eventID string) (*Event, error) {
	if eventID == "" {
		return nil, &APIError{Code: "invalid_request", Message: "event_id is required", StatusCode: 400}
	}

	respBody, err := s.client.get(ctx, fmt.Sprintf("/api/v1/events/%s", eventID), nil)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(respBody, &event); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &event, nil
}

// GetByTrace retrieves all events for a trace ID.
func (s *EventsService) GetByTrace(ctx context.Context, traceID string) ([]Event, error) {
	if traceID == "" {
		return nil, &APIError{Code: "invalid_request", Message: "trace_id is required", StatusCode: 400}
	}

	resp, err := s.Search(ctx, EventSearchRequest{
		TraceID:  traceID,
		Page:     1,
		PageSize: 1000, // Get all events
	})
	if err != nil {
		return nil, err
	}

	return resp.Events, nil
}

// EventBuilder helps build events programmatically.
type EventBuilder struct {
	event Event
}

// NewEventBuilder creates a new event builder.
func NewEventBuilder(traceID string, eventType string) *EventBuilder {
	return &EventBuilder{
		event: Event{
			EventID:   uuid.New().String(),
			TraceID:   traceID,
			EventType: eventType,
			Timestamp: time.Now(),
			Payload:   make(map[string]interface{}),
		},
	}
}

// WithEventID sets a custom event ID.
func (b *EventBuilder) WithEventID(id string) *EventBuilder {
	b.event.EventID = id
	return b
}

// WithSequence sets the event sequence number.
func (b *EventBuilder) WithSequence(seq int) *EventBuilder {
	b.event.Sequence = seq
	return b
}

// WithTimestamp sets a custom timestamp.
func (b *EventBuilder) WithTimestamp(t time.Time) *EventBuilder {
	b.event.Timestamp = t
	return b
}

// WithPrevEventHash sets the previous event hash (for chaining).
func (b *EventBuilder) WithPrevEventHash(hash string) *EventBuilder {
	b.event.PrevEventHash = hash
	return b
}

// WithPayload sets the entire payload.
func (b *EventBuilder) WithPayload(payload map[string]interface{}) *EventBuilder {
	b.event.Payload = payload
	return b
}

// AddPayloadField adds a field to the payload.
func (b *EventBuilder) AddPayloadField(key string, value interface{}) *EventBuilder {
	b.event.Payload[key] = value
	return b
}

// Build returns the constructed event.
func (b *EventBuilder) Build() Event {
	return b.event
}

// InputEvent creates an input event.
func InputEvent(traceID string, prompt string, modelID string) Event {
	return NewEventBuilder(traceID, EventTypeInput).
		AddPayloadField("prompt", prompt).
		AddPayloadField("model_id", modelID).
		Build()
}

// OutputEvent creates an output event.
func OutputEvent(traceID string, content string, tokensUsed int) Event {
	return NewEventBuilder(traceID, EventTypeOutput).
		AddPayloadField("content", content).
		AddPayloadField("tokens_used", tokensUsed).
		Build()
}

// ToolCallEvent creates a tool call event.
func ToolCallEvent(traceID string, toolName string, arguments map[string]interface{}) Event {
	return NewEventBuilder(traceID, EventTypeToolCall).
		AddPayloadField("tool_name", toolName).
		AddPayloadField("arguments", arguments).
		Build()
}

// ToolResultEvent creates a tool result event.
func ToolResultEvent(traceID string, toolName string, result interface{}) Event {
	return NewEventBuilder(traceID, EventTypeToolResult).
		AddPayloadField("tool_name", toolName).
		AddPayloadField("result", result).
		Build()
}

// ErrorEvent creates an error event.
func ErrorEvent(traceID string, errorCode string, errorMessage string) Event {
	return NewEventBuilder(traceID, EventTypeError).
		AddPayloadField("error_code", errorCode).
		AddPayloadField("error_message", errorMessage).
		Build()
}
