package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// Service represents a single service (PHP-FPM, Nginx, MySQL, etc.)
type Service struct {
	Name     string            // "php-fpm", "nginx", "mysql"
	Type     string            // "process" or "docker"
	Cmd      string            // Command to execute
	Args     []string          // Command arguments
	Env      map[string]string // Environment variables
	Running  bool
	Pid      int
	logger   *logrus.Logger
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
	logFile, err := os.OpenFile(
		filepath.Join(sm.herdDir, "logs", name+".log"),
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
	sm.mu.RLock()
	svc, exists := sm.services[name]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("service not found: %s", name)
	}

	if svc.Running {
		return fmt.Errorf("service already running: %s", name)
	}

	cmd := exec.Command(svc.Cmd, svc.Args...)

	// Set environment variables
	cmd.Env = os.Environ()
	for key, val := range svc.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
	}

	// Set output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		svc.logger.Errorf("Failed to start %s: %v", name, err)
		return err
	}

	svc.process = cmd
	svc.Pid = cmd.Process.Pid
	svc.Running = true

	svc.logger.Infof("Service started with PID %d", svc.Pid)

	// Wait for process to finish in background
	go func() {
		if err := cmd.Wait(); err != nil {
			svc.logger.Errorf("Service exited with error: %v", err)
		}
		svc.Running = false
	}()

	return nil
}

// Stop stops a specific service
func (sm *ServiceManager) Stop(name string) error {
	sm.mu.RLock()
	svc, exists := sm.services[name]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("service not found: %s", name)
	}

	if !svc.Running {
		return fmt.Errorf("service not running: %s", name)
	}

	if svc.process != nil {
		if err := svc.process.Process.Kill(); err != nil {
			svc.logger.Errorf("Failed to stop %s: %v", name, err)
			return err
		}
	}

	svc.Running = false
	svc.logger.Infof("Service stopped")

	return nil
}

// Restart restarts a service
func (sm *ServiceManager) Restart(name string) error {
	if err := sm.Stop(name); err != nil {
		// If stop fails (service not running), just start it
		if _, ok := sm.services[name]; !ok {
			return err
		}
	}

	return sm.Start(name)
}

// StartAll starts all registered services
func (sm *ServiceManager) StartAll() error {
	sm.mu.RLock()
	serviceNames := make([]string, 0, len(sm.services))
	for name := range sm.services {
		serviceNames = append(serviceNames, name)
	}
	sm.mu.RUnlock()

	for _, name := range serviceNames {
		sm.logger.Infof("Starting service: %s", name)
		if err := sm.Start(name); err != nil {
			sm.logger.Errorf("Failed to start %s: %v", name, err)
			// Continue with other services
		}
	}

	return nil
}

// StopAll stops all running services
func (sm *ServiceManager) StopAll() error {
	sm.mu.RLock()
	serviceNames := make([]string, 0, len(sm.services))
	for name := range sm.services {
		serviceNames = append(serviceNames, name)
	}
	sm.mu.RUnlock()

	for _, name := range serviceNames {
		sm.logger.Infof("Stopping service: %s", name)
		if err := sm.Stop(name); err != nil {
			sm.logger.Errorf("Failed to stop %s: %v", name, err)
			// Continue with other services
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
	svc, exists := sm.services[name]
	sm.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("service not found: %s", name)
	}

	return svc.Running, nil
}
