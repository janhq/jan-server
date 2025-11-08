#!/bin/bash
# Swagger Generation (Unix)
# ---------------------------------------------
# Generates swagger.json for llm-api and mcp-tools; optionally combines them
# into services/llm-api/docs/swagger/swagger-combined.json when the combine tool is present.
# Mirrors generate-swagger.ps1 behavior.
#
# Usage:
#   ./scripts/generate-swagger.sh
# Prereqs:
#   go install github.com/swaggo/swag/cmd/swag@latest
# ---------------------------------------------

# Generate Swagger documentation for Jan Server
# This script generates swagger docs for both llm-api and mcp-tools services
# and combines them into a single swagger spec

set -euo pipefail

# Prerequisite checks
if ! command -v swag >/dev/null 2>&1; then
  echo "❌ 'swag' CLI not installed. Install with: go install github.com/swaggo/swag/cmd/swag@latest" >&2
  exit 1
fi

echo "🔧 Generating Swagger documentation for Jan Server..."

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Directories
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LLM_API_DIR="$ROOT_DIR/services/llm-api"
MCP_TOOLS_DIR="$ROOT_DIR/services/mcp-tools"

missing_any=0

# Generate swagger for llm-api
echo -e "${BLUE}📝 Generating swagger for llm-api service...${NC}"
cd "$LLM_API_DIR"
swag init \
  --dir ./cmd/server,./internal/interfaces/httpserver/routes \
  --generalInfo server.go \
  --output ./docs/swagger \
  --parseDependency \
  --parseInternal

if [ -f "./docs/swagger/swagger.json" ]; then
  echo -e "${GREEN}✓ llm-api swagger generated successfully${NC}"
else
  echo -e "${YELLOW}⚠ llm-api swagger.json not found${NC}"
  missing_any=1
fi

# Generate swagger for mcp-tools
echo -e "${BLUE}📝 Generating swagger for mcp-tools service...${NC}"
cd "$MCP_TOOLS_DIR"
swag init \
  --dir . \
  --generalInfo main.go \
  --output ./docs/swagger \
  --parseDependency \
  --parseInternal

if [ -f "./docs/swagger/swagger.json" ]; then
  echo -e "${GREEN}✓ mcp-tools swagger generated successfully${NC}"
else
  echo -e "${YELLOW}⚠ mcp-tools swagger.json not found${NC}"
  missing_any=1
fi

echo ""
echo -e "${GREEN}✅ Swagger generation complete!${NC}"
echo ""
echo "Generated files:"
echo "  - $LLM_API_DIR/docs/swagger/swagger.json (LLM API service)"
echo "  - $MCP_TOOLS_DIR/docs/swagger/swagger.json (MCP Tools service)"
if [ -f "$LLM_API_DIR/docs/swagger/swagger-combined.json" ]; then
  echo "  - $LLM_API_DIR/docs/swagger/swagger-combined.json (Combined spec)"
else
  echo "  - (combined spec not yet generated)"
fi
echo ""
echo "View the API documentation:"
echo "  - LLM API: http://localhost:8080/api/swagger/index.html"
echo ""

# Optionally merge if both exist and combine tool present
if [ $missing_any -eq 0 ]; then
  if [ -f "$ROOT_DIR/scripts/swagger-combine.go" ]; then
    echo "🔄 Combining swagger specs into unified file..."
    (cd "$ROOT_DIR" && go run scripts/swagger-combine.go \
      -llm-api services/llm-api/docs/swagger/swagger.json \
      -mcp-tools services/mcp-tools/docs/swagger/swagger.json \
      -output services/llm-api/docs/swagger/swagger-combined.json) || echo "⚠ Failed to combine swagger specs"
    if [ -f "$LLM_API_DIR/docs/swagger/swagger-combined.json" ]; then
      echo -e "${GREEN}✓ Combined swagger created at $LLM_API_DIR/docs/swagger/swagger-combined.json${NC}"
    fi
  else
    echo "ℹ️  Combine script not found (scripts/swagger-combine.go). Skipping merge."
  fi
else
  echo "⚠ Skipping combine because one or more swagger.json files were missing."
fi
