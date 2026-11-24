# Ubuntu Herd MVP - Architecture & Implementation Plan

## Overview
A single Go binary (`herd`) that manages a complete PHP/Laravel development stack on Ubuntu. Think Herd for macOS but for Linux—zero config, everything integrated.

---

## Architecture

### Core Components (Managed by Binary)

```
herd (Go Binary)
├── Service Manager (supervise all processes)
├── PHP 8.2 FPM (embedded or bundled)
├── Nginx (embedded or bundled)
├── MySQL 8.0 (Docker backend)
├── PostgreSQL 15 (Docker backend)
├── Redis (Docker backend)
├── Elasticsearch 8.x (Docker backend)
├── Mailhog (SMTP testing)
├── phpMyAdmin (web UI for MySQL)
├── S3 MinIO (S3 API compatible local storage)
├── Supervisor (for queue workers)
└── Dnsmasq (DNS resolution for .test domains)
```

### Installation Flow
```
1. User downloads: herd-ubuntu-latest.tar.gz (~500MB with bundled PHP/Nginx)
2. tar xzf && chmod +x herd/herd
3. ./herd/herd init    # One-time setup: create ~/.herd, start services, dnsmasq config
4. ./herd/herd status  # Check all services running
5. ./herd/herd serve myapp ~/projects/myapp  # Boom: http://myapp.test running
```

---

## Implementation Strategy

### Phase 1: MVP Core (2-3 weeks)
- [ ] Go binary scaffolding + CLI framework (Cobra)
- [ ] Service manager (systemd or direct process supervision)
- [ ] Embedded dnsmasq for .test domains
- [ ] Docker Compose wrapper for heavy services (MySQL, Redis, Postgres, Elasticsearch)
- [ ] Nginx reverse proxy + dynamic site config generation
- [ ] PHP-FPM bundled (or downloaded on first run)
- [ ] Basic site registration (`herd serve myapp /path`)

### Phase 2: UX Polish (1-2 weeks)
- [ ] Web dashboard (see running sites, logs, restart services)
- [ ] TUI dashboard (terminal UI for quick status)
- [ ] Service auto-startup on system boot
- [ ] Log viewing (`herd logs nginx`, `herd logs php`)
- [ ] Config templating (auto-detect Laravel, Django, etc.)

### Phase 3: Full Integration (2-3 weeks)
- [ ] phpMyAdmin integration
- [ ] Mailhog SMTP testing UI
- [ ] MinIO S3 simulator dashboard
- [ ] Database backup/restore helpers
- [ ] Queue worker management UI

---

## Tech Stack for the Binary

### Language & Framework
- **Go 1.22+** (single binary, fast, zero dependencies)
- **Cobra** (CLI framework)
- **Docker SDK** (manage containers for heavy services)
- **embed** package (bundle static assets like phpMyAdmin)

### Key Libraries
```go
github.com/spf13/cobra       // CLI
github.com/docker/docker     // Docker API
github.com/moby/moby         // Container management
github.com/fsnotify/fsnotify // File watching (for auto-reload)
github.com/sirupsen/logrus   // Structured logging
github.com/gin-gonic/gin     // Web dashboard API
```

---

## Directory Structure

```
~/.herd/
├── config.yaml              # Global config (PHP version, port, etc.)
├── sites.yaml               # Registered sites (auto-generated)
├── services/
│   ├── php-8.2/
│   ├── nginx/
│   └── dnsmasq/
├── logs/
│   ├── nginx/
│   ├── php-fpm/
│   ├── mysql/
│   └── redis/
├── docker-compose.yml       # Heavy services (MySQL, Redis, etc.)
├── ssl/                      # Self-signed certs for HTTPS
└── data/
    ├── mysql/
    ├── postgres/
    └── elasticsearch/
```

---

## CLI Commands (MVP)

```bash
# Initialize and start all services
herd init

# Check service status
herd status

# Register a Laravel project
herd serve myapp ~/projects/myapp
herd serve osortoo ~/projects/osortoo

# Remove a site
herd forget myapp

# View logs
herd logs nginx
herd logs php
herd logs mysql

# Manage services
herd restart php
herd restart nginx
herd stop
herd start

# Dashboard
herd dashboard  # Opens http://localhost:7835 in browser

# Queue worker management
herd queue:work myapp default
herd queue:list

# Database helpers
herd db:create mydb
herd db:backup mydb
herd db:restore mydb backup.sql

# PHP utilities
herd php -v
herd composer install
herd artisan migrate
```

---

## Single Binary Distribution

