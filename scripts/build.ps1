<#
.SYNOPSIS
    Build Embrionix Dashboard for Windows.
.EXAMPLE
    .\scripts\build.ps1
    .\scripts\build.ps1 -Version "0.2.0" -OutputDir ".\release"
#>
param(
    [string]$Version = "dev",
    [string]$OutputDir = ".\dist"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "==> Building Embrionix Dashboard v$Version" -ForegroundColor Cyan

# Build frontend
Write-Host "--> Building frontend..." -ForegroundColor Yellow
Push-Location web
npm ci
npm run build
Pop-Location

# Build backend
Write-Host "--> Building backend (windows/amd64)..." -ForegroundColor Yellow
$env:GOOS    = "windows"
$env:GOARCH  = "amd64"
$env:CGO_ENABLED = "0"

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
go build `
    -ldflags="-s -w -X main.Version=$Version" `
    -o "$OutputDir\embrionix-dashboard.exe" `
    .\cmd\server\

# Package
Write-Host "--> Packaging..." -ForegroundColor Yellow
Copy-Item -Recurse -Force web\dist         "$OutputDir\web"
Copy-Item -Force configs\config.yaml       "$OutputDir\configs\"

Write-Host "==> Done. Output in $OutputDir" -ForegroundColor Green
Write-Host "    Run: $OutputDir\embrionix-dashboard.exe" -ForegroundColor Gray
