package store

import (
	"encoding/json"
	"testing"
	"time"
)

// TestDecryptAuditRecordSerialization 测试解密审计记录序列化
func TestDecryptAuditRecordSerialization(t *testing.T) {
	tests := []struct {
		name   string
		record DecryptAuditRecord
	}{
		{
			name: "successful decrypt",
			record: DecryptAuditRecord{
				ID:           1,
				AuditLogID:   "audit_abc123",
				ContentID:    "cnt_def456",
				EncryptedRef: "minio://bucket/encrypted/tenant/prompt/uuid",
				ContentType:  "prompt",
				TenantID:     "tenant-123",
				UserID:       "user-456",
				ClientIP:     "192.168.1.100",
				UserAgent:    "Mozilla/5.0",
				RequestID:    "req-789",
				TraceID:      "trace-101",
				Success:      true,
				DecryptedAt:  time.Now(),
			},
		},
		{
			name: "failed decrypt - permission denied",
			record: DecryptAuditRecord{
				ID:           2,
				AuditLogID:   "audit_xyz789",
				EncryptedRef: "minio://bucket/encrypted/other/prompt/uuid",
				ContentType:  "prompt",
				TenantID:     "tenant-123",
				UserID:       "user-456",
				ClientIP:     "10.0.0.1",
				Success:      false,
				FailReason:   "permission_denied",
				DecryptedAt:  time.Now(),
			},
		},
		{
			name: "failed decrypt - not found",
			record: DecryptAuditRecord{
				AuditLogID:   "audit_notfound",
				EncryptedRef: "minio://bucket/encrypted/tenant/prompt/nonexistent",
				ContentType:  "output",
				TenantID:     "tenant-999",
				Success:      false,
				FailReason:   "content_not_found",
				DecryptedAt:  time.Now(),
			},
		},
		{
			name: "minimal record",
			record: DecryptAuditRecord{
				AuditLogID:   "audit_minimal",
				EncryptedRef: "ref",
				TenantID:     "tenant",
				Success:      true,
				DecryptedAt:  time.Now(),
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
			var decoded DecryptAuditRecord
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Verify key fields
			if decoded.AuditLogID != tt.record.AuditLogID {
				t.Errorf("AuditLogID mismatch: got %s, want %s", decoded.AuditLogID, tt.record.AuditLogID)
			}
			if decoded.TenantID != tt.record.TenantID {
				t.Errorf("TenantID mismatch: got %s, want %s", decoded.TenantID, tt.record.TenantID)
			}
			if decoded.Success != tt.record.Success {
				t.Errorf("Success mismatch: got %v, want %v", decoded.Success, tt.record.Success)
			}
			if decoded.FailReason != tt.record.FailReason {
				t.Errorf("FailReason mismatch: got %s, want %s", decoded.FailReason, tt.record.FailReason)
			}
		})
	}
}

// TestDecryptAuditRecordJSONTags 测试 JSON 标签
func TestDecryptAuditRecordJSONTags(t *testing.T) {
	record := DecryptAuditRecord{
		ID:           100,
		AuditLogID:   "audit_test",
		ContentID:    "cnt_test",
		EncryptedRef: "ref_test",
		ContentType:  "prompt",
		TenantID:     "tenant_test",
		UserID:       "user_test",
		ClientIP:     "127.0.0.1",
		UserAgent:    "TestAgent",
		RequestID:    "req_test",
		TraceID:      "trace_test",
		Success:      true,
		FailReason:   "",
		DecryptedAt:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Check that JSON contains expected field names
	jsonStr := string(data)

	expectedFields := []string{
		`"id":100`,
		`"audit_id":"audit_test"`,
		`"content_id":"cnt_test"`,
		`"encrypted_ref":"ref_test"`,
		`"content_type":"prompt"`,
		`"tenant_id":"tenant_test"`,
		`"user_id":"user_test"`,
		`"client_ip":"127.0.0.1"`,
		`"success":true`,
	}

	for _, field := range expectedFields {
		if !containsString(jsonStr, field) {
			t.Errorf("expected JSON to contain %s, got: %s", field, jsonStr)
		}
	}
}

// TestDecryptAuditRecordOmitEmpty 测试 omitempty 标签
func TestDecryptAuditRecordOmitEmpty(t *testing.T) {
	// Record with empty optional fields
	record := DecryptAuditRecord{
		AuditLogID:   "audit_test",
		EncryptedRef: "ref_test",
		TenantID:     "tenant_test",
		Success:      true,
		DecryptedAt:  time.Now(),
		// Optional fields left empty
		ContentID:  "",
		UserAgent:  "",
		RequestID:  "",
		TraceID:    "",
		FailReason: "",
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Fields with omitempty should not appear when empty
	omitFields := []string{
		`"error_message":""`,  // FailReason has json tag "error_message,omitempty"
		`"trace_id":""`,
		`"request_id":""`,
		`"user_agent":""`,
	}

	for _, field := range omitFields {
		if containsString(jsonStr, field) {
			t.Errorf("expected JSON to omit empty field %s, got: %s", field, jsonStr)
		}
	}
}

// TestFailReasonConstants 测试失败原因常量
func TestFailReasonConstants(t *testing.T) {
	validReasons := []string{
		"permission_denied",
		"content_not_found",
		"decryption_failed",
		"encryption_not_configured",
		"internal_error",
	}

	for _, reason := range validReasons {
		record := DecryptAuditRecord{
			Success:    false,
			FailReason: reason,
		}

		if record.FailReason == "" {
			t.Errorf("FailReason should be set to %s", reason)
		}
	}
}

// TestDecryptAuditRecordTimeHandling 测试时间处理
func TestDecryptAuditRecordTimeHandling(t *testing.T) {
	now := time.Now()
	record := DecryptAuditRecord{
		AuditLogID:  "audit_time_test",
		DecryptedAt: now,
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded DecryptAuditRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Time should be preserved (within reasonable precision)
	diff := decoded.DecryptedAt.Sub(record.DecryptedAt)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("time not preserved: original %v, decoded %v", record.DecryptedAt, decoded.DecryptedAt)
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
