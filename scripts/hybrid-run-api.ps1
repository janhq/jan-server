# Script to run LLM API service natively while infrastructure runs in Docker

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
. "$ScriptDir\lib\common.ps1"

Print-Header "Running LLM API in Hybrid Mode"

# Check prerequisites
if (-not (Test-CommandExists "go")) {
    Print-Error "Go is not installed"
    exit 1
}

# Check if API is already running in Docker
$apiContainer = docker compose ps --format json | ConvertFrom-Json | Where-Object { $_.Service -eq "llm-api" -and $_.State -eq "running" }
if ($apiContainer) {
    Print-Warning "LLM API is running in Docker. Stop it first with:"
    Print-Info "  docker compose stop llm-api"
    exit 1
}

# Check if infrastructure is running
Print-Info "Checking infrastructure services..."
$postgres = docker compose ps --format json | ConvertFrom-Json | Where-Object { $_.Service -eq "postgres" -and $_.State -eq "running" }
if (-not $postgres) {
    Print-Error "Infrastructure is not running. Start it with:"
    Print-Info "  docker compose --profile infra up -d"
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
$env:DATABASE_URL = "postgres://jan_user:jan_password@localhost:5432/jan_llm_api?sslmode=disable"
$env:KEYCLOAK_BASE_URL = "http://localhost:8085"
$env:JWKS_URL = "http://localhost:8085/realms/jan/protocol/openid-connect/certs"
$env:ISSUER = "http://localhost:8085/realms/jan"
$env:HTTP_PORT = "8080"
$env:LOG_LEVEL = "debug"
$env:LOG_FORMAT = "console"
$env:AUTO_MIGRATE = "true"

# Navigate to service directory
Push-Location "$ScriptDir\..\services\llm-api"

try {
    Print-Info "Building LLM API..."
    go build -o bin\llm-api.exe .
    
    Print-Success "Starting LLM API on http://localhost:8080"
    Print-Info "Press Ctrl+C to stop"
    Write-Host ""
    
    # Run the service
    .\bin\llm-api.exe
}
finally {
    Pop-Location
}
