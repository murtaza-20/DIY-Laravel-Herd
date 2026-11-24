package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// KnownPHPVersions is the set Herd actively supports for multi-version local stacks.
// Any /usr/bin/phpX.Y outside this list is still discovered automatically.
var KnownPHPVersions = []string{
	"5.6", "7.0", "7.1", "7.2", "7.3", "7.4",
	"8.0", "8.1", "8.2", "8.3", "8.4", "8.5", "8.6",
}

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

// PHPBinaryPath returns the CLI binary path for a version (e.g. /usr/bin/php8.5)
func PHPBinaryPath(version string) string {
	return fmt.Sprintf("/usr/bin/php%s", version)
}

// PHPFPMBinaryPath returns the FPM binary path for a version
func PHPFPMBinaryPath(version string) string {
	return fmt.Sprintf("/usr/sbin/php-fpm%s", version)
}

// AptFPMPackage returns the apt package name for a PHP-FPM version
func AptFPMPackage(version string) string {
	return fmt.Sprintf("php%s-fpm", version)
}

// CheckPHPInstalled checks if a specific PHP version is installed (CLI or FPM)
func CheckPHPInstalled(version string) bool {
	return FileExists(PHPBinaryPath(version)) || FileExists(PHPFPMBinaryPath(version))
}

// GetAvailablePHPVersions returns installed PHP major.minor versions (ascending).
// Discovers versioned CLI, FPM binaries, and /etc/php/X.Y trees.
func GetAvailablePHPVersions() ([]string, error) {
	seen := map[string]struct{}{}
	add := func(v string) {
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		if !CheckPHPInstalled(v) && !DirExists(filepath.Join("/etc/php", v)) {
			return
		}
		// Prefer versions that at least have a CLI or FPM binary
		if CheckPHPInstalled(v) {
			seen[v] = struct{}{}
		}
	}

	cliRe := regexp.MustCompile(`^php(\d+\.\d+)$`)
	if entries, err := os.ReadDir("/usr/bin"); err == nil {
		for _, entry := range entries {
			if m := cliRe.FindStringSubmatch(entry.Name()); m != nil {
				add(m[1])
			}
		}
	}

	fpmRe := regexp.MustCompile(`^php-fpm(\d+\.\d+)$`)
	if entries, err := os.ReadDir("/usr/sbin"); err == nil {
		for _, entry := range entries {
			if m := fpmRe.FindStringSubmatch(entry.Name()); m != nil {
				add(m[1])
			}
		}
	}

	if entries, err := os.ReadDir("/etc/php"); err == nil {
		verRe := regexp.MustCompile(`^\d+\.\d+$`)
		for _, entry := range entries {
			if entry.IsDir() && verRe.MatchString(entry.Name()) {
				add(entry.Name())
			}
		}
	}

	for _, v := range KnownPHPVersions {
		if CheckPHPInstalled(v) {
			seen[v] = struct{}{}
		}
	}

	versions := make([]string, 0, len(seen))
	for v := range seen {
		versions = append(versions, v)
	}

	// Last resort: unversioned /usr/bin/php
	if len(versions) == 0 && FileExists("/usr/bin/php") {
		if v, err := DetectPHPVersionFromBinary("/usr/bin/php"); err == nil && v != "" {
			versions = append(versions, v)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return ComparePHPVersions(versions[i], versions[j]) < 0
	})

	return versions, nil
}

// DetectPHPVersionFromBinary runs `php -r` to get major.minor
func DetectPHPVersionFromBinary(binary string) (string, error) {
	out, err := exec.Command(binary, "-r", `echo PHP_MAJOR_VERSION.".".PHP_MINOR_VERSION;`).Output()
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(out))
	if !regexp.MustCompile(`^\d+\.\d+$`).MatchString(v) {
		return "", fmt.Errorf("unexpected PHP version output: %q", v)
	}
	return v, nil
}

// ComparePHPVersions returns <0 if a<b, 0 if equal, >0 if a>b
func ComparePHPVersions(a, b string) int {
	amaj, amin := splitVersion(a)
	bmaj, bmin := splitVersion(b)
	if amaj != bmaj {
		return amaj - bmaj
	}
	return amin - bmin
}

func splitVersion(v string) (int, int) {
	parts := strings.SplitN(v, ".", 2)
	major, _ := strconv.Atoi(parts[0])
	minor := 0
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor
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
		return fmt.Errorf("invalid PHP version format. Use: 8.5, 8.2, 7.4, etc.")
	}

	if !CheckPHPInstalled(version) {
		return fmt.Errorf("PHP %s is not installed (expected %s or %s).\nInstall with: herd php install %s",
			version, PHPBinaryPath(version), PHPFPMBinaryPath(version), version)
	}

	return nil
}

// AptPackageAvailable reports whether apt knows about a package
func AptPackageAvailable(pkg string) bool {
	cmd := exec.Command("apt-cache", "show", pkg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Package: "+pkg) || strings.HasPrefix(strings.TrimSpace(string(out)), "Package:")
}

// InstallPHPHint returns a distro-friendly apt install hint for multi-PHP
func InstallPHPHint() string {
	return strings.TrimSpace(`
sudo apt install php-fpm nginx dnsmasq
  # Default Ubuntu archive may only ship one PHP (e.g. 8.5).
  # Install more versions when packaged:
  #   sudo apt install php8.1-fpm php8.2-fpm php8.3-fpm php8.4-fpm
  # Or: herd php install 8.1 8.2 8.3
  # On older Ubuntu, enable Ondřej Surý's PPA for parallel PHP versions:
  #   sudo add-apt-repository ppa:ondrej/php && sudo apt update
`)
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
	return string(output), err
}
