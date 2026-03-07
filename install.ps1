# SOVA Protocol v1.0.0 - Windows Installer
# Run as Administrator: powershell -ExecutionPolicy Bypass -File install.ps1
#Requires -RunAsAdministrator

$ErrorActionPreference = "Stop"
$Version = "1.0.0"
$RepoURL = "https://github.com/IvanChernykh/SOVA"
$BaseURL = "$RepoURL/releases/download/v$Version"
$InstallDir = "$env:ProgramFiles\SOVA"
$ConfigDir = "$env:USERPROFILE\.sova"

function Write-Purple { param($msg) Write-Host $msg -ForegroundColor Magenta }
function Write-Ok     { param($msg) Write-Host "  [OK] $msg" -ForegroundColor Green }
function Write-Info   { param($msg) Write-Host "  [..] $msg" -ForegroundColor Cyan }
function Write-Warn   { param($msg) Write-Host "  [!!] $msg" -ForegroundColor Yellow }
function Write-Err    { param($msg) Write-Host "  [XX] $msg" -ForegroundColor Red }

function Show-AnimatedOwl {
    $owlOpen  = "         ___________`n        /   /   \   \`n       |   | O   O |  |`n       |   |   V   |  |`n        \   \_____/   /`n      // \___________/ \\`n     //   |||||||||||   \\`n    ||    |||||||||||    ||`n           ||   ||`n          _||___||_"
    $owlBlink = "         ___________`n        /   /   \   \`n       |   | *   * |  |`n       |   |   V   |  |`n        \   \_____/   /`n      // \___________/ \\`n     //   |||||||||||   \\`n    ||    |||||||||||    ||`n           ||   ||`n          _||___||_"
    $owlWings = "         ___________`n        /   /   \   \`n       |   | O   O |  |`n       |   |   V   |  |`n        \   \_____/   /`n     /  \___________/  \`n    /    |||||||||||    \`n   /     |||||||||||     \`n           ||   ||`n          _||___||_"
    $frames = @($owlOpen, $owlBlink, $owlOpen, $owlWings, $owlOpen)
    foreach ($frame in $frames) {
        Clear-Host
        Write-Host $frame -ForegroundColor Magenta
        Start-Sleep -Milliseconds 200
    }
}

function Show-Banner {
    if ([Environment]::UserInteractive) {
        try { Show-AnimatedOwl } catch {}
    }
    Write-Host ""
    Write-Purple "  ╔════════════════════════════════════════════════════╗"
    Write-Purple "  ║         ___________                               ║"
    Write-Purple "  ║        /   /   \   \                              ║"
    Write-Purple "  ║       |   | O   O |  |   S O V A  Protocol       ║"
    Write-Purple "  ║       |   |   V   |  |   v$Version                    ║"
    Write-Purple "  ║        \   \_____/   /                            ║"
    Write-Purple "  ║      // \___________/ \\                          ║"
    Write-Purple "  ║                                                   ║"
    Write-Purple "  ║   AI-Powered  |  Post-Quantum  |  Free & Open    ║"
    Write-Purple "  ╚════════════════════════════════════════════════════╝"
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
    New-Item -ItemType Directory -Force -Path "$ConfigDir\profiles" | Out-Null
    New-Item -ItemType Directory -Force -Path "$ConfigDir\logs" | Out-Null
    Write-Ok "Directories created"
}

function Install-Binary {
    param($Component, $Arch)
    $url = "$BaseURL/sova-$Component-windows-$Arch.exe"
    if ($Component -eq "client") {
        $dest = "$InstallDir\sova.exe"
    } else {
        $dest = "$InstallDir\sova-server.exe"
    }

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

function Find-SourceDir {
    # 1. Check if script is in the repo (run from local clone)
    $candidates = @()
    if ($PSScriptRoot -and (Test-Path "$PSScriptRoot\go.mod")) {
        return $PSScriptRoot
    }
    # 2. Check common locations
    $candidates += "$env:USERPROFILE\Desktop\SOVA"
    $candidates += "$env:USERPROFILE\SOVA"
    $candidates += "$env:USERPROFILE\Documents\SOVA"
    $candidates += "C:\SOVA"
    foreach ($dir in $candidates) {
        if (Test-Path "$dir\go.mod") {
            return $dir
        }
    }
    return $null
}

function Build-FromSource {
    $goCmd = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goCmd) {
        Write-Err "Go is not installed. Install Go 1.21+ or download pre-built binaries."
        exit 1
    }

    $goVer = (go version) -replace 'go version go', '' -replace ' .*', ''
    Write-Info "Building from source with Go $goVer..."

    $srcDir = Find-SourceDir
    $cloned = $false

    if (-not $srcDir) {
        # 3. Clone the repo to temp directory
        $gitCmd = Get-Command git -ErrorAction SilentlyContinue
        if ($gitCmd) {
            Write-Info "Cloning SOVA repository..."
            $srcDir = "$env:TEMP\sova-build-$([System.IO.Path]::GetRandomFileName())"
            git clone --depth 1 --branch "v$Version" "$RepoURL.git" $srcDir 2>&1 | Out-Null
            if (-not (Test-Path "$srcDir\go.mod")) {
                git clone --depth 1 "$RepoURL.git" $srcDir 2>&1 | Out-Null
            }
            if (Test-Path "$srcDir\go.mod") {
                $cloned = $true
                Write-Ok "Repository cloned to $srcDir"
            } else {
                Write-Err "Failed to clone repository"
                exit 1
            }
        } else {
            Write-Err "Cannot find SOVA source code and git is not available."
            Write-Err "Either clone the repo manually or install git:"
            Write-Err "  git clone $RepoURL.git"
            Write-Err "  cd SOVA; .\install.ps1"
            exit 1
        }
    } else {
        Write-Info "Found source at $srcDir"
    }

    Push-Location $srcDir
    try {
        go mod download 2>&1 | Out-Null
        Write-Info "Building server..."
        go build -ldflags "-s -w -X main.Version=v$Version" -o "$InstallDir\sova-server.exe" ./server/
        Write-Ok "Built sova-server.exe"

        Write-Info "Building client..."
        go build -ldflags "-s -w -X main.Version=v$Version" -o "$InstallDir\sova.exe" ./client/
        Write-Ok "Built sova.exe"
    } catch {
        Write-Err "Build failed: $($_.Exception.Message)"
        exit 1
    } finally {
        Pop-Location
        if ($cloned -and $srcDir) {
            Remove-Item -Recurse -Force $srcDir -ErrorAction SilentlyContinue
        }
    }
}

