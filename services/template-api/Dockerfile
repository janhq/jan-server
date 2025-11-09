ARG GO_VERSION=1.23

FROM golang:${GO_VERSION} as build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/template-api ./cmd/server

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/template-api /template-api
EXPOSE 8185
ENTRYPOINT ["/template-api"]
