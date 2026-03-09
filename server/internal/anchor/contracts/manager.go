//go:build blockchain
// +build blockchain

package contracts

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// ContractManager 合约管理器
type ContractManager struct {
	client      *ethclient.Client
	chainID     *big.Int
	privateKey  *ecdsa.PrivateKey
	address     common.Address
	registry    *RegistryContract
	arbitration *ArbitrationContract
	logger      *zap.SugaredLogger
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	RPCURL              string
	PrivateKey          *ecdsa.PrivateKey
	ChainID             int64
	RegistryAddress     string
	ArbitrationAddress  string
}

// NewContractManager 创建合约管理器
func NewContractManager(cfg *ManagerConfig, logger *zap.SugaredLogger) (*ContractManager, error) {
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum: %w", err)
	}

	chainID := big.NewInt(cfg.ChainID)

	manager := &ContractManager{
		client:     client,
		chainID:    chainID,
		privateKey: cfg.PrivateKey,
		logger:     logger,
	}

	if cfg.PrivateKey != nil {
		publicKey := cfg.PrivateKey.Public().(*ecdsa.PublicKey)
		manager.address = common.Address{}
		copy(manager.address[:], publicKey.X.Bytes()[:20])
	}

	// 初始化 Registry 合约
	if cfg.RegistryAddress != "" {
		registryAddr := common.HexToAddress(cfg.RegistryAddress)
		registry, err := NewRegistryContract(client, registryAddr, cfg.PrivateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create registry contract: %w", err)
		}
		manager.registry = registry
	}

	// 初始化 Arbitration 合约
	if cfg.ArbitrationAddress != "" {
		arbitrationAddr := common.HexToAddress(cfg.ArbitrationAddress)
		arbitration, err := NewArbitrationContract(client, arbitrationAddr, cfg.PrivateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create arbitration contract: %w", err)
		}
		manager.arbitration = arbitration
	}

	return manager, nil
}

// Registry 获取 Registry 合约
func (m *ContractManager) Registry() *RegistryContract {
	return m.registry
}

// Arbitration 获取 Arbitration 合约
func (m *ContractManager) Arbitration() *ArbitrationContract {
	return m.arbitration
}

// CreateAttestation 创建存证（封装方法）
func (m *ContractManager) CreateAttestation(
	ctx context.Context,
	certId string,
	merkleRoot string,
	fingerprintHash string,
	inputHash string,
	outputHash string,
	modelId string,
	tenantId string,
) (*TxResult, error) {
	if m.registry == nil {
		return nil, fmt.Errorf("registry contract not configured")
	}

	tx, err := m.registry.CreateAttestation(
		ctx,
		HashToCertId(certId),
		HashToCertId(merkleRoot),
		HashToCertId(fingerprintHash),
		HashToCertId(inputHash),
		HashToCertId(outputHash),
		modelId,
		tenantId,
	)
	if err != nil {
		return nil, err
	}

	return m.waitForTx(ctx, tx)
}

// SimpleAnchor 简化版锚定
func (m *ContractManager) SimpleAnchor(
	ctx context.Context,
	certHash string,
	rootHash string,
	timestamp time.Time,
) (*TxResult, error) {
	if m.registry == nil {
		return nil, fmt.Errorf("registry contract not configured")
	}

	tx, err := m.registry.Anchor(
		ctx,
		HashToCertId(certHash),
		HashToCertId(rootHash),
		big.NewInt(timestamp.Unix()),
	)
	if err != nil {
		return nil, err
	}

	return m.waitForTx(ctx, tx)
}

// BatchAnchor 批量锚定
func (m *ContractManager) BatchAnchor(
	ctx context.Context,
	certIds []string,
	merkleRoots []string,
	fingerprintHashes []string,
) (*TxResult, error) {
	if m.registry == nil {
		return nil, fmt.Errorf("registry contract not configured")
	}

	// 转换参数
	certIdBytes := make([][32]byte, len(certIds))
	rootBytes := make([][32]byte, len(merkleRoots))
	fpBytes := make([][32]byte, len(fingerprintHashes))

	for i := range certIds {
		certIdBytes[i] = HashToCertId(certIds[i])
		rootBytes[i] = HashToCertId(merkleRoots[i])
		fpBytes[i] = HashToCertId(fingerprintHashes[i])
	}

	tx, err := m.registry.BatchCreateAttestations(ctx, certIdBytes, rootBytes, fpBytes)
	if err != nil {
		return nil, err
	}

	return m.waitForTx(ctx, tx)
}

// VerifyAttestation 验证存证
func (m *ContractManager) VerifyAttestation(
	ctx context.Context,
	certId string,
	merkleRoot string,
) (bool, error) {
	if m.registry == nil {
		return false, fmt.Errorf("registry contract not configured")
	}

	return m.registry.VerifyAttestation(
		ctx,
		HashToCertId(certId),
		HashToCertId(merkleRoot),
	)
}

// VerifyFingerprint 验证指纹
func (m *ContractManager) VerifyFingerprint(
	ctx context.Context,
	certId string,
	fingerprintHash string,
) (bool, error) {
	if m.registry == nil {
		return false, fmt.Errorf("registry contract not configured")
	}

	return m.registry.VerifyFingerprint(
		ctx,
		HashToCertId(certId),
		HashToCertId(fingerprintHash),
	)
}

// CreateDispute 创建争议
func (m *ContractManager) CreateDispute(
	ctx context.Context,
	certId string,
	disputeType DisputeType,
	defendant common.Address,
	description string,
	evidenceHash string,
	stake *big.Int,
) (*TxResult, error) {
	if m.arbitration == nil {
		return nil, fmt.Errorf("arbitration contract not configured")
	}

	tx, err := m.arbitration.CreateDispute(
		ctx,
		HashToCertId(certId),
		disputeType,
		defendant,
		description,
		HashToCertId(evidenceHash),
		stake,
	)
	if err != nil {
		return nil, err
	}

	return m.waitForTx(ctx, tx)
}

// TxResult 交易结果
type TxResult struct {
	TxHash      string
	BlockNumber uint64
	BlockHash   string
	GasUsed     uint64
	Status      uint64
}

// waitForTx 等待交易确认
func (m *ContractManager) waitForTx(ctx context.Context, tx *types.Transaction) (*TxResult, error) {
	txHash := tx.Hash()
	m.logger.Infof("Waiting for transaction: %s", txHash.Hex())

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("transaction timeout")
		case <-ticker.C:
			receipt, err := m.client.TransactionReceipt(ctx, txHash)
			if err != nil {
				continue // 还没确认
			}

			return &TxResult{
				TxHash:      txHash.Hex(),
				BlockNumber: receipt.BlockNumber.Uint64(),
				BlockHash:   receipt.BlockHash.Hex(),
				GasUsed:     receipt.GasUsed,
				Status:      receipt.Status,
			}, nil
		}
	}
}

// GetBalance 获取账户余额
func (m *ContractManager) GetBalance(ctx context.Context) (*big.Int, error) {
	return m.client.BalanceAt(ctx, m.address, nil)
}

// GetChainID 获取链ID
func (m *ContractManager) GetChainID() *big.Int {
	return m.chainID
}

// IsAvailable 检查服务是否可用
func (m *ContractManager) IsAvailable(ctx context.Context) bool {
	_, err := m.client.ChainID(ctx)
	return err == nil
}

// Close 关闭连接
func (m *ContractManager) Close() {
	m.client.Close()
}
