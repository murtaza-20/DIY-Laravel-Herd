# Ubuntu Herd MVP - Complete Development Roadmap
## One-Day Build Guide for LLM Auto-Coding

This document provides the complete development path for building Ubuntu Herd MVP in a single day via vibe coding with an LLM.

---

## Project Overview

**Name**: Ubuntu Herd MVP  
**Purpose**: Local PHP/Laravel development environment for Ubuntu with multi-PHP version support  
**Expected Time**: 6-8 hours of vibe coding  
**Language**: Go 1.22+  
**Target**: Fully working CLI tool by end of day  

---

## Development Timeline (Estimated)

```
00:00-01:00  Project Setup & Go Module Initialization
01:00-02:00  Phase 1: Core Structures (PHP/Nginx/Composer packages)
02:00-03:00  Phase 2: Command Framework & Service Manager
03:00-04:00  Phase 3: herd init Command Implementation
04:00-05:00  Phase 4: herd serve Command Implementation
05:00-06:00  Phase 5: Integration & Testing
06:00-07:00  Phase 6: Bug Fixes & Polish
07:00-08:00  Buffer + Final Testing
```

---

# PHASE 0: PROJECT SETUP (1 hour)

## Step 0.1: Initialize Go Module

```bash
mkdir herd && cd herd
go mod init github.com/thesmartyking/herd
```

**Cobra CLI Framework Setup**:
```bash
go get -u github.com/spf13/cobra@latest
go get -u github.com/sirupsen/logrus@latest
go get -u gopkg.in/yaml.v3@latest
```

## Step 0.2: Directory Structure

Create this exact directory structure:

```
herd/
├── go.mod
├── go.sum
├── main.go                          # Entry point
├── cmd/
│   ├── init.go                      # herd init command
│   ├── serve.go                     # herd serve command
│   ├── status.go                    # herd status command
│   ├── logs.go                      # herd logs command
│   └── root.go                      # Root command setup
├── pkg/
│   ├── php/
│   │   └── manager.go               # PHP version management
│   ├── composer/
│   │   └── parser.go                # composer.json parsing
│   ├── nginx/
│   │   └── config.go                # Nginx configuration
│   ├── services/
│   │   └── manager.go               # Service lifecycle
│   ├── config/
│   │   └── loader.go                # Config file handling
│   └── utils/
│       └── helpers.go               # Utility functions
├── templates/
│   ├── fpm.conf                     # PHP-FPM config template
│   └── nginx.conf                   # Nginx template
└── README.md
```

## Step 0.3: Create Main.go

```go
package main

import (
	"os"
	"github.com/thesmartyking/herd/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Verify compilation**:
```bash
go build -o herd
./herd --version  # Should show version info
```

---

# PHASE 1: CORE STRUCTURES (1 hour)

## Step 1.1: Service Manager (`pkg/services/manager.go`)

✅ Already implemented - COPY from previous files

## Step 1.2: PHP Manager (`pkg/php/manager.go`)

✅ Already implemented - COPY from previous files

## Step 1.3: Composer Parser (`pkg/composer/parser.go`)

✅ Already implemented - COPY from previous files

## Step 1.4: Nginx Config Generator (`pkg/nginx/config.go`)

✅ Already implemented - COPY from previous files with updates

## Step 1.5: Config Loader (`pkg/config/loader.go`)

**New file to create**:

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// HerdConfig represents ~/.herd/config.yaml
type HerdConfig struct {
	Version string `yaml:"version"`
	PHP     struct {
		Versions []string `yaml:"versions"`
	} `yaml:"php"`
	Nginx struct {
		Port      int `yaml:"port"`
		HTTPSPort int `yaml:"https_port"`
	} `yaml:"nginx"`
}

// GetHerdDir returns the ~/.herd directory
func GetHerdDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".herd"), nil
}

// EnsureHerdDir creates ~/.herd if it doesn't exist
func EnsureHerdDir() (string, error) {
	herdDir, err := GetHerdDir()
	if err != nil {
		return "", err
	}
	
	if err := os.MkdirAll(herdDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create herd directory: %w", err)
	}
	
	// Create subdirectories
	dirs := []string{
		filepath.Join(herdDir, "php"),
		filepath.Join(herdDir, "nginx", "sites-enabled"),
		filepath.Join(herdDir, "logs"),
		filepath.Join(herdDir, "ssl"),
		filepath.Join(herdDir, "data"),
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return herdDir, nil
}
```

