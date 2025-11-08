#!/bin/bash
# Environment helper functions

source "$(dirname "$0")/common.sh"

# Check if .env file exists
check_env_file() {
    if [ ! -f .env ]; then
        print_warning ".env file not found"
        return 1
    fi
    print_success ".env file exists"
    return 0
}

# Create .env from template
create_env_from_template() {
    local template_file=${1:-.env.template}
    local output_file=${2:-.env}
    
    if [ ! -f "$template_file" ]; then
        print_error "Template file '$template_file' not found"
        return 1
    fi
    
    if [ -f "$output_file" ]; then
        print_warning "$output_file already exists"
        if ! confirm "Overwrite existing $output_file?"; then
            print_info "Keeping existing $output_file"
            return 0
        fi
    fi
    
    print_info "Creating $output_file from $template_file..."
    cp "$template_file" "$output_file"
    print_success "$output_file created"
    
    # On Linux/macOS, make it readable only by owner for security
    chmod 600 "$output_file"
    
    return 0
}

# Load environment file
load_env_file() {
    local env_file=${1:-.env}
    
    if [ ! -f "$env_file" ]; then
        print_error "Environment file '$env_file' not found"
        return 1
    fi
    
    print_info "Loading environment from $env_file..."
    set -a
    source "$env_file"
    set +a
    print_success "Environment loaded"
    return 0
}

# Validate required environment variables
validate_env_vars() {
    local -a required_vars=("$@")
    local missing=0
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            print_error "Required environment variable '$var' is not set"
            missing=1
        fi
    done
    
    if [ $missing -eq 1 ]; then
        print_error "Missing required environment variables"
        return 1
    fi
    
    print_success "All required environment variables are set"
    return 0
}

# Generate random string
generate_random_string() {
    local length=${1:-32}
    openssl rand -base64 $length | tr -d "=+/" | cut -c1-$length
}

# Update env file value
update_env_value() {
    local env_file=$1
    local key=$2
    local value=$3
    
    if [ ! -f "$env_file" ]; then
        print_error "Environment file '$env_file' not found"
        return 1
    fi
    
    if grep -q "^${key}=" "$env_file"; then
        # Update existing value
        sed -i.bak "s|^${key}=.*|${key}=${value}|" "$env_file"
        rm -f "${env_file}.bak"
        print_success "Updated $key in $env_file"
    else
        # Append new value
        echo "${key}=${value}" >> "$env_file"
        print_success "Added $key to $env_file"
    fi
    
    return 0
}

export -f check_env_file
export -f create_env_from_template
export -f load_env_file
export -f validate_env_vars
export -f generate_random_string
export -f update_env_value
