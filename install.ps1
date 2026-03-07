# SOVA Client Installer for Windows

Write-Host "Installing SOVA Client..."

# Download binary (placeholder)
$arch = $env:PROCESSOR_ARCHITECTURE
if ($arch -eq "AMD64") {
    $arch = "amd64"
} else {
    Write-Host "Unsupported architecture: $arch"
    exit 1
}

Write-Host "Downloading SOVA client binary for Windows $arch..."
# Invoke-WebRequest -Uri "https://github.com/sova-protocol/sova/releases/latest/download/sova-client-windows-$arch.exe" -OutFile "$env:ProgramFiles\SOVA\sova.exe"

# Download precompiled binary from releases
$arch = $env:PROCESSOR_ARCHITECTURE
if ($arch -eq "AMD64") {
    $arch = "amd64"
} elseif ($arch -eq "ARM64") {
    $arch = "arm64"
}
$url = "https://github.com/sova-protocol/sova/releases/latest/download/sova-client-windows-$arch.exe"
Write-Host "Скачивание $url..."
# Invoke-WebRequest -Uri $url -OutFile "$env:ProgramFiles\SOVA\sova.exe"

Write-Host "Binary установлен"

# Add to PATH
$path = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($path -notlike "*$env:ProgramFiles\SOVA*") {
    [Environment]::SetEnvironmentVariable("Path", "$path;$env:ProgramFiles\SOVA", "Machine")
}

Write-Host "SOVA Client installed. Use 'sova connect <json_uri>' to connect."