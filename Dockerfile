# Multi-stage Dockerfile for SDBX Web UI
# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o sdbx \
    ./cmd/sdbx

# Stage 2: Runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    docker-cli \
    tzdata

# Create non-root user (will run as root in practice for Docker socket access)
RUN addgroup -g 1000 sdbx && \
    adduser -D -u 1000 -G sdbx sdbx

# Copy binary from builder
COPY --from=builder /app/sdbx /usr/local/bin/sdbx

# Create directories
RUN mkdir -p /project && \
    chown -R sdbx:sdbx /project

WORKDIR /project

# Set environment variables
ENV SDBX_MODE=server \
    SDBX_PROJECT_DIR=/project

# Expose web UI port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Run as root for Docker socket access (production deployment handles this via service.yaml)
USER root

# Start web UI server
CMD ["sdbx", "serve", "--host", "0.0.0.0", "--port", "3000"]
