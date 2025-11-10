SERVICE := response-api
BIN := bin/$(SERVICE)

.PHONY: run build test tidy wire swagger clean

run:
	@echo "Starting $(SERVICE)..."
	go run ./cmd/server

build:
	go build -o $(BIN) ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

wire:
	@if ! command -v wire >/dev/null 2>&1; then echo "wire not installed (go install github.com/google/wire/cmd/wire@latest)"; exit 1; fi
	wire ./cmd/server

swagger:
	@if ! command -v swag >/dev/null 2>&1; then echo "swag CLI not installed (go install github.com/swaggo/swag/cmd/swag@latest)"; exit 1; fi
	swag init -g cmd/server/server.go -o docs/swagger

clean:
	rm -rf $(BIN)
