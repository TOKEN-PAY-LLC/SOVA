# SOVA Protocol v1.0.0 - Windows Installer
# Run as Administrator: powershell -ExecutionPolicy Bypass -File install.ps1
#Requires -RunAsAdministrator

$ErrorActionPreference = 'Stop'
$Version = '1.0.0'
$RepoURL = 'https://github.com/IvanChernykh/SOVA'
$BaseURL = $RepoURL + '/releases/download/v' + $Version
$InstallDir = $env:ProgramFiles + '\SOVA'
$ConfigDir = $env:USERPROFILE + '\.sova'

function Write-Purple {
    param([string]$msg)
    Write-Host $msg -ForegroundColor Magenta
}
function Write-Ok {
    param([string]$msg)
    Write-Host ('  [OK] ' + $msg) -ForegroundColor Green
}
function Write-Info {
    param([string]$msg)
    Write-Host ('  [..] ' + $msg) -ForegroundColor Cyan
}
function Write-Warn {
    param([string]$msg)
    Write-Host ('  [!!] ' + $msg) -ForegroundColor Yellow
}
function Write-Err {
    param([string]$msg)
    Write-Host ('  [XX] ' + $msg) -ForegroundColor Red
}

function Show-AnimatedOwl {
    $f1 = @()
    $f1 += '         ___________'
    $f1 += '        /   /   \   \'
    $f1 += '       |   | O   O |  |'
    $f1 += '       |   |   V   |  |'
    $f1 += '        \   \_____/   /'
    $f1 += '      // \___________/ \\'
    $f1 += '     //   |||||||||||   \\'
    $f1 += '    ||    |||||||||||    ||'
    $f1 += '           ||   ||'
    $f1 += '          _||___||_'

    $f2 = @()
    $f2 += '         ___________'
    $f2 += '        /   /   \   \'
    $f2 += '       |   | *   * |  |'
    $f2 += '       |   |   V   |  |'
    $f2 += '        \   \_____/   /'
    $f2 += '      // \___________/ \\'
    $f2 += '     //   |||||||||||   \\'
    $f2 += '    ||    |||||||||||    ||'
    $f2 += '           ||   ||'
    $f2 += '          _||___||_'

    $f3 = @()
    $f3 += '         ___________'
    $f3 += '        /   /   \   \'
    $f3 += '       |   | O   O |  |'
    $f3 += '       |   |   V   |  |'
    $f3 += '        \   \_____/   /'
    $f3 += '     /  \___________/  \'
    $f3 += '    /    |||||||||||    \'
    $f3 += '   /     |||||||||||     \'
    $f3 += '           ||   ||'
    $f3 += '          _||___||_'

    $frames = @($f1, $f2, $f1, $f3, $f1)
    foreach ($frame in $frames) {
        Clear-Host
        $text = $frame -join "`n"
        Write-Host $text -ForegroundColor Magenta
        Start-Sleep -Milliseconds 200
    }
}

function Show-Banner {
    if ([Environment]::UserInteractive) {
        try { Show-AnimatedOwl } catch { }
    }
    Write-Host ''
    Write-Purple '  +====================================================+'
    Write-Purple '  |         ___________                                 |'
    Write-Purple '  |        /   /   \   \                                |'
    Write-Purple ('  |       |   | O   O |  |   S O V A  Protocol        |')
    Write-Purple ('  |       |   |   V   |  |   v' + $Version + '                      |')
    Write-Purple '  |        \   \_____/   /                              |'
    Write-Purple '  |      // \___________/ \\                            |'
    Write-Purple '  |                                                     |'
    Write-Purple '  |   AI-Powered  |  Post-Quantum  |  Free & Open      |'
    Write-Purple '  +====================================================+'
    Write-Host ''
    Write-Host '  github.com/IvanChernykh/SOVA' -ForegroundColor Cyan
    Write-Host ''
}

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        'AMD64' { return 'amd64' }
        'ARM64' { return 'arm64' }
        'x86'   { return '386' }
        default {
            Write-Err ('Unsupported architecture: ' + $arch)
            exit 1
        }
    }
}

function Install-Directories {
    Write-Info 'Creating directories...'
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
    New-Item -ItemType Directory -Force -Path ($ConfigDir + '\profiles') | Out-Null
    New-Item -ItemType Directory -Force -Path ($ConfigDir + '\logs') | Out-Null
    Write-Ok 'Directories created'
}

