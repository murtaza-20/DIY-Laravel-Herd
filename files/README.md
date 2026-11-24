# Ubuntu Herd MVP

A local PHP/Laravel development environment for Ubuntu—like Herd for macOS but for Linux. Single binary, zero config, everything integrated.

## Features

- **Single Binary**: Download, run, done
- **All-in-one**: PHP-FPM, Nginx, MySQL, PostgreSQL, Redis, Elasticsearch, Mailhog, phpMyAdmin, MinIO
- **.test Domains**: Local domain names like `myapp.test` (no localhost:8000)
- **Service Management**: Start/stop/restart services via CLI
- **Queue Workers**: Manage Laravel queue workers from the CLI
- **Dashboard**: Web UI for monitoring (Phase 2)
- **Database Tools**: Create, backup, restore databases
- **Third-party API Ready**: Proper hostnames for webhook testing

## Quick Start

### Prerequisites

- Ubuntu 20.04+ (tested on 22.04 LTS and 24.04 LTS)
- Docker installed (for MySQL, PostgreSQL, Redis, Elasticsearch)
- Git (for cloning this repo)
- Go 1.22+ (for building from source)

### Installation (Development)

```bash
# Clone repo
git clone https://github.com/thesmartyking/herd.git
cd herd

# Install dependencies
go mod download

# Build binary
go build -o herd

# Initialize Herd
sudo ./herd init
# You'll be prompted for sudo password to configure Dnsmasq

# Check status
./herd status
```

### Usage

```bash
# Register a Laravel project
./herd serve osortoo ~/projects/osortoo

# Visit: http://osortoo.test in browser
# Everything just works!

# List all projects
./herd status

# View logs
./herd logs nginx
./herd logs php
./herd logs mysql

# Restart a service
./herd restart php

# Stop all services
./herd stop

# Start all services
./herd start

# Remove a project
./herd forget osortoo
```

## Development Guide

### Project Structure

```
herd/
├── main.go                       # CLI entry point (Cobra)
├── go.mod
├── go.sum
├── pkg/
│   ├── services/
│   │   └── manager.go            # Service lifecycle management
│   ├── nginx/
│   │   └── config_generator.go   # Dynamic Nginx config generation
│   ├── dns/                      # Dnsmasq integration (TODO)
│   ├── docker/                   # Docker container management (TODO)
│   └── config/                   # Config file handling (TODO)
└── cmd/                          # Individual command implementations (TODO)
```

### MVP Development Phases

**Phase 1: Core (Weeks 1-2)**
- [x] Cobra CLI scaffold
- [x] Service manager (start/stop/restart processes)
- [x] Nginx config generator (detects Laravel, PHP, Node.js)
- [ ] `herd init` implementation
- [ ] `herd serve` implementation
- [ ] Dnsmasq integration

**Phase 2: Docker Services (Week 3)**
- [ ] Docker container management (MySQL, PostgreSQL, Redis, Elasticsearch)
- [ ] Data persistence (~/.herd/data/)
- [ ] Service health checks

**Phase 3: Advanced Features (Week 4)**
- [ ] Queue worker management (`herd queue:work`)
- [ ] Database tools (`herd db:create`, etc.)
- [ ] Web dashboard (optional)
- [ ] Log streaming

### Current Implementation Status

✅ **Completed**:
- CLI framework with Cobra
- Service manager (process lifecycle)
- Nginx config generator
- Architecture documentation

🔄 **In Progress**:
- `herd init` command
- Dnsmasq configuration
- Docker integration

❌ **TODO**:
- Database management
- Queue worker integration
- Web dashboard
- Testing suite

## Key Components Explained

### 1. Service Manager (`pkg/services/manager.go`)

Manages the lifecycle of PHP-FPM, Nginx, Dnsmasq, and other services.

```go
sm := services.NewServiceManager(herdDir)

// Register a service
sm.RegisterService("php-fpm", "process", "/usr/bin/php-fpm", 
    []string{"-F", "-y", "/etc/php/8.2/fpm/php-fpm.conf"}, 
    map[string]string{})

// Start/stop/restart
sm.Start("php-fpm")
sm.Stop("php-fpm")
sm.Restart("php-fpm")

// Check status
status := sm.Status() // map[string]bool
```