### Build Process
```bash
# Download and embed dependencies
go mod download

# Build script bundles:
# 1. PHP 8.2 (precompiled for Ubuntu)
# 2. Nginx (precompiled)
# 3. Dnsmasq binary

# Create tarball with embedded binaries
tar czf herd-ubuntu-latest.tar.gz \
  herd \
  embedded/php-8.2/ \
  embedded/nginx/ \
  embedded/dnsmasq/

# Size: ~500MB (PHP + Nginx are large)
```

### Alternative: Download on First Run
```bash
# Binary is ~50MB
# On first `herd init`, download:
# - PHP 8.2-FPM (launchpad or compile)
# - Nginx (ubuntu repos)
# - Services via Docker
# Saves distribution size, slower initial setup
```

---

## Core Implementation: Service Manager

```go
package herd

type ServiceManager struct {
    processes map[string]*Service
    docker    *client.Client
}

type Service struct {
    Name       string
    Type       string    // "process", "docker"
    Cmd        string
    Env        []string
    Running    bool
    Logs       *logrus.Logger
}

func (sm *ServiceManager) Start(serviceName string) error {
    // Start individual service or Docker container
    // Update ~/.herd/status.json
}

func (sm *ServiceManager) Restart(serviceName string) error {
    sm.Stop(serviceName)
    return sm.Start(serviceName)
}

func (sm *ServiceManager) Status() map[string]bool {
    // Return health status of all services
}
```

---

## Dynamic Nginx Configuration

When user runs `herd serve myapp ~/projects/myapp`, the binary:

1. **Detects project type** (Laravel, generic PHP, Node.js)
2. **Generates Nginx config**:
   ```nginx
   server {
       listen 80;
       server_name myapp.test *.myapp.test;
       root /path/to/myapp/public;
       
       location / {
           try_files $uri $uri/ /index.php?$query_string;
       }
       
       location ~ \.php$ {
           fastcgi_pass unix:/var/run/herd/php-8.2.sock;
           include fastcgi_params;
       }
   }
   ```
3. **Writes to** `~/.herd/nginx/sites-enabled/myapp.conf`
4. **Reloads Nginx** (`nginx -s reload`)
5. **Updates Dnsmasq** to recognize `myapp.test`
6. **Logs registration** to `~/.herd/sites.yaml`

---

## Web Dashboard (Optional Phase 2)

```bash
# Accessible at http://localhost:7835
# Built with Go + embedded web UI

GET  /api/sites              # List all registered sites
GET  /api/sites/:name        # Site details
POST /api/sites              # Register new site
POST /api/services/:name/restart
GET  /api/logs/:service      # Stream logs
GET  /api/status             # Overall system health
```

Frontend: Preact or React (compiled + embedded as static assets)

---

## Database Management

### MySQL/PostgreSQL
```bash
# Docker containers managed by herd
herd db:create mydb --type mysql
herd db:create mydb --type postgres

# Exposed ports:
# MySQL: localhost:3306 (credentials: root/root)
# PostgreSQL: localhost:5432 (postgres/postgres)
# Both auto-started with `herd init`
```

### phpMyAdmin
```bash
# Embedded or as separate Docker container
# Access: http://localhost:8081
# Auto-configure MySQL credentials
```

### Data Persistence
```
~/.herd/data/mysql/        # MySQL volumes
~/.herd/data/postgres/     # PostgreSQL volumes
~/.herd/data/elasticsearch/
```
Docker volumes point to `~/.herd/data/` so data persists across restarts.

---

## DNS Resolution (.test Domains)

### Dnsmasq Integration
```bash
# herd init creates:
/etc/dnsmasq.d/herd.conf:
  address=/.test/127.0.0.1

# Enables: myapp.test → 127.0.0.1
```

### Alternative: systemd-resolved
```bash
# Add to /etc/systemd/resolved.conf
[Resolve]
DNS=127.0.0.1
Domains=~test
```

---

## Queue Worker Management

```bash
# Start queue worker for a site
herd queue:work osortoo default --timeout=60

# List running workers
herd queue:list
# Output:
# osortoo    default    running    PID: 12345
# osortoo    emails     running    PID: 12346

# Supervisor integration (optional)
# herd manages supervisor config in ~/.herd/supervisor/
```

---

## HTTPS & Self-Signed Certificates

```bash
# On first `herd init`, generate:
~/.herd/ssl/
├── ca.key
├── ca.crt
└── per-site certs (auto-generated)

# Update Nginx to serve HTTPS
# Browsers show warning (expected for self-signed)
```

---