function Install-FromRelease {
    param([string]$Arch)
    $zipName = 'sova-windows-' + $Arch + '-v' + $Version + '.zip'
    $url = $BaseURL + '/' + $zipName
    $zipPath = $env:TEMP + '\' + $zipName

    Write-Info ('Downloading ' + $zipName + '...')
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing -ErrorAction Stop
        Write-Ok 'Downloaded release archive'
    } catch {
        Write-Warn ('Download failed: ' + $_.Exception.Message)
        return $false
    }

    Write-Info 'Extracting binaries...'
    try {
        $rnd = [System.IO.Path]::GetRandomFileName()
        $extractDir = $env:TEMP + '\sova-extract-' + $rnd
        New-Item -ItemType Directory -Force -Path $extractDir | Out-Null
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractDir)

        $serverSrc = Get-ChildItem -Path $extractDir -Filter 'sova-server*' -Recurse | Select-Object -First 1
        $clientSrc = Get-ChildItem -Path $extractDir -Filter 'sova-windows*' -Recurse | Select-Object -First 1
        if (-not $clientSrc) {
            $clientSrc = Get-ChildItem -Path $extractDir -Filter 'sova-*' -Recurse |
                Where-Object { $_.Name -notmatch 'server' } | Select-Object -First 1
        }

        if ($serverSrc) {
            $dest = $InstallDir + '\sova-server.exe'
            Copy-Item $serverSrc.FullName $dest -Force
            Write-Ok 'Installed sova-server.exe'
        }
        if ($clientSrc) {
            $dest = $InstallDir + '\sova.exe'
            Copy-Item $clientSrc.FullName $dest -Force
            Write-Ok 'Installed sova.exe'
        }

        Remove-Item -Recurse -Force $extractDir -ErrorAction SilentlyContinue
        Remove-Item -Force $zipPath -ErrorAction SilentlyContinue

        if ($serverSrc -and $clientSrc) { return $true }
        Write-Warn 'Some binaries missing from archive'
        return $false
    } catch {
        Write-Warn ('Extract failed: ' + $_.Exception.Message)
        return $false
    }
}

function Find-SourceDir {
    if ($PSScriptRoot -and (Test-Path ($PSScriptRoot + '\go.mod'))) {
        return $PSScriptRoot
    }
    $candidates = @(
        ($env:USERPROFILE + '\Desktop\SOVA'),
        ($env:USERPROFILE + '\SOVA'),
        ($env:USERPROFILE + '\Documents\SOVA'),
        'C:\SOVA'
    )
    foreach ($dir in $candidates) {
        if (Test-Path ($dir + '\go.mod')) {
            return $dir
        }
    }
    return $null
}

function Build-FromSource {
    $goCmd = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goCmd) {
        Write-Err 'Go is not installed. Install Go 1.21+ or download pre-built binaries.'
        exit 1
    }

    $goVer = (go version) -replace 'go version go','' -replace ' .*',''
    Write-Info ('Building from source with Go ' + $goVer + '...')

    $srcDir = Find-SourceDir
    $cloned = $false

    if (-not $srcDir) {
        $gitCmd = Get-Command git -ErrorAction SilentlyContinue
        if ($gitCmd) {
            Write-Info 'Cloning SOVA repository...'
            $rnd = [System.IO.Path]::GetRandomFileName()
            $srcDir = $env:TEMP + '\sova-build-' + $rnd
            $cloneUrl = $RepoURL + '.git'
            git clone --depth 1 --branch ('v' + $Version) $cloneUrl $srcDir 2>&1 | Out-Null
            if (-not (Test-Path ($srcDir + '\go.mod'))) {
                git clone --depth 1 $cloneUrl $srcDir 2>&1 | Out-Null
            }
            if (Test-Path ($srcDir + '\go.mod')) {
                $cloned = $true
                Write-Ok ('Repository cloned to ' + $srcDir)
            } else {
                Write-Err 'Failed to clone repository'
                exit 1
            }
        } else {
            Write-Err 'Cannot find SOVA source code and git is not available.'
            Write-Err 'Either clone the repo manually or install git:'
            Write-Err ('  git clone ' + $RepoURL + '.git')
            Write-Err '  cd SOVA; .\install.ps1'
            exit 1
        }
    } else {
        Write-Info ('Found source at ' + $srcDir)
    }

    Push-Location $srcDir
    try {
        go mod download 2>&1 | Out-Null
        Write-Info 'Building server...'
        $ldflags = '-s -w -X main.Version=v' + $Version
        $serverOut = $InstallDir + '\sova-server.exe'
        go build -ldflags $ldflags -o $serverOut ./server/
        Write-Ok 'Built sova-server.exe'

        Write-Info 'Building client...'
        $clientOut = $InstallDir + '\sova.exe'
        go build -ldflags $ldflags -o $clientOut ./client/
        Write-Ok 'Built sova.exe'
    } catch {
        Write-Err ('Build failed: ' + $_.Exception.Message)
        exit 1
    } finally {
        Pop-Location
        if ($cloned -and $srcDir) {
            Remove-Item -Recurse -Force $srcDir -ErrorAction SilentlyContinue
        }
    }
}

