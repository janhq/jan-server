#!/usr/bin/env pwsh
# Install jan-cli to user's local bin directory

$ErrorActionPreference = "Stop"

$binDir = "$env:USERPROFILE\bin"
$source = "cmd\jan-cli\jan-cli.exe"
$dest = "$binDir\jan-cli.exe"

# Create bin directory if it doesn't exist
if (-not (Test-Path $binDir)) {
    Write-Host "Creating $binDir..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $binDir | Out-Null
}

# Copy binary
Write-Host "Installing jan-cli to $binDir..." -ForegroundColor Cyan
Copy-Item -Path $source -Destination $dest -Force

Write-Host ""
Write-Host "Success! Installed to $dest" -ForegroundColor Green
Write-Host ""

# Check if bin is in PATH
$currentPath = $env:PATH
if ($currentPath -notlike "*$binDir*") {
    Write-Host "WARNING: $binDir is not in your PATH" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "To use 'jan-cli' globally, add to PATH:" -ForegroundColor Cyan
    Write-Host "  Temporary (current session):" -ForegroundColor White
    $cmd1 = '$env:PATH += ";$env:USERPROFILE\bin"'
    Write-Host "    $cmd1" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Permanent (add to PowerShell profile):" -ForegroundColor White
    $cmd2 = 'notepad $PROFILE'
    Write-Host "    $cmd2" -ForegroundColor Gray
    $cmd3 = '# Add line: $env:PATH += ";$env:USERPROFILE\bin"'
    Write-Host "    $cmd3" -ForegroundColor Gray
    Write-Host ""
} else {
    Write-Host "Success! $binDir is already in your PATH" -ForegroundColor Green
    Write-Host ""
}

Write-Host "You can now run: jan-cli --help" -ForegroundColor Green
