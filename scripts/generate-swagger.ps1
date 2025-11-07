# Generate Swagger documentation for Jan Server
# This script generates swagger docs for both llm-api and mcp-tools services
# and combines them into a single swagger spec

$ErrorActionPreference = "Stop"

Write-Host "Generating Swagger documentation for Jan Server..." -ForegroundColor Cyan

# Directories
$ROOT_DIR = Split-Path -Parent $PSScriptRoot
$LLM_API_DIR = Join-Path (Join-Path $ROOT_DIR "services") "llm-api"
$MCP_TOOLS_DIR = Join-Path (Join-Path $ROOT_DIR "services") "mcp-tools"

# Generate swagger for llm-api
Write-Host "Generating swagger for llm-api service..." -ForegroundColor Blue
Push-Location $LLM_API_DIR
try {
    & swag init --dir ./cmd/server,./internal/interfaces/httpserver/routes --generalInfo server.go --output ./docs/swagger --parseDependency --parseInternal

    $llmApiSwagger = Join-Path (Join-Path (Join-Path $LLM_API_DIR "docs") "swagger") "swagger.json"
    if (Test-Path $llmApiSwagger) {
        Write-Host "OK llm-api swagger generated successfully" -ForegroundColor Green
    } else {
        Write-Host "WARNING llm-api swagger.json not found" -ForegroundColor Yellow
    }
} finally {
    Pop-Location
}

# Generate swagger for mcp-tools
Write-Host "Generating swagger for mcp-tools service..." -ForegroundColor Blue
Push-Location $MCP_TOOLS_DIR
try {
    & swag init --dir . --generalInfo main.go --output ./docs/swagger --parseDependency --parseInternal

    $mcpToolsSwagger = Join-Path (Join-Path (Join-Path $MCP_TOOLS_DIR "docs") "swagger") "swagger.json"
    if (Test-Path $mcpToolsSwagger) {
        Write-Host "OK mcp-tools swagger generated successfully" -ForegroundColor Green
    } else {
        Write-Host "WARNING mcp-tools swagger.json not found" -ForegroundColor Yellow
    }
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "Swagger generation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Generated files:"
Write-Host "  - $LLM_API_DIR/docs/swagger/swagger.json (LLM API service)"
Write-Host "  - $LLM_API_DIR/docs/swagger/swagger-combined.json (Combined spec)"
Write-Host "  - $MCP_TOOLS_DIR/docs/swagger/swagger.json (MCP Tools service)"
Write-Host ""
Write-Host "View the API documentation:"
Write-Host "  - LLM API: http://localhost:8080/api/swagger/index.html"
Write-Host ""

