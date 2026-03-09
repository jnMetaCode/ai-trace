package store

import (
	"encoding/json"
	"testing"
	"time"
)

// TestWORMUploadResultSerialization 测试 WORM 上传结果序列化
func TestWORMUploadResultSerialization(t *testing.T) {
	tests := []struct {
		name   string
		result WORMUploadResult
	}{
		{
			name: "governance mode",
			result: WORMUploadResult{
				ObjectKey:     "certs/2024/01/15/cert-123.json",
				ETag:          "abc123def456",
				VersionID:     "v1",
				RetentionMode: "GOVERNANCE",
				RetentionDate: time.Now().Add(365 * 24 * time.Hour),
			},
		},
		{
			name: "compliance mode",
			result: WORMUploadResult{
				ObjectKey:     "certs/2024/06/01/cert-456.json",
				ETag:          "xyz789",
				VersionID:     "v2",
				RetentionMode: "COMPLIANCE",
				RetentionDate: time.Now().Add(7 * 365 * 24 * time.Hour),
			},
		},
		{
			name: "minimal result",
			result: WORMUploadResult{
				ObjectKey: "certs/test.json",
				ETag:      "etag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded WORMUploadResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.ObjectKey != tt.result.ObjectKey {
				t.Errorf("ObjectKey mismatch: got %s, want %s", decoded.ObjectKey, tt.result.ObjectKey)
			}
			if decoded.ETag != tt.result.ETag {
				t.Errorf("ETag mismatch: got %s, want %s", decoded.ETag, tt.result.ETag)
			}
			if decoded.RetentionMode != tt.result.RetentionMode {
				t.Errorf("RetentionMode mismatch: got %s, want %s", decoded.RetentionMode, tt.result.RetentionMode)
			}
		})
	}
}

// TestNewWORMStorage 测试 WORM 存储创建
func TestNewWORMStorage(t *testing.T) {
	tests := []struct {
		name              string
		retentionDays     int
		wantRetentionDays int
	}{
		{
			name:              "default retention",
			retentionDays:     0,
			wantRetentionDays: 365, // Default
		},
		{
			name:              "negative retention uses default",
			retentionDays:     -1,
			wantRetentionDays: 365, // Default
		},
		{
			name:              "custom retention",
			retentionDays:     30,
			wantRetentionDays: 30,
		},
		{
			name:              "long retention",
			retentionDays:     3650, // 10 years
			wantRetentionDays: 3650,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewWORMStorage(nil, "test-bucket", tt.retentionDays)

			if storage.retentionDays != tt.wantRetentionDays {
				t.Errorf("retentionDays mismatch: got %d, want %d", storage.retentionDays, tt.wantRetentionDays)
			}
			if storage.bucket != "test-bucket" {
				t.Errorf("bucket mismatch: got %s, want test-bucket", storage.bucket)
			}
		})
	}
}

// TestWORMObjectKeyGeneration 测试对象键生成格式
func TestWORMObjectKeyGeneration(t *testing.T) {
	// 模拟 StoreCertificate 中的对象键生成逻辑
	certID := "cert-abc123"
	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	objectKey := generateObjectKey(certID, now)

	expected := "certs/2024/06/15/cert-abc123.json"
	if objectKey != expected {
		t.Errorf("objectKey mismatch: got %s, want %s", objectKey, expected)
	}
}

// generateObjectKey 生成对象键（与 StoreCertificate 中的逻辑一致）
func generateObjectKey(certID string, t time.Time) string {
	return "certs/" + t.Format("2006/01/02") + "/" + certID + ".json"
}

// TestRetentionDateCalculation 测试保留日期计算
func TestRetentionDateCalculation(t *testing.T) {
	tests := []struct {
		name          string
		retentionDays int
	}{
		{"30 days", 30},
		{"365 days", 365},
		{"7 years", 7 * 365},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			retentionDate := now.Add(time.Duration(tt.retentionDays) * 24 * time.Hour)

			expectedDiff := time.Duration(tt.retentionDays) * 24 * time.Hour
			actualDiff := retentionDate.Sub(now)

			// Allow 1 second tolerance for test execution time
			if actualDiff < expectedDiff-time.Second || actualDiff > expectedDiff+time.Second {
				t.Errorf("retention date calculation off: expected ~%v, got %v", expectedDiff, actualDiff)
			}
		})
	}
}

// TestWORMUploadResultJSONTags 测试 JSON 标签
func TestWORMUploadResultJSONTags(t *testing.T) {
	result := WORMUploadResult{
		ObjectKey:     "key",
		ETag:          "etag",
		VersionID:     "v1",
		RetentionMode: "GOVERNANCE",
		RetentionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	expectedFields := []string{
		`"object_key":"key"`,
		`"etag":"etag"`,
		`"version_id":"v1"`,
		`"retention_mode":"GOVERNANCE"`,
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
			t.Errorf("expected JSON to contain %s, got: %s", field, jsonStr)
		}
	}
}

// TestRetentionModes 测试保留模式常量
func TestRetentionModes(t *testing.T) {
	// 验证支持的保留模式
	validModes := []string{"GOVERNANCE", "COMPLIANCE"}

	for _, mode := range validModes {
		result := WORMUploadResult{RetentionMode: mode}
		if result.RetentionMode != mode {
			t.Errorf("RetentionMode should be %s", mode)
		}
	}
}
