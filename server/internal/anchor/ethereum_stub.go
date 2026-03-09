//go:build !blockchain
// +build !blockchain

// 此文件在不使用 blockchain tag 时编译
// 提供空实现，避免引入 go-ethereum 依赖

package anchor

import (
	"context"

	"go.uber.org/zap"
)

// EthereumAnchor 以太坊锚定 stub（未启用区块链）
type EthereumAnchor struct {
	logger *zap.SugaredLogger
}

// NewEthereumAnchor 创建以太坊锚定器（stub）
func NewEthereumAnchor(cfg *Config, logger *zap.SugaredLogger) (*EthereumAnchor, error) {
	logger.Warn("Blockchain support not compiled. Build with: go build -tags blockchain")
	return nil, ErrNotConfigured
}

// Anchor 执行锚定（stub）
func (e *EthereumAnchor) Anchor(ctx context.Context, req *AnchorRequest) (*AnchorResult, error) {
	return nil, ErrNotConfigured
}

// Verify 验证锚定（stub）
func (e *EthereumAnchor) Verify(ctx context.Context, result *AnchorResult) (bool, error) {
	return false, ErrNotConfigured
}

// GetAnchorType 获取锚定类型
func (e *EthereumAnchor) GetAnchorType() AnchorType {
	return AnchorTypeEthereum
}

// IsAvailable 检查服务是否可用（stub 始终返回 false）
func (e *EthereumAnchor) IsAvailable(ctx context.Context) bool {
	return false
}

// Close 关闭连接（stub）
func (e *EthereumAnchor) Close() {}
