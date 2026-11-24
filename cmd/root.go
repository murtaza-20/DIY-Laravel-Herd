package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "herd",
	Short: "Ubuntu Herd - Local PHP/Laravel development environment",
	Long: `Herd is a lightweight, all-in-one local development environment for PHP/Laravel on Ubuntu.

Start with 'herd init' to set up your environment, then 'herd serve' to register projects.

Example:
  herd init
  herd serve myproject ~/projects/myproject
  # Visit http://myproject.test in your browser`,
	Version: "0.1.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(forgetCmd)
	rootCmd.AddCommand(sitesCmd)
}
