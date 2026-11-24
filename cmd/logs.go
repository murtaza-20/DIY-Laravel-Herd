package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "View logs for a service",
	Long: `View logs for a service (nginx, php-8.0, php-8.1, etc.)

Examples:
  herd logs nginx
  herd logs php-8.5`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		if !config.IsInitialized() {
			return fmt.Errorf("herd not initialized. Run 'herd init' first")
		}

		herdDir, err := config.GetHerdDir()
		if err != nil {
			return err
		}

		logPath := filepath.Join(herdDir, "logs", service+".log")

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			return fmt.Errorf("log file not found for service '%s'. Checked: %s", service, logPath)
		}

		content, err := os.ReadFile(logPath)
		if err != nil {
			return fmt.Errorf("failed to read logs: %w", err)
		}

		if len(content) == 0 {
			fmt.Printf("(no logs for %s yet)\n", service)
		} else {
			fmt.Printf("Logs for %s:\n\n%s\n", service, string(content))
		}

		return nil
	},
}
