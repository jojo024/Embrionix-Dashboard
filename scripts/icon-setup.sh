#!/usr/bin/env bash
# Generate favicon.ico for Windows executable embedding.
# Uses a pure-Go generator (no external dependencies).
# Usage: ./scripts/icon-setup.sh

set -euo pipefail

IcoOutput="cmd/server/favicon.ico"

echo "==> Generating favicon.ico for Windows executable"

# Build the icon generator tool
echo "--> Building icon generator..."
go build -o "./cmd/icon-generator/icon-generator" "./cmd/icon-generator/main.go"

# Generate the ICO file
echo "--> Generating $IcoOutput..."
./cmd/icon-generator/icon-generator "$IcoOutput"

echo "==> Done! Icon saved to $IcoOutput"
