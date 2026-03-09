# AI-Trace Makefile
# Common commands for development and deployment

.PHONY: help build run test clean docker-build docker-up docker-down lint fmt

# Default target
help:
	@echo "AI-Trace - Available commands:"
	@echo ""
	@echo "Development:"
	@echo "  make build      - Build the binary"
	@echo "  make run        - Run the server locally"
	@echo "  make test       - Run tests"
	@echo "  make lint       - Run linter"
	@echo "  make fmt        - Format code"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-up     - Start all services"
	@echo "  make docker-down   - Stop all services"
	@echo "  make docker-logs   - View logs"
	@echo "  make docker-clean  - Remove containers and volumes"
	@echo ""
	@echo "Database:"
	@echo "  make db-migrate    - Run database migrations"
	@echo "  make db-reset      - Reset database"
	@echo ""
	@echo "Production:"
	@echo "  make prod-up       - Start production services"
	@echo "  make prod-down     - Stop production services"

# Go commands
GO=go
BINARY_NAME=ai-trace
MAIN_PATH=./cmd/ai-trace
VERSION_PKG=github.com/ai-trace/server/internal/version

# Version info
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS = -w -s \
	-X '$(VERSION_PKG).Version=$(VERSION)' \
	-X '$(VERSION_PKG).GitCommit=$(COMMIT)' \
	-X '$(VERSION_PKG).BuildTime=$(BUILD_DATE)'

# Build the binary
build:
	$(GO) build -o bin/$(BINARY_NAME) $(MAIN_PATH)

# Build with version info
build-release:
	$(GO) build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) $(MAIN_PATH)

# Run the server
run: build
	./bin/$(BINARY_NAME) serve

# Run with hot reload (requires air)
dev:
	air -c .air.toml

# Run tests
test:
	$(GO) test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	$(GO) fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker commands
DOCKER_COMPOSE=docker compose

docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-clean:
	$(DOCKER_COMPOSE) down -v --remove-orphans

docker-restart:
	$(DOCKER_COMPOSE) restart ai-trace

# Production Docker commands
prod-up:
	$(DOCKER_COMPOSE) -f docker-compose.prod.yml up -d

prod-down:
	$(DOCKER_COMPOSE) -f docker-compose.prod.yml down

prod-logs:
	$(DOCKER_COMPOSE) -f docker-compose.prod.yml logs -f

# Database commands
db-migrate:
	$(GO) run $(MAIN_PATH) migrate up

db-reset:
	$(DOCKER_COMPOSE) exec postgres psql -U postgres -d ai_trace -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	$(DOCKER_COMPOSE) exec postgres psql -U postgres -d ai_trace -f /docker-entrypoint-initdb.d/init.sql

# Initialize project
init:
	./bin/$(BINARY_NAME) init
	cp .env.example .env
	@echo "Edit .env with your configuration"

# Generate API documentation
docs:
	swag init -g internal/api/router.go -o docs/swagger

# Install development tools
tools:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/tools/cmd/goimports@latest
	$(GO) install github.com/air-verse/air@latest
	$(GO) install github.com/swaggo/swag/cmd/swag@latest

# Check if all services are healthy
health:
	@echo "Checking services..."
	@curl -s http://localhost:8006/health | jq . || echo "AI-Trace: DOWN"
	@$(DOCKER_COMPOSE) exec -T postgres pg_isready -U postgres || echo "PostgreSQL: DOWN"
	@$(DOCKER_COMPOSE) exec -T redis redis-cli ping || echo "Redis: DOWN"
