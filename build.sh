#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
echo "[INFO] Building Vue 3 frontend..."
(cd frontend && npm install --no-audit --no-fund && npm run build)
echo "[INFO] Cross-compiling Linux x86_64 binary..."
mkdir -p fnpack/app/bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o fnpack/app/bin/antigravity.harness .
echo "[OK] Built fnpack/app/bin/antigravity.harness"
