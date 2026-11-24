#!/usr/bin/env bash
# One-shot fix: point /etc/nginx/conf.d/herd.conf at the invoking user's ~/.herd
set -euo pipefail

if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
  REAL_HOME="$(getent passwd "$SUDO_USER" | cut -d: -f6)"
else
  REAL_HOME="$HOME"
fi

HERD_DIR="${HERD_DIR:-$REAL_HOME/.herd}"

tee /etc/nginx/conf.d/herd.conf >/dev/null <<EOF
# Managed by Ubuntu Herd
include ${HERD_DIR}/nginx/sites-enabled/*.conf;
EOF

chmod -R a+rX "${HERD_DIR}/nginx" "${HERD_DIR}/ssl" 2>/dev/null || true
chmod 0644 "${HERD_DIR}/ssl/"*.pem 2>/dev/null || true

nginx -t
systemctl reload nginx

echo "Fixed include:"
cat /etc/nginx/conf.d/herd.conf
ss -tlnp | grep -E ':80|:443' || true
