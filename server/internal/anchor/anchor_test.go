package anchor

import (
	"encoding/json"
	"testing"
	"time"
)

// TestAnchorTypes 测试锚定类型常量
func TestAnchorTypes(t *testing.T) {
	types := []AnchorType{
		AnchorTypeLocal,
		AnchorTypeWORM,
		AnchorTypeEthereum,
		AnchorTypePolygon,
		AnchorTypeBSC,
		AnchorTypeArbitrum,
		AnchorTypeFederated,
	}

	// 验证所有类型都有值
	for _, at := range types {
		if at == "" {
			t.Error("anchor type should not be empty")
		}
	}

	// 验证类型是唯一的
	seen := make(map[AnchorType]bool)
	for _, at := range types {
		if seen[at] {
			t.Errorf("duplicate anchor type: %s", at)
		}
		seen[at] = true
	}
}

// TestAnchorRequest 测试锚定请求结构
func TestAnchorRequest(t *testing.T) {
	req := AnchorRequest{
		CertID:    "cert_abc123",
		RootHash:  "sha256:deadbeef",
		Timestamp: time.Now(),
		Metadata:  `{"key": "value"}`,
	}

	// 测试 JSON 序列化
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded AnchorRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.CertID != req.CertID {
		t.Errorf("CertID mismatch: got %s, want %s", decoded.CertID, req.CertID)
	}
	if decoded.RootHash != req.RootHash {
		t.Errorf("RootHash mismatch: got %s, want %s", decoded.RootHash, req.RootHash)
	}
}

// TestAnchorResult 测试锚定结果结构
func TestAnchorResult(t *testing.T) {
	result := AnchorResult{
		AnchorID:        "anchor_xyz",
		AnchorType:      AnchorTypeEthereum,
		TxHash:          "0x1234567890abcdef",
		BlockNumber:     12345678,
		BlockHash:       "0xabcdef",
		ContractAddress: "0xcontract",
		ChainID:         1,
		GasUsed:         21000,
		Timestamp:       time.Now(),
		ProofURL:        "https://etherscan.io/tx/0x123",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded AnchorResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.AnchorType != result.AnchorType {
		t.Errorf("AnchorType mismatch: got %s, want %s", decoded.AnchorType, result.AnchorType)
	}
	if decoded.TxHash != result.TxHash {
		t.Errorf("TxHash mismatch: got %s, want %s", decoded.TxHash, result.TxHash)
	}
	if decoded.BlockNumber != result.BlockNumber {
		t.Errorf("BlockNumber mismatch: got %d, want %d", decoded.BlockNumber, result.BlockNumber)
	}
}

// TestAnchorResultFederated 测试联邦锚定结果
func TestAnchorResultFederated(t *testing.T) {
	result := AnchorResult{
		AnchorID:       "fed_anchor_123",
		AnchorType:     AnchorTypeFederated,
		FederatedNodes: []string{"node1.example.com", "node2.example.com", "node3.example.com"},
		Confirmations:  3,
		Timestamp:      time.Now(),
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded AnchorResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(decoded.FederatedNodes) != len(result.FederatedNodes) {
		t.Errorf("FederatedNodes count mismatch: got %d, want %d",
			len(decoded.FederatedNodes), len(result.FederatedNodes))
	}
	if decoded.Confirmations != result.Confirmations {
		t.Errorf("Confirmations mismatch: got %d, want %d", decoded.Confirmations, result.Confirmations)
	}
}

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig should not return nil")
	}

	// 验证默认值
	if cfg.EthereumChainID != 1 {
		t.Errorf("EthereumChainID: got %d, want 1", cfg.EthereumChainID)
	}
	if cfg.PolygonChainID != 137 {
		t.Errorf("PolygonChainID: got %d, want 137", cfg.PolygonChainID)
	}
	if cfg.GasLimit != 100000 {
		t.Errorf("GasLimit: got %d, want 100000", cfg.GasLimit)
	}
	if cfg.MaxGasPrice != 100000000000 {
		t.Errorf("MaxGasPrice: got %d, want 100000000000", cfg.MaxGasPrice)
	}
	if cfg.RetryAttempts != 3 {
		t.Errorf("RetryAttempts: got %d, want 3", cfg.RetryAttempts)
	}
	if cfg.RetryDelay != 5*time.Second {
		t.Errorf("RetryDelay: got %v, want 5s", cfg.RetryDelay)
	}
	if cfg.MinConfirmations != 3 {
		t.Errorf("MinConfirmations: got %d, want 3", cfg.MinConfirmations)
	}
}

// TestErrors 测试错误定义
func TestErrors(t *testing.T) {
	errors := []error{
		ErrNotConfigured,
		ErrInsufficientGas,
		ErrTxFailed,
		ErrTxTimeout,
		ErrInvalidProof,
		ErrChainUnavailable,
		ErrNoConfirmations,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("error should not be nil")
		}
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	}
}

// TestAnchorRequestMinimal 测试最小锚定请求
func TestAnchorRequestMinimal(t *testing.T) {
	req := AnchorRequest{
		CertID:    "cert_min",
		RootHash:  "hash",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal minimal request: %v", err)
	}

	// Metadata 应该被省略（omitempty）
	jsonStr := string(data)
	if containsString(jsonStr, `"metadata":""`) {
		t.Error("empty metadata should be omitted")
	}
}

// containsString 检查字符串是否包含子串
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
