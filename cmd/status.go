package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of Herd services and sites",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.IsInitialized() {
			return fmt.Errorf("herd not initialized. Run 'herd init' first")
		}

		herdDir, err := config.GetHerdDir()
		if err != nil {
			return fmt.Errorf("herd not initialized. Run 'herd init' first")
		}

		fmt.Println("Herd Status")
		fmt.Println()

		fmt.Println("PHP Versions:")
		versions, _ := utils.GetAvailablePHPVersions()
		if len(versions) == 0 {
			fmt.Println("  (none installed)")
		} else {
			for _, v := range versions {
				fmt.Printf("  ✓ PHP %s\n", v)
			}
		}

		fmt.Println("\nRegistered Sites:")
		sites, _ := config.LoadSites()
		if len(sites) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, site := range sites {
				fmt.Printf("  • %s (PHP %s) -> https://%s.test\n", site.Name, site.PHPVersion, site.Name)
			}
		}

		fmt.Printf("\nHerd directory: %s\n", herdDir)
		fmt.Println("")
		return nil
	},
}
