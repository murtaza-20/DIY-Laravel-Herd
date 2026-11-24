package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/dashboard"
)

var (
	dashboardPort  int
	dashboardNoOpen bool
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start the Herd web dashboard",
	Long: `Start the local Herd dashboard on http://127.0.0.1:7835

Shows registered sites, PHP versions, service status, and log tails.
Press Ctrl+C to stop.`,
	RunE: runDashboard,
}

func runDashboard(cmd *cobra.Command, args []string) error {
	if !config.IsInitialized() {
		return fmt.Errorf("herd not initialized. Run 'herd init' first")
	}

	port := dashboardPort
	if port == 0 {
		port = dashboard.DefaultPort
	}

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Printf("Starting Herd dashboard at %s\n", url)
	fmt.Println("Press Ctrl+C to stop.")

	if !dashboardNoOpen {
		go func() {
			time.Sleep(400 * time.Millisecond)
			_ = openBrowser(url)
		}()
	}

	srv := dashboard.New(port)
	return srv.ListenAndServe()
}

func openBrowser(url string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		c = exec.Command("xdg-open", url)
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return c.Start()
}

func init() {
	dashboardCmd.Flags().IntVar(&dashboardPort, "port", dashboard.DefaultPort, "Dashboard port")
	dashboardCmd.Flags().BoolVar(&dashboardNoOpen, "no-open", false, "Do not open a browser")
	rootCmd.AddCommand(dashboardCmd)
}
