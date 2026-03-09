// Package crypto 提供内容加密能力
// 支持 AES-256-GCM 加密，用于保护用户敏感内容
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidKey 无效密钥
	ErrInvalidKey = errors.New("invalid encryption key")
	// ErrInvalidCiphertext 无效密文
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrDecryptionFailed 解密失败
	ErrDecryptionFailed = errors.New("decryption failed")
)

// Encryptor 加密器接口
type Encryptor interface {
	// Encrypt 加密数据
	Encrypt(plaintext []byte) (ciphertext []byte, nonce []byte, err error)
	// Decrypt 解密数据
	Decrypt(ciphertext, nonce []byte) (plaintext []byte, err error)
}

// AES256GCM AES-256-GCM 加密器
type AES256GCM struct {
	key []byte
}

// NewAES256GCM 创建 AES-256-GCM 加密器
// key 必须是 32 字节（256 位）
func NewAES256GCM(key []byte) (*AES256GCM, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	return &AES256GCM{key: key}, nil
}

// NewAES256GCMFromString 从字符串密钥创建加密器
// 会对输入进行 SHA-256 哈希以生成 32 字节密钥
func NewAES256GCMFromString(keyStr string) (*AES256GCM, error) {
	hash := sha256.Sum256([]byte(keyStr))
	return NewAES256GCM(hash[:])
}

// Encrypt 加密数据
func (e *AES256GCM) Encrypt(plaintext []byte) (ciphertext []byte, nonce []byte, err error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)

	return ciphertext, nonce, nil
}

// Decrypt 解密数据
func (e *AES256GCM) Decrypt(ciphertext, nonce []byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(nonce) != gcm.NonceSize() {
		return nil, ErrInvalidCiphertext
	}

	plaintext, err = gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptToBase64 加密并返回 Base64 编码
func (e *AES256GCM) EncryptToBase64(plaintext []byte) (ciphertextB64, nonceB64 string, err error) {
	ciphertext, nonce, err := e.Encrypt(plaintext)
	if err != nil {
		return "", "", err
	}

	ciphertextB64 = base64.StdEncoding.EncodeToString(ciphertext)
	nonceB64 = base64.StdEncoding.EncodeToString(nonce)

	return ciphertextB64, nonceB64, nil
}

// DecryptFromBase64 从 Base64 解密
func (e *AES256GCM) DecryptFromBase64(ciphertextB64, nonceB64 string) (plaintext []byte, err error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	return e.Decrypt(ciphertext, nonce)
}

// GenerateKey 生成随机 256 位密钥
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyHex 生成随机密钥并返回 hex 编码
func GenerateKeyHex() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// KeyFromHex 从 hex 字符串解析密钥
func KeyFromHex(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	return key, nil
}

// DeriveKey 从主密钥和上下文派生子密钥
// 使用 SHA-256(masterKey || context) 派生
func DeriveKey(masterKey []byte, context string) []byte {
	data := append(masterKey, []byte(context)...)
	hash := sha256.Sum256(data)
	return hash[:]
}
