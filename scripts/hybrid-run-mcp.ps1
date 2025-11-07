# Script to run MCP Tools service natively while infrastructure runs in Docker

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
. "$ScriptDir\lib\common.ps1"

Print-Header "Running MCP Tools in Hybrid Mode"

# Check prerequisites
if (-not (Test-CommandExists "go")) {
    Print-Error "Go is not installed"
    exit 1
}

# Check if MCP Tools is already running in Docker
$mcpContainer = docker compose --profile mcp ps --format json | ConvertFrom-Json | Where-Object { $_.Service -eq "mcp-tools" -and $_.State -eq "running" }
if ($mcpContainer) {
    Print-Warning "MCP Tools is running in Docker. Stop it first with:"
    Print-Info "  docker compose --profile mcp stop mcp-tools"
    exit 1
}

# Check if MCP infrastructure is running
Print-Info "Checking MCP infrastructure services..."
$vectorStore = docker compose --profile mcp ps --format json | ConvertFrom-Json | Where-Object { $_.Service -eq "vector-store" -and $_.State -eq "running" }
if (-not $vectorStore) {
    Print-Error "MCP infrastructure is not running. Start it with:"
    Print-Info "  docker compose --profile mcp up -d searxng vector-store sandboxfusion"
    exit 1
}

# Load hybrid environment
Print-Info "Loading hybrid environment..."
if (Test-Path "config\hybrid.env") {
    Get-Content "config\hybrid.env" | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
            [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim(), "Process")
        }
    }
}

# Set hybrid-specific environment variables
$env:HTTP_PORT = "8091"
$env:VECTOR_STORE_URL = "http://localhost:3015"
$env:SEARXNG_URL = "http://localhost:8086"
$env:SANDBOXFUSION_URL = "http://localhost:3010"
$env:LOG_LEVEL = "debug"
$env:LOG_FORMAT = "console"

# Navigate to service directory
Push-Location "$ScriptDir\..\services\mcp-tools"

try {
    Print-Info "Building MCP Tools..."
    go build -o bin\mcp-tools.exe .
    
    Print-Success "Starting MCP Tools on http://localhost:8091"
    Print-Info "Press Ctrl+C to stop"
    Write-Host ""
    
    # Run the service
    .\bin\mcp-tools.exe
}
finally {
    Pop-Location
}
