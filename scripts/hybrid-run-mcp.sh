#!/bin/bash
# Script to run MCP Tools service natively while infrastructure runs in Docker

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"
source "$SCRIPT_DIR/lib/docker.sh"
source "$SCRIPT_DIR/lib/hybrid.sh"

print_header "Running MCP Tools in Hybrid Mode"

# Check prerequisites
if ! command_exists "go"; then
    print_error "Go is not installed"
    exit 1
fi

# Check if MCP Tools is already running in Docker
if check_service_in_docker "mcp-tools"; then
    print_warning "MCP Tools is running in Docker. Stop it first with:"
    print_info "  docker compose --profile mcp stop mcp-tools"
    exit 1
fi

# Check if MCP infrastructure is running
print_info "Checking MCP infrastructure services..."
if ! docker compose --profile mcp ps | grep -q "vector-store.*running"; then
    print_error "MCP infrastructure is not running. Start it with:"
    print_info "  docker compose --profile mcp up -d searxng vector-store sandboxfusion"
    exit 1
fi

# Load hybrid environment
load_hybrid_env "mcp-tools"

# Navigate to service directory
cd "$SCRIPT_DIR/../services/mcp-tools"

print_info "Building MCP Tools..."
go build -o bin/mcp-tools .

print_success "Starting MCP Tools on http://localhost:8091"
print_info "Press Ctrl+C to stop"
echo ""

# Run the service
./bin/mcp-tools
