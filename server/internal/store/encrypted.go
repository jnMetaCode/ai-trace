package store

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ai-trace/server/internal/crypto"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

var (
	// ErrContentNotFound 内容未找到
	ErrContentNotFound = errors.New("encrypted content not found")
	// ErrDecryptionFailed 解密失败
	ErrDecryptionFailed = errors.New("decryption failed")
)

// EncryptedStore 加密内容存储
type EncryptedStore struct {
	minio    *minio.Client
	bucket   string
	keystore crypto.KeyStore
}

// EncryptedContent 加密内容元数据
type EncryptedContent struct {
	Ref          string    `json:"ref"`           // 存储引用 (minio://bucket/key)
	ContentType  string    `json:"content_type"`  // 内容类型 (prompt/output)
	TenantID     string    `json:"tenant_id"`
	Hash         string    `json:"hash"`          // 原文哈希（用于验证）
	Size         int64     `json:"size"`          // 原文大小
	EncryptedAt  time.Time `json:"encrypted_at"`
	KeyVersion   int       `json:"key_version"`   // 使用的密钥版本
}

// NewEncryptedStore 创建加密存储
func NewEncryptedStore(minioClient *minio.Client, bucket string, keystore crypto.KeyStore) *EncryptedStore {
	return &EncryptedStore{
		minio:    minioClient,
		bucket:   bucket,
		keystore: keystore,
	}
}

// Store 加密并存储内容
func (s *EncryptedStore) Store(ctx context.Context, tenantID, contentType string, plaintext []byte) (*EncryptedContent, error) {
	// 获取或创建租户 DEK
	dek, err := s.keystore.(*crypto.MemoryKeyStore).GetOrCreateTenantDEK(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DEK: %w", err)
	}

	// 创建加密器
	encryptor, err := crypto.NewAES256GCM(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	// 加密
	ciphertext, nonce, err := encryptor.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	// 生成存储路径
	objectKey := fmt.Sprintf("encrypted/%s/%s/%s", tenantID, contentType, uuid.New().String())

	// 存储到 MinIO（nonce 存入 metadata）
	nonceB64 := base64.StdEncoding.EncodeToString(nonce)
	userMetadata := map[string]string{
		"nonce":        nonceB64,
		"content-type": contentType,
		"tenant-id":    tenantID,
	}

	reader := bytes.NewReader(ciphertext)
	_, err = s.minio.PutObject(ctx, s.bucket, objectKey, reader, int64(len(ciphertext)), minio.PutObjectOptions{
		UserMetadata: userMetadata,
		ContentType:  "application/octet-stream",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store to MinIO: %w", err)
	}

	// 返回元数据
	return &EncryptedContent{
		Ref:         fmt.Sprintf("minio://%s/%s", s.bucket, objectKey),
		ContentType: contentType,
		TenantID:    tenantID,
		Size:        int64(len(plaintext)),
		EncryptedAt: time.Now(),
	}, nil
}

// Retrieve 检索并解密内容
func (s *EncryptedStore) Retrieve(ctx context.Context, tenantID, objectKey string) ([]byte, error) {
	// 获取对象
	obj, err := s.minio.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	// 读取密文
	ciphertext, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read ciphertext: %w", err)
	}

	// 获取 metadata
	stat, err := obj.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get object stat: %w", err)
	}

	nonceB64 := stat.UserMetadata["Nonce"]
	if nonceB64 == "" {
		return nil, errors.New("nonce not found in metadata")
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	// 获取租户 DEK
	dek, err := s.keystore.(*crypto.MemoryKeyStore).GetOrCreateTenantDEK(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DEK: %w", err)
	}

	// 解密
	encryptor, _ := crypto.NewAES256GCM(dek)
	plaintext, err := encryptor.Decrypt(ciphertext, nonce)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// RetrieveByRef 通过引用检索内容
func (s *EncryptedStore) RetrieveByRef(ctx context.Context, tenantID, ref string) ([]byte, error) {
	// 解析 ref: minio://bucket/path
	// 格式示例: minio://ai-trace/encrypted/default/prompt/abc123
	const prefix = "minio://"
	if len(ref) <= len(prefix) || ref[:len(prefix)] != prefix {
		return nil, fmt.Errorf("invalid ref format: %s", ref)
	}

	// 去掉 minio:// 前缀
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
		return nil, fmt.Errorf("invalid ref format: %s", ref)
	}

	objectKey := pathPart[slashIdx+1:]
	if objectKey == "" {
		return nil, fmt.Errorf("invalid ref format: %s", ref)
	}

	return s.Retrieve(ctx, tenantID, objectKey)
}

// Delete 删除加密内容
func (s *EncryptedStore) Delete(ctx context.Context, objectKey string) error {
	return s.minio.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

// Exists 检查内容是否存在
func (s *EncryptedStore) Exists(ctx context.Context, objectKey string) (bool, error) {
	_, err := s.minio.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ContentRef 内容引用（用于存证记录）
type ContentRef struct {
	Ref      string `json:"ref"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	Nonce    string `json:"nonce,omitempty"` // 仅内部使用
}

// StorePrompt 存储加密的 prompt
func (s *EncryptedStore) StorePrompt(ctx context.Context, tenantID string, prompt []byte, hash string) (*ContentRef, error) {
	content, err := s.Store(ctx, tenantID, "prompt", prompt)
	if err != nil {
		return nil, err
	}

	return &ContentRef{
		Ref:  content.Ref,
		Hash: hash,
		Size: content.Size,
	}, nil
}

// StoreOutput 存储加密的输出
func (s *EncryptedStore) StoreOutput(ctx context.Context, tenantID string, output []byte, hash string) (*ContentRef, error) {
	content, err := s.Store(ctx, tenantID, "output", output)
	if err != nil {
		return nil, err
	}

	return &ContentRef{
		Ref:  content.Ref,
		Hash: hash,
		Size: content.Size,
	}, nil
}
