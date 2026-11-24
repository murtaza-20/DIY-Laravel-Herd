package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/nginx"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop all Herd services",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Stopping Herd-related services...")
		_ = exec.Command("sudo", "-n", "systemctl", "stop", "nginx").Run()
		fmt.Println("✓ Stop signal sent (nginx). PHP-FPM pools remain under system control.")
		fmt.Println("  Tip: sudo systemctl stop php8.5-fpm nginx   # or: php-fpm")
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start all Herd services",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Starting Herd-related services...")
		_ = exec.Command("sudo", "-n", "systemctl", "start", "nginx").Run()
		fmt.Println("✓ Start signal sent (nginx). Ensure PHP-FPM is running for your versions.")
		fmt.Println("  Tip: sudo systemctl start php8.5-fpm nginx   # or: php-fpm")
		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart [service]",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		fmt.Printf("Restarting %s...\n", service)
		if err := exec.Command("sudo", "-n", "systemctl", "restart", service).Run(); err != nil {
			fmt.Printf("⚠️  Could not restart via systemctl. Try: sudo systemctl restart %s\n", service)
			return nil
		}
		fmt.Printf("✓ %s restarted\n", service)
		return nil
	},
}

var forgetCmd = &cobra.Command{
	Use:   "forget [name]",
	Short: "Remove a registered project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		fmt.Printf("Removing project: %s\n", projectName)

		if !config.IsInitialized() {
			return fmt.Errorf("herd not initialized. Run 'herd init' first")
		}

		if err := config.UnregisterSite(projectName); err != nil {
			return fmt.Errorf("failed to remove site: %w", err)
		}

		herdDir, _ := config.GetHerdDir()
		configGen := nginx.NewConfigGenerator(filepath.Join(herdDir, "nginx", "sites-enabled"))
		if err := configGen.RemoveConfig(projectName); err != nil {
			fmt.Printf("⚠️  Could not remove nginx config: %v\n", err)
		}
		_ = exec.Command("sudo", "systemctl", "reload", "nginx").Run()

		fmt.Println("✓ Project removed")
		return nil
	},
}
