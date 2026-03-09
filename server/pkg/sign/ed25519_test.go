package sign

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"os"
	"testing"
)

// TestNewSigner 测试签名器创建
func TestNewSigner(t *testing.T) {
	// 清除可能存在的环境变量
	os.Unsetenv("SIGNING_PRIVATE_KEY")

	signer, err := NewSigner()
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	if signer.privateKey == nil {
		t.Error("private key should not be nil")
	}
	if signer.publicKey == nil {
		t.Error("public key should not be nil")
	}
	if len(signer.privateKey) != ed25519.PrivateKeySize {
		t.Errorf("private key size: got %d, want %d", len(signer.privateKey), ed25519.PrivateKeySize)
	}
	if len(signer.publicKey) != ed25519.PublicKeySize {
		t.Errorf("public key size: got %d, want %d", len(signer.publicKey), ed25519.PublicKeySize)
	}
}

// TestNewSignerWithEnvKey 测试从环境变量加载密钥
func TestNewSignerWithEnvKey(t *testing.T) {
	// 生成一个测试密钥
	_, privKey, _ := ed25519.GenerateKey(nil)
	privKeyHex := hex.EncodeToString(privKey)

	// 设置环境变量
	os.Setenv("SIGNING_PRIVATE_KEY", privKeyHex)
	defer os.Unsetenv("SIGNING_PRIVATE_KEY")

	signer, err := NewSigner()
	if err != nil {
		t.Fatalf("failed to create signer with env key: %v", err)
	}

	// 验证加载的密钥是否正确
	if hex.EncodeToString(signer.privateKey) != privKeyHex {
		t.Error("private key mismatch")
	}
}

// TestNewSignerInvalidEnvKey 测试无效的环境变量密钥
func TestNewSignerInvalidEnvKey(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
	}{
		{"invalid hex", "not-valid-hex"},
		{"wrong size", "abcd1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SIGNING_PRIVATE_KEY", tt.envVal)
			defer os.Unsetenv("SIGNING_PRIVATE_KEY")

			_, err := NewSigner()
			if err == nil {
				t.Error("expected error for invalid key")
			}
		})
	}
}

// TestSignAndVerify 测试签名和验证
func TestSignAndVerify(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer, _ := NewSigner()

	testData := []byte("hello world")

	// 签名
	signature, err := signer.Sign(testData)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	if signature == "" {
		t.Error("signature should not be empty")
	}

	// 验证 base64 格式
	_, err = base64.StdEncoding.DecodeString(signature)
	if err != nil {
		t.Errorf("signature is not valid base64: %v", err)
	}

	// 验证签名
	valid, err := signer.Verify(testData, signature)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}
	if !valid {
		t.Error("signature should be valid")
	}

	// 使用错误数据验证
	wrongData := []byte("different data")
	valid, err = signer.Verify(wrongData, signature)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}
	if valid {
		t.Error("signature should be invalid for different data")
	}
}

// TestSignString 测试字符串签名
func TestSignString(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer, _ := NewSigner()

	testStr := "test string data"

	signature, err := signer.SignString(testStr)
	if err != nil {
		t.Fatalf("failed to sign string: %v", err)
	}

	valid, err := signer.VerifyString(testStr, signature)
	if err != nil {
		t.Fatalf("failed to verify string: %v", err)
	}
	if !valid {
		t.Error("string signature should be valid")
	}
}

