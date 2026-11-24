package ssl

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Paths under ~/.herd/ssl
const (
	CertFileName = "herd.test.pem"
	KeyFileName  = "herd.test-key.pem"
)

// Dir returns ~/.herd/ssl
func Dir(herdDir string) string {
	return filepath.Join(herdDir, "ssl")
}

// CertPath returns the TLS certificate path
func CertPath(herdDir string) string {
	return filepath.Join(Dir(herdDir), CertFileName)
}

// KeyPath returns the TLS private key path
func KeyPath(herdDir string) string {
	return filepath.Join(Dir(herdDir), KeyFileName)
}

// EnsureTools checks openssl is present; prefers mkcert when available
func EnsureTools() error {
	if _, err := exec.LookPath("openssl"); err != nil {
		return fmt.Errorf("openssl not found. Install with: sudo apt install openssl")
	}
	return nil
}

// MkcertAvailable reports whether mkcert is on PATH
func MkcertAvailable() bool {
	_, err := exec.LookPath("mkcert")
	return err == nil
}

// EnsureCertificates creates or refreshes a local cert covering *.test plus
// each registered site's exact hostname. Node-based clients (Requestly, Postman)
// reject a bare *.test wildcard because "test" is a public suffix, so every
// site needs DNS:<name>.test as a SAN.
//
// mkcert is preferred: browsers and Flutter web require a CA in the OS trust
// store (ERR_CERT_AUTHORITY_INVALID otherwise). openssl is a self-signed fallback.
func EnsureCertificates(herdDir string, siteNames []string) (certPath, keyPath string, method string, err error) {
	if err := EnsureTools(); err != nil {
		return "", "", "", err
	}

	sslDir := Dir(herdDir)
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		return "", "", "", err
	}

	certPath = CertPath(herdDir)
	keyPath = KeyPath(herdDir)
	required := RequiredDNSNames(siteNames)
	all := AllDNSNames(siteNames)
	haveFiles := fileExists(certPath) && fileExists(keyPath)

	if MkcertAvailable() {
		if haveFiles && certSignedByMkcert(certPath) && certCoversDNS(certPath, required) {
			relaxCertPerms(certPath, keyPath)
			return certPath, keyPath, "existing", nil
		}
		if err := writeMkcert(sslDir, certPath, keyPath, required); err == nil && certCoversDNS(certPath, required) {
			relaxCertPerms(certPath, keyPath)
			return certPath, keyPath, "mkcert", nil
		}
	}

	if haveFiles && certCoversDNS(certPath, required) {
		relaxCertPerms(certPath, keyPath)
		return certPath, keyPath, "existing", nil
	}

	if err := writeOpenSSL(certPath, keyPath, all); err != nil {
		return "", "", "", err
	}
	relaxCertPerms(certPath, keyPath)
	return certPath, keyPath, "openssl", nil
}

func certSignedByMkcert(certPath string) bool {
	caPath := filepath.Join(mkcertCAROOT(), "rootCA.pem")
	ca, err := parseCertFile(caPath)
	if err != nil {
		return false
	}
	leaf, err := parseCertFile(certPath)
	if err != nil {
		return false
	}
	return leaf.CheckSignatureFrom(ca) == nil
}

func mkcertCAROOT() string {
	out, err := exec.Command("mkcert", "-CAROOT").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parseCertFile(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM data in %s", path)
	}
	return x509.ParseCertificate(block.Bytes)
}

func writeMkcert(sslDir, certPath, keyPath string, dns []string) error {
	args := []string{"-cert-file", certPath, "-key-file", keyPath}
	args = append(args, mkcertHosts(dns)...)
	cmd := exec.Command("mkcert", args...)
	cmd.Dir = sslDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mkcert failed: %w\n%s", err, string(out))
	}
	return nil
}

func writeOpenSSL(certPath, keyPath string, dns []string) error {
	// IPv4 only here: openssl -addext parses IP:::1 unreliably.
	san := opensslSAN(dns, []net.IP{net.ParseIP("127.0.0.1")})
	subj := "/CN=*.test/O=Ubuntu Herd/C=US"
	cmd := exec.Command("openssl", "req", "-x509", "-nodes", "-newkey", "rsa:2048",
		"-keyout", keyPath,
		"-out", certPath,
		"-days", "825",
		"-subj", subj,
		"-addext", san,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("openssl cert generation failed: %w\n%s", err, string(out))
	}
	return nil
}

// opensslSAN builds the -addext value for subjectAltName.
func opensslSAN(dns []string, ips []net.IP) string {
	parts := make([]string, 0, len(dns)+len(ips))
	seen := make(map[string]bool, len(dns)+len(ips))
	for _, name := range dns {
		name = strings.TrimSpace(name)
		if name == "" || seen["dns:"+name] {
			continue
		}
		seen["dns:"+name] = true
		parts = append(parts, "DNS:"+name)
	}
	for _, ip := range ips {
		s := opensslIPString(ip)
		if s == "" {
			continue
		}
		key := "ip:" + s
		if seen[key] {
			continue
		}
		seen[key] = true
		parts = append(parts, "IP:"+s)
	}
	return "subjectAltName=" + strings.Join(parts, ",")
}

func opensslIPString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ""
}

func certCoversDNS(certPath string, want []string) bool {
	cert, err := parseCertFile(certPath)
	if err != nil {
		return false
	}
	have := make(map[string]bool, len(cert.DNSNames)+len(cert.IPAddresses))
	for _, n := range cert.DNSNames {
		have[strings.ToLower(n)] = true
	}
	for _, ip := range cert.IPAddresses {
		have[strings.ToLower(ip.String())] = true
	}
	for _, n := range want {
		if n == "" {
			continue
		}
		if !have[strings.ToLower(n)] {
			return false
		}
	}
	return true
}

// relaxCertPerms lets nginx (www-data) read cert/key under ~/.herd/ssl.
// Local-dev keys are world-readable so www-data can load them without chgrp.
func relaxCertPerms(certPath, keyPath string) {
	_ = os.Chmod(filepath.Dir(certPath), 0755)
	_ = os.Chmod(certPath, 0644)
	_ = os.Chmod(keyPath, 0644)
	_ = exec.Command("chgrp", "www-data", keyPath, certPath).Run()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
