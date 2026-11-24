package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/php"
)

var phpCmd = &cobra.Command{
	Use:   "php",
	Short: "Run PHP CLI for the current Herd site (or manage versions)",
	Long: `Uses the PHP version pinned for the Herd site that owns your current
directory — the same version Nginx/FPM uses for that .test site.

Examples:
  cd ~/Code/myapp
  herd php -v
  herd php artisan migrate
  herd php vendor/bin/phpunit

  herd php list
  herd php install 8.2
  herd php which
  herd php --site=myapp -v`,
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	RunE:               runPHPCommand,
}

func runPHPCommand(cmd *cobra.Command, args []string) error {
	args, siteName := php.ParseSiteFlag(args)

	if len(args) == 0 {
		return printPHPWhich(siteName)
	}

	switch args[0] {
	case "list":
		return runPHPList()
	case "install":
		return runPHPInstall(args[1:])
	case "which":
		return printPHPWhich(siteName)
	case "help", "--help", "-h":
		return cmd.Help()
	}

	err := php.ExecSitePHP(siteName, args)
	if code := php.ExitCode(err); code >= 0 {
		os.Exit(code)
	}
	return err
}

func printPHPWhich(siteName string) error {
	site, bin, err := php.ResolveSitePHP(siteName)
	if err != nil {
		return err
	}
	fmt.Printf("Site:    %s\n", site.Name)
	fmt.Printf("PHP:     %s\n", site.PHPVersion)
	fmt.Printf("Binary:  %s\n", bin)
	fmt.Printf("FPM:     %s\n", php.SocketPath(site.PHPVersion))
	return nil
}

func runPHPList() error {
	infos, err := php.ListVersionInfo()
	if err != nil {
		return err
	}

	fmt.Println("PHP versions (multi-version support)")
	fmt.Println()
	fmt.Printf("%-8s %-10s %-14s %s\n", "Version", "Installed", "Apt package", "Status")
	fmt.Println(strings.Repeat("-", 72))

	installedCount := 0
	aptCount := 0
	for _, info := range infos {
		inst := "no"
		if info.Installed {
			inst = "yes"
			installedCount++
		}
		apt := "missing"
		if info.AptAvailable {
			apt = "available"
			aptCount++
		}
		fmt.Printf("%-8s %-10s %-14s %s\n", info.Version, inst, info.AptPackage, apt)
	}

	fmt.Println()
	fmt.Printf("Installed: %d  |  Installable from apt now: %d\n", installedCount, aptCount)
	fmt.Println()
	if installedCount <= 1 || aptCount <= 1 {
		fmt.Println(php.MultiPHPHint())
		fmt.Println()
	}
	fmt.Println("Per-site CLI (matches FPM):")
	fmt.Println("  cd ~/code/myapp && herd php -v")
	fmt.Println("  herd composer install")
	fmt.Println("  herd serve myapp ~/code/myapp --php=8.2")
	return nil
}

func runPHPInstall(versions []string) error {
	if len(versions) == 0 {
		return fmt.Errorf("usage: herd php install <version...>  (e.g. herd php install 8.1 8.2)")
	}
	if err := php.InstallVersions(versions); err != nil {
		return err
	}

	fmt.Println("\n✓ Packages installed. Configuring Herd pools for new versions...")
	if !config.IsInitialized() {
		fmt.Println("Run: herd init")
		return nil
	}

	herdDir, err := config.GetHerdDir()
	if err != nil {
		return err
	}
	mgr := php.NewPHPManager(herdDir, nil)
	if err := mgr.Initialize(nil); err != nil {
		return fmt.Errorf("installed packages, but herd PHP setup failed: %w\nRun: herd init", err)
	}

	fmt.Println("✓ Done. Check with: herd php list && herd status")
	return nil
}

func init() {
	rootCmd.AddCommand(phpCmd)
}
