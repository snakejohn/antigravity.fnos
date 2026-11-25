#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

echo "[INFO] Building Vue 3 frontend..."
(cd frontend && npm install --no-audit --no-fund && npm run build)

echo "[INFO] Cross-compiling Linux x86_64 binary..."
mkdir -p fnpack/app/bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o fnpack/app/bin/antigravity.harness .

echo "[INFO] Normalizing LF line endings and permissions for fnOS packaging..."
for f in fnpack/manifest fnpack/app/ui/config fnpack/config/* fnpack/wizard/* fnpack/cmd/*; do
  [ -f "$f" ] && sed -i 's/\r$//' "$f"
done
chmod 755 fnpack/cmd/* fnpack/app/bin/antigravity.harness

echo "[OK] Ready for fnOS packaging (run 'fnpack build' inside fnpack/ or 'appcenter-cli install-local -d fnpack')"
