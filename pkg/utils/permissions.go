package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const webUser = "www-data"

// EnsureWebWritable makes Laravel storage/bootstrap dirs writable by PHP-FPM (www-data).
// Prefers POSIX ACLs; falls back to a+rwX for local development.
func EnsureWebWritable(projectPath string) error {
	dirs := []string{
		filepath.Join(projectPath, "storage"),
		filepath.Join(projectPath, "storage", "app"),
		filepath.Join(projectPath, "storage", "app", "public"),
		filepath.Join(projectPath, "storage", "framework"),
		filepath.Join(projectPath, "storage", "framework", "cache"),
		filepath.Join(projectPath, "storage", "framework", "cache", "data"),
		filepath.Join(projectPath, "storage", "framework", "sessions"),
		filepath.Join(projectPath, "storage", "framework", "views"),
		filepath.Join(projectPath, "storage", "logs"),
		filepath.Join(projectPath, "bootstrap", "cache"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0775); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
		_ = os.Chmod(d, 0775)
	}

	// Prefer ACL so the owner keeps control while www-data can write
	if _, err := exec.LookPath("setfacl"); err == nil {
		targets := []string{
			filepath.Join(projectPath, "storage"),
			filepath.Join(projectPath, "bootstrap", "cache"),
		}
		for _, t := range targets {
			_ = exec.Command("setfacl", "-R", "-m", "u:"+webUser+":rwx", t).Run()
			_ = exec.Command("setfacl", "-R", "-d", "-m", "u:"+webUser+":rwx", t).Run()
		}
		return nil
	}

	// Fallback when setfacl is unavailable
	for _, t := range []string{
		filepath.Join(projectPath, "storage"),
		filepath.Join(projectPath, "bootstrap", "cache"),
	} {
		_ = filepath.Walk(t, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				_ = os.Chmod(path, 0777)
			} else {
				_ = os.Chmod(path, 0666)
			}
			return nil
		})
	}
	return nil
}
