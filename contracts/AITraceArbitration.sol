// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./AITraceRegistry.sol";

/**
 * @title AITraceArbitration
 * @notice AI 生成内容争议仲裁合约
 * @dev 处理关于 AI 生成内容的争议，支持多仲裁员投票机制
 */
contract AITraceArbitration {
    // ============ 枚举 ============

    /// @notice 争议状态
    enum DisputeStatus {
        Pending,    // 待处理
        Voting,     // 投票中
        Resolved,   // 已解决
        Rejected,   // 已拒绝
        Expired     // 已过期
    }

    /// @notice 争议类型
    enum DisputeType {
        ContentOwnership,    // 内容所有权争议
        ContentAuthenticity, // 内容真实性争议
        ModelMisuse,         // 模型滥用
        DataTampering,       // 数据篡改
        Other                // 其他
    }

    /// @notice 投票结果
    enum VoteResult {
        None,        // 未投票
        InFavor,     // 支持原告
        Against,     // 支持被告
        Abstain      // 弃权
    }

    // ============ 结构体 ============

    /// @notice 争议记录
    struct Dispute {
        uint256 disputeId;           // 争议 ID
        bytes32 certId;              // 相关存证 ID
        DisputeType disputeType;     // 争议类型
        DisputeStatus status;        // 争议状态
        address plaintiff;           // 原告
        address defendant;           // 被告
        string description;          // 争议描述
        bytes32 evidenceHash;        // 证据哈希
        uint256 createdAt;           // 创建时间
        uint256 votingDeadline;      // 投票截止时间
        uint256 votesInFavor;        // 支持票数
        uint256 votesAgainst;        // 反对票数
        uint256 votesAbstain;        // 弃权票数
        uint256 stake;               // 押金
        bool resolved;               // 是否已解决
        address winner;              // 胜诉方
    }

    /// @notice 仲裁员信息
    struct Arbitrator {
        address addr;                // 地址
        uint256 reputation;          // 信誉分
        uint256 totalVotes;          // 总投票数
        uint256 correctVotes;        // 正确投票数
        bool isActive;               // 是否激活
        uint256 registeredAt;        // 注册时间
    }

    /// @notice 投票记录
    struct Vote {
        address arbitrator;          // 仲裁员
        VoteResult result;           // 投票结果
        uint256 timestamp;           // 投票时间
        string reason;               // 投票理由
    }

    // ============ 状态变量 ============

    /// @notice 合约所有者
    address public owner;

    /// @notice 存证注册合约
    AITraceRegistry public registry;

    /// @notice 争议映射 (disputeId => Dispute)
    mapping(uint256 => Dispute) public disputes;

    /// @notice 争议投票映射 (disputeId => arbitrator => Vote)
    mapping(uint256 => mapping(address => Vote)) public disputeVotes;

    /// @notice 仲裁员映射 (address => Arbitrator)
    mapping(address => Arbitrator) public arbitrators;

    /// @notice 仲裁员列表
    address[] public arbitratorList;

    /// @notice 争议计数
    uint256 public disputeCount;

    /// @notice 最小押金
    uint256 public minStake;

    /// @notice 投票期限（秒）
    uint256 public votingPeriod;

    /// @notice 最小仲裁员数量
    uint256 public minArbitrators;

    /// @notice 初始信誉分
    uint256 public constant INITIAL_REPUTATION = 100;

    // ============ 事件 ============

    /// @notice 争议创建事件
    event DisputeCreated(
        uint256 indexed disputeId,
        bytes32 indexed certId,
        DisputeType disputeType,
        address indexed plaintiff,
        address defendant,
        uint256 stake
    );

    /// @notice 争议投票事件
    event DisputeVoted(
        uint256 indexed disputeId,
        address indexed arbitrator,
        VoteResult result
    );

    /// @notice 争议解决事件
    event DisputeResolved(
        uint256 indexed disputeId,
        DisputeStatus status,
        address winner
    );

    /// @notice 仲裁员注册事件
    event ArbitratorRegistered(
        address indexed arbitrator,
        uint256 timestamp
    );

    /// @notice 仲裁员移除事件
    event ArbitratorRemoved(
        address indexed arbitrator,
        uint256 timestamp
    );

    // ============ 修饰符 ============

    modifier onlyOwner() {
        require(msg.sender == owner, "AITraceArbitration: caller is not owner");
        _;
    }

    modifier onlyArbitrator() {
        require(
            arbitrators[msg.sender].isActive,
            "AITraceArbitration: caller is not arbitrator"
        );
        _;
    }

    modifier disputeExists(uint256 disputeId) {
        require(disputeId > 0 && disputeId <= disputeCount, "AITraceArbitration: dispute not found");
        _;
    }

    // ============ 构造函数 ============

    constructor(address _registry) {
        owner = msg.sender;
        registry = AITraceRegistry(_registry);
        minStake = 0.01 ether;
        votingPeriod = 7 days;
        minArbitrators = 3;

        // 注册部署者为首个仲裁员
        _registerArbitrator(msg.sender);
    }

    // ============ 争议管理 ============

    /**
     * @notice 创建争议
     * @param certId 相关存证 ID
     * @param disputeType 争议类型
     * @param defendant 被告地址
     * @param description 争议描述
     * @param evidenceHash 证据哈希
     */
    function createDispute(
        bytes32 certId,
        DisputeType disputeType,
        address defendant,
        string calldata description,
        bytes32 evidenceHash
    ) external payable returns (uint256 disputeId) {
        require(msg.value >= minStake, "AITraceArbitration: insufficient stake");
        require(defendant != msg.sender, "AITraceArbitration: cannot dispute self");
        require(defendant != address(0), "AITraceArbitration: invalid defendant");

        // 验证存证存在
        require(
            registry.attestationExists(certId),
            "AITraceArbitration: attestation not found"
        );

        disputeCount++;
        disputeId = disputeCount;

        disputes[disputeId] = Dispute({
            disputeId: disputeId,
            certId: certId,
            disputeType: disputeType,
            status: DisputeStatus.Pending,
            plaintiff: msg.sender,
            defendant: defendant,
            description: description,
            evidenceHash: evidenceHash,
            createdAt: block.timestamp,
            votingDeadline: block.timestamp + votingPeriod,
            votesInFavor: 0,
            votesAgainst: 0,
            votesAbstain: 0,
            stake: msg.value,
            resolved: false,
            winner: address(0)
        });

        emit DisputeCreated(
            disputeId,
            certId,
            disputeType,
            msg.sender,
            defendant,
            msg.value
        );

        return disputeId;
    }

    /**
     * @notice 开始投票阶段
     * @param disputeId 争议 ID
     */
    function startVoting(uint256 disputeId) external onlyOwner disputeExists(disputeId) {
        Dispute storage dispute = disputes[disputeId];
        require(dispute.status == DisputeStatus.Pending, "AITraceArbitration: invalid status");
        require(arbitratorList.length >= minArbitrators, "AITraceArbitration: insufficient arbitrators");

        dispute.status = DisputeStatus.Voting;
        dispute.votingDeadline = block.timestamp + votingPeriod;
    }

    /**
     * @notice 仲裁员投票
     * @param disputeId 争议 ID
     * @param result 投票结果
     * @param reason 投票理由
     */
    function vote(
        uint256 disputeId,
        VoteResult result,
        string calldata reason
    ) external onlyArbitrator disputeExists(disputeId) {
        Dispute storage dispute = disputes[disputeId];
        require(dispute.status == DisputeStatus.Voting, "AITraceArbitration: not in voting");
        require(block.timestamp <= dispute.votingDeadline, "AITraceArbitration: voting ended");
        require(
            disputeVotes[disputeId][msg.sender].result == VoteResult.None,
            "AITraceArbitration: already voted"
        );
        require(result != VoteResult.None, "AITraceArbitration: invalid vote");

        // 仲裁员不能是当事人
        require(
            msg.sender != dispute.plaintiff && msg.sender != dispute.defendant,
            "AITraceArbitration: arbitrator is party"
        );

        disputeVotes[disputeId][msg.sender] = Vote({
            arbitrator: msg.sender,
            result: result,
            timestamp: block.timestamp,
            reason: reason
        });

        if (result == VoteResult.InFavor) {
            dispute.votesInFavor++;
        } else if (result == VoteResult.Against) {
            dispute.votesAgainst++;
        } else {
            dispute.votesAbstain++;
        }

        arbitrators[msg.sender].totalVotes++;

        emit DisputeVoted(disputeId, msg.sender, result);
    }

    /**
     * @notice 结算争议
     * @param disputeId 争议 ID
     */
    function resolveDispute(uint256 disputeId) external disputeExists(disputeId) {
        Dispute storage dispute = disputes[disputeId];
        require(dispute.status == DisputeStatus.Voting, "AITraceArbitration: not in voting");
        require(
            block.timestamp > dispute.votingDeadline ||
            _getTotalVotes(disputeId) >= arbitratorList.length,
            "AITraceArbitration: voting not ended"
        );
        require(!dispute.resolved, "AITraceArbitration: already resolved");

        uint256 totalVotes = dispute.votesInFavor + dispute.votesAgainst;

        // 需要足够的有效投票
        if (totalVotes < minArbitrators) {
            dispute.status = DisputeStatus.Expired;
            dispute.resolved = true;
            // 退还押金
            payable(dispute.plaintiff).transfer(dispute.stake);
            emit DisputeResolved(disputeId, DisputeStatus.Expired, address(0));
            return;
        }

        // 判定结果
        if (dispute.votesInFavor > dispute.votesAgainst) {
            // 原告胜诉
            dispute.winner = dispute.plaintiff;
            dispute.status = DisputeStatus.Resolved;
            // 退还押金给原告
            payable(dispute.plaintiff).transfer(dispute.stake);
        } else if (dispute.votesAgainst > dispute.votesInFavor) {
            // 被告胜诉
            dispute.winner = dispute.defendant;
            dispute.status = DisputeStatus.Rejected;
            // 押金归被告
            payable(dispute.defendant).transfer(dispute.stake);
        } else {
            // 平局，退还押金
            dispute.status = DisputeStatus.Resolved;
            payable(dispute.plaintiff).transfer(dispute.stake);
        }

        dispute.resolved = true;

        // 更新仲裁员信誉
        _updateArbitratorReputation(disputeId);

        emit DisputeResolved(disputeId, dispute.status, dispute.winner);
    }

    // ============ 仲裁员管理 ============

    /**
     * @notice 注册仲裁员
     * @param arbitrator 仲裁员地址
     */
    function registerArbitrator(address arbitrator) external onlyOwner {
        _registerArbitrator(arbitrator);
    }

    /**
     * @notice 内部注册仲裁员
     */
    function _registerArbitrator(address arbitrator) internal {
        require(arbitrator != address(0), "AITraceArbitration: invalid address");
        require(!arbitrators[arbitrator].isActive, "AITraceArbitration: already registered");

        arbitrators[arbitrator] = Arbitrator({
            addr: arbitrator,
            reputation: INITIAL_REPUTATION,
            totalVotes: 0,
            correctVotes: 0,
            isActive: true,
            registeredAt: block.timestamp
        });

        arbitratorList.push(arbitrator);

        emit ArbitratorRegistered(arbitrator, block.timestamp);
    }

    /**
     * @notice 移除仲裁员
     * @param arbitrator 仲裁员地址
     */
    function removeArbitrator(address arbitrator) external onlyOwner {
        require(arbitrators[arbitrator].isActive, "AITraceArbitration: not registered");

        arbitrators[arbitrator].isActive = false;

        // 从列表中移除
        for (uint256 i = 0; i < arbitratorList.length; i++) {
            if (arbitratorList[i] == arbitrator) {
                arbitratorList[i] = arbitratorList[arbitratorList.length - 1];
                arbitratorList.pop();
                break;
            }
        }

        emit ArbitratorRemoved(arbitrator, block.timestamp);
    }

    /**
     * @notice 更新仲裁员信誉
     */
    function _updateArbitratorReputation(uint256 disputeId) internal {
        Dispute storage dispute = disputes[disputeId];
        VoteResult winningVote = dispute.winner == dispute.plaintiff
            ? VoteResult.InFavor
            : VoteResult.Against;

        for (uint256 i = 0; i < arbitratorList.length; i++) {
            address arbAddr = arbitratorList[i];
            Vote storage v = disputeVotes[disputeId][arbAddr];

            if (v.result == VoteResult.None) continue;

            Arbitrator storage arb = arbitrators[arbAddr];

            if (v.result == winningVote) {
                // 投票正确，增加信誉
                arb.correctVotes++;
                if (arb.reputation < 200) {
                    arb.reputation += 5;
                }
            } else if (v.result != VoteResult.Abstain) {
                // 投票错误，降低信誉
                if (arb.reputation > 10) {
                    arb.reputation -= 3;
                }
            }
        }
    }

    // ============ 查询功能 ============

    /**
     * @notice 获取争议详情
     * @param disputeId 争议 ID
     */
    function getDispute(uint256 disputeId)
        external
        view
        disputeExists(disputeId)
        returns (Dispute memory)
    {
        return disputes[disputeId];
    }

    /**
     * @notice 获取仲裁员的投票记录
     * @param disputeId 争议 ID
     * @param arbitrator 仲裁员地址
     */
    function getVote(uint256 disputeId, address arbitrator)
        external
        view
        returns (Vote memory)
    {
        return disputeVotes[disputeId][arbitrator];
    }

    /**
     * @notice 获取仲裁员信息
     * @param arbitrator 仲裁员地址
     */
    function getArbitrator(address arbitrator)
        external
        view
        returns (Arbitrator memory)
    {
        return arbitrators[arbitrator];
    }

    /**
     * @notice 获取仲裁员数量
     */
    function getArbitratorCount() external view returns (uint256) {
        return arbitratorList.length;
    }

    /**
     * @notice 获取争议总投票数
     */
    function _getTotalVotes(uint256 disputeId) internal view returns (uint256) {
        Dispute storage dispute = disputes[disputeId];
        return dispute.votesInFavor + dispute.votesAgainst + dispute.votesAbstain;
    }

    // ============ 管理功能 ============

    /**
     * @notice 设置最小押金
     */
    function setMinStake(uint256 _minStake) external onlyOwner {
        minStake = _minStake;
    }

    /**
     * @notice 设置投票期限
     */
    function setVotingPeriod(uint256 _votingPeriod) external onlyOwner {
        votingPeriod = _votingPeriod;
    }

    /**
     * @notice 设置最小仲裁员数量
     */
    function setMinArbitrators(uint256 _minArbitrators) external onlyOwner {
        minArbitrators = _minArbitrators;
    }

    /**
     * @notice 更新存证注册合约地址
     */
    function setRegistry(address _registry) external onlyOwner {
        registry = AITraceRegistry(_registry);
    }

    /**
     * @notice 转移所有权
     */
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "AITraceArbitration: invalid address");
        owner = newOwner;
    }

    /**
     * @notice 紧急提取（仅用于紧急情况）
     */
    function emergencyWithdraw() external onlyOwner {
        payable(owner).transfer(address(this).balance);
    }
}
