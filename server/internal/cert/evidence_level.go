// Package cert provides certificate management functionality.
package cert

import "strings"

// EvidenceLevel represents the trust level of a certificate.
// We use business-friendly names internally but support legacy L1/L2/L3 for API compatibility.
type EvidenceLevel string

const (
	// EvidenceLevelInternal is for internal audit purposes.
	// Technical: Ed25519 signature only.
	// Use case: Development, testing, internal reviews.
	EvidenceLevelInternal EvidenceLevel = "internal"

	// EvidenceLevelCompliance is for regulatory compliance.
	// Technical: Ed25519 + WORM storage + TSA timestamp.
	// Use case: SOC2, GDPR, industry regulations.
	EvidenceLevelCompliance EvidenceLevel = "compliance"

	// EvidenceLevelLegal is for legal evidence.
	// Technical: Ed25519 + WORM + TSA + Blockchain anchoring.
	// Use case: Legal disputes, court evidence, contracts.
	EvidenceLevelLegal EvidenceLevel = "legal"
)

// EvidenceLevelInfo provides human-readable information about an evidence level.
type EvidenceLevelInfo struct {
	Level       EvidenceLevel `json:"level"`
	DisplayName string        `json:"display_name"`
	Description string        `json:"description"`
	Features    []string      `json:"features"`
	UseCases    []string      `json:"use_cases"`
	LegacyCode  string        `json:"legacy_code"` // L1, L2, L3 for backward compatibility
}

// AllEvidenceLevels returns information about all available evidence levels.
func AllEvidenceLevels() []EvidenceLevelInfo {
	return []EvidenceLevelInfo{
		{
			Level:       EvidenceLevelInternal,
			DisplayName: "Internal Audit",
			Description: "Basic digital signature for internal record-keeping",
			Features: []string{
				"Ed25519 digital signature",
				"Merkle tree verification",
				"Instant generation",
			},
			UseCases: []string{
				"Development and testing",
				"Internal audit trails",
				"Team reviews",
			},
			LegacyCode: "L1",
		},
		{
			Level:       EvidenceLevelCompliance,
			DisplayName: "Compliance Evidence",
			Description: "Tamper-proof storage with trusted timestamp",
			Features: []string{
				"All Internal features",
				"WORM (Write-Once-Read-Many) storage",
				"RFC 3161 trusted timestamp",
				"Immutable audit trail",
			},
			UseCases: []string{
				"SOC2 compliance",
				"GDPR data processing records",
				"Financial regulations",
				"Healthcare (HIPAA)",
			},
			LegacyCode: "L2",
		},
		{
			Level:       EvidenceLevelLegal,
			DisplayName: "Legal Evidence",
			Description: "Blockchain-anchored proof for legal proceedings",
			Features: []string{
				"All Compliance features",
				"Blockchain anchoring (Ethereum)",
				"Public verifiability",
				"Non-repudiation",
			},
			UseCases: []string{
				"Legal disputes",
				"Contract evidence",
				"Intellectual property",
				"Regulatory investigations",
			},
			LegacyCode: "L3",
		},
	}
}

// GetEvidenceLevelInfo returns information about a specific evidence level.
func GetEvidenceLevelInfo(level EvidenceLevel) *EvidenceLevelInfo {
	for _, info := range AllEvidenceLevels() {
		if info.Level == level {
			return &info
		}
	}
	return nil
}

// ParseEvidenceLevel converts a string to an EvidenceLevel, supporting both
// new business names and legacy L1/L2/L3 codes.
func ParseEvidenceLevel(s string) EvidenceLevel {
	s = strings.ToLower(strings.TrimSpace(s))

	// Support new business names
	switch s {
	case "internal", "audit", "basic":
		return EvidenceLevelInternal
	case "compliance", "regulatory", "standard":
		return EvidenceLevelCompliance
	case "legal", "court", "blockchain":
		return EvidenceLevelLegal
	}

	// Support legacy codes for backward compatibility
	switch s {
	case "l1":
		return EvidenceLevelInternal
	case "l2":
		return EvidenceLevelCompliance
	case "l3":
		return EvidenceLevelLegal
	}

	// Default to internal
	return EvidenceLevelInternal
}

// String returns the evidence level as a string.
func (e EvidenceLevel) String() string {
	return string(e)
}

// LegacyCode returns the legacy L1/L2/L3 code for backward compatibility.
func (e EvidenceLevel) LegacyCode() string {
	switch e {
	case EvidenceLevelInternal:
		return "L1"
	case EvidenceLevelCompliance:
		return "L2"
	case EvidenceLevelLegal:
		return "L3"
	default:
		return "L1"
	}
}

// DisplayName returns a human-readable display name.
func (e EvidenceLevel) DisplayName() string {
	switch e {
	case EvidenceLevelInternal:
		return "Internal Audit"
	case EvidenceLevelCompliance:
		return "Compliance Evidence"
	case EvidenceLevelLegal:
		return "Legal Evidence"
	default:
		return "Internal Audit"
	}
}

// SuggestLevel suggests an appropriate evidence level based on context.
func SuggestLevel(industry string, modelName string, contentType string) EvidenceLevel {
	// High-risk industries default to compliance
	highRiskIndustries := map[string]bool{
		"finance":    true,
		"healthcare": true,
		"legal":      true,
		"insurance":  true,
		"government": true,
	}

	if highRiskIndustries[strings.ToLower(industry)] {
		return EvidenceLevelCompliance
	}

	// High-capability models may warrant higher evidence
	highCapModels := map[string]bool{
		"gpt-4":               true,
		"gpt-4-turbo":         true,
		"claude-3-opus":       true,
		"claude-3.5-sonnet":   true,
	}

	if highCapModels[strings.ToLower(modelName)] {
		return EvidenceLevelCompliance
	}

	// Default to internal
	return EvidenceLevelInternal
}
