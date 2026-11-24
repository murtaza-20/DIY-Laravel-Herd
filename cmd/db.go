package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/db"
	"github.com/thesmartyking/herd/pkg/docker"
)

var (
	dbType   string
	dbOut    string
	dbAll    bool
	dbMySQL  bool
	dbPG     bool
	dbRedis  bool
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage Docker databases (MySQL, Postgres, Redis)",
	Long: `Start/stop Herd database containers and create/backup/restore databases.

Defaults (local only):
  MySQL     root:root@127.0.0.1:3306
  Postgres  postgres:postgres@127.0.0.1:5432
  Redis     127.0.0.1:6379

Requires Docker (or Podman). Install: sudo ./scripts/p10-install-docker.sh`,
}

var dbStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start database containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := dockerManager()
		if err != nil {
			return err
		}
		kinds := selectedKinds()
		for _, kind := range kinds {
			fmt.Printf("→ %s\n", kind)
			if err := m.Start(kind); err != nil {
				return err
			}
			fmt.Printf("  ✓ %s  %s\n", docker.Catalog[kind].Name, docker.ConnectionHint(kind))
		}
		return nil
	},
}

var dbStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop database containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := dockerManager()
		if err != nil {
			return err
		}
		for _, kind := range selectedKinds() {
			_ = m.Stop(kind)
			fmt.Printf("✓ stopped %s\n", kind)
		}
		return nil
	},
}

var dbStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show database container status",
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
		fmt.Println("Database services")
		fmt.Println()
		for _, kind := range []string{"mysql", "postgres", "redis"} {
			running, detail, _ := m.Status(kind)
			mark := "○"
			if running {
				mark = "●"
			}
			fmt.Printf("  %s %-10s  %-12s  %s\n", mark, kind, detail, docker.ConnectionHint(kind))
		}
		fmt.Println()
		return nil
	},
}

var dbCreateCmd = &cobra.Command{
	Use:     "create [name]",
	Aliases: []string{"db:create"},
	Short:   "Create a database (mysql|postgres)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, err := requireSQLType()
		if err != nil {
			return err
		}
		client, err := dbClient()
		if err != nil {
			return err
		}
		name := args[0]
		if err := client.Create(kind, name); err != nil {
			return err
		}
		fmt.Printf("✓ Created %s database %q\n", kind, name)
		fmt.Printf("  %s/%s\n", docker.ConnectionHint(kind), name)
		return nil
	},
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List databases",
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, err := requireSQLType()
		if err != nil {
			return err
		}
		client, err := dbClient()
		if err != nil {
			return err
		}
		names, err := client.List(kind)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Println("(none)")
			return nil
		}
		for _, n := range names {
			fmt.Println(n)
		}
		return nil
	},
}

var dbBackupCmd = &cobra.Command{
	Use:     "backup [name]",
	Aliases: []string{"db:backup"},
	Short:   "Backup a database to SQL file",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, err := requireSQLType()
		if err != nil {
			return err
		}
		client, err := dbClient()
		if err != nil {
			return err
		}
		path, err := client.Backup(kind, args[0], dbOut)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Backup written to %s\n", path)
		return nil
	},
}

var dbRestoreCmd = &cobra.Command{
	Use:     "restore [name] [dump.sql]",
	Aliases: []string{"db:restore"},
	Short:   "Restore a database from SQL dump",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, err := requireSQLType()
		if err != nil {
			return err
		}
		client, err := dbClient()
		if err != nil {
			return err
		}
		if err := client.Restore(kind, args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("✓ Restored %s database %q from %s\n", kind, args[0], args[1])
		return nil
	},
}

func dockerManager() (*docker.Manager, error) {
	if !config.IsInitialized() {
		return nil, fmt.Errorf("herd not initialized. Run 'herd init' first")
	}
	herdDir, err := config.GetHerdDir()
	if err != nil {
		return nil, err
	}
	return docker.NewManager(herdDir)
}

func dbClient() (*db.Client, error) {
	if !config.IsInitialized() {
		return nil, fmt.Errorf("herd not initialized. Run 'herd init' first")
	}
	herdDir, err := config.GetHerdDir()
	if err != nil {
		return nil, err
	}
	return db.NewClient(herdDir)
}

func selectedKinds() []string {
	var kinds []string
	if dbMySQL {
		kinds = append(kinds, "mysql")
	}
	if dbPG {
		kinds = append(kinds, "postgres")
	}
	if dbRedis {
		kinds = append(kinds, "redis")
	}
	if len(kinds) == 0 || dbAll {
		return []string{"mysql", "postgres", "redis"}
	}
	return kinds
}

func requireSQLType() (string, error) {
	t := strings.ToLower(strings.TrimSpace(dbType))
	if t == "" {
		t = "mysql"
	}
	if t != "mysql" && t != "postgres" {
		return "", fmt.Errorf("--type must be mysql or postgres")
	}
	return t, nil
}

func init() {
	dbStartCmd.Flags().BoolVar(&dbAll, "all", false, "Start all DB services")
	dbStartCmd.Flags().BoolVar(&dbMySQL, "mysql", false, "Start MySQL")
	dbStartCmd.Flags().BoolVar(&dbPG, "postgres", false, "Start Postgres")
	dbStartCmd.Flags().BoolVar(&dbRedis, "redis", false, "Start Redis")
	dbStopCmd.Flags().BoolVar(&dbMySQL, "mysql", false, "Stop MySQL")
	dbStopCmd.Flags().BoolVar(&dbPG, "postgres", false, "Stop Postgres")
	dbStopCmd.Flags().BoolVar(&dbRedis, "redis", false, "Stop Redis")

	for _, c := range []*cobra.Command{dbCreateCmd, dbListCmd, dbBackupCmd, dbRestoreCmd} {
		c.Flags().StringVar(&dbType, "type", "mysql", "Database engine: mysql|postgres")
	}
	dbBackupCmd.Flags().StringVar(&dbOut, "out", "", "Output SQL file path")

	dbCmd.AddCommand(dbStartCmd, dbStopCmd, dbStatusCmd, dbCreateCmd, dbListCmd, dbBackupCmd, dbRestoreCmd)
	rootCmd.AddCommand(dbCmd)
}
