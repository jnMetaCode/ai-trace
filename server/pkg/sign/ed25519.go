package sign

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
)

// Signer Ed25519签名器
type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	mu         sync.RWMutex
}

var (
	defaultSigner *Signer
	once          sync.Once
)

// GetDefaultSigner 获取默认签名器（单例）
func GetDefaultSigner() (*Signer, error) {
	var initErr error
	once.Do(func() {
		defaultSigner, initErr = NewSigner()
	})
	return defaultSigner, initErr
}

// NewSigner 创建新的签名器
// 优先从环境变量读取私钥，否则生成新的
func NewSigner() (*Signer, error) {
	s := &Signer{}

	// 尝试从环境变量读取私钥
	privKeyHex := os.Getenv("SIGNING_PRIVATE_KEY")
	if privKeyHex != "" {
		privKey, err := hex.DecodeString(privKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid SIGNING_PRIVATE_KEY: %w", err)
		}
		if len(privKey) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("invalid private key size: expected %d, got %d", ed25519.PrivateKeySize, len(privKey))
		}
		s.privateKey = ed25519.PrivateKey(privKey)
		s.publicKey = s.privateKey.Public().(ed25519.PublicKey)
	} else {
		// 生成新的密钥对
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate key pair: %w", err)
		}
		s.privateKey = privKey
		s.publicKey = pubKey
	}

	return s, nil
}

// Sign 签名数据
func (s *Signer) Sign(data []byte) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.privateKey == nil {
		return "", fmt.Errorf("private key not initialized")
	}

	signature := ed25519.Sign(s.privateKey, data)
	return base64.StdEncoding.EncodeToString(signature), nil
}

// SignString 签名字符串
func (s *Signer) SignString(data string) (string, error) {
	return s.Sign([]byte(data))
}

// Verify 验证签名
func (s *Signer) Verify(data []byte, signatureB64 string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false, fmt.Errorf("invalid signature format: %w", err)
	}

	return ed25519.Verify(s.publicKey, data, signature), nil
}

// VerifyString 验证字符串签名
func (s *Signer) VerifyString(data, signatureB64 string) (bool, error) {
	return s.Verify([]byte(data), signatureB64)
}

// VerifyWithPublicKey 使用指定公钥验证签名
func VerifyWithPublicKey(pubKeyHex string, data []byte, signatureB64 string) (bool, error) {
	pubKey, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false, fmt.Errorf("invalid public key: %w", err)
	}
	if len(pubKey) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key size")
	}

	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false, fmt.Errorf("invalid signature format: %w", err)
	}

	return ed25519.Verify(ed25519.PublicKey(pubKey), data, signature), nil
}

// GetPublicKeyHex 获取公钥的十六进制表示
func (s *Signer) GetPublicKeyHex() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return hex.EncodeToString(s.publicKey)
}

// GetPrivateKeyHex 获取私钥的十六进制表示（用于备份）
func (s *Signer) GetPrivateKeyHex() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return hex.EncodeToString(s.privateKey)
}

// SignCertificate 签名证书数据
// 签名内容: cert_id + root_hash + evidence_level + timestamp
func (s *Signer) SignCertificate(certID, rootHash, evidenceLevel, timestamp string) (string, error) {
	data := fmt.Sprintf("%s|%s|%s|%s", certID, rootHash, evidenceLevel, timestamp)
	return s.SignString(data)
}

// VerifyCertificateSignature 验证证书签名
func (s *Signer) VerifyCertificateSignature(certID, rootHash, evidenceLevel, timestamp, signature string) (bool, error) {
	data := fmt.Sprintf("%s|%s|%s|%s", certID, rootHash, evidenceLevel, timestamp)
	return s.VerifyString(data, signature)
}
