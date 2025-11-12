.PHONY: install
install:
	@echo "🔧 Installing tools and setting up hooks..."
	@go run scripts/install_hooks.go

.PHONY: doc
doc:
	@swag init --parseDependency -g cmd/server/server.go -o swagger

.PHONY: wire
wire:
	@wire ./cmd/server

.PHONY: gormgen
gormgen:
	@echo "🔄 Generating GORM gen code..."
	@go run cmd/gormgen/gormgen.go

.PHONY: setup
setup:
	@make doc
	@make wire
