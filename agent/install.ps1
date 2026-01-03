# GPTWol Agent - One-click install script for Windows
# Run in PowerShell as Administrator:
# irm https://raw.githubusercontent.com/nbb2025/gptwol/main/agent/install.ps1 -OutFile install.ps1; .\install.ps1 -Action shutdown

param(
    [ValidateSet("shutdown", "reboot", "sleep", "hibernate")]
    [string]$Action = "shutdown",

    [string]$Mac = ""
)

$ErrorActionPreference = "Stop"

$Repo = "nbb2025/gptwol"
$BinaryName = "gptwol-agent.exe"
$InstallDir = "$env:ProgramFiles\gptwol-agent"

function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Err { Write-Host "[ERROR] $args" -ForegroundColor Red }

# Check admin
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Err "Please run as Administrator"
    exit 1
}

# Detect architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
Write-Info "Detected: windows/${arch}"

# Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Download binary
$downloadUrl = "https://github.com/${Repo}/releases/latest/download/gptwol-agent-windows-${arch}.exe"
$binaryPath = Join-Path $InstallDir $BinaryName

Write-Info "Downloading from: ${downloadUrl}"
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $binaryPath -UseBasicParsing
} catch {
    Write-Err "Download failed: $_"
    exit 1
}

# Stop existing service if running
$service = Get-Service -Name "gptwol-agent" -ErrorAction SilentlyContinue
if ($service) {
    Write-Info "Stopping existing service..."
    Stop-Service -Name "gptwol-agent" -Force -ErrorAction SilentlyContinue
    & sc.exe delete "gptwol-agent" | Out-Null
    Start-Sleep -Seconds 2
}

# Install service
Write-Info "Installing as Windows service..."
$installArgs = @("-install", "-action", $Action)
if ($Mac) {
    $installArgs += @("-mac", $Mac)
}
& $binaryPath @installArgs

Write-Info "Installation complete!"
Write-Host ""
Write-Host "  Service: gptwol-agent"
Write-Host "  Action: ${Action}"
Write-Host "  Listening on: UDP ports 7, 9"
Write-Host "  Binary: ${binaryPath}"
Write-Host "  Status: sc query gptwol-agent"
Write-Host ""
Write-Host "  To trigger ${Action}, send a Sleep-on-LAN packet (reversed MAC)"
Write-Host "  from your gptwol server."

# Add firewall rule for UDP ports
Write-Info "Adding firewall rules..."
New-NetFirewallRule -DisplayName "GPTWol Agent UDP 7" -Direction Inbound -Protocol UDP -LocalPort 7 -Action Allow -ErrorAction SilentlyContinue | Out-Null
New-NetFirewallRule -DisplayName "GPTWol Agent UDP 9" -Direction Inbound -Protocol UDP -LocalPort 9 -Action Allow -ErrorAction SilentlyContinue | Out-Null

# Add to PATH
$currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($currentPath -notlike "*$InstallDir*") {
    Write-Info "Adding to system PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "Machine")
}
