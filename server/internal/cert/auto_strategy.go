// Package cert provides automatic certificate generation strategies.
package cert

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AutoCertStrategy defines when certificates should be automatically generated.
type AutoCertStrategy struct {
	// Enabled controls whether auto-cert is active
	Enabled bool `json:"enabled" yaml:"enabled"`

	// DefaultLevel is the evidence level for auto-generated certs
	DefaultLevel EvidenceLevel `json:"default_level" yaml:"default_level"`

	// Triggers define when to auto-generate certificates
	Triggers []AutoCertTrigger `json:"triggers" yaml:"triggers"`

	// Schedule for batch certificate generation
	Schedule *AutoCertSchedule `json:"schedule,omitempty" yaml:"schedule,omitempty"`
}

// AutoCertTrigger defines a condition that triggers auto certificate generation.
type AutoCertTrigger struct {
	// Type of trigger: "model", "token_count", "content_pattern", "time_window"
	Type string `json:"type" yaml:"type"`

	// Condition is trigger-specific condition
	Condition AutoCertCondition `json:"condition" yaml:"condition"`

	// Level to use when this trigger fires (overrides default)
	Level EvidenceLevel `json:"level,omitempty" yaml:"level,omitempty"`

	// Description for logging/display
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// AutoCertCondition holds trigger-specific conditions.
type AutoCertCondition struct {
	// For "model" trigger: list of model names
	Models []string `json:"models,omitempty" yaml:"models,omitempty"`

	// For "token_count" trigger: minimum tokens
	MinTokens int `json:"min_tokens,omitempty" yaml:"min_tokens,omitempty"`

	// For "content_pattern" trigger: regex patterns to match
	Patterns []string `json:"patterns,omitempty" yaml:"patterns,omitempty"`

	// For "industry" trigger: industry codes
	Industries []string `json:"industries,omitempty" yaml:"industries,omitempty"`
}

// AutoCertSchedule defines scheduled batch certificate generation.
type AutoCertSchedule struct {
	// Interval: "hourly", "daily", "weekly"
	Interval string `json:"interval" yaml:"interval"`

	// Time of day (for daily/weekly): "02:00" (2 AM)
	TimeOfDay string `json:"time_of_day,omitempty" yaml:"time_of_day,omitempty"`

	// Day of week (for weekly): 0=Sunday, 1=Monday, ...
	DayOfWeek *int `json:"day_of_week,omitempty" yaml:"day_of_week,omitempty"`

	// MaxTraceAge: only cert traces newer than this
	MaxTraceAge string `json:"max_trace_age,omitempty" yaml:"max_trace_age,omitempty"`
}

// DefaultStrategy returns a sensible default auto-cert strategy.
func DefaultStrategy() *AutoCertStrategy {
	return &AutoCertStrategy{
		Enabled:      false, // Off by default, user must opt-in
		DefaultLevel: EvidenceLevelInternal,
		Triggers: []AutoCertTrigger{
			{
				Type:        "model",
				Description: "Auto-cert for high-capability models",
				Condition: AutoCertCondition{
					Models: []string{
						"gpt-4", "gpt-4-turbo", "gpt-4o",
						"claude-3-opus", "claude-3.5-sonnet",
						"deepseek-chat",
					},
				},
				Level: EvidenceLevelCompliance,
			},
			{
				Type:        "token_count",
				Description: "Auto-cert for large responses",
				Condition: AutoCertCondition{
					MinTokens: 2000,
				},
				Level: EvidenceLevelInternal,
			},
		},
		Schedule: &AutoCertSchedule{
			Interval:    "daily",
			TimeOfDay:   "02:00",
			MaxTraceAge: "24h",
		},
	}
}

// AutoCertEvaluator evaluates whether a trace should get an auto certificate.
type AutoCertEvaluator struct {
	strategy *AutoCertStrategy
	logger   *zap.SugaredLogger
	mu       sync.RWMutex
}

// NewAutoCertEvaluator creates a new evaluator with the given strategy.
func NewAutoCertEvaluator(strategy *AutoCertStrategy, logger *zap.SugaredLogger) *AutoCertEvaluator {
	if strategy == nil {
		strategy = DefaultStrategy()
	}
	return &AutoCertEvaluator{
		strategy: strategy,
		logger:   logger,
	}
}

// TraceContext holds context about a trace for evaluation.
type TraceContext struct {
	TraceID     string
	Model       string
	TokenCount  int
	Content     string
	Industry    string
	TenantID    string
	CreatedAt   time.Time
}

// EvaluationResult holds the result of auto-cert evaluation.
type EvaluationResult struct {
	ShouldCert    bool
	Level         EvidenceLevel
	TriggerReason string
}

// Evaluate checks if a trace should have an auto-generated certificate.
func (e *AutoCertEvaluator) Evaluate(ctx context.Context, tc *TraceContext) *EvaluationResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.strategy.Enabled {
		return &EvaluationResult{ShouldCert: false}
	}

	for _, trigger := range e.strategy.Triggers {
		if e.matchesTrigger(tc, &trigger) {
			level := trigger.Level
			if level == "" {
				level = e.strategy.DefaultLevel
			}

			return &EvaluationResult{
				ShouldCert:    true,
				Level:         level,
				TriggerReason: trigger.Description,
			}
		}
	}

	return &EvaluationResult{ShouldCert: false}
}

