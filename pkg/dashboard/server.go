package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thesmartyking/herd/pkg/config"
	"github.com/thesmartyking/herd/pkg/docker"
	"github.com/thesmartyking/herd/pkg/nginx"
	"github.com/thesmartyking/herd/pkg/php"
	"github.com/thesmartyking/herd/pkg/queue"
	"github.com/thesmartyking/herd/pkg/utils"
)

const DefaultPort = 7835

// StatusResponse is the dashboard overview payload
type StatusResponse struct {
	HerdDir   string        `json:"herd_dir"`
	PHP       []PHPInfo     `json:"php"`
	Sites     []SiteInfo    `json:"sites"`
	Services  []ServiceInfo `json:"services"`
	Databases []DBInfo      `json:"databases"`
	Extras    []DBInfo      `json:"extras"`
	Queues    []QueueInfo   `json:"queues"`
	Version   string        `json:"version"`
	Time      time.Time     `json:"time"`
}

type DBInfo struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Detail  string `json:"detail"`
	DSN     string `json:"dsn"`
}

type QueueInfo struct {
	Site       string    `json:"site"`
	Queue      string    `json:"queue"`
	PHPVersion string    `json:"php_version"`
	Pid        int       `json:"pid"`
	LogFile    string    `json:"log_file"`
	StartedAt  time.Time `json:"started_at"`
}

type PHPInfo struct {
	Version  string `json:"version"`
	Socket   string `json:"socket"`
	SocketOK bool   `json:"socket_ok"`
	Active   bool   `json:"active"`
}

type SiteInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	PHPVersion   string    `json:"php_version"`
	ProjectType  string    `json:"project_type"`
	URL          string    `json:"url"`
	RegisteredAt time.Time `json:"registered_at"`
}

type ServiceInfo struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Detail string `json:"detail"`
}

// Server hosts the dashboard API + UI
type Server struct {
	Port   int
	router *gin.Engine
}

// New creates a dashboard server
func New(port int) *Server {
	if port == 0 {
		port = DefaultPort
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	s := &Server{Port: port, router: r}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
	s.router.GET("/app.css", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/css; charset=utf-8", appCSS)
	})
	s.router.GET("/app.js", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/javascript; charset=utf-8", appJS)
	})

	api := s.router.Group("/api")
	{
		api.GET("/status", s.handleStatus)
		api.GET("/sites", s.handleSites)
		api.GET("/sites/:name", s.handleSite)
		api.DELETE("/sites/:name", s.handleForgetSite)
		api.GET("/php", s.handlePHP)
		api.GET("/logs/:service", s.handleLogs)
		api.POST("/services/:name/restart", s.handleRestart)
		api.POST("/services/nginx/reload", s.handleNginxReload)
	}
}

// ListenAndServe starts the HTTP server (blocking)
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.Port)
	return s.router.Run(addr)
}

// ListenAndServeContext runs until ctx is cancelled
func (s *Server) ListenAndServeContext(ctx context.Context) error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.Port)
	srv := &http.Server{Addr: addr, Handler: s.router}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) handleStatus(c *gin.Context) {
	herdDir, _ := config.GetHerdDir()
	sites, _ := loadSiteInfos()
	c.JSON(http.StatusOK, StatusResponse{
		HerdDir:   herdDir,
		PHP:       collectPHP(),
		Sites:     sites,
		Services:  collectServices(),
		Databases: collectDatabases(herdDir),
		Extras:    collectExtras(herdDir),
		Queues:    collectQueues(herdDir),
		Version:   "0.1.0",
		Time:      time.Now(),
	})
}

func (s *Server) handleSites(c *gin.Context) {
	sites, err := loadSiteInfos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sites": sites})
}

func (s *Server) handleSite(c *gin.Context) {
	name := c.Param("name")
	site, err := config.GetSite(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toSiteInfo(*site))
}

func (s *Server) handleForgetSite(c *gin.Context) {
	name := c.Param("name")
	if err := config.UnregisterSite(name); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	herdDir, _ := config.GetHerdDir()
	gen := nginx.NewConfigGenerator(filepath.Join(herdDir, "nginx", "sites-enabled"))
	_ = gen.RemoveConfig(name)
	_ = exec.Command("sudo", "-n", "systemctl", "reload", "nginx").Run()
	c.JSON(http.StatusOK, gin.H{"ok": true, "forgotten": name})
}

func (s *Server) handlePHP(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"php": collectPHP()})
}

