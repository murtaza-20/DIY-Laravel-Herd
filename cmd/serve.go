package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/composer"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/nginx"
	"github.com/thesmartyking/herd/pkg/php"
	"github.com/thesmartyking/herd/pkg/ssl"
	"github.com/thesmartyking/herd/pkg/utils"
)

var (
	phpVersion string
)

var serveCmd = &cobra.Command{
	Use:   "serve [name] [path]",
	Short: "Register and serve a project",
	Long: `Register and serve a Laravel/PHP project at https://name.test

The PHP version is auto-detected from composer.json. Override with --php flag.

Examples:
  herd serve myapp ~/projects/myapp
  herd serve legacy ~/projects/legacy --php=7.4
  herd serve api ~/projects/api --php=8.3`,
	Args: cobra.ExactArgs(2),
	RunE: serveProject,
}

func serveProject(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	projectPath := args[1]

	fmt.Printf("Registering project: %s\n", projectName)
	fmt.Printf("Path: %s\n", projectPath)

	if err := utils.ValidateProjectName(projectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	if !utils.DirExists(projectPath) {
		return fmt.Errorf("project path does not exist: %s", projectPath)
	}

	absPath, _ := filepath.Abs(projectPath)

	if !config.IsInitialized() {
		return fmt.Errorf("herd not initialized. Run 'herd init' first")
	}

	herdDir, err := config.GetHerdDir()
	if err != nil {
		return fmt.Errorf("herd not initialized. Run 'herd init' first")
	}

	var detectedPHPVersion string

	if phpVersion != "" {
		detectedPHPVersion = phpVersion
		fmt.Printf("[php] Using PHP %s (from --php flag)\n", phpVersion)
		if err := utils.ValidatePHPVersion(phpVersion); err != nil {
			return err
		}
	} else {
		fmt.Println("[composer] Reading composer.json...")
		requirement, err := composer.ExtractPHPRequirement(absPath)

		if err != nil || requirement == nil {
			fmt.Println("[warn] No PHP requirement found, using latest available PHP")
			versions, _ := utils.GetAvailablePHPVersions()
			if len(versions) == 0 {
				return fmt.Errorf("no PHP versions available")
			}
			detectedPHPVersion = versions[len(versions)-1]
		} else {
			fmt.Printf("[composer] Found requirement: %s\n", requirement.RawVersion)

			phpMgr := php.NewPHPManager(herdDir, nil)
			_ = phpMgr.Initialize([]string{})
			compatible, err := phpMgr.FindCompatibleVersion(requirement)
			if err != nil {
				return fmt.Errorf("no compatible PHP version found for %s: %w", requirement.RawVersion, err)
			}
			detectedPHPVersion = compatible
		}

		fmt.Printf("✓ Using PHP %s\n", detectedPHPVersion)
	}

	sock := php.SocketPath(detectedPHPVersion)
	if !utils.FileExists(sock) {
		return fmt.Errorf("PHP-FPM socket missing: %s\nStart it with: sudo systemctl start php%s-fpm", sock, detectedPHPVersion)
	}

	projectType := detectProjectType(absPath)
	fmt.Printf("✓ Project type: %s\n", projectType)

	certPath, keyPath, method, err := ssl.EnsureCertificates(herdDir, append(config.SiteNames(), projectName))
	if err != nil {
		return fmt.Errorf("TLS certificates required: %w\nRun: herd init", err)
	}
	fmt.Printf("✓ TLS (%s)\n", method)
	if method == "openssl" {
		fmt.Println("⚠️  Self-signed cert — Chrome/Flutter web will fail with ERR_CERT_AUTHORITY_INVALID.")
		fmt.Println("   sudo apt install -y mkcert libnss3-tools && mkcert -install")
		fmt.Println("   then re-run this command and fully quit Chrome.")
	}

	// Allow www-data to traverse into the project and write Laravel caches
	_ = os.Chmod(absPath, 0755)
	if projectType == "laravel" {
		if err := utils.EnsureWebWritable(absPath); err != nil {
			fmt.Printf("⚠️  could not open storage for www-data: %v\n", err)
		} else {
			fmt.Println("✓ storage/bootstrap writable by www-data")
		}
	}

	fmt.Println("[nginx] Generating Nginx configuration...")
	ngixDir := filepath.Join(herdDir, "nginx", "sites-enabled")
	configGen := nginx.NewConfigGenerator(ngixDir)
	configGen.SetTLS(certPath, keyPath)

	site := nginx.Site{
		Name:       projectName,
		Path:       absPath,
		Type:       projectType,
		PHPVersion: detectedPHPVersion,
	}

	if err := configGen.WriteConfig(site); err != nil {
		return fmt.Errorf("failed to generate nginx config: %w", err)
	}
	fmt.Println("✓ Nginx config generated")

	fmt.Println("[sites] Registering site...")
	// Allow re-serve: forget existing registration quietly
	_ = config.UnregisterSite(projectName)
	if err := config.RegisterSite(projectName, absPath, detectedPHPVersion, projectType); err != nil {
		return fmt.Errorf("failed to register site: %w", err)
	}
	fmt.Println("✓ Site registered")

	fmt.Println("[nginx] Testing and reloading Nginx...")
	if out, err := exec.Command("sudo", "nginx", "-t").CombinedOutput(); err != nil {
		fmt.Printf("⚠️  nginx -t failed: %s\n", string(out))
		fmt.Println("  Ensure system integration is applied: sudo ./scripts/p8-system-setup.sh")
	} else if err := exec.Command("sudo", "systemctl", "reload", "nginx").Run(); err != nil {
		fmt.Println("⚠️  Could not reload nginx. Run: sudo systemctl reload nginx")
	} else {
		fmt.Println("✓ Nginx reloaded")
	}

	fmt.Println("\nSuccess!")
	fmt.Printf("   Project:  %s\n", projectName)
	fmt.Printf("   PHP:      %s (%s)\n", detectedPHPVersion, sock)
	fmt.Printf("   URL:      https://%s.test\n", projectName)
	fmt.Println("")

	return nil
}

func detectProjectType(path string) string {
	if utils.FileExists(filepath.Join(path, "artisan")) {
		return "laravel"
	}
	if utils.FileExists(filepath.Join(path, "composer.json")) {
		return "php"
	}
	if utils.FileExists(filepath.Join(path, "package.json")) {
		return "node"
	}
	if utils.FileExists(filepath.Join(path, "index.html")) {
		return "static"
	}
	return "unknown"
}

func init() {
	serveCmd.Flags().StringVar(&phpVersion, "php", "", "Explicit PHP version (e.g., 8.5)")
}
