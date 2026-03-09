package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ai-trace/server/internal/fingerprint"
)

// TestFingerprintRecordSerialization 测试指纹记录序列化
func TestFingerprintRecordSerialization(t *testing.T) {
	tests := []struct {
		name   string
		record FingerprintRecord
	}{
		{
			name: "full fingerprint record",
			record: FingerprintRecord{
				ID:              1,
				TenantID:        "tenant-123",
				TraceID:         "trace-456",
				FingerprintID:   "fp_abc123",
				ModelID:         "gpt-4",
				ModelProvider:   "openai",
				FingerprintHash: "abc123hash",
				StatisticalFeatures: &fingerprint.StatisticalFeatures{
					TotalTokens:      100,
					PromptTokens:     20,
					CompletionTokens: 80,
					TokensPerSecond:  25.5,
					FirstTokenMs:     150,
					TotalLatencyMs:   4000,
					FinishReason:     "stop",
				},
				FullFingerprint: &fingerprint.InferenceFingerprint{
					Statistical: &fingerprint.StatisticalFeatures{
						TotalTokens:    100,
						TokensPerSecond: 25.5,
					},
				},
				Status:    "active",
				CreatedAt: time.Now(),
			},
		},
		{
			name: "minimal fingerprint record",
			record: FingerprintRecord{
				TenantID:        "tenant-min",
				TraceID:         "trace-min",
				FingerprintID:   "fp_minimal",
				ModelID:         "llama2",
				FingerprintHash: "minihash",
				Status:          "active",
			},
		},
		{
			name: "fingerprint with semantic layer",
			record: FingerprintRecord{
				TenantID:        "tenant-sem",
				TraceID:         "trace-sem",
				FingerprintID:   "fp_semantic",
				ModelID:         "gpt-4",
				FingerprintHash: "semhash",
				SemanticFeatures: &fingerprint.SemanticFeatures{
					TextComplexity:      65.5,
					VocabularyDiversity: 0.75,
					SentenceCount:       10,
					AvgSentenceLength:   15.5,
				},
				Status: "active",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			data, err := json.Marshal(tt.record)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Test JSON unmarshaling
			var decoded FingerprintRecord
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Verify key fields
			if decoded.TenantID != tt.record.TenantID {
				t.Errorf("TenantID mismatch: got %s, want %s", decoded.TenantID, tt.record.TenantID)
			}
			if decoded.TraceID != tt.record.TraceID {
				t.Errorf("TraceID mismatch: got %s, want %s", decoded.TraceID, tt.record.TraceID)
			}
			if decoded.FingerprintHash != tt.record.FingerprintHash {
				t.Errorf("FingerprintHash mismatch: got %s, want %s", decoded.FingerprintHash, tt.record.FingerprintHash)
			}
		})
	}
}

// TestFingerprintVerifyResult 测试验证结果结构
func TestFingerprintVerifyResult(t *testing.T) {
	tests := []struct {
		name   string
		result FingerprintVerifyResult
	}{
		{
			name: "valid result",
			result: FingerprintVerifyResult{
				TraceID:      "trace-123",
				Valid:        true,
				Message:      "Fingerprint integrity verified",
				StoredHash:   "abc123",
				ComputedHash: "abc123",
				ExpectedHash: "abc123",
			},
		},
		{
			name: "invalid result with mismatch",
			result: FingerprintVerifyResult{
				TraceID:      "trace-456",
				Valid:        false,
				Message:      "Fingerprint data has been tampered with",
				StoredHash:   "abc123",
				ComputedHash: "def456",
				ExpectedHash: "abc123",
			},
		},
		{
			name: "no expected hash",
			result: FingerprintVerifyResult{
				TraceID:      "trace-789",
				Valid:        true,
				Message:      "Fingerprint integrity verified",
				StoredHash:   "xyz789",
				ComputedHash: "xyz789",
				ExpectedHash: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded FingerprintVerifyResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Valid != tt.result.Valid {
				t.Errorf("Valid mismatch: got %v, want %v", decoded.Valid, tt.result.Valid)
			}
			if decoded.StoredHash != tt.result.StoredHash {
				t.Errorf("StoredHash mismatch: got %s, want %s", decoded.StoredHash, tt.result.StoredHash)
			}
			if decoded.TraceID != tt.result.TraceID {
				t.Errorf("TraceID mismatch: got %s, want %s", decoded.TraceID, tt.result.TraceID)
			}
		})
	}
}

