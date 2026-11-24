package composer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// PHPRequirement represents a composer.json PHP version constraint
type PHPRequirement struct {
	RawVersion string
	MinMajor   int
	MinMinor   int
	MaxMajor   int // -1 means unbounded
	MaxMinor   int // -1 means unbounded
	Exact      bool
}

type composerJSON struct {
	Require map[string]string `json:"require"`
}

// ExtractPHPRequirement reads composer.json and returns the PHP constraint
func ExtractPHPRequirement(projectPath string) (*PHPRequirement, error) {
	composerPath := filepath.Join(projectPath, "composer.json")
	data, err := os.ReadFile(composerPath)
	if err != nil {
		return nil, fmt.Errorf("composer.json not found: %w", err)
	}

	var cj composerJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return nil, fmt.Errorf("failed to parse composer.json: %w", err)
	}

	raw, ok := cj.Require["php"]
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	req, err := ParseConstraint(raw)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// ParseConstraint parses common Composer PHP constraints like ^8.1, >=8.0, ~8.2.0, 8.2.*
func ParseConstraint(raw string) (*PHPRequirement, error) {
	raw = strings.TrimSpace(raw)
	req := &PHPRequirement{RawVersion: raw, MaxMajor: -1, MaxMinor: -1}

	// Exact: 8.2 or 8.2.0
	if m := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.\d+)?$`).FindStringSubmatch(raw); m != nil {
		req.MinMajor, _ = strconv.Atoi(m[1])
		req.MinMinor, _ = strconv.Atoi(m[2])
		req.MaxMajor = req.MinMajor
		req.MaxMinor = req.MinMinor
		req.Exact = true
		return req, nil
	}

	// Caret: ^8.1 or ^8.1.0 → >=8.1 <9.0
	if m := regexp.MustCompile(`^\^(\d+)\.(\d+)(?:\.\d+)?$`).FindStringSubmatch(raw); m != nil {
		req.MinMajor, _ = strconv.Atoi(m[1])
		req.MinMinor, _ = strconv.Atoi(m[2])
		req.MaxMajor = req.MinMajor + 1
		req.MaxMinor = 0
		return req, nil
	}

	// Tilde: ~8.2.0 → >=8.2.0 <8.3.0 ; ~8.2 → >=8.2 <9.0
	if m := regexp.MustCompile(`^~(\d+)\.(\d+)(?:\.(\d+))?$`).FindStringSubmatch(raw); m != nil {
		req.MinMajor, _ = strconv.Atoi(m[1])
		req.MinMinor, _ = strconv.Atoi(m[2])
		if m[3] != "" {
			req.MaxMajor = req.MinMajor
			req.MaxMinor = req.MinMinor + 1
		} else {
			req.MaxMajor = req.MinMajor + 1
			req.MaxMinor = 0
		}
		return req, nil
	}

	// >=8.1 or >=8.1.0
	if m := regexp.MustCompile(`^>=\s*(\d+)\.(\d+)(?:\.\d+)?$`).FindStringSubmatch(raw); m != nil {
		req.MinMajor, _ = strconv.Atoi(m[1])
		req.MinMinor, _ = strconv.Atoi(m[2])
		return req, nil
	}

	// >8.1
	if m := regexp.MustCompile(`^>\s*(\d+)\.(\d+)(?:\.\d+)?$`).FindStringSubmatch(raw); m != nil {
		req.MinMajor, _ = strconv.Atoi(m[1])
		req.MinMinor, _ = strconv.Atoi(m[2])
		req.MinMinor++
		return req, nil
	}

	// Wildcard: 8.* or 8.2.*
	if m := regexp.MustCompile(`^(\d+)\.(?:\*|(\d+)\.\*)$`).FindStringSubmatch(raw); m != nil {
		req.MinMajor, _ = strconv.Atoi(m[1])
		if m[2] != "" {
			req.MinMinor, _ = strconv.Atoi(m[2])
			req.MaxMajor = req.MinMajor
			req.MaxMinor = req.MinMinor + 1
		} else {
			req.MinMinor = 0
			req.MaxMajor = req.MinMajor + 1
			req.MaxMinor = 0
		}
		return req, nil
	}

	// Range with pipe: ^8.1|^8.2 — take first clause
	if strings.Contains(raw, "|") {
		parts := strings.Split(raw, "|")
		return ParseConstraint(strings.TrimSpace(parts[0]))
	}

	// Fallback: extract first X.Y found
	if m := regexp.MustCompile(`(\d+)\.(\d+)`).FindStringSubmatch(raw); m != nil {
		req.MinMajor, _ = strconv.Atoi(m[1])
		req.MinMinor, _ = strconv.Atoi(m[2])
		return req, nil
	}

	return nil, fmt.Errorf("unsupported PHP constraint: %s", raw)
}

// Satisfies returns true if version (e.g. "8.2") satisfies the requirement.
// Upper bounds (MaxMajor/MaxMinor) are exclusive when set (MaxMajor >= 0).
func (r *PHPRequirement) Satisfies(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}

	if r.Exact {
		return major == r.MinMajor && minor == r.MinMinor
	}

	if major < r.MinMajor || (major == r.MinMajor && minor < r.MinMinor) {
		return false
	}

	if r.MaxMajor >= 0 {
		if major > r.MaxMajor {
			return false
		}
		if major == r.MaxMajor && minor >= r.MaxMinor {
			return false
		}
	}

	return true
}
