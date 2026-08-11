package router

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"router-sync/internal/models"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// Manager manages routing tables and policies using netlink.
// The hostname identifies which interface mapping on a provider applies here.
type Manager struct {
	mu       sync.RWMutex
	hostname string
}

// NewManager creates a new router manager pinned to the given hostname so it can
// resolve provider.Interfaces[hostname] consistently.
func NewManager(hostname string) (*Manager, error) {
	return &Manager{hostname: hostname}, nil
}

// Hostname returns the hostname this manager is bound to.
func (m *Manager) Hostname() string {
	return m.hostname
}

// SetupProvider sets up routing for an internet provider.
// Acquires the manager mutex; callers that already hold it (e.g. SyncProviders)
// must use setupProviderLocked instead.
func (m *Manager) SetupProvider(provider *models.InternetProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.setupProviderLocked(provider)
}

// setupProviderLocked performs the provider setup assuming m.mu is already held.
func (m *Manager) setupProviderLocked(provider *models.InternetProvider) error {
	iface := provider.InterfaceForHost(m.hostname)
	logrus.Infof("Setting up provider %s on interface %s with gateway %s",
		provider.Name, iface, provider.Gateway)

	// Get the network interface
	// link, err := netlink.LinkByName(provider.Interface)
	// if err != nil {
	// 	return fmt.Errorf("failed to get interface %s: %w", provider.Interface, err)
	// }

	// Parse gateway IP
	// gwIP := net.ParseIP(provider.Gateway)
	// if gwIP == nil {
	// 	return fmt.Errorf("invalid gateway IP: %s", provider.Gateway)
	// }

	// Add default route to the routing table
	// route := &netlink.Route{
	// 	LinkIndex: link.Attrs().Index,
	// 	Gw:        gwIP,
	// 	Table:     provider.TableID,
	// 	Priority:  100,
	// }

	// Remove existing route if it exists
	// netlink.RouteDel(route)

	// Add the new route
	// if err := netlink.RouteAdd(route); err != nil {
	// 	return fmt.Errorf("failed to add route for provider %s: %w", provider.Name, err)
	// }

	logrus.Infof("Successfully set up provider %s (route installation commented out)", provider.Name)
	return nil
}

// RemoveProvider removes routing for an internet provider
func (m *Manager) RemoveProvider(provider *models.InternetProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logrus.Infof("Removing provider %s", provider.Name)

	// Get the network interface
	// link, err := netlink.LinkByName(provider.Interface)
	// if err != nil {
	// 	return fmt.Errorf("failed to get interface %s: %w", provider.Interface, err)
	// }

	// Parse gateway IP
	// gwIP := net.ParseIP(provider.Gateway)
	// if gwIP == nil {
	// 	return fmt.Errorf("invalid gateway IP: %s", provider.Gateway)
	// }

	// Remove the route
	// route := &netlink.Route{
	// 	LinkIndex: link.Attrs().Index,
	// 	Gw:        gwIP,
	// 	Table:     provider.TableID,
	// }

	// if err := netlink.RouteDel(route); err != nil {
	// 	logrus.Warnf("Failed to remove route for provider %s: %v", provider.Name, err)
	// }

	logrus.Infof("Successfully removed provider %s (route removal commented out)", provider.Name)
	return nil
}

