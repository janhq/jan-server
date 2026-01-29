# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod file first
COPY apps/backend/go.mod ./apps/backend/

# Copy source code (needed for go mod tidy to analyze imports)
COPY apps/backend/ ./apps/backend/

# Download dependencies and generate go.sum
WORKDIR /build/apps/backend
RUN go mod tidy && go mod download

# Build the application
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X jan-server/mono/apps/backend/internal/infrastructure/config.Version=${VERSION}" \
    -o /build/jan-server \
    ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 jan && \
    adduser -u 1000 -G jan -s /bin/sh -D jan

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/jan-server .

# Copy migrations if they exist
COPY --from=builder /build/apps/backend/migrations ./migrations

# Set ownership
RUN chown -R jan:jan /app

# Switch to non-root user
USER jan

# Expose ports
EXPOSE 8080 9091

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

# Run the application
ENTRYPOINT ["/app/jan-server"]
