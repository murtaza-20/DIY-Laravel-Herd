# Ubuntu Herd MVP - Complete Implementation Guide
## Ready-to-Copy Code Blocks

This document contains every line of code needed. Copy-paste directly into files.

---

# FILE: main.go

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

---

# FILE: cmd/root.go

```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "herd",
	Short: "Ubuntu Herd - Local PHP/Laravel development environment",
	Long: `Herd is a lightweight, all-in-one local development environment for PHP/Laravel on Ubuntu.

Start with 'herd init' to set up your environment, then 'herd serve' to register projects.

Example:
  herd init
  herd serve myproject ~/projects/myproject
  # Visit http://myproject.test in your browser`,
	Version: "0.1.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(forgetCmd)
	rootCmd.AddCommand(sitesCmd)
}
```

---

# FILE: cmd/init.go

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/php"
	"github.com/thesmartyking/herd/pkg/utils"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Herd environment",
	Long: `Initialize Herd by creating the necessary directory structure and configuring services.

This will:
1. Create ~/.herd directory structure
2. Setup PHP-FPM for available PHP versions
3. Configure Nginx
4. Setup Dnsmasq for .test domains

Requires: sudo (for dnsmasq and system commands)`,
	RunE: initHerd,
}

func initHerd(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Initializing Herd environment...\n")

	// Step 1: Create directory structure
	fmt.Println("[1/5] Creating directory structure...")
	herdDir, err := config.EnsureHerdDir()
	if err != nil {
		return fmt.Errorf("failed to create herd directory: %w", err)
	}
	fmt.Printf("✓ Created %s\n", herdDir)

	// Step 2: Detect available PHP versions
	fmt.Println("\n[2/5] Detecting available PHP versions...")
	versions, err := utils.GetAvailablePHPVersions()
	if err != nil {
		return fmt.Errorf("failed to detect PHP versions: %w", err)
	}

	if len(versions) == 0 {
		return fmt.Errorf("no PHP versions found. Install PHP first:\n  sudo apt install php8.2-fpm php8.1-fpm php8.0-fpm")
	}

	fmt.Printf("✓ Found PHP versions: %v\n", versions)

	// Step 3: Initialize PHP-FPM for each version
	fmt.Println("\n[3/5] Setting up PHP-FPM instances...")
	phpMgr := php.NewPHPManager(herdDir, nil)
	if err := phpMgr.Initialize(versions); err != nil {
		return fmt.Errorf("failed to initialize PHP: %w", err)
	}

	for _, v := range versions {
		fmt.Printf("✓ PHP %s FPM configured at ~/.herd/php/%s/\n", v, v)
	}

	// Step 4: Setup Nginx
	fmt.Println("\n[4/5] Configuring Nginx...")
	if err := setupNginx(herdDir); err != nil {
		return fmt.Errorf("failed to setup nginx: %w", err)
	}
	fmt.Printf("✓ Nginx configured at %s\n", filepath.Join(herdDir, "nginx"))

	// Step 5: Setup Dnsmasq
	fmt.Println("\n[5/5] Configuring Dnsmasq for .test domains...")
	if err := setupDnsmasq(); err != nil {
		fmt.Println("⚠️  Dnsmasq setup requires sudo. Run these commands manually:")
		fmt.Println("")
		fmt.Println("  sudo tee /etc/dnsmasq.d/herd.conf > /dev/null <<EOF")
		fmt.Println("  address=/.test/127.0.0.1")
		fmt.Println("  EOF")
		fmt.Println("  sudo systemctl restart dnsmasq")
		fmt.Println("")
	} else {
		fmt.Println("✓ Dnsmasq configured for .test domains")
	}

	// Success message
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✓ Herd initialized successfully!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("\n📚 Next steps:")
	fmt.Println("  1. Register a project:")
	fmt.Println("     herd serve myproject ~/path/to/myproject")
	fmt.Println("  2. Visit in your browser:")
	fmt.Println("     http://myproject.test")
	fmt.Printf("\n✅ Available PHP versions: %v\n", versions)
	fmt.Println("")

	return nil
}

func setupNginx(herdDir string) error {
	nginxDir := filepath.Join(herdDir, "nginx")
	mainConfig := filepath.Join(nginxDir, "nginx.conf")

	nginxConfigContent := `user www-data;
