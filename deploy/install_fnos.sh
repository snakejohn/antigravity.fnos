#!/bin/bash
set -euo pipefail

APP_DIR="/vol3/@appcenter/antigravity.harness"
DATA_DIR="/vol3/@appdata/antigravity.harness"
BIN_SRC="${1:-./fnpack/app/bin/antigravity.harness}"

echo "[INFO] Installing Antigravity Harness Console to ${APP_DIR}..."
mkdir -p "${APP_DIR}/bin" "${DATA_DIR}"
cp -f "${BIN_SRC}" "${APP_DIR}/bin/antigravity.harness"
chmod +x "${APP_DIR}/bin/antigravity.harness"

cp -f "$(dirname "$0")/antigravity-harness.service" /etc/systemd/system/antigravity-harness.service
systemctl daemon-reload
systemctl enable --now antigravity-harness.service
systemctl status antigravity-harness.service --no-pager
echo "[OK] Console available at http://$(hostname -I | awk '{print $1}'):2399/"
