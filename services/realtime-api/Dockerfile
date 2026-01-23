ARG GO_VERSION=1.25

FROM golang:${GO_VERSION} as build

WORKDIR /src

# Copy go-common from named build context to match replace directive path
COPY --from=go-common . /packages/go-common

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/realtime-api ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*
RUN useradd --system --home /app --no-create-home --uid 10004 realtime
WORKDIR /app
COPY --from=build /out/realtime-api /app/realtime-api
EXPOSE 8186
USER realtime
ENTRYPOINT ["/app/realtime-api"]
