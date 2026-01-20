#!/usr/bin/env bash
# jan-cli wrapper script
# Automatically builds and runs jan-cli from cmd/jan-cli/

set -e
# Save the original working directory
ROOT_DIR="$(pwd)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="${SCRIPT_DIR}/jan-cli"
CLI_BINARY="${CLI_DIR}/jan-cli"

# Build if binary doesn't exist or any .go source is newer
needs_build=false
if [ ! -f "${CLI_BINARY}" ]; then
    needs_build=true
else
    # Check if any .go file is newer than the binary
    while IFS= read -r -d '' gofile; do
        if [ "$gofile" -nt "${CLI_BINARY}" ]; then
            needs_build=true
            break
        fi
    done < <(find "${CLI_DIR}" -maxdepth 1 -name "*.go" -print0)
fi

if [ "$needs_build" = true ]; then
    echo "Building jan-cli..." >&2
    cd "${CLI_DIR}"
    go build -o jan-cli .
fi

# Change to the original working directory
cd "${ROOT_DIR}"

# Run jan-cli with all arguments
exec "${CLI_BINARY}" "$@"
