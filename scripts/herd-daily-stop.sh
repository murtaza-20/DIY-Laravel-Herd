#!/usr/bin/env bash
# Stop Herd extras/dashboard started for the day (keeps DB containers unless --all).
#
# Usage:
#   ./scripts/herd-daily-stop.sh           # stop dashboard + extras
#   ./scripts/herd-daily-stop.sh --all     # also herd db stop

set -euo pipefail

ALL=0
[[ "${1:-}" == "--all" ]] && ALL=1

HERD_BIN="${HERD_BIN:-}"
if [[ -z "$HERD_BIN" ]]; then
  if command -v herd >/dev/null 2>&1; then
    HERD_BIN="$(command -v herd)"
  elif [[ -x "$HOME/.local/bin/herd" ]]; then
    HERD_BIN="$HOME/.local/bin/herd"
  fi
fi

printf '[herd] Stopping dashboard…\n'
pkill -f 'herd dashboard' 2>/dev/null || true

if [[ -n "${HERD_BIN:-}" ]]; then
  printf '[herd] Stopping extras…\n'
  "$HERD_BIN" extras stop 2>/dev/null || true
  if [[ "$ALL" == "1" ]]; then
    printf '[herd] Stopping databases…\n'
    "$HERD_BIN" db stop 2>/dev/null || true
  else
    printf '[herd] Databases left running (pass --all to stop them)\n'
  fi
fi

printf '[herd] Done.\n'