func (e *AutoCertEvaluator) matchesTrigger(tc *TraceContext, trigger *AutoCertTrigger) bool {
	switch trigger.Type {
	case "model":
		return e.matchesModel(tc.Model, trigger.Condition.Models)

	case "token_count":
		return tc.TokenCount >= trigger.Condition.MinTokens

	case "industry":
		return e.matchesIndustry(tc.Industry, trigger.Condition.Industries)

	case "content_pattern":
		return e.matchesPattern(tc.Content, trigger.Condition.Patterns)

	default:
		return false
	}
}

func (e *AutoCertEvaluator) matchesModel(model string, patterns []string) bool {
	modelLower := strings.ToLower(model)
	for _, pattern := range patterns {
		if strings.Contains(modelLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func (e *AutoCertEvaluator) matchesIndustry(industry string, industries []string) bool {
	industryLower := strings.ToLower(industry)
	for _, ind := range industries {
		if strings.EqualFold(industryLower, ind) {
			return true
		}
	}
	return false
}

func (e *AutoCertEvaluator) matchesPattern(content string, patterns []string) bool {
	contentLower := strings.ToLower(content)
	for _, pattern := range patterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// UpdateStrategy updates the auto-cert strategy.
func (e *AutoCertEvaluator) UpdateStrategy(strategy *AutoCertStrategy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strategy = strategy
}

// GetStrategy returns the current strategy.
func (e *AutoCertEvaluator) GetStrategy() *AutoCertStrategy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy
	strategyCopy := *e.strategy
	return &strategyCopy
}

// StrategyJSON returns the strategy as JSON for API responses.
func (e *AutoCertEvaluator) StrategyJSON() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return json.Marshal(e.strategy)
}

// ExampleStrategies returns example strategies for documentation.
func ExampleStrategies() map[string]*AutoCertStrategy {
	return map[string]*AutoCertStrategy{
		"minimal": {
			Enabled:      true,
			DefaultLevel: EvidenceLevelInternal,
			Triggers:     []AutoCertTrigger{},
			Schedule: &AutoCertSchedule{
				Interval:    "daily",
				TimeOfDay:   "03:00",
				MaxTraceAge: "24h",
			},
		},
		"compliance": {
			Enabled:      true,
			DefaultLevel: EvidenceLevelCompliance,
			Triggers: []AutoCertTrigger{
				{
					Type:        "model",
					Description: "All GPT-4 and Claude calls",
					Condition: AutoCertCondition{
						Models: []string{"gpt-4", "claude-3"},
					},
					Level: EvidenceLevelCompliance,
				},
			},
			Schedule: &AutoCertSchedule{
				Interval:    "hourly",
				MaxTraceAge: "1h",
			},
		},
		"strict": {
			Enabled:      true,
			DefaultLevel: EvidenceLevelCompliance,
			Triggers: []AutoCertTrigger{
				{
					Type:        "model",
					Description: "All AI calls",
					Condition: AutoCertCondition{
						Models: []string{"gpt", "claude", "llama", "deepseek"},
					},
					Level: EvidenceLevelCompliance,
				},
				{
					Type:        "token_count",
					Description: "Large responses",
					Condition: AutoCertCondition{
						MinTokens: 1000,
					},
					Level: EvidenceLevelCompliance,
				},
			},
		},
	}
}
