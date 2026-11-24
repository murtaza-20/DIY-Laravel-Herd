package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/thesmartyking/herd/pkg/php"
)

const siteTemplate = `
# Managed by Ubuntu Herd — {{ .Name }} (PHP {{ .PHPVersion }}, type {{ .Type }})
server {
    listen 80;
    listen [::]:80;
    server_name {{ .Name }}.test *.{{ .Name }}.test;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;

    server_name {{ .Name }}.test *.{{ .Name }}.test;
    root {{ .Root }};

    index index.php index.html index.htm;
    charset utf-8;

    client_max_body_size 120M;

    ssl_certificate     {{ .CertPath }};
    ssl_certificate_key {{ .KeyPath }};
    ssl_protocols TLSv1.2 TLSv1.3;

{{ if eq .Type "laravel" }}
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location = /favicon.ico { access_log off; log_not_found off; }
    location = /robots.txt  { access_log off; log_not_found off; }

    error_page 404 /index.php;
{{ else if or (eq .Type "static") (eq .Type "node") }}
    location / {
        try_files $uri $uri/ =404;
    }
{{ else }}
    location / {
        try_files $uri $uri/ =404;
    }
{{ end }}

{{ if ne .Type "static" }}{{ if ne .Type "node" }}
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:{{ .Socket }};
    }
{{ end }}{{ end }}

    location ~ /\.(?!well-known).* {
        deny all;
    }
}
`

// Site represents a website configuration
type Site struct {
	Name       string
	Path       string
	Type       string
	PHPVersion string
	Socket     string
	Root       string
	CertPath   string
	KeyPath    string
}

// ConfigGenerator generates Nginx configurations
type ConfigGenerator struct {
	sitesDir string
	certPath string
	keyPath  string
}

// NewConfigGenerator creates a new config generator
func NewConfigGenerator(sitesDir string) *ConfigGenerator {
	return &ConfigGenerator{sitesDir: sitesDir}
}

// SetTLS sets certificate paths used for all sites
func (cg *ConfigGenerator) SetTLS(certPath, keyPath string) {
	cg.certPath = certPath
	cg.keyPath = keyPath
}

// DetectProjectType tries to detect the project type
func DetectProjectType(path string) string {
	if fileExists(filepath.Join(path, "artisan")) {
		return "laravel"
	}
	if fileExists(filepath.Join(path, "composer.json")) {
		return "php"
	}
	if fileExists(filepath.Join(path, "package.json")) {
		return "node"
	}
	if fileExists(filepath.Join(path, "index.html")) {
		return "static"
	}
	return "php"
}

func siteRoot(site Site) string {
	if site.Type == "laravel" {
		return filepath.Join(site.Path, "public")
	}
	return site.Path
}

// GenerateConfig generates an Nginx config for a site
func (cg *ConfigGenerator) GenerateConfig(site Site) (string, error) {
	if site.Type == "" {
		site.Type = DetectProjectType(site.Path)
	}
	if site.PHPVersion != "" {
		site.Socket = php.SocketPath(site.PHPVersion)
	} else {
		site.Socket = "/run/php/php-fpm.sock"
	}
	site.Root = siteRoot(site)
	site.CertPath = cg.certPath
	site.KeyPath = cg.keyPath
	if site.CertPath == "" || site.KeyPath == "" {
		return "", fmt.Errorf("TLS certificate not configured; run: herd init")
	}

	tmpl, err := template.New("site").Parse(siteTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, site); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteConfig writes an Nginx config file to disk
func (cg *ConfigGenerator) WriteConfig(site Site) error {
	config, err := cg.GenerateConfig(site)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cg.sitesDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(cg.sitesDir, fmt.Sprintf("%s.conf", site.Name))
	return os.WriteFile(configPath, []byte(config), 0644)
}

// RemoveConfig removes an Nginx config file
func (cg *ConfigGenerator) RemoveConfig(siteName string) error {
	configPath := filepath.Join(cg.sitesDir, fmt.Sprintf("%s.conf", siteName))
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
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
			sites = append(sites, strings.TrimSuffix(entry.Name(), ".conf"))
		}
	}
	return sites, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
