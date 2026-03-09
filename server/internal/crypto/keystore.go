package crypto

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrKeyNotFound 密钥未找到
	ErrKeyNotFound = errors.New("key not found")
	// ErrKeyExpired 密钥已过期
	ErrKeyExpired = errors.New("key expired")
)

// KeyStore 密钥存储接口
type KeyStore interface {
	// GetTenantDEK 获取租户的数据加密密钥 (DEK)
	GetTenantDEK(ctx context.Context, tenantID string) ([]byte, error)

	// CreateTenantDEK 为租户创建新的 DEK
	CreateTenantDEK(ctx context.Context, tenantID string) ([]byte, error)

	// RotateKey 轮换租户密钥
	RotateKey(ctx context.Context, tenantID string) error

	// DeleteKey 删除租户密钥
	DeleteKey(ctx context.Context, tenantID string) error
}

// TenantKey 租户密钥信息
type TenantKey struct {
	TenantID    string    `json:"tenant_id"`
	KeyID       string    `json:"key_id"`
	DEK         []byte    `json:"-"`               // 加密后的 DEK（不序列化明文）
	EncryptedDEK string   `json:"encrypted_dek"`   // Base64 编码的加密 DEK
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	Version     int       `json:"version"`
}

// MemoryKeyStore 内存密钥存储（用于开发/测试）
type MemoryKeyStore struct {
	mu      sync.RWMutex
	keys    map[string]*TenantKey
	kek     []byte // Key Encryption Key（主密钥）
}

// NewMemoryKeyStore 创建内存密钥存储
// kek 是主密钥，用于加密存储的 DEK
func NewMemoryKeyStore(kek []byte) (*MemoryKeyStore, error) {
	if len(kek) != 32 {
		return nil, ErrInvalidKey
	}
	return &MemoryKeyStore{
		keys: make(map[string]*TenantKey),
		kek:  kek,
	}, nil
}

// NewMemoryKeyStoreFromString 从字符串创建内存密钥存储
func NewMemoryKeyStoreFromString(kekStr string) (*MemoryKeyStore, error) {
	encryptor, err := NewAES256GCMFromString(kekStr)
	if err != nil {
		return nil, err
	}
	return &MemoryKeyStore{
		keys: make(map[string]*TenantKey),
		kek:  encryptor.key,
	}, nil
}

// GetTenantDEK 获取租户的数据加密密钥
func (s *MemoryKeyStore) GetTenantDEK(ctx context.Context, tenantID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.keys[tenantID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// 检查是否过期
	if !key.ExpiresAt.IsZero() && time.Now().After(key.ExpiresAt) {
		return nil, ErrKeyExpired
	}

	return key.DEK, nil
}

// CreateTenantDEK 为租户创建新的 DEK
func (s *MemoryKeyStore) CreateTenantDEK(ctx context.Context, tenantID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在
	if _, exists := s.keys[tenantID]; exists {
		return s.keys[tenantID].DEK, nil
	}

	// 生成新的 DEK
	dek, err := GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}

	// 用 KEK 加密 DEK
	encryptor, _ := NewAES256GCM(s.kek)
	encryptedDEK, nonce, err := encryptor.Encrypt(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	// 存储
	keyID := fmt.Sprintf("key_%s_%d", tenantID[:8], time.Now().Unix())
	s.keys[tenantID] = &TenantKey{
		TenantID:     tenantID,
		KeyID:        keyID,
		DEK:          dek,
		EncryptedDEK: hex.EncodeToString(append(nonce, encryptedDEK...)),
		CreatedAt:    time.Now(),
		Version:      1,
	}

	return dek, nil
}

// GetOrCreateTenantDEK 获取或创建租户 DEK
func (s *MemoryKeyStore) GetOrCreateTenantDEK(ctx context.Context, tenantID string) ([]byte, error) {
	dek, err := s.GetTenantDEK(ctx, tenantID)
	if err == nil {
		return dek, nil
	}

	if errors.Is(err, ErrKeyNotFound) {
		return s.CreateTenantDEK(ctx, tenantID)
	}

	return nil, err
}

// RotateKey 轮换租户密钥
func (s *MemoryKeyStore) RotateKey(ctx context.Context, tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldKey, exists := s.keys[tenantID]
	if !exists {
		return ErrKeyNotFound
	}

	// 生成新的 DEK
	dek, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate new DEK: %w", err)
	}

	// 用 KEK 加密新 DEK
	encryptor, _ := NewAES256GCM(s.kek)
	encryptedDEK, nonce, err := encryptor.Encrypt(dek)
	if err != nil {
		return fmt.Errorf("failed to encrypt new DEK: %w", err)
	}

	// 更新（保留旧版本信息用于解密旧数据）
	keyID := fmt.Sprintf("key_%s_%d", tenantID[:8], time.Now().Unix())
	s.keys[tenantID] = &TenantKey{
		TenantID:     tenantID,
		KeyID:        keyID,
		DEK:          dek,
		EncryptedDEK: hex.EncodeToString(append(nonce, encryptedDEK...)),
		CreatedAt:    time.Now(),
		Version:      oldKey.Version + 1,
	}

	return nil
}

// DeleteKey 删除租户密钥
func (s *MemoryKeyStore) DeleteKey(ctx context.Context, tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[tenantID]; !exists {
		return ErrKeyNotFound
	}

	delete(s.keys, tenantID)
	return nil
}

// ListTenants 列出所有租户（用于管理）
func (s *MemoryKeyStore) ListTenants(ctx context.Context) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tenants := make([]string, 0, len(s.keys))
	for t := range s.keys {
		tenants = append(tenants, t)
	}
	return tenants
}

// GetKeyInfo 获取密钥信息（不含明文）
func (s *MemoryKeyStore) GetKeyInfo(ctx context.Context, tenantID string) (*TenantKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.keys[tenantID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// 返回不含明文 DEK 的副本
	return &TenantKey{
		TenantID:     key.TenantID,
		KeyID:        key.KeyID,
		EncryptedDEK: key.EncryptedDEK,
		CreatedAt:    key.CreatedAt,
		ExpiresAt:    key.ExpiresAt,
		Version:      key.Version,
	}, nil
}