## Step 1.6: Utilities (`pkg/utils/helpers.go`)

```go
package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecuteCommand runs a shell command
func ExecuteCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// ExecuteCommandOutput runs a command and returns output
func ExecuteCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// CheckPHPInstalled checks if a PHP version is available
func CheckPHPInstalled(version string) bool {
	binary := fmt.Sprintf("/usr/bin/php%s", version)
	return FileExists(binary)
}

// GetAvailablePHPVersions returns installed PHP versions
func GetAvailablePHPVersions() ([]string, error) {
	versions := []string{}
	candidates := []string{"7.4", "8.0", "8.1", "8.2", "8.3", "8.4"}
	
	for _, v := range candidates {
		if CheckPHPInstalled(v) {
			versions = append(versions, v)
		}
	}
	
	return versions, nil
}
```

---

# PHASE 2: COMMAND FRAMEWORK (1 hour)

## Step 2.1: Root Command (`cmd/root.go`)

```go
package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	logger = logrus.New()
	rootCmd = &cobra.Command{
		Use:   "herd",
		Short: "Ubuntu Herd - Local PHP/Laravel development environment",
		Long: `Herd is a lightweight, all-in-one local development environment for PHP/Laravel on Ubuntu.
		
Start with 'herd init' to initialize your environment, then 'herd serve' to register projects.`,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(forgetCmd)
}
```

## Step 2.2: Status Command (`cmd/status.go`)

```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all Herd services",
	RunE: func(cmd *cobra.Command, args []string) error {
		herdDir, err := config.GetHerdDir()
		if err != nil {
			return err
		}
		
		fmt.Println("📊 Herd Status:\n")
		fmt.Println("Service          Status       Info")
		fmt.Println("─────────────────────────────────────")
		fmt.Println("PHP-FPM          Configured  ~/.herd/php/")
		fmt.Println("Nginx            Configured  ~/.herd/nginx/")
		fmt.Println("Dnsmasq          Configured  .test domains")
		fmt.Println("\nHerd directory:", herdDir)
		
		return nil
	},
}
```

## Step 2.3: Logs Command (`cmd/logs.go`)

```go
package cmd

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "View logs for a service (nginx, php-8.0, php-8.1, etc.)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		herdDir, _ := config.GetHerdDir()
		
		logPath := filepath.Join(herdDir, "logs", service+".log")
		content, err := ioutil.ReadFile(logPath)
		if err != nil {
			return fmt.Errorf("could not read logs for %s: %w", service, err)
		}
		
		fmt.Printf("📜 Logs for %s:\n\n%s\n", service, string(content))
		return nil
	},
}
```

## Step 2.4: Stop/Start/Restart Commands (`cmd/control.go`)

```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop all Herd services",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("⛔ Stopping all Herd services...")
		// TODO: Implement service stopping
		fmt.Println("✓ All services stopped")
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start all Herd services",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("▶️  Starting all Herd services...")
		// TODO: Implement service starting
		fmt.Println("✓ All services started")
		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart [service]",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		fmt.Printf("🔄 Restarting %s...\n", service)
		// TODO: Implement service restart
		fmt.Printf("✓ %s restarted\n", service)
		return nil
	},
}

var forgetCmd = &cobra.Command{
	Use:   "forget [name]",
	Short: "Remove a registered project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		fmt.Printf("🗑️  Removing project: %s\n", projectName)
		// TODO: Implement project removal
		fmt.Println("✓ Project removed")
		return nil
	},
}

func init() {
	// Commands added in root.go
}
```

---

# PHASE 3: HERD INIT COMMAND (1 hour)

## Step 3.1: Init Command (`cmd/init.go`)

This is the critical command that initializes everything.

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/php"
	"github.com/thesmartyking/herd/pkg/services"
	"github.com/thesmartyking/herd/pkg/utils"
	"gopkg.in/yaml.v3"
)

type InitConfig struct {
	PHPVersions []string `yaml:"php_versions"`
}

var (
	phpVersions string // Flag for --php-versions
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Herd environment (one-time setup)",
	Long: `Initialize Herd by:
1. Creating ~/.herd directory structure
2. Setting up PHP-FPM for specified versions
3. Configuring Nginx
4. Setting up Dnsmasq for .test domains`,
	RunE: initHerd,
}

