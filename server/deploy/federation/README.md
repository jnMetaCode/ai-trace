# AI-Trace 联邦节点部署指南

联邦节点允许多个独立组织共同验证和存储 AI 决策存证，实现类似区块链的去中心化信任，但更轻量。

## 架构概述

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Node A        │◄───►│   Node B        │◄───►│   Node C        │
│   (公司 A)      │     │   (公司 B)      │     │   (审计机构)    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │  联邦共识 (min_confirm) │
                    │  至少 N 个节点确认      │
                    └─────────────────────────┘
```

## 快速开始

### 单节点测试

```bash
# 1. 启动基础服务
docker-compose -f docker-compose.yml up -d

# 2. 启动带联邦功能的节点
docker-compose -f docker-compose.federation.yml up -d
```

### 多节点部署

```bash
# 启动 3 节点联邦网络
docker-compose -f docker-compose.multi-node.yml up -d
```

## 配置说明

### 主节点配置 (config-node1.yaml)

```yaml
server:
  port: 8080
  mode: release

# ... 数据库等基础配置 ...

features:
  blockchain_anchor: false
  federated_nodes: true    # 启用联邦节点
  metrics: true
  reports: true

anchor:
  federated:
    enabled: true
    nodes:
      - http://node2:8080   # 其他联邦节点
      - http://node3:8080
    min_confirmations: 2    # 至少 2 个节点确认
```

### 节点发现

节点启动后会自动：
1. 生成 Ed25519 密钥对
2. 计算节点 ID（公钥前 8 字节）
3. 向配置的节点广播自己

查看节点信息：
```bash
curl http://localhost:8006/api/v1/federated/node/info
```

响应：
```json
{
  "node_id": "a1b2c3d4",
  "public_key": "302a300506032b6570032100...",
  "version": "0.1.0",
  "endpoints": ["/federated/confirm", "/federated/verify"],
  "features": ["anchor", "verify"]
}
```

## 存证流程

### 1. 用户发起存证请求

```bash
curl -X POST http://node1:8006/api/v1/certs/commit \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "trace_id": "trc_abc123",
    "evidence_level": "L3"
  }'
```

### 2. 联邦共识流程

```
Node1 收到请求
    │
    ├──► 构建锚定数据 (cert_id + root_hash + timestamp)
    │
    ├──► 用私钥签名
    │
    ├──► 并行请求 Node2, Node3 确认
    │         │
    │         ├──► Node2: 验证签名 → 签名确认 → 返回
    │         │
    │         └──► Node3: 验证签名 → 签名确认 → 返回
    │
    ├──► 收集确认（需要 >= min_confirmations）
    │
    └──► 返回锚定结果
```

### 3. 验证存证

```bash
curl http://node1:8006/api/v1/federated/verify/fed_abc123_1704067200
```

节点会向其他联邦节点查询验证。

## 节点类型

### 完整节点（Full Node）

- 存储所有事件和存证
- 参与联邦共识
- 提供 API 服务

```yaml
features:
  federated_nodes: true
  reports: true

database:
  host: postgres
  # ... 完整数据库配置
```

### 验证节点（Verifier Node）

- 只参与存证验证
- 不存储完整数据
- 轻量级部署

```yaml
features:
  federated_nodes: true
  reports: false

# 可选：使用 SQLite 或内存存储
database:
  type: sqlite
  path: /data/verifier.db
```

## 安全配置

### 节点间通信

建议使用 TLS 加密节点间通信：

```yaml
anchor:
  federated:
    nodes:
      - https://node2.company-b.com:8443
      - https://node3.audit-firm.com:8443
    tls:
      enabled: true
      cert_file: /certs/node.crt
      key_file: /certs/node.key
      ca_file: /certs/ca.crt
```

### 节点认证

每个节点使用 Ed25519 密钥对进行身份验证：

```yaml
anchor:
  federated:
    # 可选：指定密钥文件（否则自动生成）
    private_key_file: /keys/node.key
    public_key_file: /keys/node.pub
```

## 监控和运维

### 健康检查

```bash
# 检查节点健康
curl http://localhost:8006/health

# 检查联邦状态
curl http://localhost:8006/api/v1/federated/nodes
```

### Prometheus 指标

启用 metrics 后，可以监控：

- `ai_trace_anchor_operations_total{anchor_type="federated",status="success|failed"}`
- `ai_trace_federated_confirmations` - 确认数分布
- `ai_trace_federated_latency_seconds` - 联邦共识延迟

### 日志

```bash
# 查看联邦相关日志
docker logs ai-trace-node1 2>&1 | grep -i federated
```

## 故障处理

### 节点不可用

当部分节点不可用时：

1. 如果可用节点 >= min_confirmations，继续工作
2. 如果可用节点 < min_confirmations，存证失败
3. 建议设置 min_confirmations = (总节点数 / 2) + 1

### 网络分区

发生网络分区时：

1. 各分区独立工作
2. 分区恢复后自动同步
3. 冲突通过时间戳解决

## 最佳实践

1. **节点数量**: 建议 3-7 个节点，奇数个
2. **地理分布**: 节点分布在不同地区/云厂商
3. **组织多样性**: 包含不同组织（企业、审计机构、监管机构）
4. **确认数**: min_confirmations 设为 (n/2)+1 或更高
5. **备份**: 定期备份节点密钥和数据

## 常见问题

### Q: 联邦节点和区块链有什么区别？

| 特性 | 联邦节点 | 区块链 |
|------|---------|--------|
| 共识速度 | 毫秒级 | 秒/分钟级 |
| 节点数量 | 3-50 | 数千+ |
| 信任模型 | 已知节点 | 无需信任 |
| 成本 | 几乎为零 | Gas 费用 |
| 适用场景 | 企业联盟 | 公开验证 |

### Q: 如何加入已有的联邦网络？

1. 获取现有节点的 endpoint 列表
2. 配置 `anchor.federated.nodes`
3. 启动节点
4. 向现有节点注册：

```bash
curl -X POST http://existing-node:8006/api/v1/federated/nodes/register \
  -H "Content-Type: application/json" \
  -d '{"endpoint": "http://your-node:8006"}'
```

### Q: 数据如何同步？

联邦节点不需要完全同步数据。每个节点：
- 独立存储自己收到的存证请求
- 验证时向其他节点查询
- 通过签名确保数据一致性
