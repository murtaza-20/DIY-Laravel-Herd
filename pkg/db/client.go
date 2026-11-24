package db

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/thesmartyking/herd/pkg/docker"
)

var nameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// ValidateName checks a database name
func ValidateName(name string) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("invalid database name %q (use letters, numbers, underscore; start with a letter)", name)
	}
	return nil
}

// Client performs create/list/backup/restore against Herd DB containers
type Client struct {
	Docker *docker.Manager
}

func NewClient(herdDir string) (*Client, error) {
	m, err := docker.NewManager(herdDir)
	if err != nil {
		return nil, err
	}
	return &Client{Docker: m}, nil
}

// EnsureRunning starts the engine if needed and waits for readiness
func (c *Client) EnsureRunning(kind string) error {
	if err := c.Docker.Start(kind); err != nil {
		return err
	}
	return c.Docker.WaitReady(kind, 60*time.Second)
}

// Create creates a database
func (c *Client) Create(kind, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if err := c.EnsureRunning(kind); err != nil {
		return err
	}
	switch kind {
	case "mysql":
		_, err := c.Docker.Exec("mysql", "mysql", "-uroot", "-proot", "-e",
			fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", name))
		return err
	case "postgres":
		out, err := c.Docker.Exec("postgres", "psql", "-U", "postgres", "-tc",
			fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", name))
		if err != nil {
			return err
		}
		if strings.Contains(out, "1") {
			return nil
		}
		_, err = c.Docker.Exec("postgres", "psql", "-U", "postgres", "-c",
			fmt.Sprintf("CREATE DATABASE %s;", name))
		return err
	default:
		return fmt.Errorf("db create supports mysql|postgres, not %s", kind)
	}
}

// List returns database names
func (c *Client) List(kind string) ([]string, error) {
	if err := c.EnsureRunning(kind); err != nil {
		return nil, err
	}
	switch kind {
	case "mysql":
		out, err := c.Docker.Exec("mysql", "mysql", "-uroot", "-proot", "-N", "-e", "SHOW DATABASES;")
		if err != nil {
			return nil, err
		}
		skip := map[string]bool{"information_schema": true, "mysql": true, "performance_schema": true, "sys": true}
		var names []string
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || skip[line] {
				continue
			}
			names = append(names, line)
		}
		return names, nil
	case "postgres":
		out, err := c.Docker.Exec("postgres", "psql", "-U", "postgres", "-Atc",
			"SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY 1;")
		if err != nil {
			return nil, err
		}
		var names []string
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				names = append(names, line)
			}
		}
		return names, nil
	default:
		return nil, fmt.Errorf("db list supports mysql|postgres, not %s", kind)
	}
}

// Backup dumps a database to a file; returns the path written
func (c *Client) Backup(kind, name, outPath string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	if err := c.EnsureRunning(kind); err != nil {
		return "", err
	}
	if outPath == "" {
		ts := time.Now().Format("20060102-150405")
		outPath = filepath.Join(c.Docker.HerdDir, "data", "backups", fmt.Sprintf("%s-%s-%s.sql", kind, name, ts))
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return "", err
	}

	var out string
	var err error
	switch kind {
	case "mysql":
		out, err = c.Docker.Exec("mysql", "mysqldump", "-uroot", "-proot", "--single-transaction", "--routines", name)
	case "postgres":
		out, err = c.Docker.Exec("postgres", "pg_dump", "-U", "postgres", "--clean", "--if-exists", name)
	default:
		return "", fmt.Errorf("backup supports mysql|postgres")
	}
	if err != nil {
		return "", fmt.Errorf("dump failed: %w\n%s", err, out)
	}
	if err := os.WriteFile(outPath, []byte(out), 0644); err != nil {
		return "", err
	}
	return outPath, nil
}

// Restore loads a SQL dump into a database (creates DB if missing)
func (c *Client) Restore(kind, name, dumpPath string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	data, err := os.ReadFile(dumpPath)
	if err != nil {
		return err
	}
	if err := c.Create(kind, name); err != nil {
		return err
	}

	bin, err := docker.Engine()
	if err != nil {
		return err
	}
	svc := docker.Catalog[kind]
	var args []string
	switch kind {
	case "mysql":
		args = []string{"exec", "-i", svc.Name, "mysql", "-uroot", "-proot", name}
	case "postgres":
		args = []string{"exec", "-i", svc.Name, "psql", "-U", "postgres", "-d", name}
	default:
		return fmt.Errorf("restore supports mysql|postgres")
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore failed: %w\n%s", err, string(out))
	}
	return nil
}