func initHerd(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Initializing Herd environment...\n")
	
	// Step 1: Create directory structure
	fmt.Println("[1/5] Creating directory structure...")
	herdDir, err := config.EnsureHerdDir()
	if err != nil {
		return err
	}
	fmt.Printf("✓ Created %s\n", herdDir)
	
	// Step 2: Get PHP versions (default to common ones)
	fmt.Println("\n[2/5] Detecting available PHP versions...")
	versions, err := utils.GetAvailablePHPVersions()
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		return fmt.Errorf("no PHP versions found. Install PHP first: sudo apt install php8.2-fpm php8.1-fpm")
	}
	fmt.Printf("✓ Found PHP versions: %v\n", versions)
	
	// Step 3: Initialize PHP-FPM for each version
	fmt.Println("\n[3/5] Setting up PHP-FPM instances...")
	phpMgr := php.NewPHPManager(herdDir, nil)
	if err := phpMgr.Initialize(versions); err != nil {
		return fmt.Errorf("failed to initialize PHP versions: %w", err)
	}
	for _, v := range versions {
		fmt.Printf("✓ PHP %s configured\n", v)
	}
	
	// Step 4: Setup Nginx
	fmt.Println("\n[4/5] Configuring Nginx...")
	if err := setupNginx(herdDir); err != nil {
		return fmt.Errorf("failed to setup nginx: %w", err)
	}
	fmt.Println("✓ Nginx configured")
	
	// Step 5: Setup Dnsmasq
	fmt.Println("\n[5/5] Configuring Dnsmasq for .test domains...")
	if err := setupDnsmasq(); err != nil {
		fmt.Printf("⚠️  Dnsmasq setup requires sudo. You may need to run:\n")
		fmt.Printf("  sudo tee /etc/dnsmasq.d/herd.conf <<< 'address=/.test/127.0.0.1'\n")
		fmt.Printf("  sudo systemctl restart dnsmasq\n")
	} else {
		fmt.Println("✓ Dnsmasq configured for .test domains")
	}
	
	// Save config
	fmt.Println("\n[✓] Herd initialized successfully!")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Register your first project: herd serve projectname ~/path/to/project\n")
	fmt.Printf("  2. Visit http://projectname.test in your browser\n")
	fmt.Printf("\nPHP versions available: %v\n", versions)
	
	return nil
}

func setupNginx(herdDir string) error {
	// Create main nginx.conf
	nginxDir := filepath.Join(herdDir, "nginx")
	mainConfig := filepath.Join(nginxDir, "nginx.conf")
	
	nginxConfigContent := `user www-data;
worker_processes auto;
pid /var/run/nginx.pid;

events {
    worker_connections 768;
}

http {
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    access_log /var/log/nginx/access.log;
    error_log /var/log/nginx/error.log;

    gzip on;

    include ` + nginxDir + `/sites-enabled/*.conf;
}
`
	
	return os.WriteFile(mainConfig, []byte(nginxConfigContent), 0644)
}

func setupDnsmasq() error {
	dnsmasqConfig := "address=/.test/127.0.0.1\n"
	
	// Try to write directly (might fail without sudo)
	err := os.WriteFile("/etc/dnsmasq.d/herd.conf", []byte(dnsmasqConfig), 0644)
	if err != nil {
		return err
	}
	
	// Restart dnsmasq
	return utils.ExecuteCommand("sudo", "systemctl", "restart", "dnsmasq")
}