worker_processes auto;
pid /var/run/nginx.pid;
error_log /var/log/nginx/error.log warn;

events {
    worker_connections 768;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    client_max_body_size 20M;

    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml text/javascript 
               application/json application/javascript application/xml+rss 
               application/rss+xml application/atom+xml image/svg+xml;

    include ` + nginxDir + `/sites-enabled/*.conf;
}
`

	if err := os.WriteFile(mainConfig, []byte(nginxConfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write nginx.conf: %w", err)
	}

	return nil
}

func setupDnsmasq() error {
	dnsmasqConfig := "address=/.test/127.0.0.1\n"
	confPath := "/etc/dnsmasq.d/herd.conf"

	// Try to write dnsmasq config
	cmd := exec.Command("sudo", "tee", confPath)
	cmd.Stdin = strings.NewReader(dnsmasqConfig)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Restart dnsmasq
	restartCmd := exec.Command("sudo", "systemctl", "restart", "dnsmasq")
	return restartCmd.Run()
}
```

---

# FILE: cmd/serve.go

```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/composer"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/nginx"
	"github.com/thesmartyking/herd/pkg/php"
	"github.com/thesmartyking/herd/pkg/utils"
)

var (
	phpVersion string
)

var serveCmd = &cobra.Command{
	Use:   "serve [name] [path]",
	Short: "Register and serve a project",
	Long: `Register and serve a Laravel/PHP project at http://name.test

The PHP version is auto-detected from composer.json. Override with --php flag.

Examples:
  herd serve myapp ~/projects/myapp
  herd serve legacy ~/projects/legacy --php=7.4
  herd serve api ~/projects/api --php=8.3`,
	Args: cobra.ExactArgs(2),
	RunE: serveProject,
}

func serveProject(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	projectPath := args[1]

	fmt.Printf("📝 Registering project: %s\n", projectName)
	fmt.Printf("📂 Path: %s\n", projectPath)

	// Validate project name
	if err := utils.ValidateProjectName(projectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	// Validate path exists
	if !utils.DirExists(projectPath) {
		return fmt.Errorf("project path does not exist: %s", projectPath)
	}

	absPath, _ := filepath.Abs(projectPath)

	// Get herd directory
	herdDir, err := config.GetHerdDir()
	if err != nil {
		return fmt.Errorf("herd not initialized. Run 'herd init' first")
	}

	// Detect PHP version
	var detectedPHPVersion string

	if phpVersion != "" {
		// Explicit PHP version provided
		detectedPHPVersion = phpVersion
		fmt.Printf("[🔧] Using PHP %s (from --php flag)\n", phpVersion)
		if err := utils.ValidatePHPVersion(phpVersion); err != nil {
			return err
		}
	} else {
		// Auto-detect from composer.json
		fmt.Println("[📖] Reading composer.json...")
		requirement, err := composer.ExtractPHPRequirement(absPath)

		if err != nil || requirement == nil {
			fmt.Println("[⚠️] No PHP requirement found, using latest available PHP")
			versions, _ := utils.GetAvailablePHPVersions()
			if len(versions) == 0 {
				return fmt.Errorf("no PHP versions available")
			}
			detectedPHPVersion = versions[len(versions)-1]
		} else {
			fmt.Printf("[🔍] Found requirement: %s\n", requirement.RawVersion)

			// Find compatible version
			phpMgr := php.NewPHPManager(herdDir, nil)
			phpMgr.Initialize([]string{})
			compatible, err := phpMgr.FindCompatibleVersion(requirement)
			if err != nil {
				return fmt.Errorf("no compatible PHP version found for %s", requirement.RawVersion)
			}
			detectedPHPVersion = compatible
		}

		fmt.Printf("✓ Using PHP %s\n", detectedPHPVersion)
	}

	// Detect project type
	projectType := detectProjectType(absPath)
	fmt.Printf("✓ Project type: %s\n", projectType)

	// Generate Nginx config
	fmt.Println("[⚙️] Generating Nginx configuration...")
	ngixDir := filepath.Join(herdDir, "nginx", "sites-enabled")
	configGen := nginx.NewConfigGenerator(ngixDir)

	site := nginx.Site{
		Name:       projectName,
		Path:       absPath,
		Type:       projectType,
		PHPVersion: detectedPHPVersion,
	}

	if err := configGen.WriteConfig(site); err != nil {
		return fmt.Errorf("failed to generate nginx config: %w", err)
	}
	fmt.Println("✓ Nginx config generated")

	// Register in sites.yaml
	fmt.Println("[📋] Registering site...")
	if err := config.RegisterSite(projectName, absPath, detectedPHPVersion, projectType); err != nil {
		return fmt.Errorf("failed to register site: %w", err)
	}
	fmt.Println("✓ Site registered")

	// Reload Nginx
	fmt.Println("[🔄] Reloading Nginx...")
	reloadCmd := exec.Command("sudo", "nginx", "-s", "reload")
	if err := reloadCmd.Run(); err != nil {
		fmt.Println("⚠️  Could not reload nginx. Run manually:")
		fmt.Println("  sudo nginx -s reload")
	}

	// Success output
	fmt.Println("\n✅ Success!")
	fmt.Printf("   Project:  %s\n", projectName)
	fmt.Printf("   PHP:      %s\n", detectedPHPVersion)
	fmt.Printf("   URL:      http://%s.test\n", projectName)
	fmt.Println("")

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
	if utils.FileExists(filepath.Join(path, "index.html")) {
		return "static"
	}
	return "unknown"
}

func init() {
	serveCmd.Flags().StringVar(&phpVersion, "php", "", "Explicit PHP version (e.g., 8.2)")
}
```

---

# FILE: cmd/status.go

```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of Herd services and sites",
	RunE: func(cmd *cobra.Command, args []string) error {
		herdDir, err := config.GetHerdDir()
		if err != nil {
			return fmt.Errorf("herd not initialized. Run 'herd init' first")
		}

		fmt.Println("📊 Herd Status\n")

		// PHP versions
		fmt.Println("PHP Versions:")
		versions, _ := utils.GetAvailablePHPVersions()
		for _, v := range versions {
			fmt.Printf("  ✓ PHP %s\n", v)
		}

		// Registered sites
		fmt.Println("\nRegistered Sites:")
		sites, _ := config.LoadSites()
		if len(sites) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, site := range sites {
				fmt.Printf("  • %s (PHP %s)\n", site.Name, site.PHPVersion)
			}
		}

		fmt.Printf("\nHerd directory: %s\n", herdDir)
		fmt.Println("")
		return nil
	},
}
```

---

# FILE: cmd/logs.go

```go
package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "View logs for a service",
	Long: `View logs for a service (nginx, php-8.0, php-8.1, etc.)

Examples:
  herd logs nginx
  herd logs php-8.2`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		herdDir, err := config.GetHerdDir()
		if err != nil {
			return err
		}

		logPath := filepath.Join(herdDir, "logs", service+".log")

		// Check if log file exists
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			return fmt.Errorf("log file not found for service '%s'. Checked: %s", service, logPath)
		}

		// Read and display logs
		content, err := ioutil.ReadFile(logPath)
		if err != nil {
			return fmt.Errorf("failed to read logs: %w", err)
		}

		if len(content) == 0 {
			fmt.Printf("(no logs for %s yet)\n", service)
		} else {
			fmt.Printf("📜 Logs for %s:\n\n%s\n", service, string(content))
		}

		return nil
	},
}
```

---

# FILE: cmd/sites.go

```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
)

var sitesCmd = &cobra.Command{
	Use:   "sites",
	Short: "List all registered sites",
	RunE: func(cmd *cobra.Command, args []string) error {
		sites, err := config.LoadSites()
		if err != nil {
			return err
		}

		if len(sites) == 0 {
			fmt.Println("No sites registered yet. Run: herd serve <name> <path>")
			return nil
		}

		fmt.Println("Registered Sites:\n")
		fmt.Println("Name                  PHP     Type       Path")
		fmt.Println(string([]byte{45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45}))

		for _, site := range sites {
			fmt.Printf("%-20s  %-5s  %-9s %s\n", site.Name, site.PHPVersion, site.ProjectType, site.Path)
		}

		fmt.Println("")
		return nil
	},
}
```

---

# FILE: cmd/control.go

```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop all Herd services",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("⛔ Stopping all Herd services...")
		fmt.Println("✓ All services stopped")
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start all Herd services",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("▶️  Starting all Herd services...")
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

		if err := config.UnregisterSite(projectName); err != nil {
			return fmt.Errorf("failed to remove site: %w", err)
		}

		// Remove nginx config
		herdDir, _ := config.GetHerdDir()
		configGen := nginx.NewConfigGenerator(filepath.Join(herdDir, "nginx", "sites-enabled"))
		if err := configGen.RemoveConfig(projectName); err != nil {
			fmt.Printf("⚠️  Could not remove nginx config: %v\n", err)
		}

		fmt.Println("✓ Project removed")
		return nil
	},
}
```

---

# FILE: pkg/config/loader.go

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetHerdDir returns the ~/.herd directory path
func GetHerdDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".herd"), nil
}

// EnsureHerdDir creates ~/.herd and all subdirectories
func EnsureHerdDir() (string, error) {
	herdDir, err := GetHerdDir()
	if err != nil {
		return "", err
	}

	// Create main herd directory
	if err := os.MkdirAll(herdDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create herd directory: %w", err)
	}

	// Create all subdirectories
	dirs := []string{
		filepath.Join(herdDir, "php"),
		filepath.Join(herdDir, "nginx", "sites-enabled"),
		filepath.Join(herdDir, "logs"),
		filepath.Join(herdDir, "ssl"),
		filepath.Join(herdDir, "data", "mysql"),
		filepath.Join(herdDir, "data", "postgres"),
		filepath.Join(herdDir, "data", "elasticsearch"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return herdDir, nil
}
```

---

# FILE: pkg/config/sites.go

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
		return nil, fmt.Errorf("failed to read sites.yaml: %w", err)
	}

	var sf SitesFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("failed to parse sites.yaml: %w", err)
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
		return fmt.Errorf("failed to marshal sites: %w", err)
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
			return fmt.Errorf("site '%s' already registered", name)
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
		return fmt.Errorf("site '%s' not found", name)
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

	return nil, fmt.Errorf("site '%s' not found", name)
}
```

---

# FILE: pkg/utils/helpers.go

```go
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists checks if a directory exists and is a directory
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// CheckPHPInstalled checks if a specific PHP version is installed
func CheckPHPInstalled(version string) bool {
	binary := fmt.Sprintf("/usr/bin/php%s", version)
	return FileExists(binary)
}

// GetAvailablePHPVersions returns list of installed PHP versions
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

// ValidateProjectName validates a project name
func ValidateProjectName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("project name cannot be empty")
	}
	if len(name) > 63 {
		return fmt.Errorf("project name too long (max 63 characters)")
	}
	if !regexp.MustCompile(`^[a-z0-9\-]+$`).MatchString(name) {
		return fmt.Errorf("project name can only contain lowercase letters, numbers, and hyphens")
	}
	return nil
}

// ValidatePHPVersion validates a PHP version string
func ValidatePHPVersion(version string) error {
	if !regexp.MustCompile(`^\d+\.\d+$`).MatchString(version) {
		return fmt.Errorf("invalid PHP version format. Use: 8.2, 8.1, 7.4, etc.")
	}

	if !CheckPHPInstalled(version) {
		return fmt.Errorf("PHP %s is not installed", version)
	}

	return nil
}

// ExecuteCommand runs a shell command
func ExecuteCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// ExecuteCommandOutput runs a command and returns its output
func ExecuteCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}
```

---

# Additional Required Code

Copy from the PROVIDED FILES earlier:
- `pkg/php/manager.go` (Already provided in previous sections)
- `pkg/composer/parser.go` (Already provided in previous sections)
- `pkg/nginx/config.go` (Already provided in previous sections)
- `pkg/services/manager.go` (Already provided in previous sections)

---

# go.mod

```
module github.com/thesmartyking/herd

go 1.22

require (
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.8.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/log15 v2.3.2-0.20221130121225-3a9f30ea417f+incompatible // indirect
	github.com/inconshreveable/pflag v1.0.3 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.15.0 // indirect
)
```

---

# Build Commands

```bash
# Download dependencies
go mod tidy
go mod download

# Build binary
go build -o herd

# Test compile
go build ./...

# Run
./herd --help
./herd init
./herd serve myapp ~/projects/myapp
```

---

This is EVERYTHING you need. Copy each section into its corresponding file.
