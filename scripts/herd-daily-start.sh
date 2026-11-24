#!/usr/bin/env bash
# Everyday Herd startup — databases, optional extras, dashboard.
#
# Usage:
#   ./scripts/herd-daily-start.sh
#   HERD_EXTRAS=1 ./scripts/herd-daily-start.sh      # also Mailhog / phpMyAdmin / MinIO
#   HERD_OPEN_DASHBOARD=1 ./scripts/herd-daily-start.sh
#
# Autostart (optional): copy the desktop entry or enable the user systemd unit
# documented at the bottom of USER_GUIDE.md.

set -euo pipefail

HERD_BIN="${HERD_BIN:-}"
HERD_EXTRAS="${HERD_EXTRAS:-0}"
HERD_OPEN_DASHBOARD="${HERD_OPEN_DASHBOARD:-0}"
DASHBOARD_PORT="${DASHBOARD_PORT:-7835}"
LOG_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/herd"
LOG_FILE="$LOG_DIR/daily-start.log"
DASHBOARD_LOG="$LOG_DIR/dashboard.log"

mkdir -p "$LOG_DIR"

log() { printf '[herd] %s\n' "$*"; }
warn() { printf '[herd] WARN: %s\n' "$*" >&2; }

resolve_herd() {
  if [[ -n "$HERD_BIN" && -x "$HERD_BIN" ]]; then
    return
  fi
  if command -v herd >/dev/null 2>&1; then
    HERD_BIN="$(command -v herd)"
    return
  fi
  if [[ -x "$HOME/.local/bin/herd" ]]; then
    HERD_BIN="$HOME/.local/bin/herd"
    return
  fi
  # Fall back to repo-relative binary when script is run from diy-herd
  local root
  root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
  if [[ -x "$root/herd" ]]; then
    HERD_BIN="$root/herd"
    return
  fi
  warn "herd binary not found. Build/install it first:"
  warn "  cd ~/Code/diy-herd && go build -o herd . && cp herd ~/.local/bin/"
  exit 1
}

docker_ok() {
  docker info >/dev/null 2>&1
}

ensure_docker() {
  if docker_ok; then
    return 0
  fi
  if ! command -v docker >/dev/null 2>&1; then
    warn "Docker is not installed. Skip DB/extras, or run: sudo ./scripts/p10-install-docker.sh"
    return 1
  fi
  # User may be in docker group but this shell session is stale
  if id -nG 2>/dev/null | tr ' ' '\n' | grep -qx docker; then
    warn "Docker socket permission denied in this shell."
    warn "Run once:  newgrp docker"
    warn "Or log out/in, then re-run this script."
    return 1
  fi
  warn "Your user is not in the docker group."
  warn "  sudo usermod -aG docker \"\$USER\""
  warn "  newgrp docker"
  return 1
}

port_listening() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltn "( sport = :$port )" 2>/dev/null | grep -q ":$port"
  else
    curl -sS -m 1 -o /dev/null "http://127.0.0.1:$port/" 2>/dev/null
  fi
}

start_dashboard() {
  if port_listening "$DASHBOARD_PORT"; then
    log "Dashboard already running on http://127.0.0.1:$DASHBOARD_PORT"
    return
  fi

  local args=(dashboard --port "$DASHBOARD_PORT")
  if [[ "$HERD_OPEN_DASHBOARD" != "1" ]]; then
    args+=(--no-open)
  fi

  nohup "$HERD_BIN" "${args[@]}" >>"$DASHBOARD_LOG" 2>&1 &
  local pid=$!
  sleep 0.6
  if port_listening "$DASHBOARD_PORT"; then
    log "Dashboard started (pid $pid) → http://127.0.0.1:$DASHBOARD_PORT"
  else
    warn "Dashboard did not bind to :$DASHBOARD_PORT — see $DASHBOARD_LOG"
  fi
}

main() {
  {
    echo "---- $(date -Is) ----"
    resolve_herd
    export PATH="$(dirname "$HERD_BIN"):$PATH"
    log "Using $HERD_BIN"

    if ! "$HERD_BIN" status >/dev/null 2>&1; then
      warn "herd status failed. Run: herd init"
    fi

    if ensure_docker; then
      log "Starting databases (MySQL / Postgres / Redis)…"
      if "$HERD_BIN" db start; then
        log "Databases ready"
      else
        warn "herd db start failed"
      fi

      if [[ "$HERD_EXTRAS" == "1" ]]; then
        log "Starting extras (Mailhog / phpMyAdmin / MinIO)…"
        if "$HERD_BIN" extras start; then
          log "Extras ready"
        else
          warn "herd extras start failed"
        fi
      else
        log "Skipping extras (set HERD_EXTRAS=1 to enable)"
      fi
    fi

    start_dashboard

    echo
    log "Done. Useful URLs:"
    log "  Dashboard   http://127.0.0.1:$DASHBOARD_PORT"
    if [[ "$HERD_EXTRAS" == "1" ]]; then
      log "  Mailhog     http://127.0.0.1:8025"
      log "  phpMyAdmin  http://127.0.0.1:8081"
      log "  MinIO       http://127.0.0.1:9001"
    fi
    log "Sites: herd sites"
    log "Stop dashboard later: pkill -f 'herd dashboard' || true"
  } | tee -a "$LOG_FILE"
}

main "$@"