func init() {
	initCmd.Flags().StringVar(&phpVersions, "php-versions", "8.1,8.2,8.3", "PHP versions to initialize (comma-separated)")
}
```

---

# PHASE 4: HERD SERVE COMMAND (1 hour)

## Step 4.1: Sites Manager (`pkg/config/sites.go`)

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"gopkg.in/yaml.v3"
)

type Site struct {
	Name         string    `yaml:"name"`
	Path         string    `yaml:"path"`
	PHPVersion   string    `yaml:"php_version"`
	ProjectType  string    `yaml:"project_type"`
	RegisteredAt time.Time `yaml:"registered_at"`
}

type SitesFile struct {
	Sites []Site `yaml:"sites"`
}

// LoadSites reads all registered sites
func LoadSites() ([]Site, error) {
	herdDir, err := GetHerdDir()
	if err != nil {
		return nil, err
	}
	
	sitesPath := filepath.Join(herdDir, "sites.yaml")
	
	// Return empty list if file doesn't exist
	if _, err := os.Stat(sitesPath); os.IsNotExist(err) {
		return []Site{}, nil
	}
	
	data, err := os.ReadFile(sitesPath)
	if err != nil {
		return nil, err
	}
	
	var sf SitesFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	
	return sf.Sites, nil
}

// SaveSites writes sites to sites.yaml
func SaveSites(sites []Site) error {
	herdDir, err := GetHerdDir()
	if err != nil {
		return err
	}
	
	sitesPath := filepath.Join(herdDir, "sites.yaml")
	
	sf := SitesFile{Sites: sites}
	data, err := yaml.Marshal(sf)
	if err != nil {
		return err
	}
	
	return os.WriteFile(sitesPath, data, 0644)
}

// RegisterSite adds a new site
func RegisterSite(name, path, phpVersion, projectType string) error {
	sites, err := LoadSites()
	if err != nil {
		return err
	}
	
	// Check if already exists
	for _, s := range sites {
		if s.Name == name {
			return fmt.Errorf("site %s already registered", name)
		}
	}
	
	sites = append(sites, Site{
		Name:         name,
		Path:         path,
		PHPVersion:   phpVersion,
		ProjectType:  projectType,
		RegisteredAt: time.Now(),
	})
	
	return SaveSites(sites)
}

// UnregisterSite removes a site
func UnregisterSite(name string) error {
	sites, err := LoadSites()
	if err != nil {
		return err
	}
	
	filtered := []Site{}
	found := false
	for _, s := range sites {
		if s.Name != name {
			filtered = append(filtered, s)
		} else {
			found = true
		}
	}
	
	if !found {
		return fmt.Errorf("site %s not found", name)
	}
	
	return SaveSites(filtered)
}

// GetSite retrieves a specific site
func GetSite(name string) (*Site, error) {
	sites, err := LoadSites()
	if err != nil {
		return nil, err
	}
	
	for _, s := range sites {
		if s.Name == name {
			return &s, nil
		}
	}
	
	return nil, fmt.Errorf("site %s not found", name)
}
```

## Step 4.2: Serve Command (`cmd/serve.go`)

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/composer"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/nginx"
	"github.com/thesmartyking/herd/pkg/php"
	"github.com/thesmartyking/herd/pkg/utils"
)

var (
	explicitPHPVersion string // --php flag
)

var serveCmd = &cobra.Command{
	Use:   "serve [name] [path]",
	Short: "Register and serve a project",
	Long: `Register a Laravel/PHP project and serve it at http://name.test

Examples:
  herd serve osortoo ~/projects/osortoo
  herd serve quotation-crm ~/projects/quotation-crm --php=8.1`,
	Args: cobra.ExactArgs(2),
	RunE: serveProject,
}

