//go:build blockchain
// +build blockchain

package contracts

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ArbitrationABI AITraceArbitration 合约 ABI
const ArbitrationABI = `[
	{"inputs":[{"internalType":"address","name":"_registry","type":"address"}],"stateMutability":"nonpayable","type":"constructor"},
	{"inputs":[{"internalType":"bytes32","name":"certId","type":"bytes32"},{"internalType":"uint8","name":"disputeType","type":"uint8"},{"internalType":"address","name":"defendant","type":"address"},{"internalType":"string","name":"description","type":"string"},{"internalType":"bytes32","name":"evidenceHash","type":"bytes32"}],"name":"createDispute","outputs":[{"internalType":"uint256","name":"disputeId","type":"uint256"}],"stateMutability":"payable","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"disputeId","type":"uint256"}],"name":"startVoting","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"disputeId","type":"uint256"},{"internalType":"uint8","name":"result","type":"uint8"},{"internalType":"string","name":"reason","type":"string"}],"name":"vote","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"disputeId","type":"uint256"}],"name":"resolveDispute","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"address","name":"arbitrator","type":"address"}],"name":"registerArbitrator","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"address","name":"arbitrator","type":"address"}],"name":"removeArbitrator","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"disputeId","type":"uint256"}],"name":"getDispute","outputs":[{"components":[{"internalType":"uint256","name":"disputeId","type":"uint256"},{"internalType":"bytes32","name":"certId","type":"bytes32"},{"internalType":"uint8","name":"disputeType","type":"uint8"},{"internalType":"uint8","name":"status","type":"uint8"},{"internalType":"address","name":"plaintiff","type":"address"},{"internalType":"address","name":"defendant","type":"address"},{"internalType":"string","name":"description","type":"string"},{"internalType":"bytes32","name":"evidenceHash","type":"bytes32"},{"internalType":"uint256","name":"createdAt","type":"uint256"},{"internalType":"uint256","name":"votingDeadline","type":"uint256"},{"internalType":"uint256","name":"votesInFavor","type":"uint256"},{"internalType":"uint256","name":"votesAgainst","type":"uint256"},{"internalType":"uint256","name":"votesAbstain","type":"uint256"},{"internalType":"uint256","name":"stake","type":"uint256"},{"internalType":"bool","name":"resolved","type":"bool"},{"internalType":"address","name":"winner","type":"address"}],"internalType":"struct AITraceArbitration.Dispute","name":"","type":"tuple"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"disputeId","type":"uint256"},{"internalType":"address","name":"arbitrator","type":"address"}],"name":"getVote","outputs":[{"components":[{"internalType":"address","name":"arbitrator","type":"address"},{"internalType":"uint8","name":"result","type":"uint8"},{"internalType":"uint256","name":"timestamp","type":"uint256"},{"internalType":"string","name":"reason","type":"string"}],"internalType":"struct AITraceArbitration.Vote","name":"","type":"tuple"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"address","name":"arbitrator","type":"address"}],"name":"getArbitrator","outputs":[{"components":[{"internalType":"address","name":"addr","type":"address"},{"internalType":"uint256","name":"reputation","type":"uint256"},{"internalType":"uint256","name":"totalVotes","type":"uint256"},{"internalType":"uint256","name":"correctVotes","type":"uint256"},{"internalType":"bool","name":"isActive","type":"bool"},{"internalType":"uint256","name":"registeredAt","type":"uint256"}],"internalType":"struct AITraceArbitration.Arbitrator","name":"","type":"tuple"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"getArbitratorCount","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"registry","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"disputeCount","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"minStake","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"votingPeriod","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"_minStake","type":"uint256"}],"name":"setMinStake","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"_votingPeriod","type":"uint256"}],"name":"setVotingPeriod","outputs":[],"stateMutability":"nonpayable","type":"function"}
]`

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

// VoteResult 投票结果
type VoteResult uint8

const (
	VoteResultNone    VoteResult = 0
	VoteResultInFavor VoteResult = 1
	VoteResultAgainst VoteResult = 2
	VoteResultAbstain VoteResult = 3
)

// Dispute 争议记录
type Dispute struct {
	DisputeId      *big.Int
	CertId         [32]byte
	DisputeType    DisputeType
	Status         DisputeStatus
	Plaintiff      common.Address
	Defendant      common.Address
	Description    string
	EvidenceHash   [32]byte
	CreatedAt      *big.Int
	VotingDeadline *big.Int
	VotesInFavor   *big.Int
	VotesAgainst   *big.Int
	VotesAbstain   *big.Int
	Stake          *big.Int
	Resolved       bool
	Winner         common.Address
}

// Vote 投票记录
type Vote struct {
	Arbitrator common.Address
	Result     VoteResult
	Timestamp  *big.Int
	Reason     string
}

// Arbitrator 仲裁员信息
type Arbitrator struct {
	Addr         common.Address
	Reputation   *big.Int
	TotalVotes   *big.Int
	CorrectVotes *big.Int
	IsActive     bool
	RegisteredAt *big.Int
}

