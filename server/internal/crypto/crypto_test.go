package crypto

import (
	"bytes"
	"context"
	"testing"
)

func TestAES256GCM_EncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	encryptor, err := NewAES256GCM(key)
	if err != nil {
		t.Fatalf("NewAES256GCM failed: %v", err)
	}

	plaintext := []byte("Hello, World! This is a test message for encryption.")

	// 加密
	ciphertext, nonce, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// 验证密文不同于明文
	if bytes.Equal(plaintext, ciphertext) {
		t.Error("Ciphertext should not equal plaintext")
	}

	// 解密
	decrypted, err := encryptor.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	// 验证解密结果
	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted text doesn't match original: got %s, want %s", decrypted, plaintext)
	}
}

func TestAES256GCM_Base64(t *testing.T) {
	encryptor, err := NewAES256GCMFromString("my-secret-key")
	if err != nil {
		t.Fatalf("NewAES256GCMFromString failed: %v", err)
	}

	plaintext := []byte("Test message for Base64 encoding")

	// 加密到 Base64
	ciphertextB64, nonceB64, err := encryptor.EncryptToBase64(plaintext)
	if err != nil {
		t.Fatalf("EncryptToBase64 failed: %v", err)
	}

	// 从 Base64 解密
	decrypted, err := encryptor.DecryptFromBase64(ciphertextB64, nonceB64)
	if err != nil {
		t.Fatalf("DecryptFromBase64 failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Decrypted text doesn't match original")
	}
}

func TestAES256GCM_InvalidKey(t *testing.T) {
	// 测试无效密钥长度
	_, err := NewAES256GCM([]byte("short-key"))
	if err != ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey, got %v", err)
	}
}

func TestAES256GCM_TamperedCiphertext(t *testing.T) {
	key, _ := GenerateKey()
	encryptor, _ := NewAES256GCM(key)

	plaintext := []byte("Original message")
	ciphertext, nonce, _ := encryptor.Encrypt(plaintext)

	// 篡改密文
	ciphertext[0] ^= 0xFF

	// 解密应该失败
	_, err := encryptor.Decrypt(ciphertext, nonce)
	if err != ErrDecryptionFailed {
		t.Errorf("Expected ErrDecryptionFailed, got %v", err)
	}
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// 两次生成的密钥应该不同
	if bytes.Equal(key1, key2) {
		t.Error("Generated keys should be different")
	}

	// 密钥长度应该是 32 字节
	if len(key1) != 32 {
		t.Errorf("Key length should be 32, got %d", len(key1))
	}
}

func TestGenerateKeyHex(t *testing.T) {
	hexKey, err := GenerateKeyHex()
	if err != nil {
		t.Fatalf("GenerateKeyHex failed: %v", err)
	}

	// Hex 编码的 32 字节应该是 64 字符
	if len(hexKey) != 64 {
		t.Errorf("Hex key length should be 64, got %d", len(hexKey))
	}

	// 应该能解析回密钥
	key, err := KeyFromHex(hexKey)
	if err != nil {
		t.Fatalf("KeyFromHex failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Parsed key length should be 32, got %d", len(key))
	}
}

func TestDeriveKey(t *testing.T) {
	masterKey, _ := GenerateKey()

	// 相同的 context 应该产生相同的派生密钥
	derived1 := DeriveKey(masterKey, "tenant-123")
	derived2 := DeriveKey(masterKey, "tenant-123")

	if !bytes.Equal(derived1, derived2) {
		t.Error("Same context should produce same derived key")
	}

	// 不同的 context 应该产生不同的派生密钥
	derived3 := DeriveKey(masterKey, "tenant-456")

	if bytes.Equal(derived1, derived3) {
		t.Error("Different context should produce different derived key")
	}

	// 派生密钥长度应该是 32 字节
	if len(derived1) != 32 {
		t.Errorf("Derived key length should be 32, got %d", len(derived1))
	}
}

func TestMemoryKeyStore(t *testing.T) {
	kek, _ := GenerateKey()
	keystore, err := NewMemoryKeyStore(kek)
	if err != nil {
		t.Fatalf("NewMemoryKeyStore failed: %v", err)
	}

	ctx := context.Background()
	tenantID := "test-tenant-123"

	// 获取不存在的密钥应该失败
	_, err = keystore.GetTenantDEK(ctx, tenantID)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}

	// 创建密钥
	dek1, err := keystore.CreateTenantDEK(ctx, tenantID)
	if err != nil {
		t.Fatalf("CreateTenantDEK failed: %v", err)
	}

	// 再次获取应该返回相同的密钥
	dek2, err := keystore.GetTenantDEK(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTenantDEK failed: %v", err)
	}

	if !bytes.Equal(dek1, dek2) {
		t.Error("Should return same DEK")
	}

	// 轮换密钥
	err = keystore.RotateKey(ctx, tenantID)
	if err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}

	// 新密钥应该不同
	dek3, _ := keystore.GetTenantDEK(ctx, tenantID)
	if bytes.Equal(dek1, dek3) {
		t.Error("Rotated key should be different")
	}

	// 删除密钥
	err = keystore.DeleteKey(ctx, tenantID)
	if err != nil {
		t.Fatalf("DeleteKey failed: %v", err)
	}

	// 删除后应该找不到
	_, err = keystore.GetTenantDEK(ctx, tenantID)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound after delete, got %v", err)
	}
}

func TestGetOrCreateTenantDEK(t *testing.T) {
	kek, _ := GenerateKey()
	keystore, _ := NewMemoryKeyStore(kek)

	ctx := context.Background()
	tenantID := "auto-create-tenant"

	// 第一次调用应该创建
	dek1, err := keystore.GetOrCreateTenantDEK(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetOrCreateTenantDEK failed: %v", err)
	}

	// 第二次调用应该返回相同的密钥
	dek2, err := keystore.GetOrCreateTenantDEK(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetOrCreateTenantDEK failed: %v", err)
	}

	if !bytes.Equal(dek1, dek2) {
		t.Error("Should return same DEK on second call")
	}
}

func TestKeyStoreFromString(t *testing.T) {
	keystore, err := NewMemoryKeyStoreFromString("my-master-password")
	if err != nil {
		t.Fatalf("NewMemoryKeyStoreFromString failed: %v", err)
	}

	ctx := context.Background()
	tenantID := "string-key-tenant"

	dek, err := keystore.CreateTenantDEK(ctx, tenantID)
	if err != nil {
		t.Fatalf("CreateTenantDEK failed: %v", err)
	}

	if len(dek) != 32 {
		t.Errorf("DEK length should be 32, got %d", len(dek))
	}
}

func TestEncryptDecryptWithKeyStore(t *testing.T) {
	kek, _ := GenerateKey()
	keystore, _ := NewMemoryKeyStore(kek)

	ctx := context.Background()
	tenantID := "encryption-test-tenant"

	// 获取 DEK
	dek, _ := keystore.GetOrCreateTenantDEK(ctx, tenantID)

	// 使用 DEK 加密
	encryptor, _ := NewAES256GCM(dek)

	originalData := []byte("Sensitive user content that needs encryption")
	ciphertext, nonce, _ := encryptor.Encrypt(originalData)

	// 模拟稍后检索和解密
	dek2, _ := keystore.GetTenantDEK(ctx, tenantID)
	encryptor2, _ := NewAES256GCM(dek2)

	decrypted, err := encryptor2.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(originalData, decrypted) {
		t.Error("Decrypted data doesn't match original")
	}
}
