#!/bin/bash
# Hybrid development helper functions

source "$(dirname "$0")/common.sh"
source "$(dirname "$0")/docker.sh"

# Show hybrid environment variables for a service
show_hybrid_env() {
    local service=$1
    
    print_header "Environment Variables for $service (Hybrid Mode)"
    
    case "$service" in
        llm-api|api)
            cat << 'EOF'
export DATABASE_URL="postgres://jan_user:jan_password@localhost:5432/jan_llm_api?sslmode=disable"
export KEYCLOAK_BASE_URL="http://localhost:8085"
export JWKS_URL="http://localhost:8085/realms/jan/protocol/openid-connect/certs"
export ISSUER="http://localhost:8085/realms/jan"
export HTTP_PORT="8080"
export LOG_LEVEL="debug"
export LOG_FORMAT="console"
export AUTO_MIGRATE="true"
EOF
            ;;
        media-api|media)
            cat << 'EOF'
export MEDIA_DATABASE_URL="postgres://media:media@localhost:5432/media_api?sslmode=disable"
export MEDIA_SERVICE_KEY="changeme-media-key"
export MEDIA_API_KEY="changeme-media-key"
export MEDIA_API_PORT="8285"
export MEDIA_API_URL="http://localhost:8285"
export MEDIA_S3_ENDPOINT="https://s3.menlo.ai"
export MEDIA_S3_REGION="us-west-2"
export MEDIA_S3_BUCKET="platform-dev"
export MEDIA_S3_ACCESS_KEY="XXXXX"
export MEDIA_S3_SECRET_KEY="YYYY"
export MEDIA_S3_USE_PATH_STYLE="true"
export MEDIA_S3_PRESIGN_TTL="5m"
export MEDIA_MAX_BYTES="20971520"
export MEDIA_PROXY_DOWNLOAD="true"
export MEDIA_RETENTION_DAYS="30"
export MEDIA_REMOTE_FETCH_TIMEOUT="15s"
EOF
            ;;
        mcp-tools|mcp)
            cat << 'EOF'
export HTTP_PORT="8091"
export VECTOR_STORE_URL="http://localhost:3015"
export SEARXNG_URL="http://localhost:8086"
export SANDBOXFUSION_URL="http://localhost:3010"
export LOG_LEVEL="debug"
export LOG_FORMAT="console"
EOF
            ;;
        *)
            print_error "Unknown service: $service"
            print_info "Available services: llm-api, media-api, mcp-tools"
            return 1
            ;;
    esac
    
    echo ""
    print_info "Copy and paste the above export commands, or run:"
    print_info "  eval \"\$(make show-hybrid-env service=$service)\""
    echo ""
}

# Load hybrid environment for a service
load_hybrid_env() {
    local service=$1
    
    if [ -f "config/hybrid.env" ]; then
        print_info "Loading config/hybrid.env..."
        set -a
        source "config/hybrid.env"
        set +a
    fi
    
    # Override with localhost URLs
    export DATABASE_URL="postgres://jan_user:jan_password@localhost:5432/jan_llm_api?sslmode=disable"
    export KEYCLOAK_BASE_URL="http://localhost:8085"
    export JWKS_URL="http://localhost:8085/realms/jan/protocol/openid-connect/certs"
    export ISSUER="http://localhost:8085/realms/jan"
    
    case "$service" in
        llm-api|api)
            export HTTP_PORT="8080"
            export LOG_LEVEL="debug"
            ;;
        media-api|media)
            export MEDIA_DATABASE_URL="${MEDIA_DATABASE_URL:-postgres://media:media@localhost:5432/media_api?sslmode=disable}"
            export MEDIA_SERVICE_KEY="${MEDIA_SERVICE_KEY:-changeme-media-key}"
            export MEDIA_API_KEY="${MEDIA_API_KEY:-$MEDIA_SERVICE_KEY}"
            local media_port="${MEDIA_API_PORT:-8285}"
            export MEDIA_API_PORT="$media_port"
            export MEDIA_API_URL="${MEDIA_API_URL:-http://localhost:$media_port}"
            export MEDIA_S3_ENDPOINT="${MEDIA_S3_ENDPOINT:-https://s3.menlo.ai}"
            export MEDIA_S3_REGION="${MEDIA_S3_REGION:-us-west-2}"
            export MEDIA_S3_BUCKET="${MEDIA_S3_BUCKET:-platform-dev}"
            export MEDIA_S3_ACCESS_KEY="${MEDIA_S3_ACCESS_KEY:-XXXXX}"
            export MEDIA_S3_SECRET_KEY="${MEDIA_S3_SECRET_KEY:-YYYY}"
            export MEDIA_S3_USE_PATH_STYLE="${MEDIA_S3_USE_PATH_STYLE:-true}"
            export MEDIA_S3_PRESIGN_TTL="${MEDIA_S3_PRESIGN_TTL:-5m}"
            export MEDIA_MAX_BYTES="${MEDIA_MAX_BYTES:-20971520}"
            export MEDIA_PROXY_DOWNLOAD="${MEDIA_PROXY_DOWNLOAD:-true}"
            export MEDIA_RETENTION_DAYS="${MEDIA_RETENTION_DAYS:-30}"
            export MEDIA_REMOTE_FETCH_TIMEOUT="${MEDIA_REMOTE_FETCH_TIMEOUT:-15s}"
            export LOG_LEVEL="${LOG_LEVEL:-debug}"
            ;;
        mcp-tools|mcp)
            export HTTP_PORT="8091"
            export VECTOR_STORE_URL="http://localhost:3015"
            export SEARXNG_URL="http://localhost:8086"
            export SANDBOXFUSION_URL="http://localhost:3010"
            export LOG_LEVEL="debug"
            ;;
    esac
    
    print_success "Hybrid environment loaded for $service"
}

# Check if service is running in Docker
check_service_in_docker() {
    local service=$1
    docker ps --filter "name=$service" --format "{{.Names}}" | grep -q "$service"
}

export -f show_hybrid_env
export -f load_hybrid_env
export -f check_service_in_docker
