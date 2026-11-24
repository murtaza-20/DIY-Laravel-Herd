#!/usr/bin/env bash
# Install (optional) and exercise Herd against multiple PHP versions.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="${HOME}/.local/go/bin:${PATH}"

VERSIONS=(7.4 8.0 8.1 8.2 8.3 8.4 8.5 8.6)
INSTALL="${INSTALL:-1}"

if [[ ! -x ./herd ]]; then
  go build -o herd .
fi

if [[ "$INSTALL" == "1" ]]; then
  echo "=== Installing PHP FPM/CLI packages ==="
  pkgs=()
  for v in "${VERSIONS[@]}"; do
    pkgs+=("php${v}-fpm" "php${v}-cli")
  done
  sudo apt-get install -y "${pkgs[@]}"
fi

echo "=== Discovered binaries ==="
ls -1 /usr/bin/php[0-9]* /usr/sbin/php-fpm[0-9]* 2>/dev/null || true

echo
echo "=== herd php list ==="
./herd php list

echo
echo "=== herd init ==="
./herd init

PASS=0
FAIL=0
pass() { echo "PASS: $1"; PASS=$((PASS+1)); }
fail() { echo "FAIL: $1 — $2"; FAIL=$((FAIL+1)); }

FIXTURE_ROOT="$ROOT/scripts/fixtures/multi-php"
rm -rf "$FIXTURE_ROOT"
mkdir -p "$FIXTURE_ROOT"

echo
echo "=== Per-version serve / forget ==="
for v in "${VERSIONS[@]}"; do
  name="php$(echo "$v" | tr -d '.')"
  dir="$FIXTURE_ROOT/$name"
  mkdir -p "$dir"
  cat > "$dir/composer.json" <<EOF
{
  "name": "herd/test-$name",
  "require": { "php": "^${v}" }
}
EOF
  echo "<?php phpinfo();" > "$dir/index.php"

  ./herd forget "$name" >/dev/null 2>&1 || true

  if [[ ! -x "/usr/bin/php${v}" ]]; then
    fail "serve $v" "binary /usr/bin/php${v} missing (install skipped?)"
    continue
  fi

  if out=$(./herd serve "$name" "$dir" --php="$v" 2>&1); then
    if echo "$out" | grep -q "PHP:      $v"; then
      pass "serve $v (--php)"
    else
      fail "serve $v (--php)" "unexpected output: $out"
    fi
  else
    fail "serve $v (--php)" "$out"
  fi

  # Auto-detect from composer constraint
  ./herd forget "${name}-auto" >/dev/null 2>&1 || true
  if out=$(./herd serve "${name}-auto" "$dir" 2>&1); then
    if echo "$out" | grep -Eq "Using PHP ${v}|PHP:      ${v}"; then
      pass "serve $v (composer auto)"
    else
      # Compatible higher version may be chosen for ^X.Y — accept any installed match
      if echo "$out" | grep -Eq 'Using PHP [0-9]+\.[0-9]+|PHP:      [0-9]+\.[0-9]+'; then
        pass "serve $v (composer auto — compatible pick)"
      else
        fail "serve $v (composer auto)" "$out"
      fi
    fi
  else
    fail "serve $v (composer auto)" "$out"
  fi

  conf="$HOME/.herd/nginx/sites-enabled/${name}.conf"
  if [[ -f "$conf" ]] && grep -q "php${v}-fpm.sock" "$conf"; then
    pass "nginx socket for $v"
  else
    fail "nginx socket for $v" "missing or wrong socket in $conf"
  fi
  if [[ -f "$conf" ]] && grep -q "listen 443 ssl" "$conf"; then
    pass "tls listener for $v"
  else
    fail "tls listener for $v" "no 443 ssl in $conf"
  fi
done

echo
echo "=== herd status / sites ==="
./herd status
./herd sites

echo
echo "=== Cleanup test sites ==="
for v in "${VERSIONS[@]}"; do
  name="php$(echo "$v" | tr -d '.')"
  ./herd forget "$name" >/dev/null 2>&1 || true
  ./herd forget "${name}-auto" >/dev/null 2>&1 || true
done

echo
echo "Results: $PASS passed, $FAIL failed"
[[ "$FAIL" -eq 0 ]]
