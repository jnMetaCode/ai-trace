# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /ai-trace ./cmd/ai-trace

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 aitrace && \
    adduser -u 1000 -G aitrace -s /bin/sh -D aitrace

WORKDIR /app

# Copy binary from builder
COPY --from=builder /ai-trace /usr/local/bin/ai-trace

# Copy default config
COPY config.yaml.example /app/config.yaml

# Create data directories
RUN mkdir -p /app/data /app/logs && \
    chown -R aitrace:aitrace /app

# Switch to non-root user
USER aitrace

# Expose port
EXPOSE 8006

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8006/health || exit 1

# Default command
CMD ["ai-trace", "serve"]
