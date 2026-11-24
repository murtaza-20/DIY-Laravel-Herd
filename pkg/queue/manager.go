package queue

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/utils"
)

// Worker is a background Laravel queue worker
type Worker struct {
	ID         string    `yaml:"id"`
	Site       string    `yaml:"site"`
	Queue      string    `yaml:"queue"`
	Path       string    `yaml:"path"`
	PHPVersion string    `yaml:"php_version"`
	Pid        int       `yaml:"pid"`
	LogFile    string    `yaml:"log_file"`
	StartedAt  time.Time `yaml:"started_at"`
}

type workersFile struct {
	Workers []Worker `yaml:"workers"`
}

// Manager persists and controls queue workers under ~/.herd
type Manager struct {
	HerdDir string
}

func NewManager(herdDir string) *Manager {
	return &Manager{HerdDir: herdDir}
}

func (m *Manager) storePath() string {
	return filepath.Join(m.HerdDir, "queues.yaml")
}

func (m *Manager) load() ([]Worker, error) {
	path := m.storePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Worker{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wf workersFile
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, err
	}
	return wf.Workers, nil
}

func (m *Manager) save(workers []Worker) error {
	data, err := yaml.Marshal(workersFile{Workers: workers})
	if err != nil {
		return err
	}
	return os.WriteFile(m.storePath(), data, 0644)
}

func workerID(site, queueName string) string {
	return fmt.Sprintf("%s__%s", site, queueName)
}

// List returns known workers and refreshes Running state via /proc
func (m *Manager) List() ([]Worker, error) {
	workers, err := m.load()
	if err != nil {
		return nil, err
	}
	alive := make([]Worker, 0, len(workers))
	changed := false
	for _, w := range workers {
		if processAlive(w.Pid) {
			alive = append(alive, w)
		} else {
			changed = true
		}
	}
	if changed {
		_ = m.save(alive)
	}
	return alive, nil
}

// Start launches `php artisan queue:work` for a registered site
func (m *Manager) Start(siteName, queueName string, timeout int) (*Worker, error) {
	if queueName == "" {
		queueName = "default"
	}
	site, err := config.GetSite(siteName)
	if err != nil {
		return nil, fmt.Errorf("%w — register with: herd serve %s /path", err, siteName)
	}
	if !utils.FileExists(filepath.Join(site.Path, "artisan")) {
		return nil, fmt.Errorf("not a Laravel project (no artisan): %s", site.Path)
	}

	phpBin := utils.PHPBinaryPath(site.PHPVersion)
	if !utils.FileExists(phpBin) {
		return nil, fmt.Errorf("PHP %s not found at %s", site.PHPVersion, phpBin)
	}

	id := workerID(siteName, queueName)
	workers, err := m.List()
	if err != nil {
		return nil, err
	}
	for _, w := range workers {
		if w.ID == id {
			return nil, fmt.Errorf("worker already running for %s queue=%s (pid %d)", siteName, queueName, w.Pid)
		}
	}

	logDir := filepath.Join(m.HerdDir, "logs")
	_ = os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, fmt.Sprintf("queue-%s-%s.log", siteName, queueName))

	args := []string{
		"artisan", "queue:work",
		"--queue=" + queueName,
		"--tries=3",
		"--sleep=3",
	}
	if timeout > 0 {
		args = append(args, fmt.Sprintf("--timeout=%d", timeout))
	}

	cmd := exec.Command(phpBin, args...)
	cmd.Dir = site.Path
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = lf
	cmd.Stderr = lf

	if err := cmd.Start(); err != nil {
		_ = lf.Close()
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}
	// Detach: don't wait; keep log file open in child
	go func() {
		_ = cmd.Wait()
		_ = lf.Close()
	}()

	w := Worker{
		ID:         id,
		Site:       siteName,
		Queue:      queueName,
		Path:       site.Path,
		PHPVersion: site.PHPVersion,
		Pid:        cmd.Process.Pid,
		LogFile:    logFile,
		StartedAt:  time.Now(),
	}
	workers = append(workers, w)
	if err := m.save(workers); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	return &w, nil
}

// Stop kills a worker by site (+ optional queue)
func (m *Manager) Stop(siteName, queueName string) error {
	workers, err := m.List()
	if err != nil {
		return err
	}
	remaining := make([]Worker, 0, len(workers))
	stopped := 0
	for _, w := range workers {
		match := w.Site == siteName && (queueName == "" || w.Queue == queueName)
		if !match {
			remaining = append(remaining, w)
			continue
		}
		if processAlive(w.Pid) {
			_ = syscall.Kill(-w.Pid, syscall.SIGTERM) // process group
			_ = syscall.Kill(w.Pid, syscall.SIGTERM)
			time.Sleep(200 * time.Millisecond)
			if processAlive(w.Pid) {
				_ = syscall.Kill(w.Pid, syscall.SIGKILL)
			}
		}
		stopped++
	}
	if stopped == 0 {
		return fmt.Errorf("no running worker for site %q", siteName)
	}
	return m.save(remaining)
}

// StopAll stops every tracked worker
func (m *Manager) StopAll() error {
	workers, err := m.List()
	if err != nil {
		return err
	}
	for _, w := range workers {
		_ = m.Stop(w.Site, w.Queue)
	}
	return nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

// ParseID splits site__queue
func ParseID(id string) (site, queueName string) {
	parts := strings.SplitN(id, "__", 2)
	if len(parts) != 2 {
		return id, "default"
	}
	return parts[0], parts[1]
}

// FormatPid for display
func FormatPid(pid int) string {
	return strconv.Itoa(pid)
}
