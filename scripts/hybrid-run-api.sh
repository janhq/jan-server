#!/bin/bash
# Script to run LLM API service natively while infrastructure runs in Docker

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"
source "$SCRIPT_DIR/lib/docker.sh"
source "$SCRIPT_DIR/lib/hybrid.sh"

print_header "Running LLM API in Hybrid Mode"

# Check prerequisites
if ! command_exists "go"; then
    print_error "Go is not installed"
    exit 1
fi

# Check if API is already running in Docker
if check_service_in_docker "llm-api"; then
    print_warning "LLM API is running in Docker. Stop it first with:"
    print_info "  docker compose stop llm-api"
    exit 1
fi

# Check if infrastructure is running
print_info "Checking infrastructure services..."
if ! docker compose ps | grep -q "postgres.*running"; then
    print_error "Infrastructure is not running. Start it with:"
    print_info "  docker compose --profile infra up -d"
    exit 1
fi

# Load hybrid environment
load_hybrid_env "llm-api"

# Navigate to service directory
cd "$SCRIPT_DIR/../services/llm-api"

print_info "Building LLM API..."
go build -o bin/llm-api .

print_success "Starting LLM API on http://localhost:8080"
print_info "Press Ctrl+C to stop"
echo ""

# Run the service
./bin/llm-api
