# Multi-stage Dockerfile for Movie Rating API

# Stage 1: Builder
FROM golang:1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make protobuf-dev protoc

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/server \
    ./cmd/src

# Stage 2: Runtime
FROM alpine:3

# Install envsubst for environment variable substitution
RUN apk add --no-cache gettext

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /app/server
COPY --from=builder /workspace/configs /app/configs

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --spider -q http://localhost:8080/healthz || exit 1

# Use shell wrapper to substitute environment variables before starting app
ENTRYPOINT ["/bin/sh", "-c", "envsubst < /app/configs/config.yaml > /tmp/config.yaml && /app/server -conf /tmp"]
CMD []