// ArbitrationContract AITraceArbitration 合约封装
type ArbitrationContract struct {
	address common.Address
	client  *ethclient.Client
	abi     abi.ABI
	auth    *bind.TransactOpts
	chainID *big.Int
}

// NewArbitrationContract 创建 Arbitration 合约实例
func NewArbitrationContract(
	client *ethclient.Client,
	contractAddr common.Address,
	privateKey *ecdsa.PrivateKey,
	chainID *big.Int,
) (*ArbitrationContract, error) {
	parsedABI, err := abi.JSON(strings.NewReader(ArbitrationABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	var auth *bind.TransactOpts
	if privateKey != nil {
		auth, err = bind.NewKeyedTransactorWithChainID(privateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create transactor: %w", err)
		}
	}

	return &ArbitrationContract{
		address: contractAddr,
		client:  client,
		abi:     parsedABI,
		auth:    auth,
		chainID: chainID,
	}, nil
}

// CreateDispute 创建争议
func (a *ArbitrationContract) CreateDispute(
	ctx context.Context,
	certId [32]byte,
	disputeType DisputeType,
	defendant common.Address,
	description string,
	evidenceHash [32]byte,
	stake *big.Int,
) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := a.abi.Pack("createDispute", certId, uint8(disputeType), defendant, description, evidenceHash)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return a.sendTransaction(ctx, data, stake)
}

// StartVoting 开始投票
func (a *ArbitrationContract) StartVoting(
	ctx context.Context,
	disputeId *big.Int,
) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := a.abi.Pack("startVoting", disputeId)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return a.sendTransaction(ctx, data, big.NewInt(0))
}

// Vote 投票
func (a *ArbitrationContract) Vote(
	ctx context.Context,
	disputeId *big.Int,
	result VoteResult,
	reason string,
) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := a.abi.Pack("vote", disputeId, uint8(result), reason)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return a.sendTransaction(ctx, data, big.NewInt(0))
}

// ResolveDispute 结算争议
func (a *ArbitrationContract) ResolveDispute(
	ctx context.Context,
	disputeId *big.Int,
) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := a.abi.Pack("resolveDispute", disputeId)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return a.sendTransaction(ctx, data, big.NewInt(0))
}

// RegisterArbitrator 注册仲裁员
func (a *ArbitrationContract) RegisterArbitrator(
	ctx context.Context,
	arbitrator common.Address,
) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := a.abi.Pack("registerArbitrator", arbitrator)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return a.sendTransaction(ctx, data, big.NewInt(0))
}

// RemoveArbitrator 移除仲裁员
func (a *ArbitrationContract) RemoveArbitrator(
	ctx context.Context,
	arbitrator common.Address,
) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := a.abi.Pack("removeArbitrator", arbitrator)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return a.sendTransaction(ctx, data, big.NewInt(0))
}

// GetDispute 获取争议详情
func (a *ArbitrationContract) GetDispute(
	ctx context.Context,
	disputeId *big.Int,
) (*Dispute, error) {
	data, err := a.abi.Pack("getDispute", disputeId)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := a.call(ctx, data)
	if err != nil {
		return nil, err
	}

	outputs, err := a.abi.Unpack("getDispute", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("empty result")
	}

	// 类型断言处理
	if disputeData, ok := outputs[0].(struct {
		DisputeId      *big.Int       `json:"disputeId"`
		CertId         [32]byte       `json:"certId"`
		DisputeType    uint8          `json:"disputeType"`
		Status         uint8          `json:"status"`
		Plaintiff      common.Address `json:"plaintiff"`
		Defendant      common.Address `json:"defendant"`
		Description    string         `json:"description"`
		EvidenceHash   [32]byte       `json:"evidenceHash"`
		CreatedAt      *big.Int       `json:"createdAt"`
		VotingDeadline *big.Int       `json:"votingDeadline"`
		VotesInFavor   *big.Int       `json:"votesInFavor"`
		VotesAgainst   *big.Int       `json:"votesAgainst"`
		VotesAbstain   *big.Int       `json:"votesAbstain"`
		Stake          *big.Int       `json:"stake"`
		Resolved       bool           `json:"resolved"`
		Winner         common.Address `json:"winner"`
	}); ok {
		return &Dispute{
			DisputeId:      disputeData.DisputeId,
			CertId:         disputeData.CertId,
			DisputeType:    DisputeType(disputeData.DisputeType),
			Status:         DisputeStatus(disputeData.Status),
			Plaintiff:      disputeData.Plaintiff,
			Defendant:      disputeData.Defendant,
			Description:    disputeData.Description,
			EvidenceHash:   disputeData.EvidenceHash,
			CreatedAt:      disputeData.CreatedAt,
			VotingDeadline: disputeData.VotingDeadline,
			VotesInFavor:   disputeData.VotesInFavor,
			VotesAgainst:   disputeData.VotesAgainst,
			VotesAbstain:   disputeData.VotesAbstain,
			Stake:          disputeData.Stake,
			Resolved:       disputeData.Resolved,
			Winner:         disputeData.Winner,
		}, nil
	}

	return nil, fmt.Errorf("failed to parse dispute data")
}

