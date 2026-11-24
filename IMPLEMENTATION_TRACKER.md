# DIY Herd — Implementation Tracker

**Overall status:** MVP + P8–P11 complete  
**Last updated:** 2026-07-25  
**Docs:** [USER_GUIDE.md](USER_GUIDE.md) (day-to-day reference)  
**Next action:** Day-to-day use; optional Elasticsearch later

---

## Phase checklist

- [x] Phase 0 — Project setup
- [x] Phase 1 — Core packages
- [x] Phase 2 — Command framework
- [x] Phase 3 — `herd init`
- [x] Phase 4 — `herd serve` + sites
- [x] Phase 5 — Integration
- [x] Phase 6 — Manual testing
- [x] Phase 7 — Polish + README
- [ ] Post-MVP P8–P11 (deferred until host has PHP/Nginx; tracked below)

---

## Gap log

| Gap | Status |
|-----|--------|
| `pkg/php/manager.go` marked PROVIDED but missing from docs | **Resolved** — implemented in Phase 1 |
| `pkg/composer/parser.go` marked PROVIDED but missing | **Resolved** — implemented in Phase 1 |
| Multi-PHP reference MDs missing from workspace | Inferred from COMPLETE_CODE APIs + constraint parser |
| `files/` flat multi-package layout cannot build | Kept as reference; real code under `cmd/` + `pkg/` |
| Go toolchain not on PATH initially | Installed user-local Go `1.22.12` at `~/.local/go` |
| No system PHP on build host | **Resolved** — host has PHP 8.5 via `php-fpm` meta-package (`/usr/bin/php8.5`) |
| Hardcoded PHP candidates stopped at 8.4 | **Resolved** — scan CLI/FPM/`/etc/php` + KnownPHPVersions |
| Ubuntu 26.04 apt only ships PHP 8.5 | Documented; `herd php list|install` + multi-PHP hint (ondrej PPA on supported releases) |

---

## Phase notes

### Phase 0
- **Date:** 2026-07-23
- **Done:** Repo-root Go module, dirs (`cmd/`, `pkg/*`, `templates/`, `scripts/`), `main.go`, lean `go.mod` (cobra/logrus/yaml only), tracker created
- **Build result:** Module ready (full binary after Phase 1–2)
- **Blockers:** Needed user-local Go install (no system Go)

### Phase 1
- **Date:** 2026-07-23
- **Done:** `pkg/utils`, `pkg/config` (loader+sites), `pkg/services`, `pkg/nginx` (with `PHPVersion` sockets), `pkg/composer` (constraint parse), `pkg/php` (Initialize + FindCompatibleVersion)
- **Build result:** `go build ./pkg/...` OK
- **Blockers:** none

### Phase 2
- **Date:** 2026-07-23
- **Done:** `cmd/root.go`, `status.go`, `logs.go`, `sites.go`, `control.go`
- **Build result:** Commands registered in help
- **Blockers:** none

### Phase 3
- **Date:** 2026-07-23
- **Done:** `cmd/init.go` — dirs, PHP detect, FPM pool files, nginx.conf, dnsmasq with sudo `-n` + manual fallback
- **Build result:** `./herd init --help` OK; without PHP returns clear install hint
- **Blockers:** Host has no PHP packages (expected for this machine)

### Phase 4
- **Date:** 2026-07-23
- **Done:** `cmd/serve.go` — name/path validation, `--php`, composer auto-detect, nginx write, sites.yaml, reload fallback
- **Build result:** `./herd serve --help` OK
- **Blockers:** Full serve needs PHP + prior `init`

### Phase 5
- **Date:** 2026-07-23
- **Done:** `go mod tidy`, `go build -o herd`, help smoke for all commands
- **Build result:** Clean binary (~6.3MB); all commands listed
- **Blockers:** none

### Phase 6
- **Date:** 2026-07-23
- **Done:** `scripts/manual-test.sh` — help gates, validation errors, init-without-PHP path; skips live serve when no PHP
- **Build result:** **13 passed, 0 failed**
- **Blockers:** No `phpX.Y` binaries on host — integration path skipped

### Phase 7
- **Date:** 2026-07-23
- **Done:** Root `README.md` (MVP-accurate), `.gitignore`, clearer init/status/serve errors, tracker finalized
- **Build result:** MVP success criteria met (modulo live PHP/Nginx on this host)
- **Blockers:** none for code; runtime stack install remains a host ops step

---

## Post-MVP

### P8 Real deploy + SSL — complete
- [x] Use system FPM sockets `/run/php/phpX.Y-fpm.sock`
- [x] Nginx vhosts with HTTP→HTTPS + TLS (`pkg/ssl`, mkcert/openssl)
- [x] Dnsmasq bind `127.0.0.1` + systemd-resolved `Domains=~test`
- [x] System drop-ins via `pkg/system` + `scripts/p8-system-setup.sh`
- [x] Fixed sudo `$HOME` → `$SUDO_USER` include path (`p8-fix-nginx-include.sh`)
- [x] Host verify — HTTPS `.test` working

### P9 Dashboard — complete
- [x] Gin API on `127.0.0.1:7835` (`pkg/dashboard`)
- [x] Embedded UI (sites, PHP, services, logs, forget, restart)
- [x] `herd dashboard` / `--port` / `--no-open`
- [x] Smoke: `/api/status`, `/`, `/app.css`, `/app.js` → 200

### P10 Databases — code complete (Docker install on host pending)
- [x] `pkg/docker` container manager (mysql/postgres/redis via docker/podman CLI)
- [x] `pkg/db` create/list/backup/restore
- [x] `herd db start|stop|status|create|list|backup|restore`
- [x] Dashboard shows database container status
- [x] `scripts/p10-install-docker.sh`
- [ ] Host: install Docker + `herd db start` smoke test

### P11 Queues / extras — complete
- [x] `herd queue work|list|stop` — Laravel `artisan queue:work` with PID tracking
- [x] `herd extras start|stop|status` — Mailhog, phpMyAdmin, MinIO (Docker)
- [x] Shared Docker network `herd` for phpMyAdmin → MySQL
- [x] Dashboard shows extras + queue workers

### Optional later
- [ ] Elasticsearch
- [ ] Host Docker install smoke for db + extras

### Site-pinned PHP CLI — complete
- [x] `FindSiteByPath` / cwd resolution
- [x] `herd php [args…]` / `herd php which` use site PHP (not `/usr/bin/php`)
- [x] `herd composer [args…]` runs Composer via site PHP
- [x] `--site=<name>` override when outside project dir


---

## MVP success criteria (from DAY_BUILD_PACKAGE)

- [x] `go build -o herd` succeeds
- [x] Help for all commands works
- [x] `init` / `serve` / `status` / `sites` work (verified with PHP 8.5 from `php-fpm`)
- [x] Clear errors for missing deps / uninitialized herd
- [x] Accurate root README present
