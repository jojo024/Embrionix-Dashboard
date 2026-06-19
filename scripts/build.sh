#!/usr/bin/env bash
# Build Embrionix Dashboard for Linux / macOS.
# Usage: ./scripts/build.sh [VERSION] [OUTPUT_DIR]
set -euo pipefail

VERSION="${1:-dev}"
OUTPUT_DIR="${2:-./dist}"

echo "==> Building Embrionix Dashboard v${VERSION}"

# Build frontend
echo "--> Building frontend..."
(cd web && npm ci && npm run build)

# Build backend
echo "--> Building backend (linux/amd64)..."
mkdir -p "${OUTPUT_DIR}"

# For Windows builds, prepare rsrc + icon (requires ImageMagick + rsrc tool)
# On Linux/macOS, skip icon embedding
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
    echo "--> Preparing Windows icon..."
    if [ -f "cmd/server/favicon.ico" ]; then
        go install github.com/akavel/rsrc@latest
        (cd cmd/server && rsrc -ico favicon.ico)
    else
        echo "    Icon not found. Skipping rsrc. Run: ./scripts/icon-setup.ps1 on Windows"
    fi
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o "${OUTPUT_DIR}/embrionix-dashboard" \
    ./cmd/server/

# Package
echo "--> Packaging..."
cp -r web/dist        "${OUTPUT_DIR}/web"
mkdir -p              "${OUTPUT_DIR}/configs"
cp configs/config.yaml "${OUTPUT_DIR}/configs/"

echo "==> Done. Output in ${OUTPUT_DIR}"
echo "    Run: ${OUTPUT_DIR}/embrionix-dashboard"
