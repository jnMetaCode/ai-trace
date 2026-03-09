# AI-Trace Server 编译指南

本文档说明如何编译 AI-Trace Server 的不同版本。

## 快速开始

### 基础版本（推荐大多数用户）

```bash
# 标准编译 - 不包含区块链和监控依赖
go build -o ai-trace-server ./cmd/server
```

这个版本包含：
- 完整的事件追踪和存证功能
- internal (本地签名) 和 compliance (WORM存储) 存证级别
- OpenAI/Claude/Ollama 代理网关
- 联邦化验证节点支持
- 报告生成功能

**不包含**（以减少二进制大小和依赖）：
- 以太坊区块链锚定 (legal级别)
- Prometheus 监控指标

## 可选功能

### 启用区块链支持

```bash
# 编译时启用以太坊区块链锚定功能
go build -tags blockchain -o ai-trace-server ./cmd/server
```

启用后增加：
- 以太坊主网/测试网锚定
- Polygon 网络支持
- 智能合约交互（可选）

**注意**：会增加约 100MB 的二进制大小（go-ethereum 依赖较大）

### 启用 Prometheus 监控

```bash
# 编译时启用 Prometheus 指标
go build -tags metrics -o ai-trace-server ./cmd/server
```

启用后增加：
- `/metrics` 端点
- HTTP 请求延迟和计数
- 事件处理统计
- LLM 调用和 Token 使用统计
- 证书生成统计

### 启用所有功能

```bash
# 编译完整版本
go build -tags "blockchain metrics" -o ai-trace-server ./cmd/server
```

## 配置文件

编译后，通过 `config.yaml` 启用/禁用功能：

```yaml
features:
  # 区块链锚定（需要 -tags blockchain 编译）
  blockchain_anchor: false
  # 联邦化验证节点
  federated_nodes: false
  # Prometheus 监控（需要 -tags metrics 编译）
  metrics: false
  # 报告生成
  reports: true

anchor:
  ethereum:
    enabled: false
    rpc_url: "https://mainnet.infura.io/v3/YOUR_KEY"
    private_key: ""  # 或使用环境变量 ANCHOR_ETHEREUM_PRIVATE_KEY
    chain_id: 1      # 1=Mainnet, 5=Goerli, 11155111=Sepolia
```

## 编译矩阵

| 版本 | 命令 | 二进制大小 | 适用场景 |
|------|------|-----------|----------|
| 基础版 | `go build ./cmd/server` | ~30MB | 大多数用户 |
| +监控 | `go build -tags metrics ./cmd/server` | ~35MB | 需要 Prometheus 监控 |
| +区块链 | `go build -tags blockchain ./cmd/server` | ~130MB | 需要 legal 区块链存证 |
| 完整版 | `go build -tags "blockchain metrics" ./cmd/server` | ~135MB | 企业级部署 |

## Docker 构建

### 基础镜像

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ai-trace-server ./cmd/server

FROM alpine:latest
COPY --from=builder /app/ai-trace-server /usr/local/bin/
CMD ["ai-trace-server"]
```

### 完整版镜像

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -tags "blockchain metrics" -o ai-trace-server ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/ai-trace-server /usr/local/bin/
CMD ["ai-trace-server"]
```

## 依赖说明

### 基础依赖
- `github.com/gin-gonic/gin` - Web 框架
- `github.com/jackc/pgx/v5` - PostgreSQL 驱动
- `github.com/redis/go-redis/v9` - Redis 客户端
- `github.com/minio/minio-go/v7` - MinIO/S3 客户端
- `go.uber.org/zap` - 日志库

### 区块链依赖（可选）
- `github.com/ethereum/go-ethereum` - 以太坊客户端

### 监控依赖（可选）
- `github.com/prometheus/client_golang` - Prometheus 客户端

## 验证编译

```bash
# 检查二进制是否包含区块链支持
./ai-trace-server --version
# 输出示例:
# AI-Trace Server v0.2.0
# Build tags: blockchain, metrics

# 或者检查 /health 端点
curl http://localhost:8006/health
# {
#   "status": "healthy",
#   "version": "0.2.0",
#   "features": {
#     "blockchain": true,
#     "metrics": true
#   }
# }
```

## 常见问题

### Q: 为什么要分离区块链依赖？

A: `go-ethereum` 包含大量的加密和网络库，会显著增加二进制大小（约 +100MB）。大多数用户可能只需要 internal/compliance 存证级别，不需要区块链功能。通过 build tags 实现条件编译，用户可以按需选择。

### Q: 如何在不重新编译的情况下禁用功能？

A: 功能开关通过配置文件控制。例如，即使编译时包含了区块链支持，也可以在 `config.yaml` 中设置 `anchor.ethereum.enabled: false` 来禁用。

### Q: Prometheus 监控会影响性能吗？

A: 几乎不会。Prometheus 客户端使用原子操作和内存映射，开销极低。如果你确实不需要监控，可以通过配置禁用或不编译此功能。