func (s *Server) handleLogs(c *gin.Context) {
	service := c.Param("service")
	herdDir, err := config.GetHerdDir()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "herd not initialized"})
		return
	}
	path := filepath.Join(herdDir, "logs", service+".log")
	data, err := readTail(path, 64*1024)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error(), "path": path})
		return
	}
	c.JSON(http.StatusOK, gin.H{"service": service, "log": string(data)})
}

func (s *Server) handleRestart(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service name required"})
		return
	}
	out, err := exec.Command("sudo", "-n", "systemctl", "restart", name).CombinedOutput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  fmt.Sprintf("restart failed: %v", err),
			"hint":   "sudo systemctl restart " + name,
			"output": string(out),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "restarted": name})
}

func (s *Server) handleNginxReload(c *gin.Context) {
	out, err := exec.Command("sudo", "-n", "systemctl", "reload", "nginx").CombinedOutput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "output": string(out)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func loadSiteInfos() ([]SiteInfo, error) {
	sites, err := config.LoadSites()
	if err != nil {
		return nil, err
	}
	out := make([]SiteInfo, 0, len(sites))
	for _, s := range sites {
		out = append(out, toSiteInfo(s))
	}
	return out, nil
}

func toSiteInfo(s config.Site) SiteInfo {
	return SiteInfo{
		Name:         s.Name,
		Path:         s.Path,
		PHPVersion:   s.PHPVersion,
		ProjectType:  s.ProjectType,
		URL:          fmt.Sprintf("https://%s.test", s.Name),
		RegisteredAt: s.RegisteredAt,
	}
}

func collectPHP() []PHPInfo {
	versions, _ := utils.GetAvailablePHPVersions()
	out := make([]PHPInfo, 0, len(versions))
	for _, v := range versions {
		sock := php.SocketPath(v)
		out = append(out, PHPInfo{
			Version:  v,
			Socket:   sock,
			SocketOK: utils.FileExists(sock),
			Active:   systemdActive(fmt.Sprintf("php%s-fpm", v)),
		})
	}
	return out
}

func collectServices() []ServiceInfo {
	services := []ServiceInfo{
		{Name: "nginx", Active: systemdActive("nginx"), Detail: "web server"},
		{Name: "dnsmasq", Active: systemdActive("dnsmasq"), Detail: ".test DNS"},
	}
	versions, _ := utils.GetAvailablePHPVersions()
	for _, v := range versions {
		name := fmt.Sprintf("php%s-fpm", v)
		services = append(services, ServiceInfo{
			Name:   name,
			Active: systemdActive(name),
			Detail: "PHP " + v,
		})
	}
	return services
}

func collectDatabases(herdDir string) []DBInfo {
	return collectDockerKinds(herdDir, docker.DBKinds)
}

func collectExtras(herdDir string) []DBInfo {
	return collectDockerKinds(herdDir, docker.ExtraKinds)
}

func collectDockerKinds(herdDir string, kinds []string) []DBInfo {
	out := make([]DBInfo, 0, len(kinds))
	if !docker.Available() {
		for _, kind := range kinds {
			out = append(out, DBInfo{
				Kind:    kind,
				Name:    docker.Catalog[kind].Name,
				Running: false,
				Detail:  "docker not installed",
				DSN:     docker.ConnectionHint(kind),
			})
		}
		return out
	}
	m, err := docker.NewManager(herdDir)
	if err != nil {
		return out
	}
	for _, kind := range kinds {
		running, detail, _ := m.Status(kind)
		out = append(out, DBInfo{
			Kind:    kind,
			Name:    docker.Catalog[kind].Name,
			Running: running,
			Detail:  detail,
			DSN:     docker.ConnectionHint(kind),
		})
	}
	return out
}

func collectQueues(herdDir string) []QueueInfo {
	mgr := queue.NewManager(herdDir)
	workers, err := mgr.List()
	if err != nil {
		return nil
	}
	out := make([]QueueInfo, 0, len(workers))
	for _, w := range workers {
		out = append(out, QueueInfo{
			Site:       w.Site,
			Queue:      w.Queue,
			PHPVersion: w.PHPVersion,
			Pid:        w.Pid,
			LogFile:    w.LogFile,
			StartedAt:  w.StartedAt,
		})
	}
	return out
}

func systemdActive(unit string) bool {
	out, err := exec.Command("systemctl", "is-active", unit).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

func readTail(path string, max int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("log not found: %s", path)
	}
	if len(data) > max {
		data = data[len(data)-max:]
	}
	return data, nil
}
