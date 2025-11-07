# Main setup script for jan-server development environment

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
. "$ScriptDir\lib\common.ps1"

Print-Header "Jan Server Development Environment Setup"

# Check prerequisites
Print-Info "Checking prerequisites..."

$MissingDeps = $false

if (-not (Test-CommandExists "docker")) {
    Print-Error "Docker is not installed"
    Print-Info "Install from: https://docs.docker.com/get-docker/"
    $MissingDeps = $true
} else {
    $dockerVersion = docker --version
    Print-Success "Docker found: $dockerVersion"
}

if (-not (Test-CommandExists "docker")) {
    $MissingDeps = $true
} else {
    try {
        $composeVersion = docker compose version
        Print-Success "Docker Compose found: $composeVersion"
    } catch {
        Print-Error "Docker Compose V2 is not installed"
        Print-Info "Install from: https://docs.docker.com/compose/install/"
        $MissingDeps = $true
    }
}

if (-not (Test-CommandExists "make")) {
    Print-Warning "Make is not installed (optional but recommended)"
    Print-Info "On Windows: Install via chocolatey (choco install make) or use WSL"
} else {
    $makeVersion = make --version | Select-Object -First 1
    Print-Success "Make found: $makeVersion"
}

if (-not (Test-CommandExists "go")) {
    Print-Warning "Go is not installed (required for hybrid mode)"
    Print-Info "Install from: https://go.dev/dl/"
} else {
    $goVersion = go version
    Print-Success "Go found: $goVersion"
}

if (-not (Test-CommandExists "newman")) {
    Print-Warning "Newman is not installed (required for integration tests)"
    Print-Info "Install with: npm install -g newman"
} else {
    $newmanVersion = newman --version
    Print-Success "Newman found: $newmanVersion"
}

if (-not (Test-CommandExists "curl")) {
    Print-Warning "curl is not installed (optional)"
} else {
    Print-Success "curl found"
}

if ($MissingDeps) {
    Print-Error "Missing required dependencies. Please install them and run this script again."
    exit 1
}

Write-Host ""
Print-Info "All required dependencies are installed!"

# Create .env file
Print-Header "Environment Configuration"

if (Test-Path ".env") {
    Print-Warning ".env file already exists"
    $overwrite = Read-Host "Do you want to overwrite it? (y/N)"
    if ($overwrite -ne "y" -and $overwrite -ne "Y") {
        Print-Info "Keeping existing .env file"
    } else {
        if (Test-Path ".env.template") {
            Copy-Item ".env.template" ".env" -Force
            Print-Success ".env file created from template"
        } else {
            Print-Warning ".env.template not found"
        }
    }
} else {
    if (Test-Path ".env.template") {
        Copy-Item ".env.template" ".env"
        Print-Success ".env file created from template"
    } else {
        Print-Warning ".env.template not found, will be created in Phase 1.3"
        Print-Info "Using default environment variables for now"
    }
}

# Create config directory
if (-not (Test-Path "config")) {
    Print-Info "Creating config directory..."
    New-Item -ItemType Directory -Path "config" | Out-Null
    Print-Success "config directory created"
}

# Setup Docker
Print-Header "Docker Setup"

try {
    $dockerInfo = docker info 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "Docker is not running"
    }
} catch {
    Print-Error "Docker is not running. Please start Docker and run this script again."
    exit 1
}

Print-Info "Creating Docker networks..."
@("jan-network", "mcp-network") | ForEach-Object {
    $networkName = $_
    $existing = docker network ls --filter "name=$networkName" --format "{{.Name}}" | Select-String -Pattern "^$networkName$"
    if (-not $existing) {
        docker network create $networkName | Out-Null
        Print-Success "Created network: $networkName"
    } else {
        Print-Info "Network already exists: $networkName"
    }
}

# Pull base images
Print-Info "Pulling base Docker images (this may take a while)..."
try {
    docker compose pull postgres keycloak kong
} catch {
    Print-Warning "Some images could not be pulled"
}

Print-Success "Docker setup complete"

# Summary
Print-Header "Setup Complete!"

Write-Host @"

Next Steps:
-----------
1. Start infrastructure:
   make up-infra              # Start postgres, keycloak, kong
   
2. Start API:
   make up-api                # Start llm-api service
   
3. Start MCP tools (optional):
   make up-mcp                # Start all MCP services
   
4. Run tests:
   make test-all              # Run all integration tests
   
5. Hybrid development (optional):
   make down-api              # Stop API in Docker
   .\scripts\hybrid-run-api.ps1  # Run API natively

For more information:
   make help                  # Show all available commands
   Get-Content README.md      # Read the documentation

"@

Print-Success "Happy coding! 🚀"
