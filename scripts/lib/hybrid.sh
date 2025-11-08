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
            print_info "Available services: llm-api, mcp-tools"
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
