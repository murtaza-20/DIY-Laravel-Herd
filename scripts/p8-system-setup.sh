#!/usr/bin/env bash
# P8 host integration: mkcert, nginx include, dnsmasq + systemd-resolved for .test
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# When run as `sudo ./scripts/p8-system-setup.sh`, $HOME becomes /root — use the real user.
if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
  REAL_USER="$SUDO_USER"
  REAL_HOME="$(getent passwd "$SUDO_USER" | cut -d: -f6)"
else
  REAL_USER="$(id -un)"
  REAL_HOME="$HOME"
fi

HERD_DIR="${HERD_DIR:-$REAL_HOME/.herd}"

echo "=== Target user: ${REAL_USER}  HERD_DIR=${HERD_DIR} ==="

echo "=== Installing TLS tools (openssl + mkcert) ==="
apt-get install -y openssl mkcert libnss3-tools

echo "=== Writing Nginx include ==="
tee /etc/nginx/conf.d/herd.conf >/dev/null <<EOF
# Managed by Ubuntu Herd (user: ${REAL_USER})
include ${HERD_DIR}/nginx/sites-enabled/*.conf;
EOF

echo "=== Writing Dnsmasq (.test on 127.0.0.1) ==="
tee /etc/dnsmasq.d/herd.conf >/dev/null <<'EOF'
# Managed by Ubuntu Herd — coexist with systemd-resolved (127.0.0.53)
bind-interfaces
listen-address=127.0.0.1
address=/.test/127.0.0.1
EOF

echo "=== Routing ~test via systemd-resolved ==="
mkdir -p /etc/systemd/resolved.conf.d
tee /etc/systemd/resolved.conf.d/herd.conf >/dev/null <<'EOF'
# Managed by Ubuntu Herd
[Resolve]
DNS=127.0.0.1
Domains=~test
EOF

echo "=== Permissions for www-data traverse ==="
chmod 0751 "$REAL_HOME" || true
chmod -R a+rX "${HERD_DIR}/nginx" "${HERD_DIR}/ssl" 2>/dev/null || true
chmod 0644 "${HERD_DIR}/ssl/"*.pem 2>/dev/null || true

echo "=== Restart DNS services ==="
systemctl restart dnsmasq
systemctl restart systemd-resolved

echo "=== Rebuild Herd + init (certs) as ${REAL_USER} ==="
runuser -u "$REAL_USER" -- env HOME="$REAL_HOME" PATH="${REAL_HOME}/.local/go/bin:/usr/local/go/bin:$PATH" \
  bash -lc "cd '$ROOT' && go build -o herd . && ./herd init"

# Trust mkcert CA for the user
if command -v mkcert >/dev/null; then
  echo "=== mkcert -install (local CA trust) as ${REAL_USER} ==="
  runuser -u "$REAL_USER" -- env HOME="$REAL_HOME" mkcert -install || true
fi

echo "=== Nginx test + reload ==="
nginx -t
systemctl reload nginx

echo
echo "Nginx include now points at:"
cat /etc/nginx/conf.d/herd.conf
echo
echo "Verify:"
echo "  resolvectl query demo.test"
echo "  curl -kfsS https://demo.test/"
echo "Done (user=${REAL_USER})."
