// Package middleware provides HTTP middleware for AI-Trace.
package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// TraceResponseHeaders holds information to include in response headers.
type TraceResponseHeaders struct {
	TraceID      string
	EventCount   int
	PayloadHash  string
	EvidenceHint string
}

// SetTraceHeaders adds human-readable trace headers to the response.
// These headers help users understand what happened without checking logs.
//
// Headers added:
//   - X-Trace-Id: The trace ID for this request
//   - X-AI-Trace-Summary: Human-readable summary
//   - X-AI-Trace-Events: Number of events recorded
//   - X-AI-Trace-Hash: Short hash for quick verification
//   - X-AI-Trace-Hint: Suggested next action
func SetTraceHeaders(c *gin.Context, h *TraceResponseHeaders) {
	if h.TraceID != "" {
		c.Header("X-Trace-Id", h.TraceID)
	}

	// Human-readable summary
	summary := buildSummary(h)
	if summary != "" {
		c.Header("X-AI-Trace-Summary", summary)
	}

	if h.EventCount > 0 {
		c.Header("X-AI-Trace-Events", fmt.Sprintf("%d", h.EventCount))
	}

	if h.PayloadHash != "" {
		// Short hash for display (first 8 chars)
		shortHash := h.PayloadHash
		if len(shortHash) > 8 {
			shortHash = shortHash[:8]
		}
		c.Header("X-AI-Trace-Hash", shortHash)
	}

	// Provide helpful hint for next action
	if h.EvidenceHint != "" {
		c.Header("X-AI-Trace-Hint", h.EvidenceHint)
	}
}

func buildSummary(h *TraceResponseHeaders) string {
	parts := []string{}

	if h.TraceID != "" {
		parts = append(parts, fmt.Sprintf("Traced[%s]", truncateID(h.TraceID)))
	}

	if h.EventCount > 0 {
		parts = append(parts, fmt.Sprintf("%d events", h.EventCount))
	}

	if h.PayloadHash != "" {
		parts = append(parts, fmt.Sprintf("hash:%s", truncateHash(h.PayloadHash)))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " | ")
}

func truncateID(id string) string {
	// trc_abc123def456 -> abc123
	if strings.HasPrefix(id, "trc_") {
		id = id[4:]
	}
	if len(id) > 6 {
		return id[:6]
	}
	return id
}

func truncateHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

// TraceHeadersMiddleware is a middleware that initializes trace context.
func TraceHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add informational headers about AI-Trace
		c.Header("X-Powered-By", "AI-Trace")
		c.Header("X-AI-Trace-Version", "0.2.0")

		c.Next()
	}
}

// CertificateSuggestionHeader adds a header suggesting certificate generation.
func CertificateSuggestionHeader(c *gin.Context, traceID string) {
	if traceID == "" {
		return
	}

	hint := BuildCertHint(traceID, "internal")
	c.Header("X-AI-Trace-Hint", hint)
}

// BuildCertHint creates a user-friendly, copy-paste ready curl command for certificate generation.
// This makes it easy for users to generate certificates without reading documentation.
func BuildCertHint(traceID, evidenceLevel string) string {
	if evidenceLevel == "" {
		evidenceLevel = "internal"
	}
	// Single-line curl command that's easy to copy
	return fmt.Sprintf(
		`curl -X POST "$AI_TRACE_URL/api/v1/certs/commit" -H "Content-Type: application/json" -H "X-API-Key: $AI_TRACE_KEY" -d '{"trace_id":"%s","evidence_level":"%s"}'`,
		traceID,
		evidenceLevel,
	)
}

// BuildCertHintWithURL creates a hint with explicit URL (for when we know the host).
func BuildCertHintWithURL(baseURL, traceID, evidenceLevel string) string {
	if evidenceLevel == "" {
		evidenceLevel = "internal"
	}
	if baseURL == "" {
		baseURL = "http://localhost:8006"
	}
	return fmt.Sprintf(
		`curl -X POST "%s/api/v1/certs/commit" -H "Content-Type: application/json" -H "X-API-Key: YOUR_KEY" -d '{"trace_id":"%s","evidence_level":"%s"}'`,
		baseURL,
		traceID,
		evidenceLevel,
	)
}

// EvidenceLevelDescriptions provides human-readable descriptions for evidence levels.
var EvidenceLevelDescriptions = map[string]string{
	"internal":   "Fast, Ed25519 signed, ideal for development and internal audit",
	"compliance": "WORM storage + TSA timestamp, for SOC2/GDPR/HIPAA compliance",
	"legal":      "Blockchain anchored, for legal disputes and court evidence",
}

// QuickStats represents quick statistics for headers.
type QuickStats struct {
	RequestsToday    int64
	CertsGenerated   int64
	LastCertTime     string
}

// SetStatsHeaders adds quick stats to response headers.
func SetStatsHeaders(c *gin.Context, stats *QuickStats) {
	if stats == nil {
		return
	}

	if stats.RequestsToday > 0 {
		c.Header("X-AI-Trace-Requests-Today", fmt.Sprintf("%d", stats.RequestsToday))
	}

	if stats.CertsGenerated > 0 {
		c.Header("X-AI-Trace-Certs-Total", fmt.Sprintf("%d", stats.CertsGenerated))
	}
}