// TestVerifyInvalidSignature 测试无效签名
func TestVerifyInvalidSignature(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer, _ := NewSigner()

	tests := []struct {
		name      string
		signature string
		wantErr   bool
		wantValid bool
	}{
		{"invalid base64", "not-base64!!!", true, false},
		{"empty signature", "", false, false},           // empty string is valid base64, decodes to empty bytes
		{"valid base64 wrong sig", base64.StdEncoding.EncodeToString([]byte("wrong")), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := signer.Verify([]byte("data"), tt.signature)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && valid != tt.wantValid {
				t.Errorf("valid: got %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

// TestVerifyWithPublicKey 测试使用公钥验证
func TestVerifyWithPublicKey(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer, _ := NewSigner()

	testData := []byte("test data for verification")
	signature, _ := signer.Sign(testData)
	pubKeyHex := signer.GetPublicKeyHex()

	// 使用公钥验证
	valid, err := VerifyWithPublicKey(pubKeyHex, testData, signature)
	if err != nil {
		t.Fatalf("failed to verify with public key: %v", err)
	}
	if !valid {
		t.Error("signature should be valid")
	}
}

// TestVerifyWithPublicKeyInvalid 测试无效公钥
func TestVerifyWithPublicKeyInvalid(t *testing.T) {
	tests := []struct {
		name      string
		pubKey    string
		data      []byte
		signature string
		wantErr   bool
	}{
		{
			name:      "invalid hex public key",
			pubKey:    "not-valid-hex",
			data:      []byte("data"),
			signature: base64.StdEncoding.EncodeToString([]byte("sig")),
			wantErr:   true,
		},
		{
			name:      "wrong size public key",
			pubKey:    "abcd1234",
			data:      []byte("data"),
			signature: base64.StdEncoding.EncodeToString([]byte("sig")),
			wantErr:   true,
		},
		{
			name:      "invalid signature format",
			pubKey:    hex.EncodeToString(make([]byte, ed25519.PublicKeySize)),
			data:      []byte("data"),
			signature: "not-base64!!!",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyWithPublicKey(tt.pubKey, tt.data, tt.signature)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
		})
	}
}

// TestGetPublicKeyHex 测试获取公钥
func TestGetPublicKeyHex(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer, _ := NewSigner()

	pubKeyHex := signer.GetPublicKeyHex()

	if pubKeyHex == "" {
		t.Error("public key hex should not be empty")
	}

	// 验证是有效的 hex
	decoded, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		t.Errorf("public key is not valid hex: %v", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		t.Errorf("decoded public key size: got %d, want %d", len(decoded), ed25519.PublicKeySize)
	}
}

// TestGetPrivateKeyHex 测试获取私钥
func TestGetPrivateKeyHex(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer, _ := NewSigner()

	privKeyHex := signer.GetPrivateKeyHex()

	if privKeyHex == "" {
		t.Error("private key hex should not be empty")
	}

	decoded, err := hex.DecodeString(privKeyHex)
	if err != nil {
		t.Errorf("private key is not valid hex: %v", err)
	}
	if len(decoded) != ed25519.PrivateKeySize {
		t.Errorf("decoded private key size: got %d, want %d", len(decoded), ed25519.PrivateKeySize)
	}
}

// TestSignCertificate 测试证书签名
func TestSignCertificate(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer, _ := NewSigner()

	certID := "cert_abc123"
	rootHash := "sha256:deadbeef"
	evidenceLevel := "L2"
	timestamp := "2024-01-15T10:30:00Z"

	// 签名证书
	signature, err := signer.SignCertificate(certID, rootHash, evidenceLevel, timestamp)
	if err != nil {
		t.Fatalf("failed to sign certificate: %v", err)
	}

	if signature == "" {
		t.Error("certificate signature should not be empty")
	}

	// 验证证书签名
	valid, err := signer.VerifyCertificateSignature(certID, rootHash, evidenceLevel, timestamp, signature)
	if err != nil {
		t.Fatalf("failed to verify certificate signature: %v", err)
	}
	if !valid {
		t.Error("certificate signature should be valid")
	}
}

// TestSignCertificateWrongParams 测试错误参数的证书签名验证
func TestSignCertificateWrongParams(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer, _ := NewSigner()

	certID := "cert_abc123"
	rootHash := "sha256:deadbeef"
	evidenceLevel := "L2"
	timestamp := "2024-01-15T10:30:00Z"

	signature, _ := signer.SignCertificate(certID, rootHash, evidenceLevel, timestamp)

	// 使用错误的 certID 验证
	valid, _ := signer.VerifyCertificateSignature("wrong_cert", rootHash, evidenceLevel, timestamp, signature)
	if valid {
		t.Error("signature should be invalid for wrong certID")
	}

	// 使用错误的 rootHash 验证
	valid, _ = signer.VerifyCertificateSignature(certID, "wrong_hash", evidenceLevel, timestamp, signature)
	if valid {
		t.Error("signature should be invalid for wrong rootHash")
	}

	// 使用错误的 evidenceLevel 验证
	valid, _ = signer.VerifyCertificateSignature(certID, rootHash, "L1", timestamp, signature)
	if valid {
		t.Error("signature should be invalid for wrong evidenceLevel")
	}

	// 使用错误的 timestamp 验证
	valid, _ = signer.VerifyCertificateSignature(certID, rootHash, evidenceLevel, "wrong_time", signature)
	if valid {
		t.Error("signature should be invalid for wrong timestamp")
	}
}

// TestSignerKeyConsistency 测试密钥一致性
func TestSignerKeyConsistency(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer, _ := NewSigner()

	// 导出密钥
	privKeyHex := signer.GetPrivateKeyHex()

	// 使用导出的密钥创建新的签名器
	os.Setenv("SIGNING_PRIVATE_KEY", privKeyHex)
	defer os.Unsetenv("SIGNING_PRIVATE_KEY")

	signer2, err := NewSigner()
	if err != nil {
		t.Fatalf("failed to create signer from exported key: %v", err)
	}

	// 两个签名器应该产生相同的签名
	data := []byte("test consistency")
	sig1, _ := signer.Sign(data)
	sig2, _ := signer2.Sign(data)

	if sig1 != sig2 {
		t.Error("signatures from same key should be identical")
	}

	// 公钥应该相同
	if signer.GetPublicKeyHex() != signer2.GetPublicKeyHex() {
		t.Error("public keys should match")
	}
}

// TestCrossVerification 测试交叉验证
func TestCrossVerification(t *testing.T) {
	os.Unsetenv("SIGNING_PRIVATE_KEY")
	signer1, _ := NewSigner()
	signer2, _ := NewSigner()

	data := []byte("test cross verification")

	// signer1 签名
	sig1, _ := signer1.Sign(data)

	// signer2 验证 signer1 的签名应该失败
	valid, _ := signer2.Verify(data, sig1)
	if valid {
		t.Error("different signer should not verify another's signature")
	}

	// 使用 signer1 的公钥验证应该成功
	valid, _ = VerifyWithPublicKey(signer1.GetPublicKeyHex(), data, sig1)
	if !valid {
		t.Error("verification with correct public key should succeed")
	}
}
