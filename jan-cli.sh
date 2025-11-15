#!/usr/bin/env bash
# jan-cli wrapper script
# Automatically builds and runs jan-cli from cmd/jan-cli/

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="${SCRIPT_DIR}/cmd/jan-cli"
CLI_BINARY="${CLI_DIR}/jan-cli"

# Build if binary doesn't exist or source is newer
if [ ! -f "${CLI_BINARY}" ] || [ "${CLI_DIR}/main.go" -nt "${CLI_BINARY}" ]; then
    echo "Building jan-cli..." >&2
    cd "${CLI_DIR}"
    go build -o jan-cli .
    cd "${SCRIPT_DIR}"
fi

# Run jan-cli with all arguments
exec "${CLI_BINARY}" "$@"
