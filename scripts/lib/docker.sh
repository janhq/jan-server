#!/bin/bash
# Docker helper functions

source "$(dirname "$0")/common.sh"

# Check if Docker is running
check_docker() {
    if ! command_exists docker; then
        print_error "Docker is not installed"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running"
        print_info "Please start Docker Desktop and try again"
        return 1
    fi
    
    print_success "Docker is running"
    return 0
}

# Check Docker Compose version
check_docker_compose() {
    if ! docker compose version >/dev/null 2>&1; then
        print_error "Docker Compose V2 is not available"
        print_info "Please update Docker Desktop to a version that includes Compose V2"
        return 1
    fi
    
    local version=$(docker compose version --short)
    print_success "Docker Compose $version is available"
    return 0
}

# Create Docker network if it doesn't exist
create_network() {
    local network_name=$1
    
    if docker network inspect "$network_name" >/dev/null 2>&1; then
        print_info "Network '$network_name' already exists"
        return 0
    fi
    
    print_info "Creating network '$network_name'..."
    if docker network create "$network_name" >/dev/null; then
        print_success "Network '$network_name' created"
        return 0
    else
        print_error "Failed to create network '$network_name'"
        return 1
    fi
}

# Remove Docker network
remove_network() {
    local network_name=$1
    
    if ! docker network inspect "$network_name" >/dev/null 2>&1; then
        print_info "Network '$network_name' does not exist"
        return 0
    fi
    
    print_info "Removing network '$network_name'..."
    if docker network rm "$network_name" >/dev/null 2>&1; then
        print_success "Network '$network_name' removed"
        return 0
    else
        print_warning "Could not remove network '$network_name' (may have containers attached)"
        return 1
    fi
}

# Wait for service to be healthy
wait_for_service() {
    local service_name=$1
    local max_wait=${2:-60}
    local interval=2
    local waited=0
    
    print_info "Waiting for service '$service_name' to be healthy..."
    
    while [ $waited -lt $max_wait ]; do
        local health=$(docker inspect --format='{{.State.Health.Status}}' "$service_name" 2>/dev/null || echo "none")
        
        if [ "$health" = "healthy" ]; then
            print_success "Service '$service_name' is healthy"
            return 0
        fi
        
        echo -n "."
        sleep $interval
        waited=$((waited + interval))
    done
    
    echo ""
    print_error "Service '$service_name' did not become healthy within ${max_wait}s"
    return 1
}

# Get host.docker.internal IP for Linux
get_host_docker_internal_ip() {
    local platform=$(get_platform)
    
    if [ "$platform" = "linux" ]; then
        # Try to get Docker bridge IP
        local bridge_ip=$(ip -4 addr show docker0 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}' || echo "172.17.0.1")
        echo "$bridge_ip"
    else
        echo "host.docker.internal"
    fi
}

export -f check_docker
export -f check_docker_compose
export -f create_network
export -f remove_network
export -f wait_for_service
export -f get_host_docker_internal_ip