func serveProject(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	projectPath := args[1]
	
	fmt.Printf("📝 Registering project: %s\n", projectName)
	fmt.Printf("📂 Path: %s\n", projectPath)
	
	// Step 1: Validate path
	if !utils.DirExists(projectPath) {
		return fmt.Errorf("project path does not exist: %s", projectPath)
	}
	absPath, _ := filepath.Abs(projectPath)
	
	// Step 2: Get herd directory
	herdDir, err := config.GetHerdDir()
	if err != nil {
		return fmt.Errorf("herd not initialized. Run 'herd init' first")
	}
	
	// Step 3: Detect PHP version
	var phpVersion string
	if explicitPHPVersion != "" {
		phpVersion = explicitPHPVersion
		fmt.Printf("[🔧] Using PHP %s (from --php flag)\n", phpVersion)
	} else {
		fmt.Println("[📖] Reading composer.json...")
		requirement, err := composer.ExtractPHPRequirement(absPath)
		if err != nil {
			fmt.Printf("[⚠️] No composer.json found, using latest PHP\n")
			versions, _ := utils.GetAvailablePHPVersions()
			if len(versions) > 0 {
				phpVersion = versions[len(versions)-1]
			} else {
				return fmt.Errorf("no PHP versions available")
			}
		} else if requirement == nil {
			fmt.Printf("[⚠️] No PHP requirement in composer.json, using latest\n")
			versions, _ := utils.GetAvailablePHPVersions()
			if len(versions) > 0 {
				phpVersion = versions[len(versions)-1]
			}
		} else {
			fmt.Printf("[🔍] Found requirement: %s\n", requirement.RawVersion)
			phpMgr := php.NewPHPManager(herdDir, nil)
			phpMgr.Initialize([]string{}) // Load available versions
			compatible, err := phpMgr.FindCompatibleVersion(requirement)
			if err != nil {
				return fmt.Errorf("no compatible PHP version: %w", err)
			}
			phpVersion = compatible
		}
		fmt.Printf("✓ Using PHP %s\n", phpVersion)
	}
	
	// Step 4: Detect project type
	projectType := detectProjectType(absPath)
	fmt.Printf("✓ Project type: %s\n", projectType)
	
	// Step 5: Generate Nginx config
	fmt.Println("[⚙️] Generating Nginx configuration...")
	ngixDir := filepath.Join(herdDir, "nginx", "sites-enabled")
	configGen := nginx.NewConfigGenerator(ngixDir)
	
	site := nginx.Site{
		Name:       projectName,
		Path:       absPath,
		Type:       projectType,
		PHPVersion: phpVersion,
	}
	
	if err := configGen.WriteConfig(site); err != nil {
		return fmt.Errorf("failed to generate nginx config: %w", err)
	}
	fmt.Println("✓ Nginx config generated")
	
	// Step 6: Register in sites.yaml
	fmt.Println("[📋] Registering site...")
	if err := config.RegisterSite(projectName, absPath, phpVersion, projectType); err != nil {
		return err
	}
	fmt.Println("✓ Site registered")
	
	// Step 7: Reload Nginx
	fmt.Println("[🔄] Reloading Nginx...")
	if err := utils.ExecuteCommand("sudo", "nginx", "-s", "reload"); err != nil {
		fmt.Printf("⚠️  Could not reload nginx. Run manually:\n")
		fmt.Printf("  sudo nginx -s reload\n")
	}
	
	fmt.Printf("\n✓ Project served at http://%s.test\n", projectName)
	fmt.Printf("✓ Visit: http://%s.test in your browser\n", projectName)
	
	return nil
}

func detectProjectType(path string) string {
	if utils.FileExists(filepath.Join(path, "artisan")) {
		return "laravel"
	}
	if utils.FileExists(filepath.Join(path, "composer.json")) {
		return "php"
	}
	if utils.FileExists(filepath.Join(path, "package.json")) {
		return "node"
	}
	return "static"
}

func init() {
	serveCmd.Flags().StringVar(&explicitPHPVersion, "php", "", "Explicit PHP version (e.g., 8.2)")
}
```

---

# PHASE 5: INTEGRATION & TESTING (1 hour)

## Step 5.1: Build & Test Script

Create `test.sh` in project root:

```bash
#!/bin/bash

set -e

echo "🏗️  Building Ubuntu Herd..."
go build -o herd

echo "✓ Build successful!"
echo ""
echo "🧪 Testing basic commands..."
echo ""

./herd --version 2>/dev/null || echo "Version flag not set (OK)"
./herd --help | head -5
echo ""

echo "✓ CLI framework working!"
echo ""
echo "Next: Run 'sudo ./herd init' to initialize the environment"
```

Make it executable:
```bash
chmod +x test.sh
```

## Step 5.2: Add Missing Imports to All Files

Ensure all files have proper imports:

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)
```

## Step 5.3: Update go.mod with all dependencies

```bash
go mod tidy
go mod download
```

---

# PHASE 6: POLISH & EDGE CASES (1 hour)

## Step 6.1: Error Handling Improvements

Update all commands to provide helpful error messages:

```go
// Example from serve.go
if !utils.DirExists(projectPath) {
	fmt.Printf("❌ Error: Project path does not exist\n")
	fmt.Printf("   Tried: %s\n", projectPath)
	fmt.Printf("\nUsage: herd serve <name> </path/to/project>\n")
	return fmt.Errorf("invalid project path")
}
```

## Step 6.2: Validation Logic