// TestComputeFingerprintHash 测试指纹哈希计算
func TestComputeFingerprintHash(t *testing.T) {
	tests := []struct {
		name        string
		fingerprint *fingerprint.InferenceFingerprint
		wantEmpty   bool
	}{
		{
			name: "statistical fingerprint",
			fingerprint: &fingerprint.InferenceFingerprint{
				Statistical: &fingerprint.StatisticalFeatures{
					TotalTokens:     100,
					TokensPerSecond: 25.5,
				},
			},
			wantEmpty: false,
		},
		{
			name: "full fingerprint",
			fingerprint: &fingerprint.InferenceFingerprint{
				Statistical: &fingerprint.StatisticalFeatures{
					TotalTokens: 200,
				},
				Semantic: &fingerprint.SemanticFeatures{
					TextComplexity: 50.0,
				},
			},
			wantEmpty: false,
		},
		{
			name:        "nil fingerprint",
			fingerprint: nil,
			wantEmpty:   true,
		},
		{
			name:        "empty fingerprint",
			fingerprint: &fingerprint.InferenceFingerprint{},
			wantEmpty:   false, // Will still produce a hash of empty struct
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := ComputeFingerprintHash(tt.fingerprint)

			if tt.wantEmpty && hash != "" {
				t.Errorf("expected empty hash, got %s", hash)
			}
			if !tt.wantEmpty && hash == "" {
				t.Error("expected non-empty hash, got empty")
			}

			// Verify hash consistency
			if tt.fingerprint != nil {
				hash2 := ComputeFingerprintHash(tt.fingerprint)
				if hash != hash2 {
					t.Errorf("hash not consistent: %s vs %s", hash, hash2)
				}
			}
		})
	}
}

// TestComputeFingerprintHashDifferentInputs 测试不同输入产生不同哈希
func TestComputeFingerprintHashDifferentInputs(t *testing.T) {
	fp1 := &fingerprint.InferenceFingerprint{
		Statistical: &fingerprint.StatisticalFeatures{
			TotalTokens: 100,
		},
	}
	fp2 := &fingerprint.InferenceFingerprint{
		Statistical: &fingerprint.StatisticalFeatures{
			TotalTokens: 101,
		},
	}

	hash1 := ComputeFingerprintHash(fp1)
	hash2 := ComputeFingerprintHash(fp2)

	if hash1 == hash2 {
		t.Error("different fingerprints should produce different hashes")
	}
}

