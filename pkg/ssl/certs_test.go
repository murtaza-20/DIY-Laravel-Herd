package ssl

import (
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSiteDNSNames(t *testing.T) {
	got := SiteDNSNames([]string{"mahakali-ornaments-api", "mahakali-ornaments-api", " demo "})
	want := []string{
		"mahakali-ornaments-api.test",
		"mahakali-ornaments-api.test.",
		"*.mahakali-ornaments-api.test",
		"demo.test",
		"demo.test.",
		"*.demo.test",
	}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestAllDNSNamesIncludesExactHost(t *testing.T) {
	got := AllDNSNames([]string{"mahakali-ornaments-api"})
	need := []string{
		"*.test",
		"localhost",
		"mahakali-ornaments-api.test",
		"mahakali-ornaments-api.test.",
		"*.mahakali-ornaments-api.test",
	}
	have := map[string]bool{}
	for _, n := range got {
		have[n] = true
	}
	for _, n := range need {
		if !have[n] {
			t.Errorf("missing SAN %q in %v", n, got)
		}
	}
}

func TestOpensslSAN(t *testing.T) {
	san := opensslSAN(
		[]string{"*.test", "mahakali-ornaments-api.test", "mahakali-ornaments-api.test."},
		[]net.IP{net.ParseIP("127.0.0.1")},
	)
	if !strings.HasPrefix(san, "subjectAltName=") {
		t.Fatalf("prefix: %s", san)
	}
	for _, part := range []string{
		"DNS:*.test",
		"DNS:mahakali-ornaments-api.test",
		"DNS:mahakali-ornaments-api.test.",
		"IP:127.0.0.1",
	} {
		if !strings.Contains(san, part) {
			t.Errorf("missing %q in %s", part, san)
		}
	}
}

func TestMkcertHostsDropsTrailingDot(t *testing.T) {
	got := mkcertHosts([]string{"foo.test", "foo.test.", "*.foo.test", ""})
	if len(got) < 2 || got[0] != "foo.test" || got[1] != "*.foo.test" {
		t.Fatalf("got %v", got)
	}
	hasLoopback := false
	for _, n := range got {
		if n == "127.0.0.1" {
			hasLoopback = true
		}
		if strings.HasSuffix(n, ".") && net.ParseIP(strings.TrimSuffix(n, ".")) == nil && n != "::1" {
			t.Errorf("trailing-dot host leaked to mkcert: %q", n)
		}
	}
	if !hasLoopback {
		t.Errorf("missing 127.0.0.1 in %v", got)
	}
}

func TestRequiredDNSNamesOmitsTrailingDot(t *testing.T) {
	got := RequiredDNSNames([]string{"mahakali-ornaments-api"})
	for _, n := range got {
		if strings.HasSuffix(n, ".") {
			t.Errorf("required SAN should not be a trailing-dot FQDN: %q", n)
		}
	}
	have := map[string]bool{}
	for _, n := range got {
		have[n] = true
	}
	for _, n := range []string{"*.test", "localhost", "mahakali-ornaments-api.test", "*.mahakali-ornaments-api.test"} {
		if !have[n] {
			t.Errorf("missing required SAN %q in %v", n, got)
		}
	}
}

func TestEnsureCertificatesAddsSiteSANs(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not installed")
	}
	dir := t.TempDir()
	t.Setenv("CAROOT", filepath.Join(dir, "caroot"))
	certPath, _, method, err := EnsureCertificates(dir, []string{"mahakali-ornaments-api"})
	if err != nil {
		t.Fatal(err)
	}
	if method == "" {
		t.Fatal("empty method")
	}
	need := RequiredDNSNames([]string{"mahakali-ornaments-api"})
	if !certCoversDNS(certPath, need) {
		t.Fatalf("cert from %s missing site SANs", method)
	}
	if method == "openssl" && !certCoversDNS(certPath, []string{"mahakali-ornaments-api.test."}) {
		t.Fatal("openssl cert missing trailing-dot SAN")
	}
	_, _, method2, err := EnsureCertificates(dir, []string{"mahakali-ornaments-api"})
	if err != nil {
		t.Fatal(err)
	}
	if method2 != "existing" {
		t.Fatalf("expected existing, got %s", method2)
	}
}
