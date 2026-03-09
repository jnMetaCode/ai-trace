# Contributing to AI-Trace

Thank you for your interest in contributing to AI-Trace! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone.

## How to Contribute

### Reporting Issues

1. **Search existing issues** first to avoid duplicates
2. Use the issue template when available
3. Include:
   - Clear description of the problem
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version, etc.)

### Submitting Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Follow the coding style** described below
3. **Write tests** for new functionality
4. **Update documentation** if needed
5. **Sign your commits** (see DCO below)

## Development Setup

### Prerequisites

- Go 1.24+
- Docker and Docker Compose
- Python 3.9+ (for SDK development)
- Make (optional but recommended)

### Quick Start

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/ai-trace.git
cd ai-trace/server

# Add upstream remote
git remote add upstream https://github.com/jnMetaCode/ai-trace.git

# Install Go dependencies
go mod download

# Copy config
cp config.yaml.example config.yaml
cp .env.example .env

# Start all services with Docker
docker compose up -d

# Or run the server directly (requires running PostgreSQL, Redis, MinIO)
make run
```

### Using Make

```bash
make help          # Show all available commands
make build         # Build the binary
make test          # Run tests
make lint          # Run linter
make docker-up     # Start Docker services
make docker-down   # Stop Docker services
```

### Python SDK Development

```bash
cd sdk/python

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install in development mode
pip install -e ".[dev]"

# Run tests
pytest -v

# Run linter
ruff check ai_trace
mypy ai_trace
```

## Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Use `golint` and `go vet` for linting
- Keep functions focused and small
- Write descriptive variable names

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code change
- `refactor`: Code change without feature/fix
- `test`: Adding tests
- `chore`: Maintenance tasks

Example:
```
feat(federation): add signature verification for confirm requests

- Add timestamp validation (±5 minutes)
- Implement Ed25519 signature verification
- Add trusted node registry

Closes #123
```

### Code Structure

```
ai-trace/server/
├── cmd/
│   └── ai-trace/          # CLI application
│       └── cmd/           # Cobra commands
├── internal/              # Private packages
│   ├── api/               # HTTP handlers and router
│   ├── anchor/            # Anchoring (federation, blockchain)
│   ├── cert/              # Certificate management
│   ├── config/            # Configuration loading
│   ├── gateway/           # LLM provider proxy
│   ├── merkle/            # Merkle tree (standard & incremental)
│   ├── middleware/        # HTTP middleware (auth, logging)
│   ├── queue/             # Message queue handling
│   ├── report/            # Audit report generation
│   ├── store/             # Data storage (PostgreSQL, Redis, MinIO)
│   └── version/           # Version information
├── sdk/
│   └── python/            # Python SDK
├── examples/              # Usage examples
├── docs/                  # Documentation
├── scripts/               # Build and release scripts
└── deploy/                # Deployment configs (K8s, etc.)
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/merkle/...

# Run integration tests
go test -tags=integration ./...
```

## Pull Request Process

1. **Create a feature branch**
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make your changes** with clear commits

3. **Run tests and linting**
   ```bash
   go test ./...
   go vet ./...
   golint ./...
   ```

4. **Push and create PR**
   ```bash
   git push origin feat/my-feature
   ```

5. **Wait for review** - maintainers will review your PR

6. **Address feedback** - make requested changes

7. **Merge** - once approved, your PR will be merged

## Developer Certificate of Origin (DCO)

By contributing, you certify that:

1. The contribution was created by you
2. You have the right to submit it under the project license
3. You understand the contribution is public

Sign your commits with `git commit -s`:

```
Signed-off-by: Your Name <your.email@example.com>
```

## Areas for Contribution

### High Priority

- [ ] Azure OpenAI provider support
- [ ] Additional blockchain anchoring (Polygon, BSC)
- [ ] Enhanced audit report templates
- [ ] Performance optimizations
- [ ] Documentation improvements

### Good First Issues

Look for issues labeled `good first issue` - these are suitable for newcomers.

### Documentation

- Improve API documentation
- Add usage examples
- Translate documentation
- Write tutorials

## Getting Help

- **GitHub Issues**: For bugs and features
- **Discussions**: For questions and ideas
- **Email**: contributors@aitrace.cc

## Recognition

Contributors are recognized in:
- CONTRIBUTORS.md file
- Release notes
- Project website

Thank you for contributing to AI-Trace!
