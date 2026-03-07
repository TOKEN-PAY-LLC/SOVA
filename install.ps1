# SOVA Protocol v1.0 - Windows Installer
# Run as Administrator: powershell -ExecutionPolicy Bypass -File install.ps1
#Requires -RunAsAdministrator

$ErrorActionPreference = "Stop"
$Version = "1.0.0"
$BaseURL = "https://github.com/IvanChernykh/SOVA/releases/download/v$Version"
$InstallDir = "$env:ProgramFiles\SOVA"
$ConfigDir = "$env:ProgramData\SOVA"

function Write-Purple { param($msg) Write-Host $msg -ForegroundColor Magenta }
function Write-Ok     { param($msg) Write-Host "  [OK] $msg" -ForegroundColor Green }
function Write-Info   { param($msg) Write-Host "  [..] $msg" -ForegroundColor Cyan }
function Write-Warn   { param($msg) Write-Host "  [!!] $msg" -ForegroundColor Yellow }
function Write-Err    { param($msg) Write-Host "  [XX] $msg" -ForegroundColor Red }

function Show-AnimatedOwl {
    $frames = @(
        "    ,___,`n    {o,o}`n    /)  )`n    -`"  `"-",
        "    ,___,`n    {O,O}`n    /)  )`n    -`"  `"-",
        "    ,___,`n    {o,o}`n    /)  )`n    -`"  `"-",
        "    ,___,`n    {-,-}`n    /)  )`n    -`"  `"-",
        "    ,___,`n    {o,o}`n    /)  )`n    -`"  `"-"
    )
    foreach ($frame in $frames) {
        Clear-Host
        Write-Host $frame -ForegroundColor Magenta
        Start-Sleep -Milliseconds 150
    }
}

function Show-Banner {
    if ([Environment]::UserInteractive) {
        Show-AnimatedOwl
    }
    Write-Host ""
    Write-Host "    +========================================+" -ForegroundColor Magenta
    Write-Host "    |            ,___,                       |" -ForegroundColor Magenta
    Write-Host "    |            {o,o}    S O V A            |" -ForegroundColor Magenta
    Write-Host "    |            /)  )    Protocol v$Version      |" -ForegroundColor Magenta
    Write-Host '    |            -"  "-                      |' -ForegroundColor Magenta
    Write-Host "    |                                        |" -ForegroundColor Magenta
    Write-Host "    |  Autonomous AI-Powered Anti-DPI        |" -ForegroundColor Magenta
    Write-Host "    |  Post-Quantum  |  100% Free & Open     |" -ForegroundColor Magenta
    Write-Host "    +========================================+" -ForegroundColor Magenta
    Write-Host ""
    Write-Host "  github.com/IvanChernykh/SOVA" -ForegroundColor Cyan
    Write-Host ""
}

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86"   { return "386" }
        default {
            Write-Err "Unsupported architecture: $arch"
            exit 1
        }
    }
}

function Install-Directories {
    Write-Info "Creating directories..."
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
    New-Item -ItemType Directory -Force -Path "$ConfigDir\logs" | Out-Null
    Write-Ok "Directories created"
}

function Install-Binary {
    param($Component, $Arch)
    $url = "$BaseURL/sova-$Component-windows-$Arch.exe"
    $dest = "$InstallDir\sova-$Component.exe"

    Write-Info "Downloading sova-$Component..."
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing -ErrorAction Stop
        Write-Ok "Downloaded sova-$Component to $dest"
        return $true
    } catch {
        Write-Warn "Download failed: $($_.Exception.Message)"
        return $false
    }
}

function Build-FromSource {
    $goCmd = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goCmd) {
        Write-Err "Go is not installed. Install Go 1.21+ or download pre-built binaries."
        exit 1
    }

    $goVer = (go version) -replace 'go version go', '' -replace ' .*', ''
    Write-Info "Building from source with Go $goVer..."

    $scriptDir = Split-Path -Parent $MyInvocation.ScriptName
    if (-not (Test-Path "$scriptDir\go.mod")) {
        $scriptDir = $PSScriptRoot
    }
    if (-not (Test-Path "$scriptDir\go.mod")) {
        Write-Err "Cannot find SOVA source code"
        exit 1
    }

    Push-Location $scriptDir
    try {
        go mod download
        Write-Info "Building server..."
        go build -ldflags "-s -w" -o "$InstallDir\sova-server.exe" ./server/
        Write-Ok "Built sova-server.exe"

        Write-Info "Building client..."
        go build -ldflags "-s -w" -o "$InstallDir\sova.exe" ./client/
        Write-Ok "Built sova.exe"
    } finally {
        Pop-Location
    }
}

function Install-Config {
    $configFile = "$ConfigDir\config.json"
    if (Test-Path $configFile) {
        Write-Warn "Config already exists at $configFile, skipping"
        return
    }

    Write-Info "Generating default configuration..."
    @'
{
  "port": 443,
  "api": {
    "enabled": true,
    "port": 8080
  },
  "security": {
    "enable_pq": true,
    "allowed_users": [],
    "rate_limit": 100
  },
  "transports": ["web_mirror", "cloud_carrier", "shadow_websocket"],
  "sni_list": ["sova.example.com", "cdn.cloudflare.com", "aws.amazon.com"]
}
'@ | Set-Content $configFile -Encoding UTF8
    Write-Ok "Configuration at $configFile"
}

function Add-ToPath {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ($currentPath -notlike "*$InstallDir*") {
        Write-Info "Adding SOVA to system PATH..."
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "Machine")
        $env:Path = "$env:Path;$InstallDir"
        Write-Ok "Added to PATH"
    } else {
        Write-Info "SOVA already in PATH"
    }
}

function Install-WindowsService {
    Write-Info "Registering Windows service..."
    try {
        $svc = Get-Service -Name "SOVA" -ErrorAction SilentlyContinue
        if ($svc) {
            Write-Warn "Service already exists"
            return
        }
        New-Service -Name "SOVA" -BinaryPathName "$InstallDir\sova-server.exe" `
            -DisplayName "SOVA Protocol Server" `
            -Description "Autonomous AI-Powered Protocol Server" `
            -StartupType Automatic | Out-Null
        Write-Ok "Windows service registered"
        Write-Info "Start with: Start-Service SOVA"
    } catch {
        Write-Warn "Service registration skipped: $($_.Exception.Message)"
    }
}

# === Main ===
Show-Banner
$arch = Get-Architecture
Write-Info "Platform: windows/$arch"

Install-Directories

$downloaded = (Install-Binary "server" $arch) -and (Install-Binary "client" $arch)
if (-not $downloaded) {
    Write-Info "Falling back to build from source..."
    Build-FromSource
}

Install-Config
Add-ToPath
Install-WindowsService

Write-Host ""
Write-Ok "SOVA Protocol v$Version installed successfully!"
Write-Host ""
Write-Info "Dashboard:  http://localhost:8080"
Write-Info "Server:     sova-server.exe"
Write-Info "Client:     sova connect <config>"
Write-Info "Config:     $ConfigDir\config.json"
Write-Host ""