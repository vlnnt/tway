#!/usr/bin/env bash

set -e

cd "$(dirname "$0")"

echo "Building Tway for Linux..."

mkdir -p tway-linux

GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o tway-linux/tway \
    ./cmd/tway

cp -f assets/tway.ico tway-linux/tway.ico

echo
echo "Build completed:"
echo "  tway-linux/tway"
echo "  tway-linux/tway.ico"