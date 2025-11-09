param(
    [Parameter(Mandatory = $true)]
    [string]$Name,

    [string]$DestinationRoot = "services"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$serviceName = $Name.Trim()
if (-not $serviceName) {
    throw "Service name cannot be empty."
}

$safeName = $serviceName.ToLower()
$source = Join-Path -Path "services" -ChildPath "template-api"
if (-not (Test-Path $source)) {
    throw "Template API not found. Make sure services/template-api exists."
}

$destination = Join-Path -Path $DestinationRoot -ChildPath $safeName
if (Test-Path $destination) {
    throw "Destination '$destination' already exists."
}

Write-Host "Cloning template-api to $destination"
Copy-Item -Path $source -Destination $destination -Recurse

$pascalName = ($safeName -split "[-_]" | ForEach-Object { $_.Substring(0,1).ToUpper() + $_.Substring(1) }) -join " "

$files = Get-ChildItem -Path $destination -Recurse -File |
    Where-Object { $_.Extension -notin ".exe", ".dll" }

foreach ($file in $files) {
    (Get-Content -Path $file.FullName) |
        ForEach-Object {
            $_ -replace "Template API", $pascalName `
               -replace "template-api", $safeName
        } |
        Set-Content -Path $file.FullName -Encoding UTF8
}

# Update go module path explicitly.
$goModPath = Join-Path $destination "go.mod"
if (Test-Path $goModPath) {
    (Get-Content $goModPath) |
        ForEach-Object {
            $_ -replace "jan-server/services/template-api", "jan-server/services/$safeName"
        } |
        Set-Content $goModPath -Encoding UTF8
}

Write-Host "Service cloned. Next steps:"
Write-Host "  1. Update config/example.env and README for $serviceName."
Write-Host "  2. Run 'cd $destination && go mod tidy'."
Write-Host "  3. Wire up routes and dependencies."
