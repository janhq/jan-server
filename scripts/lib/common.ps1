# Common helper functions for Windows PowerShell

# Print functions with colors
function Print-Success {
    param([string]$Message)
    Write-Host "OK $Message" -ForegroundColor Green
}

function Print-Error {
    param([string]$Message)
    Write-Host "ERROR $Message" -ForegroundColor Red
}

function Print-Warning {
    param([string]$Message)
    Write-Host "WARNING $Message" -ForegroundColor Yellow
}

function Print-Info {
    param([string]$Message)
    Write-Host "INFO $Message" -ForegroundColor Cyan
}

function Print-Header {
    param([string]$Message)
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "$Message" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""
}

# Check if command exists
function Test-CommandExists {
    param([string]$Command)
    $null -ne (Get-Command $Command -ErrorAction SilentlyContinue)
}

# Wait for user confirmation
function Confirm-Action {
    param(
        [string]$Prompt = "Are you sure?",
        [string]$Default = "N"
    )
    $response = Read-Host "$Prompt [y/N]"
    return ($response -match '^[Yy](es)?$')
}

# Sleep with message
function Start-SleepWithMessage {
    param(
        [int]$Seconds,
        [string]$Message = "Waiting"
    )
    Write-Host -NoNewline "$Message"
    for ($i = 1; $i -le $Seconds; $i++) {
        Write-Host -NoNewline "."
        Start-Sleep -Seconds 1
    }
    Write-Host ""
}

# Check if running in CI
function Test-IsCI {
    return ($env:CI -or $env:GITHUB_ACTIONS -or $env:GITLAB_CI)
}

# Get platform
function Get-Platform {
    return "windows"
}

# Get architecture
function Get-Architecture {
    return $env:PROCESSOR_ARCHITECTURE
}
