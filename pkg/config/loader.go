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

	if err := os.MkdirAll(herdDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create herd directory: %w", err)
	}

	dirs := []string{
		filepath.Join(herdDir, "php"),
		filepath.Join(herdDir, "nginx", "sites-enabled"),
		filepath.Join(herdDir, "logs"),
		filepath.Join(herdDir, "ssl"),
		filepath.Join(herdDir, "data", "mysql"),
		filepath.Join(herdDir, "data", "postgres"),
		filepath.Join(herdDir, "data", "redis"),
		filepath.Join(herdDir, "data", "minio"),
		filepath.Join(herdDir, "data", "backups"),
		filepath.Join(herdDir, "data", "elasticsearch"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return herdDir, nil
}

// IsInitialized returns true if ~/.herd exists
func IsInitialized() bool {
	herdDir, err := GetHerdDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(herdDir)
	return err == nil && info.IsDir()
}
