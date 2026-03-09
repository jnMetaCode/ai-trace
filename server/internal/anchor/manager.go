package anchor

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Manager 锚定管理器
// 统一管理多种锚定方式
type Manager struct {
	anchors map[AnchorType]Anchorer
	logger  *zap.SugaredLogger
	config  *Config
}

// NewManager 创建锚定管理器
func NewManager(cfg *Config, logger *zap.SugaredLogger) (*Manager, error) {
	m := &Manager{
		anchors: make(map[AnchorType]Anchorer),
		logger:  logger,
		config:  cfg,
	}

	// 初始化以太坊锚定器
	if cfg.EthereumRPCURL != "" {
		eth, err := NewEthereumAnchor(cfg, logger)
		if err != nil {
			logger.Warnf("Failed to initialize Ethereum anchor: %v", err)
		} else {
			m.anchors[AnchorTypeEthereum] = eth
			logger.Info("Ethereum anchor initialized")
		}
	}

	// 初始化联邦锚定器
	if len(cfg.FederatedNodes) > 0 {
		fed, err := NewFederatedAnchor(cfg, logger)
		if err != nil {
			logger.Warnf("Failed to initialize Federated anchor: %v", err)
		} else {
			m.anchors[AnchorTypeFederated] = fed
			logger.Infof("Federated anchor initialized with %d nodes", len(cfg.FederatedNodes))
		}
	}

	return m, nil
}

// Anchor 执行锚定
func (m *Manager) Anchor(ctx context.Context, anchorType AnchorType, req *AnchorRequest) (*AnchorResult, error) {
	anchorer, ok := m.anchors[anchorType]
	if !ok {
		return nil, fmt.Errorf("anchor type %s not configured", anchorType)
	}

	if !anchorer.IsAvailable(ctx) {
		return nil, ErrChainUnavailable
	}

	return anchorer.Anchor(ctx, req)
}

// Verify 验证锚定
func (m *Manager) Verify(ctx context.Context, result *AnchorResult) (bool, error) {
	anchorer, ok := m.anchors[result.AnchorType]
	if !ok {
		return false, fmt.Errorf("anchor type %s not configured", result.AnchorType)
	}

	return anchorer.Verify(ctx, result)
}

// GetAvailableAnchors 获取可用的锚定类型
func (m *Manager) GetAvailableAnchors(ctx context.Context) []AnchorType {
	var available []AnchorType
	for t, a := range m.anchors {
		if a.IsAvailable(ctx) {
			available = append(available, t)
		}
	}
	return available
}

// GetAnchorer 获取指定类型的锚定器
func (m *Manager) GetAnchorer(anchorType AnchorType) (Anchorer, bool) {
	a, ok := m.anchors[anchorType]
	return a, ok
}

// HasAnchor 检查是否有指定类型的锚定器
func (m *Manager) HasAnchor(anchorType AnchorType) bool {
	_, ok := m.anchors[anchorType]
	return ok
}

// Close 关闭所有锚定器
func (m *Manager) Close() {
	for _, a := range m.anchors {
		if closer, ok := a.(interface{ Close() }); ok {
			closer.Close()
		}
	}
}