// GetVote 获取投票记录
func (a *ArbitrationContract) GetVote(
	ctx context.Context,
	disputeId *big.Int,
	arbitrator common.Address,
) (*Vote, error) {
	data, err := a.abi.Pack("getVote", disputeId, arbitrator)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := a.call(ctx, data)
	if err != nil {
		return nil, err
	}

	outputs, err := a.abi.Unpack("getVote", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("empty result")
	}

	if voteData, ok := outputs[0].(struct {
		Arbitrator common.Address `json:"arbitrator"`
		Result     uint8          `json:"result"`
		Timestamp  *big.Int       `json:"timestamp"`
		Reason     string         `json:"reason"`
	}); ok {
		return &Vote{
			Arbitrator: voteData.Arbitrator,
			Result:     VoteResult(voteData.Result),
			Timestamp:  voteData.Timestamp,
			Reason:     voteData.Reason,
		}, nil
	}

	return nil, fmt.Errorf("failed to parse vote data")
}

// GetArbitrator 获取仲裁员信息
func (a *ArbitrationContract) GetArbitrator(
	ctx context.Context,
	arbitrator common.Address,
) (*Arbitrator, error) {
	data, err := a.abi.Pack("getArbitrator", arbitrator)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := a.call(ctx, data)
	if err != nil {
		return nil, err
	}

	outputs, err := a.abi.Unpack("getArbitrator", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("empty result")
	}

	if arbData, ok := outputs[0].(struct {
		Addr         common.Address `json:"addr"`
		Reputation   *big.Int       `json:"reputation"`
		TotalVotes   *big.Int       `json:"totalVotes"`
		CorrectVotes *big.Int       `json:"correctVotes"`
		IsActive     bool           `json:"isActive"`
		RegisteredAt *big.Int       `json:"registeredAt"`
	}); ok {
		return &Arbitrator{
			Addr:         arbData.Addr,
			Reputation:   arbData.Reputation,
			TotalVotes:   arbData.TotalVotes,
			CorrectVotes: arbData.CorrectVotes,
			IsActive:     arbData.IsActive,
			RegisteredAt: arbData.RegisteredAt,
		}, nil
	}

	return nil, fmt.Errorf("failed to parse arbitrator data")
}

// GetArbitratorCount 获取仲裁员数量
func (a *ArbitrationContract) GetArbitratorCount(ctx context.Context) (*big.Int, error) {
	data, err := a.abi.Pack("getArbitratorCount")
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := a.call(ctx, data)
	if err != nil {
		return nil, err
	}

	var count *big.Int
	err = a.abi.UnpackIntoInterface(&count, "getArbitratorCount", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	return count, nil
}

// GetDisputeCount 获取争议总数
func (a *ArbitrationContract) GetDisputeCount(ctx context.Context) (*big.Int, error) {
	data, err := a.abi.Pack("disputeCount")
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := a.call(ctx, data)
	if err != nil {
		return nil, err
	}

	var count *big.Int
	err = a.abi.UnpackIntoInterface(&count, "disputeCount", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	return count, nil
}

// GetMinStake 获取最小押金
func (a *ArbitrationContract) GetMinStake(ctx context.Context) (*big.Int, error) {
	data, err := a.abi.Pack("minStake")
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := a.call(ctx, data)
	if err != nil {
		return nil, err
	}

	var stake *big.Int
	err = a.abi.UnpackIntoInterface(&stake, "minStake", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	return stake, nil
}

// SetMinStake 设置最小押金
func (a *ArbitrationContract) SetMinStake(
	ctx context.Context,
	minStake *big.Int,
) (*types.Transaction, error) {
	if a.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := a.abi.Pack("setMinStake", minStake)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return a.sendTransaction(ctx, data, big.NewInt(0))
}

// sendTransaction 发送交易
func (a *ArbitrationContract) sendTransaction(
	ctx context.Context,
	data []byte,
	value *big.Int,
) (*types.Transaction, error) {
	nonce, err := a.client.PendingNonceAt(ctx, a.auth.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := a.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	msg := ethereum.CallMsg{
		From:  a.auth.From,
		To:    &a.address,
		Value: value,
		Data:  data,
	}
	gasLimit, err := a.client.EstimateGas(ctx, msg)
	if err != nil {
		gasLimit = 300000
	}

	tx := types.NewTransaction(
		nonce,
		a.address,
		value,
		gasLimit,
		gasPrice,
		data,
	)

	signedTx, err := a.auth.Signer(a.auth.From, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = a.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx, nil
}

// call 执行只读调用
func (a *ArbitrationContract) call(ctx context.Context, data []byte) ([]byte, error) {
	msg := ethereum.CallMsg{
		To:   &a.address,
		Data: data,
	}

	result, err := a.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	return result, nil
}
