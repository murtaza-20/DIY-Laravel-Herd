package php

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/utils"
)

// ResolveSitePHP returns the site for cwd (or siteName) and its CLI binary.
func ResolveSitePHP(siteName string) (*config.Site, string, error) {
	var site *config.Site
	var err error
	if siteName != "" {
		site, err = config.GetSite(siteName)
	} else {
		site, err = config.FindSiteForCwd()
	}
	if err != nil {
		return nil, "", err
	}

	bin := utils.PHPBinaryPath(site.PHPVersion)
	if !utils.FileExists(bin) {
		return site, "", fmt.Errorf("PHP %s CLI not found at %s\nInstall with: herd php install %s", site.PHPVersion, bin, site.PHPVersion)
	}
	return site, bin, nil
}

// ExecSitePHP runs the site-pinned PHP binary with the given args (stdio inherited).
func ExecSitePHP(siteName string, args []string) error {
	_, bin, err := ResolveSitePHP(siteName)
	if err != nil {
		return err
	}
	return execInherited(bin, args)
}

// FindComposerBinary locates composer on PATH or common install locations.
func FindComposerBinary() (string, error) {
	if p, err := exec.LookPath("composer"); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "composer"),
		"/usr/local/bin/composer",
		"/usr/bin/composer",
		"composer.phar",
	}
	for _, c := range candidates {
		if utils.FileExists(c) {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs, nil
			}
			return c, nil
		}
	}
	return "", fmt.Errorf("composer not found on PATH\nInstall: curl -sS https://getcomposer.org/installer | php -- --install-dir=$HOME/.local/bin --filename=composer")
}

// ExecSiteComposer runs composer via the site-pinned PHP CLI.
func ExecSiteComposer(siteName string, args []string) error {
	site, phpBin, err := ResolveSitePHP(siteName)
	if err != nil {
		return err
	}
	composerBin, err := FindComposerBinary()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "[herd] %s → PHP %s (%s)\n", site.Name, site.PHPVersion, phpBin)
	cmdArgs := append([]string{composerBin}, args...)
	return execInherited(phpBin, cmdArgs)
}

func execInherited(bin string, args []string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

// ExitCode extracts a process exit code from an exec error, or -1 if unknown.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return -1
}

// DescribeSitePHP returns a one-line summary for the current directory's PHP.
func DescribeSitePHP(siteName string) (string, error) {
	site, bin, err := ResolveSitePHP(siteName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s (site %s) → %s", site.PHPVersion, site.Name, bin), nil
}

// ParseSiteFlag strips a leading --site=name / --site name from args.
// Returns remaining args and the site name (empty if unset).
func ParseSiteFlag(args []string) (rest []string, site string) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case strings.HasPrefix(a, "--site="):
			site = strings.TrimPrefix(a, "--site=")
		case a == "--site" && i+1 < len(args):
			i++
			site = args[i]
		default:
			rest = append(rest, a)
		}
	}
	return rest, site
}