// SetupPolicy sets up a routing policy based on source IP
func (m *Manager) SetupPolicy(policy *models.RoutingPolicy, provider *models.InternetProvider) error {
	logrus.Debugf("=== SetupPolicy called for policy: %s ===", policy.Name)

	// Note: This function is called from SyncPolicies which already holds the mutex
	// so we don't need to lock again here

	logrus.Debugf("SetupPolicy: Checking if policy is enabled")
	if !policy.Enabled {
		logrus.Debugf("Policy %s is disabled, removing existing rules", policy.Name)

		srcNet, err := parsePolicySource(policy.ID)
		if err != nil {
			return err
		}

		// Remove all rules for this source IP and clear conntrack
		if err := m.removeAllRulesForSource(srcNet); err != nil {
			logrus.Warnf("Failed to remove rules for disabled policy %s: %v", policy.Name, err)
		}

		logrus.Debugf("Successfully disabled policy %s", policy.Name)
		return nil
	}

	// Log enabled policy at INFO level
	logrus.Infof("Policy: %s, Source: %s, Provider: %s", policy.Name, policy.ID, provider.Name)

	logrus.Debugf("SetupPolicy: Policy is enabled, proceeding with setup")
	logrus.Debugf("Setting up policy %s (ID: %s) to use provider %s (TableID: %d)",
		policy.Name, policy.ID, provider.Name, provider.TableID)

	srcNet, err := parsePolicySource(policy.ID)
	if err != nil {
		return err
	}

	wantPriority := calculatePriority(srcNet)
	logrus.Debugf("Parsed source network: %s (want priority %d)", srcNet.String(), wantPriority)

	// Check if a rule already exists for this source network
	exists, existingPriority, existingTable := m.checkRoutingRuleExists(srcNet)

	if exists && existingPriority == wantPriority && existingTable == provider.TableID {
		logrus.Debugf("SKIPPING: Routing rule already exists and is correct for policy %s: priority=%d, table=%d, src=%s",
			policy.Name, existingPriority, existingTable, srcNet.String())
		return nil
	}

	if exists {
		logrus.Infof("Reconciling rule for %s: have priority=%d table=%d, want priority=%d table=%d",
			srcNet.String(), existingPriority, existingTable, wantPriority, provider.TableID)
		if err := m.removeAllRulesForSource(srcNet); err != nil {
			return fmt.Errorf("failed to remove old routing rules for policy %s: %w", policy.Name, err)
		}
	}

	// Add routing rule using ip command
	logrus.Debugf("ADDING: New routing rule for policy %s: src=%s, table=%d, priority=%d",
		policy.Name, srcNet.String(), provider.TableID, wantPriority)
	if err := m.addRoutingRule(srcNet, provider.TableID); err != nil {
		return fmt.Errorf("failed to add routing rule for policy %s: %w", policy.Name, err)
	}

	logrus.Debugf("Successfully set up policy %s", policy.Name)
	return nil
}

// RemovePolicy removes a routing policy
func (m *Manager) RemovePolicy(policy *models.RoutingPolicy, provider *models.InternetProvider) error {
	logrus.Infof("Removing policy %s (ID: %s)", policy.Name, policy.ID)

	// Note: This function is called from SyncPolicies which already holds the mutex
	// so we don't need to lock again here

	srcNet, err := parsePolicySource(policy.ID)
	if err != nil {
		return err
	}

	// Remove routing rule using ip command
	if err := m.removeRoutingRule(srcNet); err != nil {
		return fmt.Errorf("failed to remove routing rule for policy %s: %w", policy.Name, err)
	}

	logrus.Infof("Successfully removed policy %s", policy.Name)
	return nil
}

// SyncProviders synchronizes all providers with the current routing configuration
func (m *Manager) SyncProviders(providers []*models.InternetProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logrus.Info("Synchronizing providers with routing configuration")
	logrus.Infof("Processing %d providers", len(providers))

	// Clear existing routes for our tables
	for _, provider := range providers {
		logrus.Debugf("Clearing routes for provider: %s", provider.Name)
		if err := m.clearProviderRoutes(provider); err != nil {
			logrus.Warnf("Failed to clear routes for provider %s: %v", provider.Name, err)
		}
	}

	// Set up new routes. We already hold m.mu, so call the locked variant.
	for _, provider := range providers {
		logrus.Debugf("Setting up provider: %s", provider.Name)
		if err := m.setupProviderLocked(provider); err != nil {
			logrus.Errorf("Failed to set up provider %s: %v", provider.Name, err)
			continue
		}
	}

	logrus.Info("Provider synchronization completed")
	return nil
}

