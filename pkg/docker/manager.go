package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const NetworkName = "herd"

// Engine is the container CLI (docker or podman)
func Engine() (string, error) {
	if path, err := exec.LookPath("docker"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("podman"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("neither docker nor podman found. Install with:\n  sudo apt install docker.io\n  sudo usermod -aG docker $USER\n  # then log out/in, or: newgrp docker")
}

// Available reports whether a container engine is present
func Available() bool {
	_, err := Engine()
	return err == nil
}

// Service describes a Herd-managed container
type Service struct {
	Name    string
	Kind    string
	Image   string
	Ports   []string // "3306:3306"
	Env     []string
	Volume  string   // "mysql:/var/lib/mysql" relative to ~/.herd/data
	Cmd     []string // optional container command
	Network bool     // join herd network
	URLHint string
}

// Catalog of all managed containers (databases + extras)
var Catalog = map[string]Service{
	"mysql": {
		Name: "herd-mysql", Kind: "mysql", Image: "mysql:8.0",
		Ports: []string{"3306:3306"},
		Env:   []string{"MYSQL_ROOT_PASSWORD=root", "MYSQL_ROOT_HOST=%"},
		Volume: "mysql:/var/lib/mysql", Network: true,
		URLHint: "mysql://root:root@127.0.0.1:3306",
	},
	"postgres": {
		Name: "herd-postgres", Kind: "postgres", Image: "postgres:16",
		Ports: []string{"5432:5432"},
		Env:   []string{"POSTGRES_PASSWORD=postgres", "POSTGRES_USER=postgres"},
		Volume: "postgres:/var/lib/postgresql/data", Network: true,
		URLHint: "postgres://postgres:postgres@127.0.0.1:5432/postgres",
	},
	"redis": {
		Name: "herd-redis", Kind: "redis", Image: "redis:7",
		Ports: []string{"6379:6379"},
		Volume: "redis:/data", Network: true,
		URLHint: "redis://127.0.0.1:6379",
	},
	"mailhog": {
		Name: "herd-mailhog", Kind: "mailhog", Image: "mailhog/mailhog:latest",
		Ports: []string{"1025:1025", "8025:8025"}, Network: true,
		URLHint: "SMTP 127.0.0.1:1025 · UI http://127.0.0.1:8025",
	},
	"phpmyadmin": {
		Name: "herd-phpmyadmin", Kind: "phpmyadmin", Image: "phpmyadmin:latest",
		Ports: []string{"8081:80"},
		Env: []string{
			"PMA_HOST=herd-mysql",
			"PMA_USER=root",
			"PMA_PASSWORD=root",
			"UPLOAD_LIMIT=64M",
		},
		Network: true,
		URLHint: "http://127.0.0.1:8081 (root/root → herd-mysql)",
	},
	"minio": {
		Name: "herd-minio", Kind: "minio", Image: "minio/minio:latest",
		Ports: []string{"9000:9000", "9001:9001"},
		Env: []string{
			"MINIO_ROOT_USER=minio",
			"MINIO_ROOT_PASSWORD=minioadmin",
		},
		Volume:  "minio:/data",
		Cmd:     []string{"server", "/data", "--console-address", ":9001"},
		Network: true,
		URLHint: "S3 http://127.0.0.1:9000 · Console http://127.0.0.1:9001 (minio/minioadmin)",
	},
}

// DBKinds are database services
var DBKinds = []string{"mysql", "postgres", "redis"}

// ExtraKinds are P11 helper UIs/services
var ExtraKinds = []string{"mailhog", "phpmyadmin", "minio"}

// Manager starts/stops Herd containers via docker/podman CLI
type Manager struct {
	HerdDir string
	bin     string
}

func NewManager(herdDir string) (*Manager, error) {
	bin, err := Engine()
	if err != nil {
		return nil, err
	}
	return &Manager{HerdDir: herdDir, bin: bin}, nil
}

func (m *Manager) run(args ...string) (string, error) {
	cmd := exec.Command(m.bin, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// CheckAccess verifies the engine CLI can talk to the daemon.
func (m *Manager) CheckAccess() error {
	out, err := m.run("info")
	if err == nil {
		return nil
	}
	msg := out
	if msg == "" {
		msg = err.Error()
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "permission denied") || strings.Contains(lower, "connect to the docker") {
		return fmt.Errorf("cannot access Docker (permission denied).\n"+
			"Add yourself to the docker group and refresh the session:\n"+
			"  sudo usermod -aG docker $USER\n"+
			"  newgrp docker\n"+
			"# or log out and back in, then retry: herd db start\n\n%s", msg)
	}
	return fmt.Errorf("docker not usable: %w\n%s", err, msg)
}

// EnsureNetwork creates the shared herd network (idempotent).
func (m *Manager) EnsureNetwork() error {
	if err := m.CheckAccess(); err != nil {
		return err
	}
	if _, err := m.run("network", "inspect", NetworkName); err == nil {
		return nil
	}
	out, err := m.run("network", "create", NetworkName)
	if err == nil {
		return nil
	}
	// Race / flaky inspect: network may already exist
	if strings.Contains(strings.ToLower(out), "already exists") {
		return nil
	}
	if _, inspectErr := m.run("network", "inspect", NetworkName); inspectErr == nil {
		return nil
	}
	if out == "" {
		out = err.Error()
	}
	return fmt.Errorf("could not create docker network %q: %s", NetworkName, out)
}

// EnsureDataDirs creates persistence directories
func (m *Manager) EnsureDataDirs() error {
	for _, svc := range Catalog {
		if svc.Volume == "" {
			continue
		}
		subdir := strings.SplitN(svc.Volume, ":", 2)[0]
		dir := filepath.Join(m.HerdDir, "data", subdir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// Status returns whether a named kind container is running
func (m *Manager) Status(kind string) (running bool, detail string, err error) {
	svc, ok := Catalog[kind]
	if !ok {
		return false, "", fmt.Errorf("unknown service: %s", kind)
	}
	out, err := m.run("inspect", "-f", "{{.State.Status}}", svc.Name)
	if err != nil {
		return false, "not created", nil
	}
	return out == "running", out, nil
}

// Start starts a service (pulls image if needed)
func (m *Manager) Start(kind string) error {
	svc, ok := Catalog[kind]
	if !ok {
		return fmt.Errorf("unknown service: %s", kind)
	}
	if err := m.EnsureDataDirs(); err != nil {
		return err
	}
	if svc.Network {
		if err := m.EnsureNetwork(); err != nil {
			return fmt.Errorf("docker network: %w", err)
		}
	}
	// phpMyAdmin works best if MySQL is up
	if kind == "phpmyadmin" {
		_ = m.Start("mysql")
	}

	running, _, _ := m.Status(kind)
	if running {
		return nil
	}

	if out, err := m.run("inspect", "-f", "{{.Id}}", svc.Name); err == nil && out != "" {
		startOut, err := m.run("start", svc.Name)
		if err != nil {
			return fmt.Errorf("failed to start existing container %s: %w\n%s", svc.Name, err, startOut)
		}
		return nil
	}

	args := []string{
		"run", "-d",
		"--name", svc.Name,
		"--restart", "unless-stopped",
	}
	if svc.Network {
		args = append(args, "--network", NetworkName)
	}
	for _, p := range svc.Ports {
		args = append(args, "-p", p)
	}
	for _, e := range svc.Env {
		args = append(args, "-e", e)
	}
	if svc.Volume != "" {
		parts := strings.SplitN(svc.Volume, ":", 2)
		host := filepath.Join(m.HerdDir, "data", parts[0])
		args = append(args, "-v", host+":"+parts[1])
	}
	args = append(args, svc.Image)
	args = append(args, svc.Cmd...)

	fmt.Printf("Starting %s (%s)…\n", svc.Name, svc.Image)
	out, err := m.run(args...)
	if err != nil {
		return fmt.Errorf("failed to start %s: %w\n%s", svc.Name, err, out)
	}
	return nil
}

// Stop stops a service container
func (m *Manager) Stop(kind string) error {
	svc, ok := Catalog[kind]
	if !ok {
		return fmt.Errorf("unknown service: %s", kind)
	}
	_, err := m.run("stop", svc.Name)
	return err
}

// StartAll starts mysql, postgres, redis
func (m *Manager) StartAll() error {
	var errs []string
	for _, kind := range DBKinds {
		if err := m.Start(kind); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", kind, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// StopAll stops all DB containers
func (m *Manager) StopAll() error {
	for _, kind := range DBKinds {
		_ = m.Stop(kind)
	}
	return nil
}

// StartExtras starts mailhog/phpmyadmin/minio (or given kinds)
func (m *Manager) StartExtras(kinds []string) error {
	if len(kinds) == 0 {
		kinds = ExtraKinds
	}
	var errs []string
	for _, kind := range kinds {
		if err := m.Start(kind); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", kind, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// StopExtras stops extra containers
func (m *Manager) StopExtras(kinds []string) error {
	if len(kinds) == 0 {
		kinds = ExtraKinds
	}
	for _, kind := range kinds {
		_ = m.Stop(kind)
	}
	return nil
}

// WaitReady waits briefly for a service to accept connections
func (m *Manager) WaitReady(kind string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, _, _ := m.Status(kind)
		if !running {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if m.ping(kind) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%s not ready within %s", kind, timeout)
}

func (m *Manager) ping(kind string) bool {
	svc := Catalog[kind]
	var args []string
	switch kind {
	case "mysql":
		args = []string{"exec", svc.Name, "mysqladmin", "ping", "-h", "127.0.0.1", "-uroot", "-proot"}
	case "postgres":
		args = []string{"exec", svc.Name, "pg_isready", "-U", "postgres"}
	case "redis":
		args = []string{"exec", svc.Name, "redis-cli", "ping"}
	default:
		return true
	}
	out, err := m.run(args...)
	if err != nil {
		return false
	}
	if kind == "redis" {
		return strings.Contains(strings.ToUpper(out), "PONG")
	}
	return true
}

// Exec runs a command inside the container
func (m *Manager) Exec(kind string, command ...string) (string, error) {
	svc, ok := Catalog[kind]
	if !ok {
		return "", fmt.Errorf("unknown service: %s", kind)
	}
	args := append([]string{"exec", svc.Name}, command...)
	return m.run(args...)
}

// ConnectionHint returns DSN-style hints for apps
func ConnectionHint(kind string) string {
	if svc, ok := Catalog[kind]; ok && svc.URLHint != "" {
		return svc.URLHint
	}
	return ""
}