### 2. Nginx Config Generator (`pkg/nginx/config_generator.go`)

Automatically generates Nginx configs based on project type.

```go
cg := nginx.NewConfigGenerator("/home/user/.herd/nginx/sites-enabled")

site := nginx.Site{
    Name: "osortoo",
    Path: "/home/user/projects/osortoo",
    // Type auto-detected: "laravel" (detects artisan file)
}

cg.WriteConfig(site) // Creates /home/user/.herd/nginx/sites-enabled/osortoo.conf
```

**Auto-detection logic**:
- If `artisan` file exists → Laravel
- If `composer.json` exists → PHP framework
- If `package.json` exists → Node.js
- If `index.html` exists → Static site
- Default → Generic PHP

### 3. CLI Architecture

Built with Cobra for elegant command structure:

```
herd
├── init                          # Initialize environment
├── serve [name] [path]          # Register a project
├── forget [name]                # Remove a project
├── status                       # Show all services
├── logs [service]               # View logs
├── start                        # Start all services
├── stop                         # Stop all services
├── restart [service]            # Restart a service
├── dashboard                    # Open web UI
├── queue
│   └── work [project] [queue]  # Start queue worker
└── db
    ├── create [name]
    ├── backup [name]
    └── restore [name] [file]
```

## Next Implementation Steps

### Step 1: Implement `herd init`

```bash
# In cmd/init.go (create this file):
# 1. Create ~/.herd directory structure
# 2. Create ~/.herd/config.yaml (empty for now)
# 3. Create ~/.herd/sites.yaml (empty)
# 4. Create ~/.herd/nginx/sites-enabled
# 5. Download/verify PHP binary
# 6. Start Docker Compose (MySQL, PostgreSQL, Redis, Elasticsearch)
# 7. Configure Dnsmasq (requires sudo)
# 8. Start PHP-FPM via ServiceManager
# 9. Start Nginx via ServiceManager
```

### Step 2: Implement `herd serve`

```bash
# In cmd/serve.go (create this file):
# 1. Validate project path exists
# 2. Detect project type (Laravel, PHP, etc.)
# 3. Generate Nginx config via ConfigGenerator
# 4. Add entry to ~/.herd/sites.yaml
# 5. Run: nginx -s reload
# 6. Output: "Project served at http://osortoo.test"
```

### Step 3: Dnsmasq Integration

```bash
# In pkg/dns/dnsmasq.go (create this file):
# 1. Create /etc/dnsmasq.d/herd.conf with: address=/.test/127.0.0.1
# 2. Restart dnsmasq: sudo systemctl restart dnsmasq
# 3. Verify: nslookup osortoo.test → 127.0.0.1
```

## Configuration File (~/.herd/config.yaml)

Once fully implemented:

```yaml
version: 1

php:
  version: "8.2"
  socket: "/var/run/herd/php-fpm.sock"

nginx:
  port: 80
  https_port: 443

services:
  mysql:
    enabled: true
    port: 3306
  redis:
    enabled: true
    port: 6379
```

## Troubleshooting

### Dnsmasq permission denied
```bash
# Make sure Dnsmasq can be managed without password
sudo visudo
# Add: username ALL=(ALL) NOPASSWD: /etc/dnsmasq.d/*
```

### Port conflicts
```bash
# Check if port 80 is in use
sudo lsof -i :80

# Check if port 3306 is in use (MySQL)
sudo lsof -i :3306
```

### PHP socket permission issues
```bash
# Ensure socket is readable by Nginx
sudo chown www-data:www-data /var/run/herd/php-fpm.sock
sudo chmod 660 /var/run/herd/php-fpm.sock
```

## Contributing

This is a personal MVP. If you'd like to extend it:

1. Fork the repo
2. Create a feature branch
3. Send a PR with your improvements

## License

MIT (or your preferred license)

## Roadmap

- [x] Architecture documentation
- [x] CLI scaffold with Cobra
- [x] Service manager
- [x] Nginx config generator
- [ ] `herd init` command
- [ ] Dnsmasq integration
- [ ] Docker service management
- [ ] Database tools
- [ ] Web dashboard
- [ ] Queue worker UI
- [ ] Automated testing

## Questions?

Open an issue or check the architecture doc: `ubuntu-herd-architecture.md`