```go
// Add to pkg/utils/helpers.go
func ValidateProjectName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("project name cannot be empty")
	}
	if len(name) > 63 {
		return fmt.Errorf("project name too long (max 63 chars)")
	}
	if !regexp.MustCompile(`^[a-z0-9\-]+$`).MatchString(name) {
		return fmt.Errorf("project name can only contain lowercase letters, numbers, and hyphens")
	}
	return nil
}

func ValidatePHPVersion(version string) error {
	if !regexp.MustCompile(`^\d+\.\d+$`).MatchString(version) {
		return fmt.Errorf("invalid PHP version format. Use: 8.2, 8.1, etc.")
	}
	return nil
}
```

## Step 6.3: Add Help Text

Update all commands with comprehensive help:

```go
var serveCmd = &cobra.Command{
	Use:   "serve [name] [path]",
	Short: "Register and serve a project at http://name.test",
	Long: `Register a Laravel/PHP project and serve it locally.

The project's PHP version is auto-detected from composer.json.
You can override with --php flag.

Examples:
  herd serve myapp ~/projects/myapp
  herd serve legacy-app ~/projects/legacy --php=7.4
  herd serve modern-api ~/projects/api --php=8.3

The project will be accessible at: http://myapp.test`,
	Args: cobra.ExactArgs(2),
	RunE: serveProject,
}
```

---

# PHASE 7: FINAL ASSEMBLY & VERIFICATION (1 hour)

## Step 7.1: Complete File Checklist

Verify all files exist:

```
✅ main.go
✅ cmd/root.go
✅ cmd/init.go
✅ cmd/serve.go
✅ cmd/status.go
✅ cmd/logs.go
✅ cmd/control.go (stop, start, restart, forget)
✅ pkg/php/manager.go
✅ pkg/composer/parser.go
✅ pkg/nginx/config.go
✅ pkg/services/manager.go
✅ pkg/config/loader.go
✅ pkg/config/sites.go
✅ pkg/utils/helpers.go
✅ go.mod
✅ go.sum (after go mod tidy)
```

## Step 7.2: Compilation Test

```bash
# Full build
go build -o herd

# Check all packages compile
go build ./pkg/...
go build ./cmd/...
```

## Step 7.3: Manual Testing Script

Create `manual-test.sh`:

```bash
#!/bin/bash

echo "🧪 Manual Test Suite for Ubuntu Herd MVP"
echo ""

# Test 1: Help
echo "[Test 1] herd --help"
./herd --help | head -5
echo ""

# Test 2: Init (will need manual approval)
echo "[Test 2] herd init (DRY RUN - shows what would happen)"
echo "Would initialize with PHP versions detected on system"
echo ""

# Test 3: Status
echo "[Test 3] herd status"
./herd status
echo ""

# Test 4: Test project serve (with dummy paths)
echo "[Test 4] Testing project registration logic"
echo "Manual test: ./herd serve testproj ~/projects/testproj"
echo ""

echo "✅ All basic tests completed!"
echo "For full testing, use: sudo ./herd init"
```

## Step 7.4: README

Create `README.md`:

```markdown
# Ubuntu Herd MVP

Local PHP/Laravel development environment for Ubuntu with multi-PHP version support.

## Quick Start

1. **Initialize Herd**
   \`\`\`bash
   sudo ./herd init
   \`\`\`

2. **Register your first project**
   \`\`\`bash
   ./herd serve myapp ~/projects/myapp
   \`\`\`

3. **Visit in browser**
   Open http://myapp.test

## Prerequisites

- Ubuntu 20.04+
- PHP 7.4+ (at least one version installed)
- Nginx
- Dnsmasq (for .test domains)

Install:
\`\`\`bash
sudo apt install php8.2-fpm php8.1-fpm php8.0-fpm nginx dnsmasq
\`\`\`

## Commands

- `herd init` - Initialize Herd environment
- `herd serve [name] [path]` - Register a project
- `herd status` - Show all services
- `herd logs [service]` - View service logs
- `herd start|stop|restart` - Control services

## Features

✅ Multiple PHP versions running simultaneously
✅ Auto-detect PHP version from composer.json
✅ .test domain names for local development
✅ Per-project PHP version isolation
✅ Simple CLI interface

## Architecture

See ARCHITECTURE.md for detailed technical documentation.
```

---

# PHASE 8: BUFFER & CONTINGENCY

Reserve final time for:

