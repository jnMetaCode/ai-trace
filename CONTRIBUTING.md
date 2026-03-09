# Contributing to AI-Trace

Thank you for your interest in contributing to AI-Trace! This guide will help you get started.

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL 15+ (or use Docker)
- Redis 7+ (or use Docker)

### Development Setup

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/ai-trace.git
cd ai-trace

# 2. Start infrastructure
docker-compose up -d postgres redis minio

# 3. Copy environment config
cp .env.example .env

# 4. Run backend
cd server
go mod download
go run ./cmd/ai-trace-server

# 5. Run frontend (new terminal)
cd console
npm install
npm run dev
```

### Useful Make Commands

```bash
make build          # Build server binary
make test           # Run all tests
make test-coverage  # Run tests with coverage report
make lint           # Run linter
make fmt            # Format code
make run-dev        # Run with hot reload
make docker-up      # Start all services via Docker
```

## How to Contribute

### Reporting Bugs

- Search [existing issues](https://github.com/jnMetaCode/ai-trace/issues) first
- Use the bug report template
- Include reproduction steps, expected vs actual behavior
- Include Go/Node version, OS, and relevant logs

### Suggesting Features

- Open a [Discussion](https://github.com/jnMetaCode/ai-trace/discussions) first
- Describe the use case, not just the solution
- Check existing issues and discussions for planned features

### Submitting Code

1. **Fork** the repository
2. **Create a branch** from `main`: `git checkout -b feat/your-feature`
3. **Make changes** and add tests
4. **Run checks**: `make fmt && make lint && make test`
5. **Commit** with a clear message (see below)
6. **Push** and open a Pull Request

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add streaming support for event capture
fix: resolve certificate verification timeout
docs: update SDK quick start guide
test: add Merkle tree edge case tests
refactor: simplify event store interface
```

### Pull Request Guidelines

- Keep PRs focused — one feature or fix per PR
- Add tests for new functionality
- Update docs if you change public APIs
- Ensure CI passes before requesting review
- Link related issues with `Closes #123`

## Project Structure

```
ai-trace/
├── server/              # Go backend
│   ├── cmd/             # Entry points
│   └── internal/        # Core packages
│       ├── gateway/     # AI request proxy
│       ├── event/       # Event capture & storage
│       ├── merkle/      # Merkle tree implementation
│       ├── cert/        # Certificate generation
│       ├── anchor/      # Blockchain anchoring
│       ├── zkp/         # Zero-knowledge proofs
│       └── middleware/   # Auth, rate limiting
├── console/             # React frontend
├── sdk/                 # Multi-language SDKs
├── verifier/            # Standalone CLI verifier
├── contracts/           # Smart contracts
└── docs/                # Documentation
```

## Code Style

### Go
- Follow standard Go conventions (`gofmt`, `go vet`)
- Use `golangci-lint` for additional checks
- Error messages should be lowercase, no trailing punctuation
- Add comments for exported functions

### TypeScript/React
- Follow the existing ESLint config
- Use functional components with hooks
- Use TypeScript strict mode

## License

By contributing, you agree that your contributions will be licensed under:
- **Apache 2.0** for SDK, verifier, and schema components
- **AGPL-3.0** for server and console components

## Questions?

- [GitHub Discussions](https://github.com/jnMetaCode/ai-trace/discussions)
- [Issues](https://github.com/jnMetaCode/ai-trace/issues)

Thank you for helping make AI decisions trustworthy and verifiable!
