#!/bin/bash
# Health check helper functions

source "$(dirname "$0")/common.sh"

# Check HTTP endpoint
check_http_endpoint() {
    local name=$1
    local url=$2
    local max_retries=${3:-30}
    local interval=2
    
    for i in $(seq 1 $max_retries); do
        if curl -sf "$url" >/dev/null 2>&1; then
            print_success "$name is healthy ($url)"
            return 0
        fi
        
        if [ $i -lt $max_retries ]; then
            echo -n "."
            sleep $interval
        fi
    done
    
    echo ""
    print_error "$name is not responding ($url)"
    return 1
}

# Check all services
check_all_services() {
    local services=(
        "API:http://localhost:8080/healthz"
        "Keycloak:http://localhost:8085"
        "Kong:http://localhost:8000"
    )
    
    local failed=0
    print_header "Checking Service Health"
    
    for service in "${services[@]}"; do
        local name="${service%%:*}"
        local url="${service#*:}"
        
        if ! check_http_endpoint "$name" "$url" 3; then
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        print_success "All services are healthy"
        return 0
    else
        print_error "Some services are not healthy"
        return 1
    fi
}

# Check MCP services
check_mcp_services() {
    local services=(
        "MCP-Tools:http://localhost:8091/healthz"
        "SearXNG:http://localhost:8086"
        "Vector-Store:http://localhost:3015/health"
        "SandboxFusion:http://localhost:3010"
    )
    
    local failed=0
    print_header "Checking MCP Services"
    
    for service in "${services[@]}"; do
        local name="${service%%:*}"
        local url="${service#*:}"
        
        if ! check_http_endpoint "$name" "$url" 3; then
            failed=1
        fi
    done
    
    if [ $failed -eq 0 ]; then
        print_success "All MCP services are healthy"
        return 0
    else
        print_error "Some MCP services are not healthy"
        return 1
    fi
}

# Wait for all services to be ready
wait_for_services_ready() {
    print_info "Waiting for services to be ready..."
    sleep 5
    
    check_all_services
    return $?
}

export -f check_http_endpoint
export -f check_all_services
export -f check_mcp_services
export -f wait_for_services_ready
