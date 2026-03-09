<p align="center">
  <img src="docs/assets/logo.svg" alt="AI-Trace Logo" width="120" height="120">
</p>

<h1 align="center">AI-Trace</h1>

<p align="center">
  <strong>企业级AI决策审计与防篡改存证系统</strong>
</p>

<p align="center">
  <a href="#特性">特性</a> •
  <a href="#快速开始">快速开始</a> •
  <a href="#文档">文档</a> •
  <a href="#sdk">SDK</a> •
  <a href="./README.md">English</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/license-AGPL--3.0-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/go-%3E%3D1.24-00ADD8.svg" alt="Go Version">
  <img src="https://img.shields.io/badge/node-%3E%3D20-339933.svg" alt="Node Version">
  <img src="https://img.shields.io/badge/python-%3E%3D3.9-3776AB.svg" alt="Python Version">
</p>

---

## 为什么选择 AI-Trace？

随着AI在企业决策中的应用日益广泛，组织面临越来越多的挑战：

| 挑战 | AI-Trace 解决方案 |
|------|------------------|
| **监管合规压力** | 为AI决策提供防篡改的审计证据 |
| **AI黑箱问题** | 完整透明的AI推理链路记录 |
| **数据篡改风险** | Merkle树 + 区块链锚定 |
| **隐私保护需求** | 最小披露证明（零知识） |

## 特性

```
┌─────────────────────────────────────────────────────────────┐
│                      AI-Trace 架构图                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   你的应用 ──→ AI-Trace 网关 ──→ LLM (OpenAI/Claude/Gemini)  │
│                      │                                      │
│                      ▼                                      │
│              ┌──────────────┐                               │
│              │   事件存储    │                               │
│              │ INPUT→MODEL  │                               │
│              │ →OUTPUT      │                               │
│              └──────┬───────┘                               │
│                     │                                       │
│                     ▼                                       │
│              ┌──────────────┐                               │
│              │  Merkle 树   │                               │
│              │   存证证书    │                               │
│              └──────┬───────┘                               │
│                     │                                       │
│          ┌─────────┼─────────┐                              │
│          ▼         ▼         ▼                              │
│    internal  compliance   legal                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 核心能力

- **全链路事件采集** - INPUT / MODEL / RETRIEVAL / TOOL_CALL / OUTPUT / POST_EDIT
- **防篡改存证** - 基于Merkle树的密码学哈希绑定
- **三级存证体系** - internal(内部) / compliance(合规) / legal(法律)
- **最小披露证明** - 只披露必要内容，保护其余数据
- **多模型代理** - 支持 OpenAI、Claude、Gemini 等，一行代码即可接入
- **开源验证器** - 独立离线验证

## 快速开始

### 方式一：Docker部署（推荐）

```bash
# 克隆项目
git clone https://github.com/jnMetaCode/ai-trace.git
cd ai-trace

# 启动所有服务
docker-compose up -d

# 查看状态
docker-compose ps
```

访问：
- 控制台：http://localhost:3006
- API：http://localhost:8006

### 方式二：本地开发

```bash
# 环境要求
# - Go 1.21+
# - Node.js 20+
# - PostgreSQL 15+
# - Redis 7+

# 1. 启动依赖服务
docker-compose up -d postgres redis minio

# 2. 运行后端
cd server
go run ./cmd/ai-trace-server

# 3. 运行前端（新终端）
cd console
npm install && npm run dev
```

## 使用方法

### 1. 代理AI请求（自动存证）

```bash
# 只需更改 OpenAI 的 base URL
curl -X POST http://localhost:8006/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "你好！"}]
  }'
```

### 2. 生成存证

```bash
curl -X POST http://localhost:8006/api/v1/certs/commit \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"trace_id": "trc_xxx", "evidence_level": "internal"}'
```

### 3. 验证存证

```bash
# 通过 API
curl -X POST http://localhost:8006/api/v1/certs/verify \
  -H "Content-Type: application/json" \
  -d '{"cert_id": "cert_xxx"}'

# 通过 CLI（离线验证）
ai-trace-verify --cert certificate.json
```

## SDK

### Python SDK

```bash
pip install ai-trace-sdk
```

```python
from ai_trace import AITraceClient

client = AITraceClient(
    base_url="http://localhost:8006",
    api_key="your-api-key"
)

# 搜索事件
events = client.events.search(trace_id="trc_xxx")

# 创建存证
cert = client.certs.commit(trace_id="trc_xxx", evidence_level="L1")

# 验证
result = client.certs.verify(cert_id=cert.cert_id)
print(f"验证结果: {result.valid}")
```

### OpenAI 无缝替换

```python
# 修改前（标准 OpenAI）
from openai import OpenAI
client = OpenAI(api_key="sk-...")

# 修改后（接入 AI-Trace）
from openai import OpenAI
client = OpenAI(
    api_key="sk-...",
    base_url="http://localhost:8006/api/v1"  # 只需加这一行
)

# 代码完全不用改
response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "你好！"}]
)
# 现在每个请求都会自动存证！
```

### Claude 接入

```python
import anthropic
client = anthropic.Anthropic(
    api_key="sk-ant-...",
    base_url="http://localhost:8006/api/v1"  # 只需加这一行
)
message = client.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=1024,
    messages=[{"role": "user", "content": "你好！"}]
)
# 每个请求都会自动存证！
```

## 存证级别

| 级别 | 时间证明 | 锚定方式 | 适用场景 | 成本 |
|------|---------|---------|---------|------|
| **internal** | 本地签名 | 本地数据库 | 内部审计 | 免费 |
| **compliance** | TSA时间戳 | WORM存储 | SOC2/GDPR/HIPAA | $ |
| **legal** | TSA时间戳 | 区块链 | 法律诉讼 | $$ |

## 项目结构

```
ai-trace/
├── server/          # Go 后端 (Gin + pgx)
├── console/         # React 前端 (Vite + Ant Design)
├── sdk/python/      # Python SDK
├── verifier/        # 开源验证器 CLI
├── docs/            # 文档
└── deploy/          # 部署配置
```

## 文档

- [快速开始指南](./docs/quick-start.md)
- [API 参考](./docs/api-reference.md)
- [SDK 指南](./docs/sdk-guide.md)
- [部署指南](./docs/deployment.md)
- [架构设计](./docs/architecture.md)

## 路线图

- [x] 核心事件采集与Merkle树
- [x] internal/compliance存证级别
- [x] Python SDK
- [x] CLI验证器
- [ ] legal区块链锚定
- [ ] Java/Go SDK
- [ ] 审计报告生成
- [ ] 多模型支持（Claude、Gemini）

## 贡献

欢迎贡献！请查看 [CONTRIBUTING.md](./CONTRIBUTING.md) 了解详情。

```bash
# 运行测试
make test

# 代码检查
make lint

# 格式化代码
make fmt
```

## 许可证

- **Server & Console**：[AGPL-3.0](./LICENSE)
- **SDK, Verifier, Schema**：[Apache-2.0](./sdk/LICENSE)

## 社区

- [GitHub Issues](https://github.com/jnMetaCode/ai-trace/issues) - Bug反馈与功能请求
- [Discussions](https://github.com/jnMetaCode/ai-trace/discussions) - 问答与想法
- [Twitter](https://twitter.com/ai_trace) - 动态与新闻

---

<p align="center">
  <strong>让AI决策可信、可验证</strong>
</p>
