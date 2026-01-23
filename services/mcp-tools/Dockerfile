ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

# Copy go-common from named build context to match replace directive path
COPY --from=go-common . /packages/go-common

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mcp-tools ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/mcp-tools .
COPY configs /app/configs

EXPOSE 8091

CMD ["./mcp-tools"]
