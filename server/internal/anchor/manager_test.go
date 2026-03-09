package anchor

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// mockAnchorer 模拟锚定器
type mockAnchorer struct {
	anchorType AnchorType
	available  bool
	anchorErr  error
	verifyOK   bool
	verifyErr  error
}

func (m *mockAnchorer) Anchor(ctx context.Context, req *AnchorRequest) (*AnchorResult, error) {
	if m.anchorErr != nil {
		return nil, m.anchorErr
	}
	return &AnchorResult{
		AnchorID:   "mock_anchor_123",
		AnchorType: m.anchorType,
		Timestamp:  time.Now(),
	}, nil
}

func (m *mockAnchorer) Verify(ctx context.Context, result *AnchorResult) (bool, error) {
	return m.verifyOK, m.verifyErr
}

func (m *mockAnchorer) GetAnchorType() AnchorType {
	return m.anchorType
}

func (m *mockAnchorer) IsAvailable(ctx context.Context) bool {
	return m.available
}

// TestNewManager 测试管理器创建
func TestNewManager(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := DefaultConfig()

	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("manager should not be nil")
	}
	if manager.anchors == nil {
		t.Error("anchors map should be initialized")
	}
}

// TestManagerWithMockAnchor 测试带模拟锚定器的管理器
func TestManagerWithMockAnchor(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := DefaultConfig()

	manager, _ := NewManager(cfg, logger)

	// 添加模拟锚定器
	mock := &mockAnchorer{
		anchorType: AnchorTypeLocal,
		available:  true,
		verifyOK:   true,
	}
	manager.anchors[AnchorTypeLocal] = mock

	// 测试 HasAnchor
	if !manager.HasAnchor(AnchorTypeLocal) {
		t.Error("should have local anchor")
	}
	if manager.HasAnchor(AnchorTypeEthereum) {
		t.Error("should not have ethereum anchor")
	}

	// 测试 GetAnchorer
	anchorer, ok := manager.GetAnchorer(AnchorTypeLocal)
	if !ok {
		t.Error("should get local anchorer")
	}
	if anchorer.GetAnchorType() != AnchorTypeLocal {
		t.Error("anchor type mismatch")
	}
}

// TestManagerAnchor 测试管理器锚定
func TestManagerAnchor(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := DefaultConfig()
	manager, _ := NewManager(cfg, logger)

	mock := &mockAnchorer{
		anchorType: AnchorTypeLocal,
		available:  true,
	}
	manager.anchors[AnchorTypeLocal] = mock

	ctx := context.Background()
	req := &AnchorRequest{
		CertID:    "cert_test",
		RootHash:  "hash_test",
		Timestamp: time.Now(),
	}

	result, err := manager.Anchor(ctx, AnchorTypeLocal, req)
	if err != nil {
		t.Fatalf("anchor failed: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.AnchorType != AnchorTypeLocal {
		t.Errorf("anchor type: got %s, want %s", result.AnchorType, AnchorTypeLocal)
	}
}

// TestManagerAnchorNotConfigured 测试未配置的锚定类型
func TestManagerAnchorNotConfigured(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := DefaultConfig()
	manager, _ := NewManager(cfg, logger)

	ctx := context.Background()
	req := &AnchorRequest{
		CertID:   "cert_test",
		RootHash: "hash_test",
	}

	_, err := manager.Anchor(ctx, AnchorTypeEthereum, req)
	if err == nil {
		t.Error("should return error for unconfigured anchor type")
	}
}

// TestManagerAnchorUnavailable 测试不可用的锚定器
func TestManagerAnchorUnavailable(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := DefaultConfig()
	manager, _ := NewManager(cfg, logger)

	mock := &mockAnchorer{
		anchorType: AnchorTypeLocal,
		available:  false, // 不可用
	}
	manager.anchors[AnchorTypeLocal] = mock

	ctx := context.Background()
	req := &AnchorRequest{
		CertID:   "cert_test",
		RootHash: "hash_test",
	}

	_, err := manager.Anchor(ctx, AnchorTypeLocal, req)
	if err != ErrChainUnavailable {
		t.Errorf("expected ErrChainUnavailable, got %v", err)
	}
}

// TestManagerVerify 测试管理器验证
func TestManagerVerify(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := DefaultConfig()
	manager, _ := NewManager(cfg, logger)

	mock := &mockAnchorer{
		anchorType: AnchorTypeLocal,
		available:  true,
		verifyOK:   true,
	}
	manager.anchors[AnchorTypeLocal] = mock

	ctx := context.Background()
	result := &AnchorResult{
		AnchorID:   "anchor_123",
		AnchorType: AnchorTypeLocal,
	}

	valid, err := manager.Verify(ctx, result)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !valid {
		t.Error("should be valid")
	}
}

// TestManagerVerifyNotConfigured 测试未配置的验证
func TestManagerVerifyNotConfigured(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := DefaultConfig()
	manager, _ := NewManager(cfg, logger)

	ctx := context.Background()
	result := &AnchorResult{
		AnchorID:   "anchor_123",
		AnchorType: AnchorTypePolygon, // 未配置
	}

	_, err := manager.Verify(ctx, result)
	if err == nil {
		t.Error("should return error for unconfigured anchor type")
	}
}

// TestManagerGetAvailableAnchors 测试获取可用锚定器
func TestManagerGetAvailableAnchors(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := DefaultConfig()
	manager, _ := NewManager(cfg, logger)

	// 添加多个模拟锚定器
	manager.anchors[AnchorTypeLocal] = &mockAnchorer{
		anchorType: AnchorTypeLocal,
		available:  true,
	}
	manager.anchors[AnchorTypeWORM] = &mockAnchorer{
		anchorType: AnchorTypeWORM,
		available:  true,
	}
	manager.anchors[AnchorTypeEthereum] = &mockAnchorer{
		anchorType: AnchorTypeEthereum,
		available:  false, // 不可用
	}

	ctx := context.Background()
	available := manager.GetAvailableAnchors(ctx)

	// 应该有 2 个可用的
	if len(available) != 2 {
		t.Errorf("available count: got %d, want 2", len(available))
	}

	// 验证包含正确的类型
	hasLocal := false
	hasWORM := false
	for _, at := range available {
		if at == AnchorTypeLocal {
			hasLocal = true
		}
		if at == AnchorTypeWORM {
			hasWORM = true
		}
	}
	if !hasLocal {
		t.Error("should include local anchor")
	}
	if !hasWORM {
		t.Error("should include WORM anchor")
	}
}

// TestManagerClose 测试关闭管理器
func TestManagerClose(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := DefaultConfig()
	manager, _ := NewManager(cfg, logger)

	// 添加一个可关闭的模拟锚定器
	closed := false
	closer := &closableMockAnchorer{
		mockAnchorer: mockAnchorer{
			anchorType: AnchorTypeLocal,
			available:  true,
		},
		onClose: func() { closed = true },
	}
	manager.anchors[AnchorTypeLocal] = closer

	manager.Close()

	if !closed {
		t.Error("anchorer should be closed")
	}
}

// closableMockAnchorer 可关闭的模拟锚定器
type closableMockAnchorer struct {
	mockAnchorer
	onClose func()
}

func (c *closableMockAnchorer) Close() {
	if c.onClose != nil {
		c.onClose()
	}
}
