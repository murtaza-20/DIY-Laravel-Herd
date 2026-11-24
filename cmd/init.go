package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/php"
	"github.com/thesmartyking/herd/pkg/ssl"
	"github.com/thesmartyking/herd/pkg/system"
	"github.com/thesmartyking/herd/pkg/utils"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Herd environment",
	Long: `Initialize Herd by creating the necessary directory structure and configuring services.

This will:
1. Create ~/.herd directory structure
2. Detect PHP-FPM versions (system sockets under /run/php/)
3. Generate local TLS certs for *.test (mkcert or openssl)
4. Write Nginx site helpers under ~/.herd/nginx
5. Integrate system Nginx + Dnsmasq + systemd-resolved (sudo)

Requires: sudo (for nginx/dnsmasq/resolved drop-ins); openssl; mkcert recommended`,
	RunE: initHerd,
}

func initHerd(cmd *cobra.Command, args []string) error {
	fmt.Println("Initializing Herd environment...")
	fmt.Println()

	fmt.Println("[1/6] Creating directory structure...")
	herdDir, err := config.EnsureHerdDir()
	if err != nil {
		return fmt.Errorf("failed to create herd directory: %w", err)
	}
	fmt.Printf("✓ Created %s\n", herdDir)

	fmt.Println("\n[2/6] Detecting available PHP versions...")
	versions, err := utils.GetAvailablePHPVersions()
	if err != nil {
		return fmt.Errorf("failed to detect PHP versions: %w", err)
	}
	if len(versions) == 0 {
		return fmt.Errorf("no PHP versions found. Install PHP first:\n  %s", utils.InstallPHPHint())
	}
	fmt.Printf("✓ Found PHP versions: %v\n", versions)

	fmt.Println("\n[3/6] Mapping PHP-FPM system sockets...")
	phpMgr := php.NewPHPManager(herdDir, nil)
	if err := phpMgr.Initialize(versions); err != nil {
		return fmt.Errorf("failed to initialize PHP: %w", err)
	}
	for _, v := range versions {
		fmt.Printf("✓ PHP %s → %s\n", v, php.SocketPath(v))
	}

	fmt.Println("\n[4/6] Ensuring local TLS certificates (*.test + per-site hosts)...")
	if !ssl.MkcertAvailable() {
		fmt.Println("⚠️  mkcert not found — will use openssl self-signed (browser warnings).")
		fmt.Println("   Recommended: sudo apt install -y mkcert libnss3-tools")
	} else {
		_ = exec.Command("mkcert", "-install").Run()
	}
	certPath, keyPath, method, err := ssl.EnsureCertificates(herdDir, config.SiteNames())
	if err != nil {
		return fmt.Errorf("TLS setup failed: %w", err)
	}
	fmt.Printf("✓ TLS cert (%s): %s\n", method, certPath)
	_ = keyPath

	fmt.Println("\n[5/6] Writing Nginx helpers under ~/.herd/nginx...")
	if err := setupNginx(herdDir); err != nil {
		return fmt.Errorf("failed to setup nginx helpers: %w", err)
	}
	fmt.Printf("✓ Nginx dir: %s\n", filepath.Join(herdDir, "nginx"))

	fmt.Println("\n[6/6] Applying system integration (nginx include, dnsmasq, resolved)...")
	if err := system.ApplyIntegration(herdDir); err != nil {
		fmt.Println("⚠️  Could not apply system config automatically (sudo required).")
		fmt.Println()
		fmt.Println(system.ManualInstructions(herdDir))
	} else {
		fmt.Println("✓ Nginx include + Dnsmasq (.test) + systemd-resolved wired")
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✓ Herd initialized successfully!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("\nNext steps:")
	fmt.Println("  1. herd serve myproject ~/path/to/myproject")
	fmt.Println("  2. Open https://myproject.test")
	if method == "openssl" {
		fmt.Println("  3. Trust the cert or install mkcert, then re-run: herd init")
	}
	fmt.Printf("\nAvailable PHP versions: %v\n\n", versions)
	return nil
}

func setupNginx(herdDir string) error {
	nginxDir := filepath.Join(herdDir, "nginx")
	if err := os.MkdirAll(filepath.Join(nginxDir, "sites-enabled"), 0755); err != nil {
		return err
	}
	// Keep a small note file; system nginx loads sites via conf.d/herd.conf
	readme := filepath.Join(nginxDir, "README")
	return os.WriteFile(readme, []byte("Site vhosts live in sites-enabled/*.conf and are included by /etc/nginx/conf.d/herd.conf\n"), 0644)
}
