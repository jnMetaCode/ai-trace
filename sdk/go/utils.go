package aitrace

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// HashContent computes SHA256 hash of content.
func HashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// HashBytes computes SHA256 hash of bytes.
func HashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// HashJSON computes SHA256 hash of a JSON-serializable object.
func HashJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return HashBytes(data), nil
}

// NewTraceID generates a new trace ID using UUID v4.
func NewTraceID() string {
	return NewUUID()
}

// NewEventID generates a new event ID using UUID v4.
func NewEventID() string {
	return NewUUID()
}

// NewUUID generates a new UUID v4.
func NewUUID() string {
	// Simple UUID v4 generation without external dependency
	// In production, use github.com/google/uuid
	return generateUUID()
}

// generateUUID generates a UUID v4.
func generateUUID() string {
	// Generate 16 random bytes
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("fallback-%d", hashTimestamp())
	}

	// Set version (4) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// hashTimestamp returns a simple hash for fallback ID generation.
func hashTimestamp() int64 {
	b := make([]byte, 8)
	rand.Read(b)
	return int64(b[0]) | int64(b[1])<<8 | int64(b[2])<<16 | int64(b[3])<<24 |
		int64(b[4])<<32 | int64(b[5])<<40 | int64(b[6])<<48 | int64(b[7])<<56
}

// Helper functions for convenience

// IsValidEvidenceLevel checks if the evidence level is valid.
func IsValidEvidenceLevel(level string) bool {
	switch level {
	case EvidenceLevelL1, EvidenceLevelL2, EvidenceLevelL3:
		return true
	default:
		return false
	}
}

// IsValidEventType checks if the event type is valid.
func IsValidEventType(eventType string) bool {
	switch eventType {
	case EventTypeInput, EventTypeOutput, EventTypeChunk,
		EventTypeToolCall, EventTypeToolResult, EventTypeError:
		return true
	default:
		return false
	}
}