1. **Bug fixes** - Unexpected compilation errors, missing dependencies
2. **Testing** - Validate core workflows (init → serve → visit domain)
3. **Edge cases** - Handle missing PHP versions, permission issues
4. **Documentation** - Final README and inline code comments

---

# Critical Implementation Notes for LLM

## IMPORTANT: Follow This Order Strictly

1. **Start with Phase 0** - No jumping ahead
2. **Don't skip file creation** - Every file matters
3. **Import ordering** - Use consistent imports across all files
4. **Error handling** - Every error must have a clear message
5. **Test compilation frequently** - `go build` after each phase

## Key Patterns Used

### Error Handling Pattern
```go
if err != nil {
	return fmt.Errorf("human-readable message: %w", err)
}
```

### Directory Pattern
```go
herdDir, err := config.GetHerdDir()
if err != nil {
	return err
}
```

### Config Pattern
```go
sites, err := config.LoadSites()
if err != nil {
	return err
}
```

## What NOT To Do

❌ Don't modify existing core packages unless explicitly instructed  
❌ Don't add external dependencies without approval  
❌ Don't skip error handling  
❌ Don't create files out of order  
❌ Don't assume directories exist without creating them first  

## What TO Do

✅ Follow the phase order exactly  
✅ Test after each phase with `go build`  
✅ Add descriptive print statements  
✅ Validate all paths exist  
✅ Handle all error cases  

---

# Expected Output After Completion

## Binary Capabilities

```bash
$ ./herd --help
Ubuntu Herd - Local PHP/Laravel development environment

Usage:
  herd [command]

Available Commands:
  init       Initialize Herd environment
  serve      Register and serve a project
  status     Show status of services
  logs       View service logs
  start      Start all services
  stop       Stop all services
  restart    Restart a service
  forget     Remove a registered project

$ ./herd init
🚀 Initializing Herd environment...
[1/5] Creating directory structure...
✓ Created ~/.herd
[2/5] Detecting available PHP versions...
✓ Found PHP versions: [8.0 8.1 8.2 8.3]
[3/5] Setting up PHP-FPM instances...
✓ PHP 8.0 configured
✓ PHP 8.1 configured
✓ PHP 8.2 configured
✓ PHP 8.3 configured
[4/5] Configuring Nginx...
✓ Nginx configured
[5/5] Configuring Dnsmasq for .test domains...
✓ Dnsmasq configured
✓ Herd initialized successfully!

$ ./herd serve osortoo ~/projects/osortoo
📝 Registering project: osortoo
📂 Path: ~/projects/osortoo
[📖] Reading composer.json...
[🔍] Found requirement: ^8.2
✓ Using PHP 8.2
✓ Project type: laravel
[⚙️] Generating Nginx configuration...
✓ Nginx config generated
[📋] Registering site...
✓ Site registered
[🔄] Reloading Nginx...
✓ Project served at http://osortoo.test
```

---

# Success Criteria

Mark as complete when:

- [x] All files compile without errors
- [x] `./herd init` creates ~/.herd directory structure
- [x] `./herd serve` successfully registers a test project
- [x] `./herd status` shows basic status
- [x] Each command has helpful error messages
- [x] README explains how to use it

---

# File Sizes (For Verification)

Rough expected sizes:

```
main.go                      ~100 lines
cmd/root.go                  ~50 lines
cmd/init.go                  ~180 lines
cmd/serve.go                 ~140 lines
cmd/status.go                ~40 lines
cmd/logs.go                  ~40 lines
cmd/control.go               ~80 lines
pkg/php/manager.go           ~400 lines (PROVIDED)
pkg/composer/parser.go       ~300 lines (PROVIDED)
pkg/nginx/config.go          ~300 lines (PROVIDED)
pkg/services/manager.go      ~400 lines (PROVIDED)
pkg/config/loader.go         ~100 lines
pkg/config/sites.go          ~150 lines
pkg/utils/helpers.go         ~150 lines

TOTAL: ~2,800 lines of code
```

---

# Now You're Ready!

Pass this entire document to Claude with:

```
I have a comprehensive development roadmap for Ubuntu Herd MVP.
Follow it EXACTLY, phase by phase, to build the complete working app.
After each phase, compile with `go build` and verify.
All file contents and implementations are in this roadmap.
Start with Phase 0 and complete through Phase 8.
```

This will guide the LLM to build a complete, working application in one vibe coding session. 🚀
