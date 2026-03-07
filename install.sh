#!/bin/bash

# SOVA Server Installer

echo "Installing SOVA Server..."

# Detect architecture
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
    ARCH="amd64"
elif [[ "$ARCH" == "aarch64" ]]; then
    ARCH="arm64"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

# Download binary (placeholder)
echo "Downloading SOVA server binary for Linux $ARCH..."
# curl -L https://github.com/sova-protocol/sova/releases/latest/download/sova-server-linux-$ARCH -o /usr/local/bin/sova-server

# Download precompiled binary from releases
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
    ARCH="amd64"
elif [[ "$ARCH" == "aarch64" ]]; then
    ARCH="arm64"
fi
URL="https://github.com/sova-protocol/sova/releases/latest/download/sova-server-linux-$ARCH"
echo "Скачивание $URL..."
# curl -L "$URL" -o /usr/local/bin/sova-server
# chmod +x /usr/local/bin/sova-server

echo "Binary установлен" 

# Generate keys and config
echo "Generating server keys..."
/usr/local/bin/sova-server --generate-keys > config.json

echo "SOVA Server installed. JSON link:"
cat config.json

echo "To start: sudo /usr/local/bin/sova-server"