// Package state collects a snapshot of the router's local network state
// (interfaces, routing tables, ip rules) for the agent to publish to NATS.
//
// The real implementation lives in collector_linux.go and uses netlink + the
// `ip` shell tool. On other platforms a stub returns an empty snapshot so the
// codebase still compiles for development/testing on macOS.
package state

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"router-sync/internal/models"
)

// Collector reads kernel-level network state.
type Collector struct {
	hostname   string
	tableNames map[int]string
}

// NewCollector creates a collector identified by the given hostname.
func NewCollector(hostname string) *Collector {
	return &Collector{
		hostname:   hostname,
		tableNames: make(map[int]string),
	}
}

// SetTableNames replaces the table-id -> friendly-name mapping the collector
// reports alongside routing tables. Pass the names of the known providers
// from the agent's current cache.
func (c *Collector) SetTableNames(names map[int]string) {
	c.tableNames = names
}

// collectRules parses `ip rule show` (the same path the manager uses) and is
// reused on all platforms because exec.Command compiles everywhere even though
// the binary itself only exists on Linux at runtime.
func (c *Collector) collectRules() ([]models.IPRule, error) {
	cmd := exec.Command("ip", "rule", "show")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ip rule show failed: %w", err)
	}

	var rules []models.IPRule
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		rule, ok := parseIPRule(line)
		if !ok {
			continue
		}
		rule.TableName = c.tableNames[rule.Table]
		rules = append(rules, rule)
	}
	return rules, nil
}

// parseIPRule extracts priority, source CIDR and table from an `ip rule show` line, e.g.:
//
//	"100: from 192.168.2.25 lookup 99"
//	"32766: from all lookup main"
func parseIPRule(line string) (models.IPRule, bool) {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return models.IPRule{}, false
	}

	priorityStr := strings.TrimSuffix(parts[0], ":")
	priority, err := strconv.Atoi(priorityStr)
	if err != nil {
		return models.IPRule{}, false
	}

	rule := models.IPRule{Priority: priority}

	for i, p := range parts {
		switch p {
		case "from":
			if i+1 < len(parts) {
				rule.From = parts[i+1]
			}
		case "lookup":
			if i+1 < len(parts) {
				rule.Table = lookupTableID(parts[i+1])
			}
		}
	}

	return rule, true
}

// lookupTableID returns either the parsed integer or, for symbolic names
// (main, default, local), the standard kernel ID.
func lookupTableID(s string) int {
	switch s {
	case "main":
		return 254
	case "default":
		return 253
	case "local":
		return 255
	}
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return id
}

// ParseCIDRString is a defensive helper for callers that want to validate a value
// that may be either an IP or CIDR (used in tests).
func ParseCIDRString(s string) (*net.IPNet, error) {
	if _, n, err := net.ParseCIDR(s); err == nil {
		return n, nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP or CIDR: %s", s)
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}, nil
}
