#!/bin/bash

# Generate Swagger documentation for Jan Server
# This script generates swagger docs for both llm-api and mcp-tools services
# and combines them into a single swagger spec

set -e

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
fi

echo ""
echo -e "${GREEN}✅ Swagger generation complete!${NC}"
echo ""
echo "Generated files:"
echo "  - $LLM_API_DIR/docs/swagger/swagger.json (LLM API service)"
echo "  - $LLM_API_DIR/docs/swagger/swagger-combined.json (Combined spec)"
echo "  - $MCP_TOOLS_DIR/docs/swagger/swagger.json (MCP Tools service)"
echo ""
echo "View the API documentation:"
echo "  - LLM API: http://localhost:8080/api/swagger/index.html"
echo ""
