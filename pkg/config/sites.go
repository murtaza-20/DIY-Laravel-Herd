package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// SiteNames returns the registered site names in registry order.
func SiteNames() []string {
	sites, err := LoadSites()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(sites))
	for _, s := range sites {
		names = append(names, s.Name)
	}
	return names
}

// LoadSites reads all registered sites
func LoadSites() ([]Site, error) {
	herdDir, err := GetHerdDir()
	if err != nil {
		return nil, err
	}

	sitesPath := filepath.Join(herdDir, "sites.yaml")

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

// FindSiteByPath returns the registered site whose Path is the longest
// prefix of target (exact match or target is inside the project tree).
func FindSiteByPath(target string) (*Site, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	abs = filepath.Clean(abs)

	sites, err := LoadSites()
	if err != nil {
		return nil, err
	}

	var best *Site
	bestLen := -1
	for i := range sites {
		sitePath, err := filepath.Abs(sites[i].Path)
		if err != nil {
			continue
		}
		sitePath = filepath.Clean(sitePath)
		if abs == sitePath || strings.HasPrefix(abs, sitePath+string(os.PathSeparator)) {
			if len(sitePath) > bestLen {
				bestLen = len(sitePath)
				best = &sites[i]
			}
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no Herd site registered for %s\nRun from a served project directory, or: herd serve <name> <path>", abs)
	}
	return best, nil
}

// FindSiteForCwd resolves the site for the current working directory.
func FindSiteForCwd() (*Site, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return FindSiteByPath(cwd)
}
