#!/usr/bin/env bash

set -e

LINUX_DIR="tway-linux-amd64"
WINDOWS_DIR="tway-windows-amd64"

echo "==> Cleaning old builds..."

rm -rf "$LINUX_DIR" "$WINDOWS_DIR"
rm -f "$LINUX_DIR.tar.gz" "$WINDOWS_DIR.zip"

echo "==> Creating release directories..."

mkdir -p "$LINUX_DIR"
mkdir -p "$WINDOWS_DIR"

echo "==> Building Linux amd64..."

GOOS=linux GOARCH=amd64 \
go build \
    -o "$LINUX_DIR/tway" \
    ./cmd/tway

echo "==> Building Windows amd64..."

GOOS=windows GOARCH=amd64 \
CGO_ENABLED=1 \
CC=x86_64-w64-mingw32-gcc \
go build \
    -ldflags="-H windowsgui" \
    -o "$WINDOWS_DIR/tway.exe" \
    ./cmd/tway

echo "==> Copying config and icon..."

cp config/config.json "$LINUX_DIR/"
cp assets/tway.ico "$LINUX_DIR/"

cp config/config.json "$WINDOWS_DIR/"
cp assets/tway.ico "$WINDOWS_DIR/"

echo "==> Creating Linux archive..."

tar -czf "$LINUX_DIR.tar.gz" "$LINUX_DIR"

echo "==> Creating Windows archive..."

7z a "$WINDOWS_DIR.zip" "$WINDOWS_DIR"

echo
echo "==> Build complete!"
echo "    $LINUX_DIR.tar.gz"
echo "    $WINDOWS_DIR.zip"