// SyncPolicies synchronizes all policies with the current routing configuration
func (m *Manager) SyncPolicies(policies []*models.RoutingPolicy, providers []*models.InternetProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logrus.Debug("Synchronizing policies with routing configuration")
	logrus.Debugf("Found %d policies and %d providers", len(policies), len(providers))

	// Clean up any duplicate rules before processing
	if err := m.cleanupDuplicateRules(); err != nil {
		logrus.Warnf("Failed to cleanup duplicate rules: %v", err)
	}

	// Create provider lookup map
	providerMap := make(map[string]*models.InternetProvider)
	for _, provider := range providers {
		providerMap[provider.ID] = provider
		logrus.Debugf("Provider: %s (ID: %s, TableID: %d)", provider.Name, provider.ID, provider.TableID)
	}

	// Set up rules for all policies
	for _, policy := range policies {
		logrus.Debugf("Setting up policy: %s (ID: %s, ProviderID: %s)", policy.Name, policy.ID, policy.ProviderID)
		if provider, exists := providerMap[policy.ProviderID]; exists {
			logrus.Debugf("Found provider for policy %s: %s (TableID: %d)", policy.Name, provider.Name, provider.TableID)
			if err := m.SetupPolicy(policy, provider); err != nil {
				logrus.Errorf("Failed to set up policy %s: %v", policy.Name, err)
				continue
			}
			logrus.Debugf("Successfully set up policy: %s", policy.Name)
		} else {
			logrus.Warnf("Provider %s not found for policy %s", policy.ProviderID, policy.Name)
		}
	}

	logrus.Debug("Policy synchronization completed")

	// Clean up rules for policies that no longer exist
	if err := m.cleanupStaleRules(policies); err != nil {
		logrus.Warnf("Failed to cleanup stale rules: %v", err)
	}

	// Validate that we have only one rule per source IP
	if err := m.validateSingleRulePerSource(); err != nil {
		logrus.Warnf("Failed to validate single rule per source: %v", err)
	}

	return nil
}

// clearProviderRoutes clears all routes for a provider
func (m *Manager) clearProviderRoutes(provider *models.InternetProvider) error {
	logrus.Debugf("Clearing routes for provider %s (table %d)", provider.Name, provider.TableID)

	// Get all routes for the table
	// Note: RouteListFiltered is not available, so we'll use RouteList and filter manually
	routes, err := netlink.RouteList(nil, 0) // 0 for all families
	if err != nil {
		logrus.Errorf("Failed to list routes: %v", err)
		return fmt.Errorf("failed to list routes: %w", err)
	}

	logrus.Debugf("Found %d total routes, checking for table %d", len(routes), provider.TableID)

	// Remove all routes in the table
	for _, route := range routes {
		if route.Table == provider.TableID {
			logrus.Debugf("Removing route in table %d: %v", provider.TableID, route)
			if err := netlink.RouteDel(&route); err != nil {
				logrus.Warnf("Failed to remove route: %v", err)
			}
		}
	}

	logrus.Debugf("Finished clearing routes for provider %s", provider.Name)
	return nil
}

// GetRoutingStats returns statistics about the current routing configuration
func (m *Manager) GetRoutingStats() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	// Count routes
	routes, err := netlink.RouteList(nil, 0) // 0 for all families
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}
	stats["total_routes"] = len(routes)

	// Count rules (not available in current netlink library)
	stats["total_rules"] = 0
	stats["rules_note"] = "Rule management not implemented in current netlink library"

	// Count interfaces
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}
	stats["total_interfaces"] = len(links)

	return stats, nil
}

// Managed ip rule priority range. Lower numbers are evaluated first by Linux.
// Priority = managedPriorityMin + (32 - prefixLen), so /32 hosts win over
// overlapping /25s and /24s, etc. Documented map covers /8–/32; /0–/7 still work.
const (
	managedPriorityMin = 2000
	managedPriorityMax = 2032
)

// parsePolicySource parses a policy ID as a CIDR or bare IPv4 host (/32).
func parsePolicySource(id string) (*net.IPNet, error) {
	_, ipnet, err := net.ParseCIDR(id)
	if err == nil {
		return ipnet, nil
	}
	srcIP := net.ParseIP(id)
	if srcIP == nil {
		return nil, fmt.Errorf("invalid policy ID as source IP/CIDR: %s", id)
	}
	if v4 := srcIP.To4(); v4 != nil {
		srcIP = v4
	}
	return &net.IPNet{
		IP:   srcIP,
		Mask: net.CIDRMask(32, 32),
	}, nil
}

