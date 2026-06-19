<#
.SYNOPSIS
    Generate favicon.ico for Windows executable embedding.
    Uses a pure-Go generator (no external dependencies).
.EXAMPLE
    .\scripts\icon-setup.ps1
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$IcoOutput = "cmd\server\favicon.ico"

Write-Host "==> Generating favicon.ico for Windows executable" -ForegroundColor Cyan

# Build the icon generator tool
Write-Host "--> Building icon generator..." -ForegroundColor Yellow
go build -o ".\cmd\icon-generator\icon-generator.exe" ".\cmd\icon-generator\main.go"
if ($LASTEXITCODE -ne 0) {
    throw "Failed to build icon generator"
}

# Generate the ICO file
Write-Host "--> Generating $IcoOutput..." -ForegroundColor Yellow
& ".\cmd\icon-generator\icon-generator.exe" "$IcoOutput"
if ($LASTEXITCODE -ne 0) {
    throw "Failed to generate icon"
}

Write-Host "==> Done! Icon saved to $IcoOutput" -ForegroundColor Green
