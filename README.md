# Ubuntu Herd MVP

Local PHP/Laravel development environment for Ubuntu — a DIY Herd-style CLI.

Single Go binary (`herd`) that manages multi-PHP projects with `.test` domains via Nginx + Dnsmasq.

**Everyday reference:** [USER_GUIDE.md](USER_GUIDE.md) — what Herd can do, how to use every command, workflows, and troubleshooting.

## Status

**MVP + P8–P11** — implemented. See [IMPLEMENTATION_TRACKER.md](IMPLEMENTATION_TRACKER.md).

Not yet included: Elasticsearch and other stretch goals.

## Prerequisites

- Ubuntu 20.04+ (or similar)
- Go 1.22+ (to build from source)
- One or more system PHP-FPM versions (`/usr/bin/php8.1`, `php8.2`, …)

```bash
sudo apt install php-fpm nginx dnsmasq
# Ubuntu 26.04 currently ships PHP 8.5 via this meta-package.
```

### Multiple PHP versions

Herd is multi-version by design: each site gets its own Nginx vhost → versioned FPM socket (`php81`, `php82`, `php85`, …).

```bash
./herd php list                 # installed vs apt-available
./herd php install 8.1 8.2 8.3  # when those packages exist in apt
./herd serve app ~/code/app --php=8.1

# CLI uses the site's pinned PHP (not system /usr/bin/php):
cd ~/code/app
herd php -v
herd composer install
herd php artisan migrate
```

If apt only has one PHP, add Ondřej Surý's PPA, then install and test:

```bash
sudo add-apt-repository ppa:ondrej/php && sudo apt update
./herd php install 7.4 8.0 8.1 8.2 8.3 8.4 8.5 8.6
./herd init
./scripts/test-multi-php.sh   # INSTALL=0 to skip apt
```

## Build

```bash
go build -o herd .
./herd --help
```

## Quick start

```bash
# One-time host wiring (sudo): nginx include, .test DNS, mkcert + TLS
sudo ./scripts/p8-system-setup.sh

# Register a project → https://myapp.test
./herd serve myapp ~/projects/myapp

# Optional: pin PHP version
./herd serve legacy ~/projects/legacy --php=8.1

# Verify end-to-end
./scripts/p8-verify.sh

# Web dashboard
./herd dashboard

# Databases (requires Docker)
sudo ./scripts/p10-install-docker.sh
newgrp docker
./herd db start
./herd db create myapp --type mysql

# Extras (requires Docker)
./herd extras start
# Mailhog UI :8025 · phpMyAdmin :8081 · MinIO :9000/:9001

# Laravel queues (no Docker needed)
./herd queue work myapp
./herd queue list

./herd status
./herd sites
./herd forget myapp
```

## Commands

| Command | Description |
|---------|-------------|
| `herd init` | Create `~/.herd`, TLS certs, PHP socket map; apply system drop-ins |
| `herd php list` | Show installed + apt-available PHP versions |
| `herd php install <ver...>` | Install `phpX.Y-fpm` via apt when packaged |
| `herd php [args…]` | Run site-pinned PHP CLI from a served project dir |
| `herd php which` | Show which PHP binary the current site uses |
| `herd composer [args…]` | Run Composer with the site-pinned PHP (not global php) |
| `herd serve <name> <path>` | Register project, write Nginx vhost, update `sites.yaml` |
| `herd dashboard` | Open web UI at `http://127.0.0.1:7835` |
| `herd db start\|stop\|status` | Manage MySQL/Postgres/Redis containers |
| `herd db create\|list\|backup\|restore` | Database tools (`--type mysql\|postgres`) |
| `herd extras start\|stop\|status` | Mailhog, phpMyAdmin, MinIO |
| `herd queue work\|list\|stop` | Laravel queue workers |
| `herd status` | Show PHP versions and registered sites |
| `herd sites` | List sites |
| `herd logs <service>` | Tail/show `~/.herd/logs/<service>.log` |
| `herd forget <name>` | Unregister site and remove Nginx config |
| `herd start` / `stop` / `restart` | Service control helpers |

## How it works

- Config and state live under `~/.herd/` (`sites.yaml`, `nginx/sites-enabled/`, `php/`, `logs/`)
- PHP version is taken from `composer.json` (`require.php`) unless `--php` is set
- Sites are served as `https://<name>.test` (HTTP redirects to HTTPS)
- PHP via system FPM sockets: `/run/php/phpX.Y-fpm.sock`
- TLS: `mkcert` preferred (trusted local CA); falls back to openssl self-signed under `~/.herd/ssl/`
- DNS: dnsmasq on `127.0.0.1:53` + systemd-resolved `Domains=~test` (coexists with stub resolver)
- Databases: Docker containers `herd-mysql` / `herd-postgres` / `herd-redis` with data in `~/.herd/data/`
- Extras: Mailhog (:8025), phpMyAdmin (:8081), MinIO (:9000/:9001) via `herd extras`
- Queues: background `php artisan queue:work` via `herd queue work`

Host integration needs sudo once (`scripts/p8-system-setup.sh`). Docker: `scripts/p10-install-docker.sh`.

## Project layout

```
cmd/           # Cobra commands
pkg/php/       # Multi-PHP FPM setup + constraint matching
pkg/composer/  # composer.json PHP requirement parser
pkg/nginx/     # Vhost generation
pkg/docker/    # MySQL/Postgres/Redis containers
pkg/db/        # create/list/backup/restore
pkg/dashboard/ # Web UI on :7835
pkg/config/    # ~/.herd + sites.yaml
pkg/utils/     # Helpers
scripts/       # setup + verify helpers
files/         # Original reference stubs (not built)
```

## Testing

```bash
chmod +x scripts/manual-test.sh
./scripts/manual-test.sh
```

## License

MIT
