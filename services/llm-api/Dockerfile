FROM golang:1.22 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/llm-api ./

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/llm-api /app/llm-api
COPY config /app/config
WORKDIR /app
USER nonroot:nonroot
ENTRYPOINT ["/app/llm-api"]
