//go:build !blockchain
// +build !blockchain

// Package contracts 提供智能合约的 Go 绑定
// 此文件为非区块链环境的 stub 实现
package contracts

import (
	"context"
	"errors"
	"math/big"
	"time"

	"go.uber.org/zap"
)

var ErrBlockchainNotEnabled = errors.New("blockchain support not enabled, rebuild with -tags blockchain")

// AttestationRecord 存证记录结构 (stub)
type AttestationRecord struct {
	CertId          [32]byte
	MerkleRoot      [32]byte
	FingerprintHash [32]byte
	InputHash       [32]byte
	OutputHash      [32]byte
	Timestamp       *big.Int
	BlockNumber     *big.Int
	ModelId         string
	TenantId        string
	IsValid         bool
}

// DisputeStatus 争议状态
type DisputeStatus uint8

const (
	DisputeStatusPending  DisputeStatus = 0
	DisputeStatusVoting   DisputeStatus = 1
	DisputeStatusResolved DisputeStatus = 2
	DisputeStatusRejected DisputeStatus = 3
	DisputeStatusExpired  DisputeStatus = 4
)

// DisputeType 争议类型
type DisputeType uint8

const (
	DisputeTypeContentOwnership    DisputeType = 0
	DisputeTypeContentAuthenticity DisputeType = 1
	DisputeTypeModelMisuse         DisputeType = 2
	DisputeTypeDataTampering       DisputeType = 3
	DisputeTypeOther               DisputeType = 4
)

// Dispute 争议记录 (stub)
type Dispute struct {
	DisputeId   *big.Int
	CertId      [32]byte
	DisputeType DisputeType
	Status      DisputeStatus
	Description string
}

// TxResult 交易结果 (stub)
type TxResult struct {
	TxHash      string
	BlockNumber uint64
	BlockHash   string
	GasUsed     uint64
	Status      uint64
}

// ContractManager 合约管理器 (stub)
type ContractManager struct{}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	RPCURL              string
	ChainID             int64
	RegistryAddress     string
	ArbitrationAddress  string
}

// NewContractManager 创建合约管理器 (stub)
func NewContractManager(cfg *ManagerConfig, logger *zap.SugaredLogger) (*ContractManager, error) {
	return nil, ErrBlockchainNotEnabled
}

// CreateAttestation stub
func (m *ContractManager) CreateAttestation(ctx context.Context, certId, merkleRoot, fingerprintHash, inputHash, outputHash, modelId, tenantId string) (*TxResult, error) {
	return nil, ErrBlockchainNotEnabled
}

// SimpleAnchor stub
func (m *ContractManager) SimpleAnchor(ctx context.Context, certHash, rootHash string, timestamp time.Time) (*TxResult, error) {
	return nil, ErrBlockchainNotEnabled
}

// BatchAnchor stub
func (m *ContractManager) BatchAnchor(ctx context.Context, certIds, merkleRoots, fingerprintHashes []string) (*TxResult, error) {
	return nil, ErrBlockchainNotEnabled
}

// VerifyAttestation stub
func (m *ContractManager) VerifyAttestation(ctx context.Context, certId, merkleRoot string) (bool, error) {
	return false, ErrBlockchainNotEnabled
}

// VerifyFingerprint stub
func (m *ContractManager) VerifyFingerprint(ctx context.Context, certId, fingerprintHash string) (bool, error) {
	return false, ErrBlockchainNotEnabled
}

// GetBalance stub
func (m *ContractManager) GetBalance(ctx context.Context) (*big.Int, error) {
	return nil, ErrBlockchainNotEnabled
}

// GetChainID stub
func (m *ContractManager) GetChainID() *big.Int {
	return nil
}

// IsAvailable stub
func (m *ContractManager) IsAvailable(ctx context.Context) bool {
	return false
}

// Close stub
func (m *ContractManager) Close() {}

// HashToCertId 将字符串哈希转换为 certId
func HashToCertId(hash string) [32]byte {
	var certId [32]byte
	return certId
}

// CertIdToHash 将 certId 转换为字符串哈希
func CertIdToHash(certId [32]byte) string {
	return ""
}