// calculatePriority maps IPv4 prefix length to an ip-rule priority.
// More specific CIDRs get lower priority numbers (higher precedence):
// /32 → 2000, /25 → 2007, /24 → 2008, /8 → 2024, /0 → 2032.
func calculatePriority(srcNet *net.IPNet) int {
	ones, bits := srcNet.Mask.Size()
	if bits == 0 {
		return managedPriorityMax
	}
	// IPv4 only: bits should be 32. Clamp ones into 0..32 for safety.
	if ones < 0 {
		ones = 0
	}
	if ones > 32 {
		ones = 32
	}
	return managedPriorityMin + (32 - ones)
}

// fromSelectors returns the "from" tokens Linux may print for srcNet.
// Host /32 rules often appear as a bare IP; CIDRs always include the mask.
func fromSelectors(srcNet *net.IPNet) []string {
	cidr := srcNet.String()
	ones, bits := srcNet.Mask.Size()
	if ones == bits {
		return []string{cidr, srcNet.IP.String()}
	}
	return []string{cidr}
}

// normalizeFromToken canonicalizes an ip-rule "from" token to CIDR form
// (hosts become /32) for set membership checks.
func normalizeFromToken(from string) string {
	if from == "" || from == "all" {
		return from
	}
	if strings.Contains(from, "/") {
		_, n, err := net.ParseCIDR(from)
		if err == nil {
			return n.String()
		}
		return from
	}
	ip := net.ParseIP(from)
	if ip == nil {
		return from
	}
	return (&net.IPNet{IP: ip.To4(), Mask: net.CIDRMask(32, 32)}).String()
}

