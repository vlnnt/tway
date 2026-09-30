#!/usr/bin/env bash

set -e

cd "$(dirname "$0")"

echo "Building Tway for Linux..."

mkdir -p tway

GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o tway/tway \
    ./cmd/tway

cp -f assets/tway.ico tway/tway.ico

echo
echo "Build completed:"
echo "  tway/tway"
echo "  tway/tway.ico"