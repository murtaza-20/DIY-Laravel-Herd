package php

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/thesmartyking/herd/pkg/composer"
	"github.com/thesmartyking/herd/pkg/utils"
)

// Manager manages multi-version PHP-FPM configuration under ~/.herd/php
type Manager struct {
	herdDir  string
	versions []string
	logger   *logrus.Logger
}

// NewPHPManager creates a PHP manager rooted at herdDir
func NewPHPManager(herdDir string, logger *logrus.Logger) *Manager {
	if logger == nil {
		logger = logrus.New()
	}
	return &Manager{
		herdDir: herdDir,
		logger:  logger,
	}
}

// Initialize sets up FPM pool config dirs for the given versions.
// If versions is empty, discovers installed PHP binaries.
func (m *Manager) Initialize(versions []string) error {
	if len(versions) == 0 {
		discovered, err := utils.GetAvailablePHPVersions()
		if err != nil {
			return err
		}
		versions = discovered
	}

	m.versions = versions
	phpRoot := filepath.Join(m.herdDir, "php")
	if err := os.MkdirAll(phpRoot, 0755); err != nil {
		return fmt.Errorf("failed to create php dir: %w", err)
	}

	// Ensure shared socket directory exists (best-effort; may need sudo in production)
	_ = os.MkdirAll("/var/run/herd", 0755)

	for _, v := range versions {
		if err := m.setupVersion(v); err != nil {
			return fmt.Errorf("failed to setup PHP %s: %w", v, err)
		}
	}

	return nil
}

func (m *Manager) setupVersion(version string) error {
	versionDir := filepath.Join(m.herdDir, "php", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return err
	}

	// P8: use system php-fpm sockets (/run/php/phpX.Y-fpm.sock) which are already running.
	sockPath := SocketPath(version)
	poolConf := fmt.Sprintf(`; Reference pool for Herd — system FPM already listens on:
;   %s
; Optional: copy to /etc/php/%s/fpm/pool.d/herd.conf if you want a dedicated Herd pool.
[herd]
user = www-data
group = www-data
listen = %s
listen.owner = www-data
listen.group = www-data
pm = dynamic
pm.max_children = 5
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
catch_workers_output = yes
`, sockPath, version, sockPath)

	poolPath := filepath.Join(versionDir, "pool.conf")
	if err := os.WriteFile(poolPath, []byte(poolConf), 0644); err != nil {
		return err
	}

	meta := fmt.Sprintf("binary=%s\nfpm=%s\nsocket=%s\nnote=using-system-fpm-socket\n",
		utils.PHPBinaryPath(version),
		utils.PHPFPMBinaryPath(version),
		sockPath,
	)
	if err := os.WriteFile(filepath.Join(versionDir, "meta.txt"), []byte(meta), 0644); err != nil {
		return err
	}

	m.logger.Infof("Configured PHP %s FPM pool at %s", version, versionDir)
	return nil
}

// SocketPath returns the system php-fpm unix socket for a version (e.g. /run/php/php8.2-fpm.sock)
func SocketPath(version string) string {
	return fmt.Sprintf("/run/php/php%s-fpm.sock", version)
}

// FindCompatibleVersion picks the highest installed PHP that satisfies the requirement
func (m *Manager) FindCompatibleVersion(requirement *composer.PHPRequirement) (string, error) {
	if requirement == nil {
		return "", fmt.Errorf("nil PHP requirement")
	}

	versions := m.versions
	if len(versions) == 0 {
		discovered, err := utils.GetAvailablePHPVersions()
		if err != nil {
			return "", err
		}
		versions = discovered
		m.versions = versions
	}

	var best string
	for _, v := range versions {
		if requirement.Satisfies(v) {
			best = v // candidates are ascending; keep last match = highest
		}
	}

	if best == "" {
		return "", fmt.Errorf("no installed PHP version satisfies %s (have: %v)", requirement.RawVersion, versions)
	}
	return best, nil
}

// Versions returns configured PHP versions
func (m *Manager) Versions() []string {
	return m.versions
}