// extractFromToken returns the token after "from" in an `ip rule show` line.
func extractFromToken(line string) string {
	parts := strings.Fields(line)
	for i, part := range parts {
		if part == "from" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// lineMatchesSource reports whether an `ip rule show` line is for srcNet.
func lineMatchesSource(line string, srcNet *net.IPNet) bool {
	from := extractFromToken(line)
	if from == "" || from == "all" {
		return false
	}
	for _, sel := range fromSelectors(srcNet) {
		if from == sel {
			return true
		}
	}
	return false
}

// parsePriorityAndTable extracts priority and lookup table from an ip rule line.
func parsePriorityAndTable(line string) (priority int, table int, ok bool) {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return 0, 0, false
	}
	priorityStr := strings.TrimSuffix(parts[0], ":")
	priority, err := strconv.Atoi(priorityStr)
	if err != nil {
		return 0, 0, false
	}
	tableStr := parts[len(parts)-1]
	table, err = strconv.Atoi(tableStr)
	if err != nil {
		return priority, 0, false
	}
	return priority, table, true
}

// deleteRuleFrom removes one fib rule matching the given from selector.
func deleteRuleFrom(from string) error {
	cmd := exec.Command("ip", "rule", "del", "from", from)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip rule del from %s: %w: %s", from, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// checkRoutingRuleExists checks if a routing rule already exists for a given source network
func (m *Manager) checkRoutingRuleExists(srcNet *net.IPNet) (bool, int, int) {
	cmd := exec.Command("ip", "rule", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("Failed to check existing rules: %v", err)
		return false, 0, 0
	}

	ruleOutput := string(output)
	logrus.Debugf("Current rules: %s", ruleOutput)

	for _, line := range strings.Split(ruleOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !lineMatchesSource(line, srcNet) {
			continue
		}
		priority, table, ok := parsePriorityAndTable(line)
		if !ok {
			continue
		}
		logrus.Debugf("Found existing rule: %s (priority: %d, table: %d)", line, priority, table)
		return true, priority, table
	}

	logrus.Debugf("No existing rule found for source %s", srcNet.String())
	return false, 0, 0
}

// removeAllRulesForSource removes all routing rules for a given source network
func (m *Manager) removeAllRulesForSource(srcNet *net.IPNet) error {
	removedCount := 0
	maxAttempts := 10 // Prevent infinite loops

	for attempt := 0; attempt < maxAttempts; attempt++ {
		cmd := exec.Command("ip", "rule", "show")
		output, err := cmd.CombinedOutput()
		if err != nil {
			logrus.Warnf("Failed to check existing rules: %v", err)
			return err
		}

		foundRule := false
		for _, line := range strings.Split(string(output), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !lineMatchesSource(line, srcNet) {
				continue
			}
			from := extractFromToken(line)
			if from == "" {
				continue
			}
			logrus.Infof("Removing rule for source %s: %s", srcNet.String(), line)
			if err := deleteRuleFrom(from); err != nil {
				logrus.Warnf("Failed to remove rule: %v", err)
			} else {
				removedCount++
				foundRule = true
				break // Remove one rule at a time
			}
		}

		if !foundRule {
			break
		}
	}

	if removedCount > 0 {
		logrus.Infof("Removed %d rules for source %s", removedCount, srcNet.String())
	}

	return nil
}

// removeRoutingRule removes a routing rule for a given source network
func (m *Manager) removeRoutingRule(srcNet *net.IPNet) error {
	exists, _, _ := m.checkRoutingRuleExists(srcNet)
	if !exists {
		logrus.Debugf("No rule to remove for source %s", srcNet.String())
		return nil
	}

	if err := m.removeAllRulesForSource(srcNet); err != nil {
		return fmt.Errorf("failed to remove routing rule: %v", err)
	}

	logrus.Infof("Removed routing rule for source %s", srcNet.String())

	// Clear conntrack entries for this source network to ensure connections stop using the old routing
	if err := m.clearConntrack(srcNet); err != nil {
		logrus.Warnf("Failed to clear conntrack entries for %s: %v", srcNet.String(), err)
	}

	return nil
}

// addRoutingRule adds a routing rule for a given source network and table
func (m *Manager) addRoutingRule(srcNet *net.IPNet, tableID int) error {
	priority := calculatePriority(srcNet)

	cmd := exec.Command("ip", "rule", "add", "priority", strconv.Itoa(priority), "table", strconv.Itoa(tableID), "from", srcNet.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Errorf("Command failed: %v", err)
		logrus.Errorf("Command output: %s", string(output))
		return fmt.Errorf("failed to add routing rule: %v", err)
	}

	logrus.Infof("Added routing rule: priority %d, source %s, table %d", priority, srcNet.String(), tableID)

	// Clear conntrack entries for this source network to ensure new connections use the updated routing
	if err := m.clearConntrack(srcNet); err != nil {
		logrus.Warnf("Failed to clear conntrack entries for %s: %v", srcNet.String(), err)
	}

	return nil
}

// clearConntrack clears conntrack entries for a given source network
func (m *Manager) clearConntrack(srcNet *net.IPNet) error {
	cmd := exec.Command("conntrack", "-D", "--src", srcNet.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		// It's okay if there are no entries to delete
		logrus.Debugf("Conntrack clear result for %s: %s", srcNet.String(), string(output))
		return nil
	}

	// conntrack exits 0 even when nothing is deleted (e.g. "0 flow entries have been deleted").
	// Avoid noisy INFO logs during periodic sync when policies are disabled/removed.
	out := strings.ToLower(string(output))
	if strings.Contains(out, "0 flow") || strings.Contains(out, "0 entries") {
		logrus.Debugf("No conntrack entries to clear for source %s", srcNet.String())
		return nil
	}

	logrus.Infof("Cleared conntrack entries for source %s", srcNet.String())
	return nil
}

// cleanupStaleRules removes routing rules for policies that no longer exist in the configuration
func (m *Manager) cleanupStaleRules(activePolicies []*models.RoutingPolicy) error {
	// Get all current routing rules
	cmd := exec.Command("ip", "rule", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("Failed to get current rules for cleanup: %v", err)
		return err
	}

	// Create a set of active policy source networks (canonical CIDR form)
	activeSources := make(map[string]bool)
	for _, policy := range activePolicies {
		srcNet, err := parsePolicySource(policy.ID)
		if err != nil {
			logrus.Warnf("Invalid policy ID as source IP/CIDR: %s", policy.ID)
			continue
		}
		activeSources[srcNet.String()] = true
	}

	// Parse rules and remove those that don't correspond to active policies
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		priorityStr := strings.TrimSuffix(parts[0], ":")
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			continue // Skip lines that don't have valid priority
		}

		// Only manage rules in our priority range
		if priority < managedPriorityMin || priority > managedPriorityMax {
			continue // Skip rules outside our managed range
		}

		from := extractFromToken(line)
		if from == "" || from == "all" {
			continue
		}

		canonical := normalizeFromToken(from)
		if activeSources[canonical] {
			continue
		}

		logrus.Infof("Removing stale rule for inactive policy: %s (priority: %d)", line, priority)
		if err := deleteRuleFrom(from); err != nil {
			logrus.Warnf("Failed to remove stale rule: %v", err)
		}
	}

	return nil
}

