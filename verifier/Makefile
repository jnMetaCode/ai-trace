BINARY  := ai-trace-verify
MODULE  := github.com/ai-trace/ai-trace-verify
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test install clean vet fmt lint

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/ai-trace-verify

test:
	go test -v -race ./...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/ai-trace-verify

clean:
	rm -rf bin/ dist/

vet:
	go vet ./...

fmt:
	gofmt -s -w .

lint: vet
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi
