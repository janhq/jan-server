#!/bin/bash
# Common helper functions for Unix-like systems (Linux/macOS)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_header() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Wait for user confirmation
confirm() {
    local prompt="${1:-Are you sure?}"
    local response
    read -p "$prompt [y/N] " response
    case "$response" in
        [yY][eE][sS]|[yY]) 
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Sleep with message
sleep_with_message() {
    local seconds=$1
    local message=${2:-"Waiting"}
    echo -n "$message"
    for i in $(seq 1 $seconds); do
        echo -n "."
        sleep 1
    done
    echo ""
}

# Check if running in CI
is_ci() {
    [ -n "$CI" ] || [ -n "$GITHUB_ACTIONS" ] || [ -n "$GITLAB_CI" ]
}

# Get platform
get_platform() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    echo "macos";;
        *)          echo "unknown";;
    esac
}

# Get architecture
get_arch() {
    uname -m
}

# Export functions for use in other scripts
export -f print_success
export -f print_error
export -f print_warning
export -f print_info
export -f print_header
export -f command_exists
export -f confirm
export -f sleep_with_message
export -f is_ci
export -f get_platform
export -f get_arch
