package php

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/thesmartyking/herd/pkg/utils"
)

// VersionInfo describes one PHP version relative to this machine
type VersionInfo struct {
	Version      string
	Installed    bool
	CLIPath      string
	FPMPath      string
	AptPackage   string
	AptAvailable bool
}

// ListVersionInfo returns status for known + currently installed PHP versions
func ListVersionInfo() ([]VersionInfo, error) {
	installed, err := utils.GetAvailablePHPVersions()
	if err != nil {
		return nil, err
	}
	installedSet := map[string]bool{}
	for _, v := range installed {
		installedSet[v] = true
	}

	seen := map[string]struct{}{}
	var order []string
	for _, v := range utils.KnownPHPVersions {
		seen[v] = struct{}{}
		order = append(order, v)
	}
	for _, v := range installed {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			order = append(order, v)
		}
	}

	out := make([]VersionInfo, 0, len(order))
	for _, v := range order {
		out = append(out, VersionInfo{
			Version:      v,
			Installed:    installedSet[v],
			CLIPath:      utils.PHPBinaryPath(v),
			FPMPath:      utils.PHPFPMBinaryPath(v),
			AptPackage:   utils.AptFPMPackage(v),
			AptAvailable: utils.AptPackageAvailable(utils.AptFPMPackage(v)),
		})
	}
	return out, nil
}

// InstallVersions installs phpX.Y-fpm / phpX.Y-cli packages via apt
func InstallVersions(versions []string) error {
	if len(versions) == 0 {
		return fmt.Errorf("no versions specified")
	}

	var pkgs []string
	var missing []string
	for _, v := range versions {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		pkg := utils.AptFPMPackage(v)
		if !utils.AptPackageAvailable(pkg) {
			missing = append(missing, fmt.Sprintf("%s (%s)", v, pkg))
			continue
		}
		pkgs = append(pkgs, pkg)
		cli := fmt.Sprintf("php%s-cli", v)
		if utils.AptPackageAvailable(cli) {
			pkgs = append(pkgs, cli)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("not available in apt for this Ubuntu release: %s\n\n%s\n\nTip: herd php list",
			strings.Join(missing, ", "),
			MultiPHPHint())
	}

	if len(pkgs) == 0 {
		return fmt.Errorf("nothing to install")
	}

	fmt.Printf("Installing: %s\n", strings.Join(pkgs, " "))
	args := append([]string{"apt-get", "install", "-y"}, pkgs...)
	cmd := exec.Command("sudo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apt install failed: %w\nTry manually:\n  sudo apt-get install -y %s", err, strings.Join(pkgs, " "))
	}
	return nil
}

// MultiPHPHint explains how to get parallel PHP versions on Ubuntu
func MultiPHPHint() string {
	return strings.TrimSpace(`
Ubuntu's default archive often ships only one PHP (e.g. 8.5 on Ubuntu 26.04).
Herd already supports many versions at once once they are installed
(/usr/bin/php8.1, php8.2, ... each with its own FPM socket).

To get older/parallel versions:
  1) Prefer packages from your distro when present:
       sudo apt install php8.1-fpm php8.2-fpm php8.3-fpm php8.4-fpm
  2) On Ubuntu releases supported by Ondřej Surý's PPA:
       sudo add-apt-repository ppa:ondrej/php
       sudo apt update
       sudo apt install php8.1-fpm php8.2-fpm php8.3-fpm php8.4-fpm
  3) Re-run: herd init && herd php list
  4) Smoke-test all versions: ./scripts/test-multi-php.sh`)
}
