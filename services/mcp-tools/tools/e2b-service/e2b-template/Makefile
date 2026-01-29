# E2B Template Build Makefile

.PHONY: build clean help init list

# Path to jan-browser-extension submodule dist
EXTENSION_DIST := ../../../packages/jan-browser-extension/dist

help:
	@echo "E2B Template Build Commands:"
	@echo "  make build     - Build E2B template (requires E2B_ACCESS_TOKEN)"
	@echo "  make init      - Initialize fresh template (first time only)"
	@echo "  make list      - List available E2B templates"
	@echo "  make clean     - Clean up temporary files"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - E2B_ACCESS_TOKEN environment variable must be set"
	@echo "  - jan-browser-extension must be built (dist/ exists)"
	@echo ""
	@echo "Template includes:"
	@echo "  - Chromium browser with jan-browser-mcp extension"
	@echo "  - search-mcp-server for dynamic browser automation tools"
	@echo "  - mcp-proxy HTTP bridge on port 17390"

build:
	@echo "=== Building E2B Template ==="
	@echo ""
	@echo "Checking prerequisites..."
	@if [ -z "$(E2B_ACCESS_TOKEN)" ]; then \
		echo "ERROR: E2B_ACCESS_TOKEN environment variable is not set"; \
		exit 1; \
	fi
	@if [ ! -d "$(EXTENSION_DIST)" ]; then \
		echo "ERROR: jan-browser-extension dist not found at $(EXTENSION_DIST)"; \
		echo "Please build the extension first:"; \
		echo "  cd ../../../packages/jan-browser-extension && npm install && npm run build"; \
		exit 1; \
	fi
	@echo ""
	@echo "Copying browser extension from submodule..."
	@rm -rf jan-browser-mcp
	@cp -r $(EXTENSION_DIST) ./jan-browser-mcp
	@echo "Extension copied to ./jan-browser-mcp"
	@echo ""
	@echo "Building E2B template..."
	E2B_ACCESS_TOKEN=$(E2B_ACCESS_TOKEN) e2b template build
	@echo ""
	@echo "Cleaning up temporary files..."
	@rm -rf jan-browser-mcp
	@echo ""
	@echo "=== Build Complete ==="

init:
	@echo "=== Initializing E2B Template ==="
	@if [ -z "$(E2B_ACCESS_TOKEN)" ]; then \
		echo "ERROR: E2B_ACCESS_TOKEN environment variable is not set"; \
		exit 1; \
	fi
	E2B_ACCESS_TOKEN=$(E2B_ACCESS_TOKEN) e2b template init
	@echo "=== Init Complete ==="

list:
	@echo "=== Listing E2B Templates ==="
	@if [ -z "$(E2B_ACCESS_TOKEN)" ]; then \
		echo "ERROR: E2B_ACCESS_TOKEN environment variable is not set"; \
		exit 1; \
	fi
	E2B_ACCESS_TOKEN=$(E2B_ACCESS_TOKEN) e2b template list
	@echo "=== List Complete ==="

clean:
	@echo "Cleaning up..."
	@rm -rf jan-browser-mcp
	@echo "Done"
