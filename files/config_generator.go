package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const laravelTemplate = `
server {
    listen 80;
    listen [::]:80;
    
    server_name {{ .Name }}.test *.{{ .Name }}.test;
    root {{ .Path }}/public;
    
    index index.php index.html index.htm;
    
    charset utf-8;
    
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
    
    location = /favicon.ico { access_log off; log_not_found off; }
    location = /robots.txt  { access_log off; log_not_found off; }
    
    error_page 404 /index.php;
    
    location ~ \.php$ {
        fastcgi_pass unix:/var/run/herd/php-fpm.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
    }
    
    location ~ /\.(?!well-known).* {
        deny all;
    }
}
`

const genericPHPTemplate = `
server {
    listen 80;
    listen [::]:80;
    
    server_name {{ .Name }}.test *.{{ .Name }}.test;
    root {{ .Path }};
    
    index index.php index.html index.htm;
    
    location / {
        try_files $uri $uri/ =404;
    }
    
    location ~ \.php$ {
        fastcgi_pass unix:/var/run/herd/php-fpm.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
    }
}
`

// Site represents a website configuration
type Site struct {
	Name string // Project name (e.g., "osortoo")
	Path string // Full path to project (e.g., "/home/user/projects/osortoo")
	Type string // Project type: "laravel", "php", "node", "static"
}

// ConfigGenerator generates Nginx configurations
type ConfigGenerator struct {
	sitesDir string // e.g., /home/user/.herd/nginx/sites-enabled
}

// NewConfigGenerator creates a new config generator
func NewConfigGenerator(sitesDir string) *ConfigGenerator {
	return &ConfigGenerator{
		sitesDir: sitesDir,
	}
}

// DetectProjectType tries to detect the project type
func DetectProjectType(path string) string {
	// Check for Laravel indicators
	if fileExists(filepath.Join(path, "artisan")) {
		return "laravel"
	}

	// Check for other framework indicators
	if fileExists(filepath.Join(path, "composer.json")) {
		return "php"
	}

	if fileExists(filepath.Join(path, "package.json")) {
		return "node"
	}

	if fileExists(filepath.Join(path, "index.html")) {
		return "static"
	}

	// Default to generic PHP
	return "php"
}

// GenerateConfig generates an Nginx config for a site
func (cg *ConfigGenerator) GenerateConfig(site Site) (string, error) {
	// Auto-detect project type if not specified
	if site.Type == "" {
		site.Type = DetectProjectType(site.Path)
	}

	var tmpl *template.Template
	var err error

	// Select template based on project type
	switch site.Type {
	case "laravel":
		tmpl, err = template.New("laravel").Parse(laravelTemplate)
	case "php", "static":
		tmpl, err = template.New("php").Parse(genericPHPTemplate)
	default:
		tmpl, err = template.New("php").Parse(genericPHPTemplate)
	}

	if err != nil {
		return "", err
	}

	// Render template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, site); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// WriteConfig writes an Nginx config file to disk
func (cg *ConfigGenerator) WriteConfig(site Site) error {
	// Generate config
	config, err := cg.GenerateConfig(site)
	if err != nil {
		return err
	}

	// Create sites-enabled directory if it doesn't exist
	if err := os.MkdirAll(cg.sitesDir, 0755); err != nil {
		return err
	}

	// Write config file
	configPath := filepath.Join(cg.sitesDir, fmt.Sprintf("%s.conf", site.Name))
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return err
	}

	return nil
}

// RemoveConfig removes an Nginx config file
func (cg *ConfigGenerator) RemoveConfig(siteName string) error {
	configPath := filepath.Join(cg.sitesDir, fmt.Sprintf("%s.conf", siteName))
	if err := os.Remove(configPath); err != nil {
		return err
	}
	return nil
}

// ListConfigs lists all configured sites
func (cg *ConfigGenerator) ListConfigs() ([]string, error) {
	entries, err := os.ReadDir(cg.sitesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var sites []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".conf") {
			siteName := strings.TrimSuffix(entry.Name(), ".conf")
			sites = append(sites, siteName)
		}
	}

	return sites, nil
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
