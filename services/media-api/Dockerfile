ARG GO_VERSION=1.23

FROM golang:${GO_VERSION} as build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/media-api ./cmd/server

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/media-api /media-api
EXPOSE 8285
ENTRYPOINT ["/media-api"]
