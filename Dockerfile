# Multi-stage Dockerfile for SDBX Web UI
# Stage 1: Build the Go binary
FROM golang:1.25-alpine@sha256:8e02eb337d9e0ea459e041f1ee5eece41cbb61f1d83e7d883a3e2fb4862063fa AS builder

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
FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

# Install runtime dependencies and apply security patches
RUN apk --no-cache upgrade && \
    apk --no-cache add \
    ca-certificates \
    docker-cli \
    su-exec \
    tzdata

# Create non-root user
RUN addgroup -g 1000 sdbx && \
    adduser -D -u 1000 -G sdbx sdbx

# Copy binary from builder
COPY --from=builder /app/sdbx /usr/local/bin/sdbx

# Copy entrypoint script that dynamically matches Docker socket GID
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

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

# Entrypoint runs as root to match Docker socket GID, then drops to sdbx user
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["sdbx", "serve", "--host", "0.0.0.0", "--port", "3000"]
