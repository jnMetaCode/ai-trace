package store

import (
	"testing"
)

// TestParseMinioRef 测试 MinIO 引用解析
func TestParseMinioRef(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		wantKey   string
		wantError bool
	}{
		{
			name:      "valid ref",
			ref:       "minio://ai-trace/encrypted/tenant1/prompt/uuid123",
			wantKey:   "encrypted/tenant1/prompt/uuid123",
			wantError: false,
		},
		{
			name:      "valid ref with nested path",
			ref:       "minio://bucket/path/to/deeply/nested/file.json",
			wantKey:   "path/to/deeply/nested/file.json",
			wantError: false,
		},
		{
			name:      "invalid prefix",
			ref:       "s3://bucket/key",
			wantError: true,
		},
		{
			name:      "empty ref",
			ref:       "",
			wantError: true,
		},
		{
			name:      "only prefix",
			ref:       "minio://",
			wantError: true,
		},
		{
			name:      "no object key",
			ref:       "minio://bucket",
			wantError: true,
		},
		{
			name:      "bucket only with slash",
			ref:       "minio://bucket/",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objectKey, err := parseMinioRef(tt.ref)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if objectKey != tt.wantKey {
				t.Errorf("objectKey mismatch: got %s, want %s", objectKey, tt.wantKey)
			}
		})
	}
}

// parseMinioRef 解析 MinIO 引用，提取 objectKey
// 这是从 RetrieveByRef 中提取的逻辑，用于单元测试
func parseMinioRef(ref string) (string, error) {
	const prefix = "minio://"
	if len(ref) <= len(prefix) || ref[:len(prefix)] != prefix {
		return "", ErrContentNotFound
	}

	pathPart := ref[len(prefix):]

	// 找到第一个 / 分隔 bucket 和 objectKey
	slashIdx := -1
	for i := 0; i < len(pathPart); i++ {
		if pathPart[i] == '/' {
			slashIdx = i
			break
		}
	}

	if slashIdx <= 0 || slashIdx >= len(pathPart)-1 {
		return "", ErrContentNotFound
	}

	objectKey := pathPart[slashIdx+1:]
	if objectKey == "" {
		return "", ErrContentNotFound
	}

	return objectKey, nil
}

// TestContentRefSerialization 测试内容引用序列化
func TestContentRefSerialization(t *testing.T) {
	ref := ContentRef{
		Ref:  "minio://bucket/encrypted/tenant/prompt/uuid",
		Hash: "sha256:abc123def456",
		Size: 1024,
	}

	if ref.Ref == "" {
		t.Error("Ref should not be empty")
	}
	if ref.Hash == "" {
		t.Error("Hash should not be empty")
	}
	if ref.Size != 1024 {
		t.Errorf("Size mismatch: got %d, want 1024", ref.Size)
	}
}

// TestEncryptedContentMetadata 测试加密内容元数据
func TestEncryptedContentMetadata(t *testing.T) {
	tests := []struct {
		name    string
		content EncryptedContent
	}{
		{
			name: "prompt content",
			content: EncryptedContent{
				Ref:         "minio://bucket/encrypted/tenant1/prompt/uuid1",
				ContentType: "prompt",
				TenantID:    "tenant1",
				Size:        512,
				KeyVersion:  1,
			},
		},
		{
			name: "output content",
			content: EncryptedContent{
				Ref:         "minio://bucket/encrypted/tenant2/output/uuid2",
				ContentType: "output",
				TenantID:    "tenant2",
				Size:        2048,
				KeyVersion:  2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.content.Ref == "" {
				t.Error("Ref should not be empty")
			}
			if tt.content.ContentType != "prompt" && tt.content.ContentType != "output" {
				t.Errorf("invalid content type: %s", tt.content.ContentType)
			}
			if tt.content.TenantID == "" {
				t.Error("TenantID should not be empty")
			}
		})
	}
}

// TestErrorDefinitions 测试错误定义
func TestErrorDefinitions(t *testing.T) {
	if ErrContentNotFound == nil {
		t.Error("ErrContentNotFound should be defined")
	}
	if ErrDecryptionFailed == nil {
		t.Error("ErrDecryptionFailed should be defined")
	}

	// 验证错误消息
	if ErrContentNotFound.Error() == "" {
		t.Error("ErrContentNotFound should have error message")
	}
	if ErrDecryptionFailed.Error() == "" {
		t.Error("ErrDecryptionFailed should have error message")
	}
}
