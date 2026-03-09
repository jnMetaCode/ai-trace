# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-01-15

### Added

#### Core Features
- **Zero-intrusion LLM Proxy**: OpenAI-compatible API proxy that requires no code changes
- **API Key Passthrough**: Upstream API keys are passed directly to providers, never stored
- **Multi-provider Support**: OpenAI, Claude, and Ollama support via unified endpoint

#### CLI Tool (`ai-trace`)
- `ai-trace init` - Initialize configuration
- `ai-trace serve` - Start the server
- `ai-trace trace create/list/get/events/add-event` - Trace management
- `ai-trace cert commit/verify/get/list/export/prove` - Certificate management
- `ai-trace version` - Display version information

#### Python SDK (`ai-trace`)
- Full client library with Pydantic models
- OpenAI integration for automatic tracing
- Async support
- Comprehensive exception handling

#### Evidence & Certification
- **Three-level Evidence System**:
  - L1: Local Ed25519 signatures for internal audit
  - L2: WORM storage with TSA timestamps for regulatory compliance
  - L3: Blockchain anchoring for legal evidence (optional)
- **Merkle Tree Proofs**: Cryptographically verifiable decision trails
- **Minimal Disclosure Proofs**: Selective disclosure for third-party verification
- **Certificate Generation**: Automated certificate creation with digital signatures

#### Federation
- **Multi-node Federation**: Decentralized verification across independent nodes
- **Signature Verification**: Ed25519 signature validation between federated nodes
- **Trust Management**: APIs for managing trusted node relationships
- **Automatic Discovery**: Node discovery and registration protocols

#### API & Integration
- **OpenAI-compatible Endpoint**: `/api/v1/chat/completions`
- **Event Ingestion API**: `/api/v1/events/ingest`
- **Certificate APIs**: Commit, verify, and search certificates
- **Proof Generation**: Minimal disclosure proof generation
- **Report Generation**: Audit report generation in HTML/PDF formats
- **Swagger Documentation**: Interactive API documentation

#### Infrastructure
- **PostgreSQL Storage**: Primary data storage with optimized indexes
- **Redis Caching**: High-performance caching and rate limiting
- **MinIO Integration**: WORM-compliant object storage for L2 evidence
- **Docker Support**: Docker Compose for development and production
- **Kubernetes Ready**: K8s manifests for production deployment
- **GitHub Actions**: CI/CD pipelines for testing and releases
- **Multi-platform Builds**: Linux, macOS, and Windows binaries

#### Security
- **Rate Limiting**: Token bucket algorithm with per-IP and per-key limits
- **CORS Middleware**: Configurable cross-origin request handling
- **Security Headers**: XSS protection, clickjacking prevention
- **Request ID Tracking**: Distributed tracing support

#### Monitoring
- **Prometheus Metrics**: Request counts, latency histograms, error rates
- **Health Endpoint**: `/health` for load balancer probes
- **Structured Logging**: JSON logging with zap

### Technical Details

- Go 1.24+ required (due to gnark/go-ethereum dependencies)
- PostgreSQL 15+ for database
- Redis 7+ for caching
- MinIO for object storage
- Optional blockchain support via build tags

### Known Limitations

- Azure OpenAI support not yet available
- Blockchain anchoring requires separate build tag
- Maximum request size: 10MB

## [0.2.0] - 2025-01-15

### Added

- **Simple Mode Deployment**: New single-file SQLite mode for quick setup
  - No external dependencies required (PostgreSQL, Redis, MinIO)
  - Single `docker-compose.simple.yml` for instant deployment
  - Ideal for development, testing, and small teams
  - Configurable via `DEPLOY_MODE=simple` environment variable or `deploy_mode: simple` in config

- **Human-Readable Response Headers**: New headers for better developer experience
  - `X-AI-Trace-Summary`: Brief description of what was captured
  - `X-AI-Trace-Events`: Number of events recorded
  - `X-AI-Trace-Hash`: Payload hash for verification
  - `X-AI-Trace-Hint`: Suggested next API call for certificate generation

- **Auto-Certificate Evaluation**: Automatic certificate generation triggers
  - Model-based triggers (e.g., auto-cert for GPT-4, Claude-3-Opus)
  - Token count thresholds (e.g., auto-cert for responses with 1000+ tokens)
  - Configurable via `auto_cert` settings in config.yaml

- **Enhanced API UX**: Self-documenting API responses
  - `next_steps` array in all major responses with actionable guidance
  - `summary` section in verification responses for non-technical stakeholders
  - Improved error messages with `message` and `suggestions` fields
  - Copy-paste ready curl commands in `X-AI-Trace-Hint` header
  - Pagination metadata (`total_count`, `total_pages`, `has_more`) in list endpoints

- **Getting Started Endpoint**: New `/api/v1/getting-started` endpoint
  - Step-by-step onboarding guide
  - Complete curl examples for core workflows
  - Evidence level descriptions and use cases

- **Health Endpoint Improvements**: Enhanced `/health` response
  - `deploy_mode` field (simple/standard)
  - `links` section with documentation URLs

### Changed

- **Evidence Level Naming**: Business-friendly names for better clarity
  - `internal` (formerly L1): Ed25519 signature, instant, for internal audit
  - `compliance` (formerly L2): WORM storage + TSA timestamp, for SOC2/GDPR/HIPAA
  - `legal` (formerly L3): Blockchain anchoring, for legal evidence
  - Legacy L1/L2/L3 names still supported for backward compatibility

- Updated all documentation with new naming conventions
- Configuration now supports `deploy_mode: simple | standard`
- SQLite path configurable via `sqlite.path` or `SQLITE_PATH` env var

### Fixed

- Data race in batch processing tests (`batch_test.go`)
- CLI build errors with missing dependencies

### Security

- KEK (Key Encryption Key) now required in production mode
- Clear warnings when using default keys in development mode

## [Unreleased]

### Planned
- Azure OpenAI provider support
- Additional blockchain networks (Polygon, BSC)
- Enhanced audit report templates
- GraphQL API
- Real-time event streaming
- Multi-tenant isolation improvements
- JavaScript/TypeScript SDK
- Kubernetes Helm charts for Simple Mode
- Web console for certificate management