function Install-Config {
    $configFile = "$ConfigDir\config.json"
    if (Test-Path $configFile) {
        Write-Warn "Config already exists at $configFile, skipping"
        return
    }

    Write-Info "Generating default configuration..."
    $json = '{' + "`n"
    $json += '  "mode": "local",' + "`n"
    $json += '  "listen_addr": "127.0.0.1",' + "`n"
    $json += '  "listen_port": 1080,' + "`n"
    $json += '  "server_addr": "",' + "`n"
    $json += '  "server_port": 443,' + "`n"
    $json += '  "encryption": {' + "`n"
    $json += '    "algorithm": "aes-256-gcm",' + "`n"
    $json += '    "pq_enabled": true,' + "`n"
    $json += '    "zkp_enabled": true' + "`n"
    $json += '  },' + "`n"
    $json += '  "stealth": {' + "`n"
    $json += '    "enabled": true,' + "`n"
    $json += '    "profile": "chrome",' + "`n"
    $json += '    "jitter_ms": 50,' + "`n"
    $json += '    "padding_enabled": true,' + "`n"
    $json += '    "decoy_enabled": false,' + "`n"
    $json += '    "tls_fingerprint": "chrome"' + "`n"
    $json += '  },' + "`n"
    $json += '  "api": {' + "`n"
    $json += '    "enabled": true,' + "`n"
    $json += '    "port": 8080,' + "`n"
    $json += '    "host": "127.0.0.1",' + "`n"
    $json += '    "auth_key": ""' + "`n"
    $json += '  },' + "`n"
    $json += '  "dns": {' + "`n"
    $json += '    "enabled": false,' + "`n"
    $json += '    "port": 5353,' + "`n"
    $json += '    "upstream": "8.8.8.8:53"' + "`n"
    $json += '  },' + "`n"
    $json += '  "log_level": "info",' + "`n"
    $json += '  "log_file": "",' + "`n"
    $json += '  "features": {' + "`n"
    $json += '    "compression": true,' + "`n"
    $json += '    "connection_pool": true,' + "`n"
    $json += '    "smart_routing": true,' + "`n"
    $json += '    "mesh_network": false,' + "`n"
    $json += '    "offline_first": false,' + "`n"
    $json += '    "ai_adapter": true,' + "`n"
    $json += '    "dashboard": true,' + "`n"
    $json += '    "auto_proxy": false' + "`n"
    $json += '  },' + "`n"
    $json += '  "transport": {' + "`n"
    $json += '    "mode": "auto",' + "`n"
    $json += '    "sni_list": ["www.google.com", "cdn.cloudflare.com", "aws.amazon.com"],' + "`n"
    $json += '    "cdn_list": ["cdn.cloudflare.com", "fastly.net"],' + "`n"
    $json += '    "fallback": true' + "`n"
    $json += '  }' + "`n"
    $json += '}'
    $json | Set-Content $configFile -Encoding UTF8
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
            -Description "SOVA Protocol - AI-Powered Post-Quantum Tunnel Server" `
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

$serverOk = Install-Binary "server" $arch
$clientOk = Install-Binary "client" $arch
if (-not $serverOk -or -not $clientOk) {
    Write-Info "Falling back to build from source..."
    Build-FromSource
}

Install-Config
Add-ToPath
Install-WindowsService

Write-Host ""
Write-Purple "  ╔════════════════════════════════════════════════════╗"
Write-Purple "  ║  SOVA Protocol v$Version installed successfully!       ║"
Write-Purple "  ╚════════════════════════════════════════════════════╝"
Write-Host ""
Write-Info "Client:     sova                     (SOCKS5 tunnel)"
Write-Info "Server:     sova-server              (relay server)"
Write-Info "API:        http://127.0.0.1:8080/api/"
Write-Info "Config:     $ConfigDir\config.json"
Write-Info "Proxy:      SOCKS5 127.0.0.1:1080"
Write-Host ""
Write-Info "Quick start:"
Write-Host "  sova                               # Start tunnel" -ForegroundColor White
Write-Host "  sova connect server.example.com    # Remote server" -ForegroundColor White
Write-Host "  sova help                          # All commands" -ForegroundColor White
Write-Host ""