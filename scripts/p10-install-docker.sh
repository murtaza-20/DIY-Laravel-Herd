#!/usr/bin/env bash
# Install Docker Engine for Herd P10 databases
set -euo pipefail

if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
  REAL_USER="$SUDO_USER"
else
  REAL_USER="$(id -un)"
fi

echo "=== Installing docker.io ==="
apt-get update
apt-get install -y docker.io

echo "=== Enabling Docker ==="
systemctl enable --now docker

echo "=== Adding ${REAL_USER} to docker group ==="
usermod -aG docker "$REAL_USER" || true

echo
echo "Docker installed. Next:"
echo "  newgrp docker          # or log out/in"
echo "  herd db start"
echo "  herd db create myapp --type mysql"
echo "  herd db status"
