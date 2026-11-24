package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
)

var sitesCmd = &cobra.Command{
	Use:   "sites",
	Short: "List all registered sites",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.IsInitialized() {
			return fmt.Errorf("herd not initialized. Run 'herd init' first")
		}

		sites, err := config.LoadSites()
		if err != nil {
			return err
		}

		if len(sites) == 0 {
			fmt.Println("No sites registered yet. Run: herd serve <name> <path>")
			return nil
		}

		fmt.Println("Registered Sites:")
		fmt.Println()
		fmt.Println("Name                  PHP     Type       Path")
		fmt.Println(strings.Repeat("-", 72))

		for _, site := range sites {
			fmt.Printf("%-20s  %-5s  %-9s %s\n", site.Name, site.PHPVersion, site.ProjectType, site.Path)
		}

		fmt.Println("")
		return nil
	},
}
