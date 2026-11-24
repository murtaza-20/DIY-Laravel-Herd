package system

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// NginxIncludePath is the system drop-in that loads Herd site configs
func NginxIncludePath() string {
	return "/etc/nginx/conf.d/herd.conf"
}

// DnsmasqConfigPath is the dnsmasq drop-in for .test
func DnsmasqConfigPath() string {
	return "/etc/dnsmasq.d/herd.conf"
}

// ResolvedConfigPath routes the ~test domain to local dnsmasq
func ResolvedConfigPath() string {
	return "/etc/systemd/resolved.conf.d/herd.conf"
}

// WriteNginxIncludeContent returns the include snippet for the current user herd dir
func WriteNginxIncludeContent(herdDir string) string {
	sites := filepath.Join(herdDir, "nginx", "sites-enabled", "*.conf")
	// client_max_body_size must live here (or in each site conf): system nginx
	// loads this drop-in, not ~/.herd/nginx/nginx.conf.
	return fmt.Sprintf("# Managed by Ubuntu Herd\nclient_max_body_size 120M;\ninclude %s;\n", sites)
}

// DnsmasqContent binds only 127.0.0.1:53 so it coexists with systemd-resolved
func DnsmasqContent() string {
	return strings.TrimSpace(`
# Managed by Ubuntu Herd — coexist with systemd-resolved (127.0.0.53)
bind-interfaces
listen-address=127.0.0.1
address=/.test/127.0.0.1
`) + "\n"
}

// ResolvedContent sends *.test queries to local dnsmasq
func ResolvedContent() string {
	return strings.TrimSpace(`
# Managed by Ubuntu Herd
[Resolve]
DNS=127.0.0.1
Domains=~test
`) + "\n"
}

// SudoWriteFile writes content to a root-owned path via sudo tee
func SudoWriteFile(path, content string) error {
	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo tee %s failed: %w", path, err)
	}
	return nil
}

// SudoRun runs a sudo command
func SudoRun(args ...string) error {
	cmd := exec.Command("sudo", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sudo %s failed: %w\n%s", strings.Join(args, " "), err, string(out))
	}
	return nil
}

// ApplyIntegration writes nginx/dnsmasq/resolved drop-ins and reloads services
func ApplyIntegration(herdDir string) error {
	if err := SudoWriteFile(NginxIncludePath(), WriteNginxIncludeContent(herdDir)); err != nil {
		return err
	}
	if err := SudoWriteFile(DnsmasqConfigPath(), DnsmasqContent()); err != nil {
		return err
	}
	if err := SudoRun("mkdir", "-p", "/etc/systemd/resolved.conf.d"); err != nil {
		return err
	}
	if err := SudoWriteFile(ResolvedConfigPath(), ResolvedContent()); err != nil {
		return err
	}

	// Ensure home + .herd are traversable by www-data
	if err := ensureWWWAccess(herdDir); err != nil {
		fmt.Printf("⚠️  www-data path access: %v\n", err)
	}

	if err := SudoRun("nginx", "-t"); err != nil {
		return fmt.Errorf("nginx config test failed: %w", err)
	}
	_ = SudoRun("systemctl", "reload", "nginx")
	_ = SudoRun("systemctl", "restart", "dnsmasq")
	_ = SudoRun("systemctl", "restart", "systemd-resolved")
	return nil
}

func ensureWWWAccess(herdDir string) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	home := u.HomeDir
	// Allow traverse of home (o+x) so nginx/php-fpm can reach project roots
	_ = os.Chmod(home, 0751)
	_ = os.Chmod(herdDir, 0755)
	_ = os.Chmod(filepath.Join(herdDir, "nginx"), 0755)
	_ = os.Chmod(filepath.Join(herdDir, "nginx", "sites-enabled"), 0755)
	_ = os.Chmod(filepath.Join(herdDir, "ssl"), 0755)
	return nil
}

// ManualInstructions returns copy-paste commands when sudo is unavailable
func ManualInstructions(herdDir string) string {
	var b strings.Builder
	b.WriteString("Run these with sudo to finish host integration:\n\n")
	b.WriteString(fmt.Sprintf("sudo tee %s >/dev/null <<'EOF'\n%sEOF\n\n", NginxIncludePath(), WriteNginxIncludeContent(herdDir)))
	b.WriteString(fmt.Sprintf("sudo tee %s >/dev/null <<'EOF'\n%sEOF\n\n", DnsmasqConfigPath(), DnsmasqContent()))
	b.WriteString("sudo mkdir -p /etc/systemd/resolved.conf.d\n")
	b.WriteString(fmt.Sprintf("sudo tee %s >/dev/null <<'EOF'\n%sEOF\n\n", ResolvedConfigPath(), ResolvedContent()))
	b.WriteString("sudo apt install -y mkcert libnss3-tools\n")
	b.WriteString("sudo nginx -t && sudo systemctl reload nginx\n")
	b.WriteString("sudo systemctl restart dnsmasq systemd-resolved\n")
	b.WriteString("\nOr: sudo ./scripts/p8-system-setup.sh\n")
	return b.String()
}
