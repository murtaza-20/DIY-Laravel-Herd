package ssl

import "strings"

// BaseDNSNames are always present on the shared Herd certificate.
func BaseDNSNames() []string {
	return []string{"*.test", "localhost"}
}

// baseIPs are passed to mkcert as-is and encoded as IP SANs for openssl.
var baseIPHosts = []string{"127.0.0.1", "::1"}

// SiteDNSNames returns SANs Node-based HTTPS clients can verify.
//
// *.test is not enough: "test" is a public suffix, so Requestly/Postman/Node
// reject it as covering the whole TLD. Each site therefore gets:
//   - name.test          (exact host)
//   - name.test.         (FQDN with trailing dot — some clients keep it)
//   - *.name.test        (matches nginx server_name *.name.test)
func SiteDNSNames(siteNames []string) []string {
	out := make([]string, 0, len(siteNames)*3)
	seen := make(map[string]bool, len(siteNames)*3)
	add := func(s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, name := range siteNames {
		name = strings.Trim(strings.TrimSpace(name), ".")
		if name == "" {
			continue
		}
		add(name + ".test")
		add(name + ".test.")
		add("*." + name + ".test")
	}
	return out
}

// AllDNSNames is the full SAN list: base names plus per-site hosts.
func AllDNSNames(siteNames []string) []string {
	base := BaseDNSNames()
	sites := SiteDNSNames(siteNames)
	out := make([]string, 0, len(base)+len(sites))
	seen := make(map[string]bool, len(base)+len(sites))
	for _, n := range append(base, sites...) {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// RequiredDNSNames are SANs we insist on. Trailing-dot FQDNs are omitted:
// mkcert cannot emit them, and browsers/Flutter strip the trailing dot.
func RequiredDNSNames(siteNames []string) []string {
	out := make([]string, 0)
	for _, n := range AllDNSNames(siteNames) {
		if strings.HasSuffix(n, ".") {
			continue
		}
		out = append(out, n)
	}
	return out
}

// mkcertHosts drops trailing-dot FQDNs (mkcert rejects them) and appends loopback IPs.
func mkcertHosts(dns []string) []string {
	out := make([]string, 0, len(dns)+len(baseIPHosts))
	seen := make(map[string]bool, len(dns)+len(baseIPHosts))
	for _, n := range dns {
		if n == "" || strings.HasSuffix(n, ".") {
			continue
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	for _, ip := range baseIPHosts {
		if seen[ip] {
			continue
		}
		seen[ip] = true
		out = append(out, ip)
	}
	return out
}
