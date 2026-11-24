package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// Service represents a single managed process
type Service struct {
	Name    string
	Type    string // "process" or "docker"
	Cmd     string
	Args    []string
	Env     map[string]string
	Running bool
	Pid     int
	logger  *logrus.Logger
	stopCh  chan struct{}
	process *exec.Cmd
}

// ServiceManager manages all Herd services
type ServiceManager struct {
	services map[string]*Service
	mu       sync.RWMutex
	logger   *logrus.Logger
	herdDir  string
}

// NewServiceManager creates a new service manager
func NewServiceManager(herdDir string) *ServiceManager {
	return &ServiceManager{
		services: make(map[string]*Service),
		logger:   logrus.New(),
		herdDir:  herdDir,
	}
}

// RegisterService adds a new service to be managed
func (sm *ServiceManager) RegisterService(name, srvType, cmd string, args []string, env map[string]string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	logger := logrus.New()
	logDir := filepath.Join(sm.herdDir, "logs")
	_ = os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(
		filepath.Join(logDir, name+".log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		logFile = os.Stdout
	}
	logger.SetOutput(logFile)

	sm.services[name] = &Service{
		Name:    name,
		Type:    srvType,
		Cmd:     cmd,
		Args:    args,
		Env:     env,
		Running: false,
		Pid:     0,
		logger:  logger,
		stopCh:  make(chan struct{}),
	}
}

// Start starts a specific service
func (sm *ServiceManager) Start(name string) error {
	sm.mu.Lock()
	svc, exists := sm.services[name]
	sm.mu.Unlock()

	if !exists {
		return fmt.Errorf("service not found: %s", name)
	}

	if svc.Running {
		return fmt.Errorf("service already running: %s", name)
	}

	cmd := exec.Command(svc.Cmd, svc.Args...)
	cmd.Env = os.Environ()
	for key, val := range svc.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
	}

	logPath := filepath.Join(sm.herdDir, "logs", name+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		svc.logger.Errorf("Failed to start %s: %v", name, err)
		return err
	}

	sm.mu.Lock()
	svc.process = cmd
	svc.Pid = cmd.Process.Pid
	svc.Running = true
	sm.mu.Unlock()

	svc.logger.Infof("Service started with PID %d", svc.Pid)

	go func() {
		_ = cmd.Wait()
		sm.mu.Lock()
		svc.Running = false
		sm.mu.Unlock()
	}()

	return nil
}

// Stop stops a specific service
func (sm *ServiceManager) Stop(name string) error {
	sm.mu.Lock()
	svc, exists := sm.services[name]
	if !exists {
		sm.mu.Unlock()
		return fmt.Errorf("service not found: %s", name)
	}
	if !svc.Running || svc.process == nil || svc.process.Process == nil {
		sm.mu.Unlock()
		return fmt.Errorf("service not running: %s", name)
	}
	proc := svc.process.Process
	sm.mu.Unlock()

	if err := proc.Kill(); err != nil {
		svc.logger.Errorf("Failed to stop %s: %v", name, err)
		return err
	}

	sm.mu.Lock()
	svc.Running = false
	sm.mu.Unlock()
	svc.logger.Infof("Service stopped")
	return nil
}

// Restart restarts a service
func (sm *ServiceManager) Restart(name string) error {
	_ = sm.Stop(name)
	return sm.Start(name)
}

// StartAll starts all registered services
func (sm *ServiceManager) StartAll() error {
	sm.mu.RLock()
	names := make([]string, 0, len(sm.services))
	for name := range sm.services {
		names = append(names, name)
	}
	sm.mu.RUnlock()

	for _, name := range names {
		if err := sm.Start(name); err != nil {
			sm.logger.Errorf("Failed to start %s: %v", name, err)
		}
	}
	return nil
}

// StopAll stops all running services
func (sm *ServiceManager) StopAll() error {
	sm.mu.RLock()
	names := make([]string, 0, len(sm.services))
	for name := range sm.services {
		names = append(names, name)
	}
	sm.mu.RUnlock()

	for _, name := range names {
		if err := sm.Stop(name); err != nil {
			sm.logger.Errorf("Failed to stop %s: %v", name, err)
		}
	}
	return nil
}

// Status returns the status of all services
func (sm *ServiceManager) Status() map[string]bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	status := make(map[string]bool)
	for name, svc := range sm.services {
		status[name] = svc.Running
	}
	return status
}

// GetStatus returns the status of a specific service
func (sm *ServiceManager) GetStatus(name string) (bool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.services[name]
	if !exists {
		return false, fmt.Errorf("service not found: %s", name)
	}
	return svc.Running, nil
}
