#!/bin/bash
# Hybrid Run: Media API (Unix)
# ---------------------------------------------
# Purpose: Run the media-api locally while infra runs in Docker.
# Mirrors hybrid-run-api.sh behavior.
# ---------------------------------------------
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"
source "$SCRIPT_DIR/lib/docker.sh"
source "$SCRIPT_DIR/lib/hybrid.sh"

print_header "Running Media API in Hybrid Mode"

if ! command_exists "go"; then
    print_error "Go is not installed"
    exit 1
fi

if check_service_in_docker "media-api"; then
    print_warning "Media API is running in Docker. Stop it first with:"
    print_info "  docker compose stop media-api"
    exit 1
fi

print_info "Checking infrastructure services..."
if ! docker compose ps | grep -qE "api-db.*Up"; then
    print_error "Infrastructure is not running. Start it with:"
    print_info "  docker compose --profile infra up -d"
    exit 1
fi

load_hybrid_env "media-api"

cd "$SCRIPT_DIR/../services/media-api"

print_info "Building Media API..."
go build -o bin/media-api .

print_success "Starting Media API on http://localhost:${MEDIA_API_PORT:-8285}"
print_info "Press Ctrl+C to stop"
echo ""

./bin/media-api
