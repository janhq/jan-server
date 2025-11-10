ARG GO_VERSION=1.23

FROM golang:${GO_VERSION} as build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/response-api ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*
RUN useradd --system --home /app --no-create-home --uid 10003 response
WORKDIR /app
COPY --from=build /out/response-api /app/response-api
EXPOSE 8082
USER response
ENTRYPOINT ["/app/response-api"]
