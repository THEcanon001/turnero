# =============================================================================
# Multi-stage Dockerfile for Turnero API Server
# Stage 1: Build Go binary
# Stage 2: Minimal runtime image
# =============================================================================

# --- Builder stage ---
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /build/turnero \
    ./cmd/server

# --- Runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -S turnero && adduser -S turnero -G turnero

WORKDIR /opt/turnero

# Copy binary from builder
COPY --from=builder /build/turnero .

# Copy migrations (needed for auto-migration on startup)
COPY --from=builder /build/internal/platform/database/migrations ./migrations

# Create directories for data
RUN mkdir -p /var/data/qr /var/log/turnero && \
    chown -R turnero:turnero /opt/turnero /var/data/qr /var/log/turnero

USER turnero

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/opt/turnero/turnero"]
