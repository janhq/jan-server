FROM golang:1.23 AS builder
WORKDIR /src
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
COPY --from=builder /out/llm-api /app/llm-api
COPY config /app/config
COPY docs /app/docs
USER appuser
ENTRYPOINT ["/app/llm-api"]
