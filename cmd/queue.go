package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/queue"
)

var (
	queueName    string
	queueTimeout int
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage Laravel queue workers",
	Long: `Start and stop background Laravel queue workers for registered sites.

Example:
  herd queue work myapp
  herd queue work myapp --queue=emails --timeout=90
  herd queue list
  herd queue stop myapp
  herd queue stop myapp --queue=emails`,
}

var queueWorkCmd = &cobra.Command{
	Use:     "work [site]",
	Aliases: []string{"queue:work"},
	Short:   "Start a queue worker for a registered site",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.IsInitialized() {
			return fmt.Errorf("herd not initialized. Run 'herd init' first")
		}
		herdDir, err := config.GetHerdDir()
		if err != nil {
			return err
		}
		mgr := queue.NewManager(herdDir)
		q := queueName
		if q == "" {
			q = "default"
		}
		w, err := mgr.Start(args[0], q, queueTimeout)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Queue worker started\n")
		fmt.Printf("  Site:    %s\n", w.Site)
		fmt.Printf("  Queue:   %s\n", w.Queue)
		fmt.Printf("  PHP:     %s\n", w.PHPVersion)
		fmt.Printf("  PID:     %d\n", w.Pid)
		fmt.Printf("  Log:     %s\n", w.LogFile)
		fmt.Printf("  Stop with: herd queue stop %s --queue=%s\n", w.Site, w.Queue)
		return nil
	},
}

var queueListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"queue:list"},
	Short:   "List running queue workers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.IsInitialized() {
			return fmt.Errorf("herd not initialized. Run 'herd init' first")
		}
		herdDir, err := config.GetHerdDir()
		if err != nil {
			return err
		}
		mgr := queue.NewManager(herdDir)
		workers, err := mgr.List()
		if err != nil {
			return err
		}
		if len(workers) == 0 {
			fmt.Println("No queue workers running.")
			return nil
		}
		fmt.Printf("%-16s %-12s %-6s %-8s %s\n", "SITE", "QUEUE", "PHP", "PID", "STARTED")
		for _, w := range workers {
			fmt.Printf("%-16s %-12s %-6s %-8d %s\n",
				w.Site, w.Queue, w.PHPVersion, w.Pid, w.StartedAt.Format(time.RFC822))
		}
		return nil
	},
}

var queueStopCmd = &cobra.Command{
	Use:   "stop [site]",
	Short: "Stop queue worker(s) for a site",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !config.IsInitialized() {
			return fmt.Errorf("herd not initialized. Run 'herd init' first")
		}
		herdDir, err := config.GetHerdDir()
		if err != nil {
			return err
		}
		mgr := queue.NewManager(herdDir)
		if err := mgr.Stop(args[0], queueName); err != nil {
			return err
		}
		if queueName == "" {
			fmt.Printf("✓ Stopped all queue workers for %s\n", args[0])
		} else {
			fmt.Printf("✓ Stopped queue worker %s/%s\n", args[0], queueName)
		}
		return nil
	},
}

func init() {
	queueWorkCmd.Flags().StringVar(&queueName, "queue", "default", "Queue name")
	queueWorkCmd.Flags().IntVar(&queueTimeout, "timeout", 60, "Job timeout seconds")
	queueStopCmd.Flags().StringVar(&queueName, "queue", "", "Queue name (empty = all for site)")

	queueCmd.AddCommand(queueWorkCmd, queueListCmd, queueStopCmd)
	rootCmd.AddCommand(queueCmd)
}
