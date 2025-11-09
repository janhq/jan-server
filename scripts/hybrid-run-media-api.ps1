# Script to run Media API service natively while infrastructure runs in Docker

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
. "$ScriptDir\lib\common.ps1"

Print-Header "Running Media API in Hybrid Mode"

if (-not (Test-CommandExists "go")) {
    Print-Error "Go is not installed"
    exit 1
}

$mediaContainer = docker compose ps --format json | ConvertFrom-Json | Where-Object { $_.Service -eq "media-api" -and $_.State -eq "running" }
if ($mediaContainer) {
    Print-Warning "Media API is running in Docker. Stop it first with:"
    Print-Info "  docker compose stop media-api"
    exit 1
}

Print-Info "Checking infrastructure services..."
$postgres = docker compose ps --format json | ConvertFrom-Json | Where-Object { $_.Service -eq "api-db" -and $_.State -eq "running" }
if (-not $postgres) {
    Print-Error "Infrastructure is not running. Start it with:"
    Print-Info "  docker compose --profile infra up -d"
    exit 1
}

if (Test-Path "config\hybrid.env") {
    Print-Info "Loading config/hybrid.env..."
    Get-Content "config\hybrid.env" | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
            [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim(), "Process")
        }
    }
}

if (-not $env:MEDIA_DATABASE_URL) { $env:MEDIA_DATABASE_URL = "postgres://media:media@localhost:5432/media_api?sslmode=disable" }
if (-not $env:MEDIA_SERVICE_KEY) { $env:MEDIA_SERVICE_KEY = "changeme-media-key" }
if (-not $env:MEDIA_API_KEY) { $env:MEDIA_API_KEY = $env:MEDIA_SERVICE_KEY }
if (-not $env:MEDIA_API_PORT) { $env:MEDIA_API_PORT = "8285" }
if (-not $env:MEDIA_API_URL) { $env:MEDIA_API_URL = "http://localhost:$($env:MEDIA_API_PORT)" }
if (-not $env:MEDIA_S3_ENDPOINT) { $env:MEDIA_S3_ENDPOINT = "https://s3.menlo.ai" }
if (-not $env:MEDIA_S3_REGION) { $env:MEDIA_S3_REGION = "us-west-2" }
if (-not $env:MEDIA_S3_BUCKET) { $env:MEDIA_S3_BUCKET = "platform-dev" }
if (-not $env:MEDIA_S3_ACCESS_KEY) { $env:MEDIA_S3_ACCESS_KEY = "XXXXX" }
if (-not $env:MEDIA_S3_SECRET_KEY) { $env:MEDIA_S3_SECRET_KEY = "YYYY" }
if (-not $env:MEDIA_S3_USE_PATH_STYLE) { $env:MEDIA_S3_USE_PATH_STYLE = "true" }
if (-not $env:MEDIA_S3_PRESIGN_TTL) { $env:MEDIA_S3_PRESIGN_TTL = "5m" }
if (-not $env:MEDIA_MAX_BYTES) { $env:MEDIA_MAX_BYTES = "20971520" }
if (-not $env:MEDIA_PROXY_DOWNLOAD) { $env:MEDIA_PROXY_DOWNLOAD = "true" }
if (-not $env:MEDIA_RETENTION_DAYS) { $env:MEDIA_RETENTION_DAYS = "30" }
if (-not $env:MEDIA_REMOTE_FETCH_TIMEOUT) { $env:MEDIA_REMOTE_FETCH_TIMEOUT = "15s" }

Push-Location "$ScriptDir\..\services\media-api"
try {
    Print-Info "Building Media API..."
    go build -o bin\media-api.exe .

    Print-Success "Starting Media API on http://localhost:$($env:MEDIA_API_PORT)"
    Print-Info "Press Ctrl+C to stop"
    Write-Host ""

    .\bin\media-api.exe
}
finally {
    Pop-Location
}
