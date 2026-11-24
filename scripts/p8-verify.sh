#!/usr/bin/env bash
# Verify P8 real-deploy: DNS, TLS, PHP via https://demo.test
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="${HOME}/.local/go/bin:${PATH}"

[[ -x ./herd ]] || go build -o herd .

FIXTURE="$ROOT/scripts/fixtures/p8-demo"
mkdir -p "$FIXTURE"
cat > "$FIXTURE/composer.json" <<'EOF'
{ "name": "herd/p8-demo", "require": { "php": "^8.2" } }
EOF
cat > "$FIXTURE/index.php" <<'EOF'
<?php
header('Content-Type: text/plain');
echo "herd-ok php=" . PHP_VERSION . "\n";
EOF
chmod 0755 "$FIXTURE" "$HOME" || true

./herd forget demo >/dev/null 2>&1 || true
./herd serve demo "$FIXTURE" --php=8.2

echo "=== DNS ==="
resolvectl query demo.test || getent hosts demo.test || true

echo "=== HTTPS curl ==="
curl -kfsS --resolve demo.test:443:127.0.0.1 "https://demo.test/" || \
  curl -kfsS "https://demo.test/"

echo
echo "Open: https://demo.test"