## Configuration File (~/.herd/config.yaml)

```yaml
version: 1

php:
  version: "8.2"
  port: 9000
  socket: "/var/run/herd/php-8.2.sock"
  extensions:
    - mbstring
    - xml
    - bcmath
    - gd
    - pdo_mysql
    - pdo_pgsql
    - redis

nginx:
  port: 80
  https_port: 443
  worker_processes: 4

database:
  mysql:
    enabled: true
    port: 3306
    root_password: "root"
  postgres:
    enabled: true
    port: 5432
    password: "postgres"

services:
  redis:
    enabled: true
    port: 6379
  elasticsearch:
    enabled: true
    port: 9200
  mailhog:
    enabled: true
    port: 1025
    web_port: 8025
  minio:
    enabled: true
    port: 9000
    web_port: 9001

dnsmasq:
  tld: "test"

dashboard:
  enabled: true
  port: 7835
```

---

## Development Workflow Example

```bash
# One-time setup
$ herd init
[✓] Creating ~/.herd directory
[✓] Starting MySQL container
[✓] Starting PostgreSQL container
[✓] Starting Redis container
[✓] Starting Elasticsearch container
[✓] Configuring Dnsmasq for .test domains
[✓] Starting PHP-FPM
[✓] Starting Nginx
[✓] Dashboard running at http://localhost:7835

# Register projects
$ herd serve osortoo ~/projects/osortoo
[✓] Site registered: osortoo.test
[✓] Visit: http://osortoo.test

$ herd serve quotation-crm ~/projects/quotation-crm
[✓] Site registered: quotation-crm.test
[✓] Visit: http://quotation-crm.test

# Run migrations
$ cd ~/projects/osortoo && php artisan migrate

# Start queue worker
$ herd queue:work osortoo default

# View logs
$ herd logs nginx
$ herd logs php

# That's it. Everything just works.
```

---

## Comparison: Herd vs Ubuntu Herd MVP

| Feature | Herd (macOS) | Ubuntu Herd MVP |
|---------|------------|-----------------|
| Single Binary | ✓ | ✓ |
| PHP-FPM | ✓ | ✓ |
| Nginx | ✓ | ✓ |
| MySQL | ✓ | ✓ (Docker) |
| PostgreSQL | ✓ | ✓ (Docker) |
| Redis | ✓ | ✓ (Docker) |
| Elasticsearch | ✗ | ✓ |
| phpMyAdmin | ✗ | ✓ |
| MinIO S3 | ✗ | ✓ |
| .test domains | ✓ | ✓ |
| Dashboard | ✗ | ✓ (Phase 2) |
| Queue Workers | Manual | ✓ (CLI) |
| Price | Paid | Free (Personal) |

---

## Challenges & Solutions

### Challenge 1: PHP-FPM Bundling
- **Problem**: PHP is large (~500MB compiled)
- **Solution**: Download on first run OR use Alpine/minimal builds

### Challenge 2: Root Access (for Dnsmasq)
- **Problem**: Dnsmasq needs root to modify DNS
- **Solution**: 
  - Prompt for sudo during `herd init`
  - Or use systemd-resolved (no sudo needed)
  - Or fallback to hosts file + local DNS

### Challenge 3: Docker Dependency
- **Problem**: MySQL, Redis, Elasticsearch need Docker
- **Solution**:
  - Require Docker to be installed
  - Or provide fallback (MySQL native, etc.)
  - Clear documentation during `herd init`

### Challenge 4: Port Conflicts
- **Problem**: User might have MySQL running already
- **Solution**: Auto-detect and use alternative ports, or skip that service

---

## MVP Timeline

**Week 1-2**: Core binary + service manager
- Go project scaffold
- CLI (Cobra) + basic commands
- Service process management
- Nginx config generation

**Week 3**: Docker integration
- MySQL + PostgreSQL containers
- Redis + Elasticsearch
- Data persistence

**Week 4**: Dnsmasq + Testing
- .test domain resolution
- Integration testing
- Documentation

**Week 5**: Polishing
- Error handling
- Logging
- Help/usage text

---

## Next Steps (for you)

1. **Create GitHub repo**: `ubuntu-herd` (or internal if personal)
2. **Initialize Go project**: `go mod init github.com/thesmartyking/herd`
3. **Set up Cobra CLI scaffold**
4. **Start with `herd init` command**
5. **Build service manager layer**
6. **Add `herd serve` command**
7. **Test with one of your projects** (Osortoo or Quotation CRM)

This is a genuinely useful tool that solves a real problem. Ship it! 🚀
