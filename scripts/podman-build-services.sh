#!/bin/bash
# Build Go services for Podman
# Workaround for podman-compose not supporting additional_contexts
#
# Usage: ./scripts/podman-build-services.sh [service...]
# Example: ./scripts/podman-build-services.sh llm-api media-api

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

# Services that need building with go-common context
GO_SERVICES=(
    "llm-api"
    "media-api"
    "response-api"
    "mcp-tools"
    "memory-tools"
    "realtime-api"
)

# If specific services provided, use those
if [ $# -gt 0 ]; then
    GO_SERVICES=("$@")
fi

echo "Building Go services for Podman..."
echo "Project root: $PROJECT_ROOT"
echo ""

for svc in "${GO_SERVICES[@]}"; do
    svc_dir="$PROJECT_ROOT/services/$svc"

    if [ ! -d "$svc_dir" ]; then
        echo "Warning: $svc_dir not found, skipping"
        continue
    fi

    dockerfile="$svc_dir/Dockerfile"
    if [ ! -f "$dockerfile" ]; then
        echo "Warning: $dockerfile not found, skipping"
        continue
    fi

    echo "Building $svc..."

    # Create a temporary build context with go-common included
    BUILD_CONTEXT=$(mktemp -d)
    trap "rm -rf $BUILD_CONTEXT" EXIT

    # Copy service files
    cp -r "$svc_dir"/* "$BUILD_CONTEXT/"

    # Copy go-common to the build context
    mkdir -p "$BUILD_CONTEXT/packages"
    cp -r "$PROJECT_ROOT/packages/go-common" "$BUILD_CONTEXT/packages/"

    # Create modified Dockerfile that doesn't use additional_contexts
    sed 's|COPY --from=go-common \. /packages/go-common|COPY packages/go-common /packages/go-common|g' \
        "$dockerfile" > "$BUILD_CONTEXT/Dockerfile.podman"

    # Build with podman
    podman build \
        -t "jan-server-$svc:latest" \
        -f "$BUILD_CONTEXT/Dockerfile.podman" \
        "$BUILD_CONTEXT"

    echo "✓ Built jan-server-$svc:latest"
    echo ""

    # Clean up temp dir for this iteration
    rm -rf "$BUILD_CONTEXT"
    trap - EXIT
done

echo "All services built successfully!"
echo ""
echo "To use in podman-compose, add to docker-compose.podman.yml:"
echo "  services:"
echo "    llm-api:"
echo "      image: jan-server-llm-api:latest"
