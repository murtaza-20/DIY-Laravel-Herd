#!/usr/bin/env bash
# Manual smoke tests for Ubuntu Herd MVP
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -x ./herd ]]; then
  echo "Building herd..."
  export PATH="${HOME}/.local/go/bin:${PATH}"
  go build -o herd .
fi

PASS=0
FAIL=0

assert_ok() {
  local name="$1"
  shift
  if "$@" >/tmp/herd-test-out.txt 2>/tmp/herd-test-err.txt; then
    echo "PASS: $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $name"
    echo "  stdout: $(cat /tmp/herd-test-out.txt)"
    echo "  stderr: $(cat /tmp/herd-test-err.txt)"
    FAIL=$((FAIL + 1))
  fi
}

assert_contains() {
  local name="$1"
  local needle="$2"
  shift 2
  local out
  out="$("$@" 2>&1)" || true
  if echo "$out" | grep -q "$needle"; then
    echo "PASS: $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $name (expected to contain: $needle)"
    echo "  got: $out"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Herd manual smoke tests ==="

assert_ok "herd --help" ./herd --help
assert_ok "herd init --help" ./herd init --help
assert_ok "herd serve --help" ./herd serve --help
assert_ok "herd status --help" ./herd status --help
assert_ok "herd sites --help" ./herd sites --help
assert_ok "herd logs --help" ./herd logs --help
assert_ok "herd forget --help" ./herd forget --help
assert_ok "herd start --help" ./herd start --help
assert_ok "herd stop --help" ./herd stop --help
assert_ok "herd restart --help" ./herd restart --help
assert_ok "herd php list" ./herd php list
assert_ok "herd php install --help" ./herd php install --help

assert_contains "serve rejects bad name" "invalid project name" ./herd serve 'Bad_Name' /tmp
assert_contains "serve rejects missing path" "does not exist" ./herd serve demo /tmp/herd-missing-project-xyz

# Fixture project
FIXTURE="$ROOT/scripts/fixtures/demo-php"
mkdir -p "$FIXTURE"
cat > "$FIXTURE/composer.json" <<'EOF'
{
  "name": "herd/demo",
  "require": {
    "php": "^8.0"
  }
}
EOF
echo '<?php echo "ok";' > "$FIXTURE/index.php"

PHP_AVAILABLE=0
# Discover versioned CLI binaries (php8.5, php8.2, ...) or unversioned php
shopt -s nullglob
for bin in /usr/bin/php[0-9].[0-9]; do
  if [[ -x "$bin" ]]; then
    PHP_AVAILABLE=1
    break
  fi
done
shopt -u nullglob
if [[ "$PHP_AVAILABLE" -eq 0 && -x /usr/bin/php ]]; then
  PHP_AVAILABLE=1
fi

if [[ "$PHP_AVAILABLE" -eq 1 ]]; then
  echo "PHP detected — running init/serve/status/forget flow"
  assert_ok "herd init" ./herd init
  # forget if leftover
  ./herd forget demo-php >/dev/null 2>&1 || true
  assert_ok "herd serve" ./herd serve demo-php "$FIXTURE"
  assert_ok "herd status" ./herd status
  assert_ok "herd sites" ./herd sites
  assert_contains "sites lists demo" "demo-php" ./herd sites
  assert_ok "herd forget" ./herd forget demo-php
else
  echo "SKIP: no system PHP binaries — init/serve integration skipped"
  assert_contains "init errors without PHP" "no PHP versions found" ./herd init
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
