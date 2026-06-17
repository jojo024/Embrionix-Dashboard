# Install Embrionix Dashboard as a Windows service using NSSM.
#
# Prerequisites:
#   - NSSM (https://nssm.cc) on PATH, or set $Nssm below to its full path.
#   - embrionix-dashboard-windows-amd64.exe and config.yaml in $InstallDir.
#
# IMPORTANT: set `updates.restart_mode: "exit"` in config.yaml so an in-app
# self-update exits cleanly and NSSM restarts the new binary (NSSM restarts on
# exit by default). With "self" the app would fight NSSM for the port.
#
# Run this script from an elevated (Administrator) PowerShell.

param(
  [string]$InstallDir = "C:\Embrionix",
  [string]$ServiceName = "EmbrionixDashboard",
  [string]$Nssm = "nssm"
)

$exe = Join-Path $InstallDir "embrionix-dashboard-windows-amd64.exe"
$cfg = Join-Path $InstallDir "config.yaml"

if (-not (Test-Path $exe)) { throw "Binary not found: $exe" }
if (-not (Test-Path $cfg)) { throw "Config not found: $cfg" }

# Install + configure the service
& $Nssm install $ServiceName $exe $cfg
& $Nssm set $ServiceName AppDirectory $InstallDir
& $Nssm set $ServiceName Start SERVICE_AUTO_START
& $Nssm set $ServiceName AppExit Default Restart          # restart on exit (needed for self-update)
& $Nssm set $ServiceName AppStdout (Join-Path $InstallDir "logs\service-out.log")
& $Nssm set $ServiceName AppStderr (Join-Path $InstallDir "logs\service-err.log")

& $Nssm start $ServiceName
Write-Host "Service '$ServiceName' installed and started. Open http://localhost:8081"
