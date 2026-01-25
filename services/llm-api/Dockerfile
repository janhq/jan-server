ARG GO_VERSION=1.25

FROM golang:${GO_VERSION} AS builder

WORKDIR /src

# Copy go-common from named build context to match replace directive path
# replace github.com/janhq/jan-server/packages/go-common => ../../packages/go-common
# From /src, ../../packages/go-common resolves to /packages/go-common
COPY --from=go-common . /packages/go-common

COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/llm-api ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*
RUN useradd --system --home /app --no-create-home --uid 10001 appuser
WORKDIR /app
RUN mkdir -p /app/.cache && chown appuser:appuser /app/.cache
COPY --from=builder /out/llm-api /app/llm-api
COPY configs /app/configs
COPY docs /app/docs
COPY migrations /app/migrations
USER appuser
ENTRYPOINT ["/app/llm-api"]
