# DIY Herd — User Guide

Personal local PHP/Laravel environment for Ubuntu (Herd-style).  
Use this as your everyday reference.

**Binary:** `herd` (usually `~/.local/bin/herd`)  
**State:** `~/.herd/`  
**URLs:** `https://<site-name>.test`

---

## Table of contents

1. [What Herd does](#1-what-herd-does)
2. [One-time install](#2-one-time-install)
3. [Core concepts](#3-core-concepts)
4. [Sites (`.test` domains)](#4-sites-test-domains)
5. [PHP versions (CLI + FPM)](#5-php-versions-cli--fpm)
6. [Composer & Artisan](#6-composer--artisan)
7. [Databases (Docker)](#7-databases-docker)
8. [Extras (Mailhog, phpMyAdmin, MinIO)](#8-extras-mailhog-phpmyadmin-minio)
9. [Laravel queues](#9-laravel-queues)
10. [Dashboard](#10-dashboard)
11. [Services, logs, status](#11-services-logs-status)
12. [Typical Laravel workflow](#12-typical-laravel-workflow)
13. [Useful paths](#13-useful-paths)
14. [Troubleshooting](#14-troubleshooting)
15. [Command cheat sheet](#15-command-cheat-sheet)

---

## 1. What Herd does

| Capability | How |
|------------|-----|
| Local HTTPS sites | `https://name.test` via Nginx + local TLS |
| Multi-PHP | Each site pinned to PHP 7.4–8.6 (FPM socket) |
| Site-matched CLI | `herd php` / `herd composer` use that site’s PHP |
| DNS for `.test` | Dnsmasq + systemd-resolved |
| Databases | Docker: MySQL, Postgres, Redis |
| Dev extras | Mailhog, phpMyAdmin, MinIO |
| Queues | Background `artisan queue:work` |
| Dashboard | Web UI on `http://127.0.0.1:7835` |

Herd does **not** replace Laravel’s own tooling (`npm`, Vite, Horizon, Octane). It wires the host stack so those tools work with stable local URLs and DBs.

---

## 2. One-time install

### 2.1 System packages

```bash
sudo apt update
sudo apt install -y nginx dnsmasq php-fpm openssl
# Optional but recommended for trusted HTTPS:
# mkcert (install via your preferred method)
```

### 2.2 Multiple PHP versions (recommended)

Ubuntu’s default archive may ship only one PHP. For parallel versions:

```bash
sudo add-apt-repository ppa:ondrej/php
sudo apt update
herd php install 8.1 8.2 8.3 8.4   # or whatever you need
```

Also install common Laravel extensions for each version you use:

```bash
# Example for 8.4 — repeat for other versions
sudo apt install -y \
  php8.4-fpm php8.4-cli php8.4-xml php8.4-mbstring \
  php8.4-curl php8.4-zip php8.4-gd php8.4-intl \
  php8.4-mysql php8.4-sqlite3 php8.4-bcmath
sudo systemctl restart php8.4-fpm
```

### 2.3 Build / install the `herd` binary

From the diy-herd repo:

```bash
cd ~/Code/diy-herd
export PATH="$HOME/.local/go/bin:$PATH"   # if Go is user-local
go build -o herd .
mkdir -p ~/.local/bin
cp -f ./herd ~/.local/bin/herd
# Ensure ~/.local/bin is on PATH (e.g. in ~/.bashrc)
```

### 2.4 Initialize Herd + host wiring

```bash
herd init
sudo ./scripts/p8-system-setup.sh   # Nginx include, .test DNS, TLS helpers
./scripts/p8-verify.sh              # optional end-to-end check
```

### 2.5 Docker (for `herd db` / `herd extras`)

```bash
# If Docker not installed:
sudo ./scripts/p10-install-docker.sh

# Allow your user to talk to Docker (once):
sudo usermod -aG docker "$USER"
newgrp docker          # or log out/in
docker info            # must work without sudo
```

---

## 3. Core concepts

### Site

A registered project under a short name:

- URL: `https://<name>.test`
- Path: absolute project directory
- PHP: major.minor (e.g. `8.4`) used by both **Nginx→FPM** and **`herd php` / `herd composer`**

Registry: `~/.herd/sites.yaml`

### PHP pinning

1. `herd serve … --php=8.2` → explicit  
2. Else `composer.json` `require.php` → best installed match  
3. Else latest installed PHP

### CLI vs FPM

| Context | Binary / socket |
|---------|-----------------|
| Browser / Nginx | `/run/php/phpX.Y-fpm.sock` as `www-data` |
| `herd php` / `herd composer` | `/usr/bin/phpX.Y` |
| Bare `php` / `composer` | System default (often newest) — **avoid inside projects** |

### Permissions

PHP-FPM runs as `www-data`. On `herd serve` for Laravel sites, Herd opens `storage/` and `bootstrap/cache` for `www-data` (ACL preferred).

---

## 4. Sites (`.test` domains)

### Register / update a site

```bash
herd serve myapp ~/Code/myapp
herd serve myapp ~/Code/myapp --php=8.2
```

Creates/updates:

- Nginx vhost under `~/.herd/nginx/sites-enabled/`
- Entry in `~/.herd/sites.yaml`
- HTTPS with local certs under `~/.herd/ssl/`
- Laravel storage permissions for `www-data`

If Nginx reload asks for sudo and fails:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

### List & inspect

```bash
herd sites
herd status
```

### Remove a site

```bash
herd forget myapp
```

Removes registry entry and Nginx config. Does **not** delete project files or databases.

---

## 5. PHP versions (CLI + FPM)

```bash
herd php list                 # installed vs apt-available
herd php install 8.2 8.3      # apt install phpX.Y-fpm
herd php which                # from inside a served project
herd php -v
herd php --site=myapp -v      # when not inside the project
```

Change a site’s PHP:

```bash
herd serve myapp ~/Code/myapp --php=8.2
sudo systemctl reload nginx
sudo systemctl restart php8.2-fpm
```

Confirm FPM is up:

```bash
systemctl status php8.2-fpm
ls -la /run/php/php8.2-fpm.sock
```

---

## 6. Composer & Artisan

**Always prefer Herd wrappers** so CLI PHP matches the site.

```bash
cd ~/Code/myapp

herd composer install
herd composer update
herd composer require package/name

herd php artisan migrate
herd php artisan migrate --seed
herd php artisan key:generate
herd php artisan optimize:clear
herd php vendor/bin/phpunit
```

Outside the project:

```bash
herd composer --site=myapp install
herd php --site=myapp artisan migrate
```

Optional aliases in `~/.bashrc` (only reliable when cwd is a served site):

```bash
alias php='herd php'
alias composer='herd composer'
```

---

## 7. Databases (Docker)

Requires Docker group access (`docker info` without sudo).

### Start / stop / status

```bash
herd db start                 # MySQL + Postgres + Redis
herd db start --mysql         # one engine
herd db status
herd db stop
```

### Defaults

| Engine   | Host        | Port | User / password   |
|----------|-------------|------|-------------------|
| MySQL    | `127.0.0.1` | 3306 | `root` / `root`   |
| Postgres | `127.0.0.1` | 5432 | `postgres` / `postgres` |
| Redis    | `127.0.0.1` | 6379 | (none)            |

Containers: `herd-mysql`, `herd-postgres`, `herd-redis`  
Data persists in: `~/.herd/data/`

### Create / list / backup / restore

```bash
herd db create myapp --type mysql
herd db create myapp --type postgres
herd db list --type mysql

herd db backup myapp --type mysql
herd db backup myapp --type mysql --out ~/backups/myapp.sql

herd db restore myapp ~/backups/myapp.sql --type mysql
```

### Laravel `.env` example (MySQL)

```env
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=myapp
DB_USERNAME=root
DB_PASSWORD=root

REDIS_HOST=127.0.0.1
REDIS_PORT=6379
```

---

## 8. Extras (Mailhog, phpMyAdmin, MinIO)

```bash
herd extras start
herd extras status
herd extras stop

herd extras start mailhog phpmyadmin   # subset
```

| Service    | URL / port | Notes |
|------------|------------|--------|
| Mailhog UI | http://127.0.0.1:8025 | Catch local email |
| Mailhog SMTP | `127.0.0.1:1025` | Point `MAIL_HOST` / port here |
| phpMyAdmin | http://127.0.0.1:8081 | `root` / `root` → `herd-mysql` |
| MinIO API  | http://127.0.0.1:9000 | S3-compatible |
| MinIO Console | http://127.0.0.1:9001 | `minio` / `minioadmin` |

Example Mailhog mail config:

```env
MAIL_MAILER=smtp
MAIL_HOST=127.0.0.1
MAIL_PORT=1025
MAIL_USERNAME=null
MAIL_PASSWORD=null
```

---

## 9. Laravel queues

Uses the site’s PHP binary; no Docker required.

```bash
herd queue work myapp
herd queue work myapp --queue=emails --timeout=90
herd queue list
herd queue stop myapp
herd queue stop myapp --queue=emails
```

Worker PIDs are tracked under `~/.herd/` (queues state file).

---

## 10. Dashboard

```bash
herd dashboard              # opens http://127.0.0.1:7835
herd dashboard --no-open
herd dashboard --port 9000
```

Shows sites, PHP versions, DB/extras status, queues, and log snippets. Stop with Ctrl+C.

---

## 11. Services, logs, status

```bash
herd status
herd start
herd stop
herd restart nginx          # or other service name Herd knows

herd logs nginx
herd logs php-8.4
```

App logs still live in the project: `storage/logs/laravel.log`.

---

## 12. Typical Laravel workflow

```bash
# 1) Docker session (once per login if needed)
newgrp docker   # if `docker info` fails

# 2) Databases & helpers
herd db start
herd extras start          # optional

# 3) Register site
herd serve myapp ~/Code/myapp --php=8.4
sudo nginx -t && sudo systemctl reload nginx   # if reload failed

# 4) App bootstrap
cd ~/Code/myapp
cp -n .env.example .env
# edit DB_* / APP_URL=https://myapp.test
herd composer install
herd php artisan key:generate
herd db create myapp --type mysql
herd php artisan migrate --seed
npm install && npm run build   # or: npm run dev

# 5) Develop
# Browser: https://myapp.test
herd dashboard
herd queue work myapp      # if needed
```

### Frontend note

Inertia/Vue apps need Vite assets:

- Dev: `npm run dev`
- Or build once: `npm run build`

Without that you may see `ViteManifestNotFoundException`.

---

## 13. Useful paths

| Path | Purpose |
|------|---------|
| `~/.local/bin/herd` | Installed CLI |
| `~/.herd/` | Runtime state |
| `~/.herd/sites.yaml` | Registered sites |
| `~/.herd/nginx/sites-enabled/` | Per-site Nginx configs |
| `~/.herd/ssl/` | Local TLS cert + key |
| `~/.herd/data/` | Docker DB/MinIO volumes |
| `~/.herd/logs/` | Herd service logs |
| `/etc/nginx/conf.d/herd.conf` | Includes Herd Nginx configs |
| `/run/php/phpX.Y-fpm.sock` | FPM sockets |

Repo scripts:

| Script | Purpose |
|--------|---------|
| `scripts/p8-system-setup.sh` | One-time Nginx/DNS/TLS host wiring |
| `scripts/p8-verify.sh` | Verify `.test` + HTTPS |
| `scripts/p10-install-docker.sh` | Install Docker Engine |
| `scripts/test-multi-php.sh` | Multi-PHP smoke test |
| `scripts/manual-test.sh` | CLI smoke tests |

---

## 14. Troubleshooting

### `herd db start` → permission denied / docker network error

Your shell doesn’t have the `docker` group yet:

```bash
sudo usermod -aG docker "$USER"
newgrp docker
docker info
herd db start
```

### Browser 500: `tempnam(): file created in the system's temporary directory`

`www-data` cannot write Laravel `storage/`:

```bash
cd ~/Code/myapp
mkdir -p storage/framework/{sessions,views,cache/data} storage/logs bootstrap/cache
setfacl -R -m u:www-data:rwx storage bootstrap/cache
setfacl -R -d -m u:www-data:rwx storage bootstrap/cache
# or re-run: herd serve myapp ~/Code/myapp
```

### Browser 500: `MissingAppKeyException`

```bash
herd php artisan key:generate
```

### Browser 500: `could not find driver` (MySQL)

Missing PDO MySQL for that PHP version:

```bash
sudo apt install -y php8.4-mysql    # match your site PHP
sudo systemctl restart php8.4-fpm
php8.4 -m | grep pdo_mysql
```

### `Class "DOMDocument" not found` on `php artisan …`

You used global `php` (e.g. 8.6) without extensions. Use:

```bash
herd php artisan …
```

And/or install `phpX.Y-xml` for that version.

### `ViteManifestNotFoundException`

```bash
npm install && npm run build
# or keep npm run dev running
```

### Site opens HTTP but not HTTPS / cert warnings

```bash
herd init
sudo ./scripts/p8-system-setup.sh
# Prefer mkcert for a trusted local CA
```

### `.test` does not resolve

```bash
sudo ./scripts/p8-system-setup.sh
./scripts/p8-verify.sh
resolvectl query myapp.test
```

### Wrong PHP version in CLI

```bash
cd ~/Code/myapp
herd php which
herd serve myapp ~/Code/myapp --php=8.2
```

### Nginx config changed but site unchanged

```bash
sudo nginx -t && sudo systemctl reload nginx
```

### Check app errors

```bash
tail -n 80 ~/Code/myapp/storage/logs/laravel.log
curl -skI https://myapp.test/
```

---

## 15. Command cheat sheet

```bash
# Lifecycle
herd init
herd serve <name> <path> [--php=X.Y]
herd forget <name>
herd sites
herd status
herd start | stop | restart [service]
herd logs <service>

# PHP / Composer (from project dir)
herd php list | install <ver...> | which
herd php [php-args…]
herd composer [composer-args…]
herd php --site=<name> …
herd composer --site=<name> …

# Docker data plane
herd db start|stop|status
herd db create|list|backup|restore …
herd extras start|stop|status

# Laravel helpers
herd queue work|list|stop …
herd dashboard [--port N] [--no-open]
```

---

## Quick daily start

### One command (recommended)

```bash
~/Code/diy-herd/scripts/herd-daily-start.sh
```

What it does:

- Starts MySQL / Postgres / Redis (`herd db start`)
- Optionally starts Mailhog / phpMyAdmin / MinIO (`HERD_EXTRAS=1`)
- Starts the dashboard on http://127.0.0.1:7835 if it isn’t already up
- Logs to `~/.local/state/herd/`

```bash
# With extras + open browser for dashboard
HERD_EXTRAS=1 HERD_OPEN_DASHBOARD=1 ~/Code/diy-herd/scripts/herd-daily-start.sh

# End of day
~/Code/diy-herd/scripts/herd-daily-stop.sh        # dashboard + extras
~/Code/diy-herd/scripts/herd-daily-stop.sh --all  # also stop DBs
```

Optional alias in `~/.bashrc`:

```bash
alias herd-up='HERD_EXTRAS=1 ~/Code/diy-herd/scripts/herd-daily-start.sh'
alias herd-down='~/Code/diy-herd/scripts/herd-daily-stop.sh'
```

### Autostart when you log in

**Option A — GNOME / desktop autostart**

```bash
mkdir -p ~/.config/autostart
cp ~/Code/diy-herd/scripts/herd-daily-start.desktop ~/.config/autostart/
# Edit the file if you want HERD_EXTRAS=1
```

**Option B — systemd user service**

```bash
mkdir -p ~/.config/systemd/user
cp ~/Code/diy-herd/scripts/herd-daily.service ~/.config/systemd/user/
# Edit paths if your diy-herd repo is not ~/Code/diy-herd
systemctl --user daemon-reload
systemctl --user enable --now herd-daily.service
systemctl --user status herd-daily.service
```

Ensure Docker works in a normal login session (`docker info`). If not:

```bash
sudo usermod -aG docker "$USER"
# then log out and back in once
```

Manual fallback:

```bash
newgrp docker 2>/dev/null || true
herd db start
herd extras start          # optional
herd dashboard --no-open &
```

---

*DIY Herd — personal Ubuntu stack. See also [README.md](README.md) and [IMPLEMENTATION_TRACKER.md](IMPLEMENTATION_TRACKER.md).*
