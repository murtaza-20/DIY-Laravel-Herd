package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/docker"
)

var extrasCmd = &cobra.Command{
	Use:   "extras",
	Short: "Manage Mailhog, phpMyAdmin, and MinIO",
	Long: `Start helper containers for local development:

  mailhog     SMTP catcher — UI http://127.0.0.1:8025  SMTP :1025
  phpmyadmin  MySQL UI     — http://127.0.0.1:8081
  minio       S3 API       — http://127.0.0.1:9000  Console :9001

Requires Docker. Install: sudo ./scripts/p10-install-docker.sh`,
}

var extrasStartCmd = &cobra.Command{
	Use:   "start [names...]",
	Short: "Start extras (default: all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := dockerManager()
		if err != nil {
			return err
		}
		kinds := normalizeExtraArgs(args)
		if err := m.StartExtras(kinds); err != nil {
			return err
		}
		for _, kind := range kinds {
			fmt.Printf("✓ %-12s  %s\n", kind, docker.ConnectionHint(kind))
		}
		return nil
	},
}

var extrasStopCmd = &cobra.Command{
	Use:   "stop [names...]",
	Short: "Stop extras (default: all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := dockerManager()
		if err != nil {
			return err
		}
		kinds := normalizeExtraArgs(args)
		_ = m.StopExtras(kinds)
		fmt.Printf("✓ stopped: %s\n", strings.Join(kinds, ", "))
		return nil
	},
}

var extrasStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show extras container status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !docker.Available() {
			fmt.Println("Docker/Podman: not installed")
			fmt.Println("  sudo ./scripts/p10-install-docker.sh")
			return nil
		}
		m, err := dockerManager()
		if err != nil {
			return err
		}
		fmt.Println("Extras")
		fmt.Println()
		for _, kind := range docker.ExtraKinds {
			running, detail, _ := m.Status(kind)
			mark := "○"
			if running {
				mark = "●"
			}
			fmt.Printf("  %s %-12s  %-12s  %s\n", mark, kind, detail, docker.ConnectionHint(kind))
		}
		fmt.Println()
		return nil
	},
}

func normalizeExtraArgs(args []string) []string {
	if len(args) == 0 {
		return append([]string{}, docker.ExtraKinds...)
	}
	var kinds []string
	for _, a := range args {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "all" {
			return append([]string{}, docker.ExtraKinds...)
		}
		allowed := false
		for _, k := range docker.ExtraKinds {
			if k == a {
				allowed = true
				break
			}
		}
		if allowed {
			kinds = append(kinds, a)
		}
	}
	if len(kinds) == 0 {
		return append([]string{}, docker.ExtraKinds...)
	}
	return kinds
}

func init() {
	extrasCmd.AddCommand(extrasStartCmd, extrasStopCmd, extrasStatusCmd)
	rootCmd.AddCommand(extrasCmd)
}