// cleanupDuplicateRules removes duplicate rules for the same IP/CIDR, keeping only the first one
func (m *Manager) cleanupDuplicateRules() error {
	logrus.Info("Cleaning up duplicate routing rules")

	// Get all current routing rules
	cmd := exec.Command("ip", "rule", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("Failed to get current rules for cleanup: %v", err)
		return err
	}

	// Track seen source IPs and their rules
	sourceRules := make(map[string][]string)
	lines := strings.Split(string(output), "\n")

	// Parse all rules and group by source (only for our managed priority range)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		priorityStr := strings.TrimSuffix(parts[0], ":")
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			continue // Skip lines that don't have valid priority
		}

		if priority < managedPriorityMin || priority > managedPriorityMax {
			continue
		}

		from := extractFromToken(line)
		if from == "" || from == "all" {
			continue
		}
		key := normalizeFromToken(from)
		sourceRules[key] = append(sourceRules[key], line)
	}

	// Remove duplicate rules, keeping only the first one for each source
	removedCount := 0
	for srcKey, rules := range sourceRules {
		if len(rules) <= 1 {
			continue
		}
		logrus.Infof("Found %d duplicate rules for source %s, keeping first one", len(rules), srcKey)

		// Keep the first rule, remove the rest
		for i := 1; i < len(rules); i++ {
			rule := rules[i]
			from := extractFromToken(rule)
			if from == "" {
				continue
			}
			logrus.Infof("Removing duplicate rule: %s", rule)
			if err := deleteRuleFrom(from); err != nil {
				logrus.Warnf("Failed to remove duplicate rule: %v", err)
			} else {
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		logrus.Infof("Cleanup completed: removed %d duplicate routing rules", removedCount)
	} else {
		logrus.Info("No duplicate rules found")
	}

	return nil
}

// suppressDefaultRulePriority is the priority of the "fall through to main but
// ignore its default route" rule. It must sit BEFORE the per-policy rules
// (which live in managedPriorityMin–managedPriorityMax) so local traffic to other LAN subnets always
// resolves via the main table, while default-route traffic falls through to
// the policy rules and out the chosen provider table.
const suppressDefaultRulePriority = 10

// suppressDefaultRuleSignature is the substring we look for in `ip rule show`
// output to detect that the rule is already installed (regardless of who
// installed it: us on a previous run, an operator, etc.).
const suppressDefaultRuleSignature = "from all lookup main suppress_prefixlength 0"

// EnsureSuppressDefaultRule installs the global "lookup main with
// suppress_prefixlength 0" rule at priority 10 if it is not already present.
// This makes policy-based routing safe for local LAN traffic: anything that
// matches a more-specific prefix in the main table (LAN, docker, tailscale,
// etc.) stays in main, and only default-route traffic falls through to the
// per-source policy rules.
func (m *Manager) EnsureSuppressDefaultRule() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	present, err := m.hasSuppressDefaultRule()
	if err != nil {
		return fmt.Errorf("failed to check suppress-default rule: %w", err)
	}
	if present {
		logrus.Debugf("Suppress-default rule already present at priority %d", suppressDefaultRulePriority)
		return nil
	}

	logrus.Infof("Installing suppress-default rule: priority=%d, lookup main, suppress_prefixlength=0",
		suppressDefaultRulePriority)

	cmd := exec.Command("ip", "rule", "add",
		"from", "all",
		"lookup", "main",
		"suppress_prefixlength", "0",
		"priority", strconv.Itoa(suppressDefaultRulePriority),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install suppress-default rule: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// RemoveSuppressDefaultRule deletes the rule installed by
// EnsureSuppressDefaultRule, matching on the full rule signature so we never
// remove an unrelated priority-10 rule the operator might have set.
func (m *Manager) RemoveSuppressDefaultRule() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	present, err := m.hasSuppressDefaultRule()
	if err != nil {
		return fmt.Errorf("failed to check suppress-default rule: %w", err)
	}
	if !present {
		logrus.Debug("Suppress-default rule not present; nothing to remove")
		return nil
	}

	logrus.Infof("Removing suppress-default rule at priority %d", suppressDefaultRulePriority)

	cmd := exec.Command("ip", "rule", "del",
		"from", "all",
		"lookup", "main",
		"suppress_prefixlength", "0",
		"priority", strconv.Itoa(suppressDefaultRulePriority),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove suppress-default rule: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// hasSuppressDefaultRule returns true if a rule at suppressDefaultRulePriority
// with the suppress-default signature is currently installed. Caller must hold
// m.mu.
func (m *Manager) hasSuppressDefaultRule() (bool, error) {
	cmd := exec.Command("ip", "rule", "show")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("ip rule show failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	prefix := strconv.Itoa(suppressDefaultRulePriority) + ":"
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		if strings.Contains(line, suppressDefaultRuleSignature) {
			return true, nil
		}
	}
	return false, nil
}

// CleanupAllRules removes all routing rules managed by this application
// (priority managedPriorityMin–managedPriorityMax).
func (m *Manager) CleanupAllRules() error {
	logrus.Infof("Cleaning up all routing rules (priority %d-%d)", managedPriorityMin, managedPriorityMax)

	// Get all current routing rules
	cmd := exec.Command("ip", "rule", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("Failed to get current rules for cleanup: %v", err)
		return err
	}

	// Parse rules and remove those in our managed range (delete by from, not priority)
	lines := strings.Split(string(output), "\n")
	removedCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		priorityStr := strings.TrimSuffix(parts[0], ":")
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			continue // Skip lines that don't have valid priority
		}

		if priority < managedPriorityMin || priority > managedPriorityMax {
			continue
		}

		from := extractFromToken(line)
		if from == "" || from == "all" {
			continue
		}

		logrus.Infof("Removing rule during cleanup: %s (priority: %d)", line, priority)
		if err := deleteRuleFrom(from); err != nil {
			logrus.Warnf("Failed to remove rule during cleanup: %v", err)
		} else {
			removedCount++
		}
	}

	logrus.Infof("Cleanup completed: removed %d routing rules", removedCount)
	return nil
}

// validateSingleRulePerSource validates that there's only one rule per IP/CIDR in the managed priority range
func (m *Manager) validateSingleRulePerSource() error {
	cmd := exec.Command("ip", "rule", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("Failed to get current rules for validation: %v", err)
		return err
	}

	// Track source IPs and their rules (only for our managed priority range)
	sourceRules := make(map[string][]string)
	lines := strings.Split(string(output), "\n")

	// Parse all rules and group by source IP
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		priorityStr := strings.TrimSuffix(parts[0], ":")
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			continue // Skip lines that don't have valid priority
		}

		if priority < managedPriorityMin || priority > managedPriorityMax {
			continue
		}

		from := extractFromToken(line)
		if from == "" || from == "all" {
			continue
		}
		key := normalizeFromToken(from)
		sourceRules[key] = append(sourceRules[key], line)
	}

	// Check for violations
	violations := 0
	for srcIP, rules := range sourceRules {
		if len(rules) > 1 {
			logrus.Warnf("VALIDATION VIOLATION: Found %d rules for source %s:", len(rules), srcIP)
			for i, rule := range rules {
				logrus.Warnf("  Rule %d: %s", i+1, rule)
			}
			violations++
		}
	}

	if violations > 0 {
		logrus.Warnf("Validation found %d sources with multiple rules", violations)
	} else {
		logrus.Debugf("Validation passed: all sources have single rules")
	}

	return nil
}
