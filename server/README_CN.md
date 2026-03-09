# AI-Trace

<p align="center">
  <strong>企业级 AI 决策审计与合规平台</strong>
</p>

<p align="center">
  <a href="https://aitrace.cc">官网</a> |
  <a href="https://docs.aitrace.cc">文档</a> |
  <a href="#快速开始">快速开始</a> |
  <a href="./README.md">English</a>
</p>

---

## AI-Trace 是什么？

AI-Trace 为企业提供**零侵入**的 AI 调用追踪、存证和审计能力。它作为透明代理运行在你的应用和 LLM 提供商之间，自动记录每一次 AI 决策，满足合规和问责需求。

### 核心特性

- **零代码改动** - 只需替换 API 端点，无需集成 SDK
- **API Key 透传** - 你的密钥直接传递给上游服务商，本平台不存储
- **多级存证** - L1（本地签名）/ L2（WORM 存储）/ L3（区块链锚定）
- **Merkle 树证明** - 可加密验证的决策轨迹
- **最小披露** - 面向第三方的选择性披露证明
- **联邦化支持** - 跨多节点的去中心化验证

### 支持的 LLM 服务

| 服务商 | 状态 | 端点 |
|--------|------|------|
| OpenAI | ✅ 完整支持 | `/api/v1/chat/completions` |
| Claude | ✅ 完整支持 | `/api/v1/chat/completions` |
| Ollama | ✅ 完整支持 | `/api/v1/chat/completions` |
| Azure OpenAI | 🚧 即将支持 | - |

---

## 快速开始

### 方式一：Docker（推荐）

```bash
# 克隆项目
git clone https://github.com/jnMetaCode/ai-trace.git
cd server

# 使用 Docker Compose 启动
docker-compose up -d

# 验证运行状态
curl http://localhost:8006/health
```

### 方式二：源码编译

```bash
# 前置要求：Go 1.21+、PostgreSQL 15+、Redis 7+、MinIO

# 编译
go build -o ai-trace-server ./cmd/ai-trace-server

# 运行
./ai-trace-server
```

### 第一个追踪的 API 调用

```bash
# 将你的 OpenAI 端点替换为 AI-Trace
curl -X POST http://localhost:8006/api/v1/chat/completions \
  -H "X-API-Key: test-api-key-12345" \
  -H "X-Upstream-API-Key: sk-your-openai-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "你好！"}]
  }'
```

现在你的 API 调用已被追踪！获取存证证书：

```bash
# 为追踪生成存证证书
curl -X POST http://localhost:8006/api/v1/certs/commit \
  -H "X-API-Key: test-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{"trace_id": "trc_xxx", "evidence_level": "L2"}'
```

---

## 存证级别

| 级别 | 描述 | 信任模型 | 适用场景 |
|------|------|----------|----------|
| **L1** | 本地 Ed25519 签名 | 自签名 | 内部审计 |
| **L2** | WORM 存储 + TSA 时间戳 | 防篡改存储 | 监管合规 |
| **L3** | 区块链锚定 | 去中心化共识 | 法律举证 |

---

## API 文档

交互式 API 文档：

```
http://localhost:8006/swagger/index.html
```

### 核心接口

| 接口 | 方法 | 描述 |
|------|------|------|
| `/api/v1/chat/completions` | POST | OpenAI 兼容代理 |
| `/api/v1/events/ingest` | POST | 写入追踪事件 |
| `/api/v1/events/search` | GET | 搜索事件 |
| `/api/v1/certs/commit` | POST | 生成存证证书 |
| `/api/v1/certs/verify` | POST | 验证证书 |
| `/api/v1/certs/{id}/prove` | POST | 生成最小披露证明 |
| `/api/v1/reports/generate` | POST | 生成审计报告 |

---

## 联邦化部署

AI-Trace 支持跨多个独立节点的联邦化验证：

```bash
# 启动 3 节点联邦网络
cd deploy/federation
./start-federation.sh start

# 节点会自动发现并相互验证
# 证书需要 2 个以上节点确认
```

详见 [联邦化部署指南](./deploy/federation/README.md)。

---

## 配置

```yaml
# config.yaml
server:
  port: 8006
  mode: release

features:
  blockchain_anchor: false  # 需要 -tags blockchain 编译
  federated_nodes: true
  metrics: true
  reports: true

anchor:
  federated:
    enabled: true
    nodes:
      - http://node2:8006
      - http://node3:8006
    min_confirmations: 2
```

---

## SDK

官方 SDK：

- **Python**: `pip install ai-trace` - [文档](./sdk/python/)
- **JavaScript**: `npm install @ai-trace/sdk` - [文档](./sdk/javascript/)

### Python 示例

```python
from ai_trace import AITraceClient

client = AITraceClient(
    api_key="your-ai-trace-key",
    upstream_api_key="sk-your-openai-key"  # 透传，不存储
)

# 像使用 OpenAI 一样使用
response = client.chat.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "你好！"}]
)

# 获取存证证书
cert = client.certs.commit(trace_id=response.trace_id, evidence_level="L2")
print(f"证书 ID: {cert.cert_id}")
```

---

## 部署

详见 [部署指南](./docs/DEPLOYMENT.md)。

---

## 贡献

欢迎贡献！请查看 [CONTRIBUTING.md](./CONTRIBUTING.md)。

---

## 许可证

Apache License 2.0 - 详见 [LICENSE](./LICENSE)

---

<p align="center">
  <sub>为 AI 问责而生 ❤️</sub>
</p>