// TestFingerprintCompareResultFields 测试比较结果字段
func TestFingerprintCompareResultFields(t *testing.T) {
	result := FingerprintCompareResult{
		TraceID1:              "trace-1",
		TraceID2:              "trace-2",
		OverallSimilarity:     0.95,
		StatisticalSimilarity: 0.98,
		TokenProbSimilarity:   0.92,
		SemanticSimilarity:    0.90,
		Conclusion:            "highly_similar",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded FingerprintCompareResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.OverallSimilarity != result.OverallSimilarity {
		t.Errorf("OverallSimilarity mismatch: got %f, want %f", decoded.OverallSimilarity, result.OverallSimilarity)
	}
	if decoded.Conclusion != result.Conclusion {
		t.Errorf("Conclusion mismatch: got %s, want %s", decoded.Conclusion, result.Conclusion)
	}
}

// TestGetSimilarityConclusion 测试相似度结论生成
func TestGetSimilarityConclusion(t *testing.T) {
	tests := []struct {
		similarity float64
		want       string
	}{
		{0.99, "highly_similar"},
		{0.95, "highly_similar"},
		{0.85, "similar"},
		{0.80, "similar"},
		{0.70, "moderately_similar"},
		{0.60, "moderately_similar"},
		{0.50, "somewhat_different"},
		{0.40, "somewhat_different"},
		{0.30, "very_different"},
		{0.0, "very_different"},
	}

	for _, tt := range tests {
		got := getSimilarityConclusion(tt.similarity)
		if got != tt.want {
			t.Errorf("getSimilarityConclusion(%f) = %s, want %s", tt.similarity, got, tt.want)
		}
	}
}

// TestHelperFunctions 测试辅助函数
func TestHelperFunctions(t *testing.T) {
	// Test absFloat
	if absFloat(-5.5) != 5.5 {
		t.Error("absFloat(-5.5) should be 5.5")
	}
	if absFloat(3.3) != 3.3 {
		t.Error("absFloat(3.3) should be 3.3")
	}

	// Test minFloat
	if minFloat(3.0, 5.0) != 3.0 {
		t.Error("minFloat(3.0, 5.0) should be 3.0")
	}
	if minFloat(7.0, 2.0) != 2.0 {
		t.Error("minFloat(7.0, 2.0) should be 2.0")
	}

	// Test maxFloat
	if maxFloat(3.0, 5.0) != 5.0 {
		t.Error("maxFloat(3.0, 5.0) should be 5.0")
	}
	if maxFloat(7.0, 2.0) != 7.0 {
		t.Error("maxFloat(7.0, 2.0) should be 7.0")
	}
}

// TestCompareStatistical 测试统计特征比较
func TestCompareStatistical(t *testing.T) {
	// Test nil cases
	if compareStatistical(nil, nil) != 0 {
		t.Error("compareStatistical(nil, nil) should be 0")
	}
	if compareStatistical(&fingerprint.StatisticalFeatures{}, nil) != 0 {
		t.Error("compareStatistical with one nil should be 0")
	}

	// Test identical features
	a := &fingerprint.StatisticalFeatures{
		TokensPerSecond: 25.0,
		FirstTokenMs:    100,
	}
	b := &fingerprint.StatisticalFeatures{
		TokensPerSecond: 25.0,
		FirstTokenMs:    100,
	}
	similarity := compareStatistical(a, b)
	if similarity < 0.99 {
		t.Errorf("identical features should have similarity ~1.0, got %f", similarity)
	}

	// Test different features
	c := &fingerprint.StatisticalFeatures{
		TokensPerSecond: 50.0,
		FirstTokenMs:    200,
	}
	simDiff := compareStatistical(a, c)
	if simDiff >= similarity {
		t.Error("different features should have lower similarity")
	}
}

// TestCompareSemantic 测试语义特征比较
func TestCompareSemantic(t *testing.T) {
	// Test nil cases
	if compareSemantic(nil, nil) != 0 {
		t.Error("compareSemantic(nil, nil) should be 0")
	}

	// Test identical features
	a := &fingerprint.SemanticFeatures{
		VocabularyDiversity: 0.8,
		AvgSentenceLength:   15.0,
		TextComplexity:      60.0,
	}
	b := &fingerprint.SemanticFeatures{
		VocabularyDiversity: 0.8,
		AvgSentenceLength:   15.0,
		TextComplexity:      60.0,
	}
	similarity := compareSemantic(a, b)
	if similarity < 0.99 {
		t.Errorf("identical features should have similarity ~1.0, got %f", similarity)
	}
}

// TestCompareTokenProb 测试 Token 概率特征比较
func TestCompareTokenProb(t *testing.T) {
	// Test nil cases
	if compareTokenProb(nil, nil) != 0 {
		t.Error("compareTokenProb(nil, nil) should be 0")
	}

	// Test identical features
	a := &fingerprint.TokenProbFeatures{
		AvgLogProb:  -2.5,
		AvgEntropy:  1.5,
	}
	b := &fingerprint.TokenProbFeatures{
		AvgLogProb:  -2.5,
		AvgEntropy:  1.5,
	}
	similarity := compareTokenProb(a, b)
	if similarity < 0.99 {
		t.Errorf("identical features should have similarity ~1.0, got %f", similarity)
	}
}

// TestFingerprintRecordJSONTags 测试 JSON 标签正确性
func TestFingerprintRecordJSONTags(t *testing.T) {
	record := FingerprintRecord{
		ID:              1,
		FingerprintID:   "fp_test",
		TraceID:         "trace_test",
		TenantID:        "tenant_test",
		FingerprintHash: "hash_test",
		ModelID:         "model_test",
		ModelProvider:   "provider_test",
		Status:          "active",
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Check that JSON contains expected field names
	expectedFields := []string{
		`"fingerprint_id":"fp_test"`,
		`"trace_id":"trace_test"`,
		`"tenant_id":"tenant_test"`,
		`"fingerprint_hash":"hash_test"`,
		`"model_id":"model_test"`,
		`"model_provider":"provider_test"`,
	}

	for _, field := range expectedFields {
		found := false
		for i := 0; i <= len(jsonStr)-len(field); i++ {
			if jsonStr[i:i+len(field)] == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected JSON to contain %s", field)
		}
	}
}

// TestErrorDefinitions 测试错误定义
func TestFingerprintErrorDefinitions(t *testing.T) {
	if ErrFingerprintNotFound == nil {
		t.Error("ErrFingerprintNotFound should be defined")
	}
	if ErrFingerprintNotFound.Error() == "" {
		t.Error("ErrFingerprintNotFound should have error message")
	}
}