function Install-Config {
    $configFile = $ConfigDir + '\config.json'
    if (Test-Path $configFile) {
        Write-Warn ('Config already exists at ' + $configFile + ', skipping')
        return
    }

    Write-Info 'Generating default configuration...'
    $lines = @()
    $lines += '{'
    $lines += '  "mode": "local",'
    $lines += '  "listen_addr": "127.0.0.1",'
    $lines += '  "listen_port": 1080,'
    $lines += '  "server_addr": "",'
    $lines += '  "server_port": 443,'
    $lines += '  "encryption": {'
    $lines += '    "algorithm": "aes-256-gcm",'
    $lines += '    "pq_enabled": true,'
    $lines += '    "zkp_enabled": true'
    $lines += '  },'
    $lines += '  "stealth": {'
    $lines += '    "enabled": true,'
    $lines += '    "profile": "chrome",'
    $lines += '    "jitter_ms": 50,'
    $lines += '    "padding_enabled": true,'
    $lines += '    "decoy_enabled": false,'
    $lines += '    "tls_fingerprint": "chrome"'
    $lines += '  },'
    $lines += '  "api": {'
    $lines += '    "enabled": true,'
    $lines += '    "port": 8080,'
    $lines += '    "host": "127.0.0.1",'
    $lines += '    "auth_key": ""'
    $lines += '  },'
    $lines += '  "dns": {'
    $lines += '    "enabled": false,'
    $lines += '    "port": 5353,'
    $lines += '    "upstream": "8.8.8.8:53"'
    $lines += '  },'
    $lines += '  "log_level": "info",'
    $lines += '  "log_file": "",'
    $lines += '  "features": {'
    $lines += '    "compression": true,'
    $lines += '    "connection_pool": true,'
    $lines += '    "smart_routing": true,'
    $lines += '    "mesh_network": false,'
    $lines += '    "offline_first": false,'
    $lines += '    "ai_adapter": true,'
    $lines += '    "dashboard": true,'
    $lines += '    "auto_proxy": false'
    $lines += '  },'
    $lines += '  "transport": {'
    $lines += '    "mode": "auto",'
    $lines += '    "sni_list": ["www.google.com", "cdn.cloudflare.com", "aws.amazon.com"],'
    $lines += '    "cdn_list": ["cdn.cloudflare.com", "fastly.net"],'
    $lines += '    "fallback": true'
    $lines += '  }'
    $lines += '}'
    $lines | Set-Content $configFile -Encoding UTF8
    Write-Ok ('Configuration saved to ' + $configFile)
}

function Add-ToPath {
    $target = [System.EnvironmentVariableTarget]::Machine
    $currentPath = [Environment]::GetEnvironmentVariable('Path', $target)
    if ($currentPath -notlike ('*' + $InstallDir + '*')) {
        Write-Info 'Adding SOVA to system PATH...'
        $newPath = $currentPath + ';' + $InstallDir
        [Environment]::SetEnvironmentVariable('Path', $newPath, $target)
        $env:Path = $env:Path + ';' + $InstallDir
        Write-Ok 'Added to PATH'
    } else {
        Write-Info 'SOVA already in PATH'
    }
}

function Install-WindowsService {
    Write-Info 'Registering Windows service...'
    try {
        $svc = Get-Service -Name 'SOVA' -ErrorAction SilentlyContinue
        if ($svc) {
            Write-Warn 'Service already exists'
            return
        }
        $binPath = $InstallDir + '\sova-server.exe'
        New-Service -Name 'SOVA' -BinaryPathName $binPath `
            -DisplayName 'SOVA Protocol Server' `
            -Description 'SOVA Protocol - AI-Powered Post-Quantum Tunnel Server' `
            -StartupType Automatic | Out-Null
        Write-Ok 'Windows service registered'
        Write-Info 'Start with: Start-Service SOVA'
    } catch {
        Write-Warn ('Service registration skipped: ' + $_.Exception.Message)
    }
}

# === Main ===
Show-Banner
$arch = Get-Architecture
Write-Info ('Platform: windows/' + $arch)

Install-Directories

$downloaded = Install-FromRelease $arch
if (-not $downloaded) {
    Write-Info 'Falling back to build from source...'
    Build-FromSource
}

Install-Config
Add-ToPath
Install-WindowsService

Write-Host ''
Write-Purple '  +====================================================+'
Write-Purple ('  |  SOVA Protocol v' + $Version + ' installed successfully!         |')
Write-Purple '  +====================================================+'
Write-Host ''
Write-Info 'Client:     sova                     (SOCKS5 tunnel)'
Write-Info 'Server:     sova-server              (relay server)'
Write-Info 'API:        http://127.0.0.1:8080/api/'
Write-Info ('Config:     ' + $ConfigDir + '\config.json')
Write-Info 'Proxy:      SOCKS5 127.0.0.1:1080'
Write-Host ''
Write-Host '  !!! RESTART YOUR TERMINAL TO USE sova COMMANDS !!!' -ForegroundColor Red -BackgroundColor Black
Write-Host '  !!! PEREZAPUSTITE TERMINAL DLYA ISPOLZOVANIYA sova !!!' -ForegroundColor Red -BackgroundColor Black
Write-Host ''
Write-Info 'Quick start (after terminal restart):'
Write-Host '  sova                               # Interactive menu' -ForegroundColor White
Write-Host '  sova start                         # Start tunnel' -ForegroundColor White
Write-Host '  sova connect server.example.com    # Remote server' -ForegroundColor White
Write-Host '  sova help                          # All commands' -ForegroundColor White
Write-Host ''