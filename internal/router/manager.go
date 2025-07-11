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

// Manager manages routing tables and policies using netlink
type Manager struct {
	mu sync.RWMutex
}

// NewManager creates a new router manager
func NewManager() (*Manager, error) {
	return &Manager{}, nil
}

// SetupProvider sets up routing for an internet provider
func (m *Manager) SetupProvider(provider *models.InternetProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logrus.Infof("Setting up provider %s on interface %s with gateway %s",
		provider.Name, provider.Interface, provider.Gateway)

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

		// Parse policy ID as source IP/CIDR
		var srcNet *net.IPNet

		// Try to parse as CIDR first
		_, ipnet, err := net.ParseCIDR(policy.ID)
		if err != nil {
			// Try as single IP
			srcIP := net.ParseIP(policy.ID)
			if srcIP == nil {
				return fmt.Errorf("invalid policy ID as source IP/CIDR: %s", policy.ID)
			}
			// Create a /32 network for single IP
			srcNet = &net.IPNet{
				IP:   srcIP,
				Mask: net.CIDRMask(32, 32),
			}
		} else {
			srcNet = ipnet
		}

		// Remove all rules for this source IP and clear conntrack
		if err := m.removeAllRulesForSource(srcNet); err != nil {
			logrus.Warnf("Failed to remove rules for disabled policy %s: %v", policy.Name, err)
		}

		// Clear conntrack entries for this source network
		if err := m.clearConntrack(srcNet); err != nil {
			logrus.Warnf("Failed to clear conntrack entries for disabled policy %s: %v", policy.Name, err)
		}

		logrus.Debugf("Successfully disabled policy %s", policy.Name)
		return nil
	}

	// Log enabled policy at INFO level
	logrus.Infof("Policy: %s, Source: %s, Provider: %s", policy.Name, policy.ID, provider.Name)

	logrus.Debugf("SetupPolicy: Policy is enabled, proceeding with setup")
	logrus.Debugf("Setting up policy %s (ID: %s) to use provider %s (TableID: %d)",
		policy.Name, policy.ID, provider.Name, provider.TableID)

	// Parse policy ID as source IP/CIDR
	var srcNet *net.IPNet

	// Try to parse as CIDR first
	_, ipnet, err := net.ParseCIDR(policy.ID)
	if err != nil {
		// Try as single IP
		srcIP := net.ParseIP(policy.ID)
		if srcIP == nil {
			return fmt.Errorf("invalid policy ID as source IP/CIDR: %s", policy.ID)
		}
		// Create a /32 network for single IP
		srcNet = &net.IPNet{
			IP:   srcIP,
			Mask: net.CIDRMask(32, 32),
		}
	} else {
		srcNet = ipnet
	}

	logrus.Debugf("Parsed source network: %s", srcNet.String())

	// Check if a rule already exists for this source network
	exists, existingPriority, existingTable := m.checkRoutingRuleExists(srcNet)

	if exists {
		// If the rule exists and points to the correct table, no changes needed
		if existingTable == provider.TableID {
			logrus.Debugf("SKIPPING: Routing rule already exists and is correct for policy %s: priority=%d, table=%d, src=%s",
				policy.Name, existingPriority, existingTable, srcNet.String())
			return nil
		}

		// If the rule exists but points to a different table, remove all rules for this source
		logrus.Debugf("Policy changed: removing all rules for source %s and adding new rule (table: %d)",
			srcNet.String(), provider.TableID)
		if err := m.removeAllRulesForSource(srcNet); err != nil {
			return fmt.Errorf("failed to remove old routing rules for policy %s: %w", policy.Name, err)
		}
	}

	// Add routing rule using ip command
	logrus.Debugf("ADDING: New routing rule for policy %s: src=%s, table=%d", policy.Name, srcNet.String(), provider.TableID)
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

	// Parse policy ID as source IP/CIDR
	var srcNet *net.IPNet

	// Try to parse as CIDR first
	_, ipnet, err := net.ParseCIDR(policy.ID)
	if err != nil {
		// Try as single IP
		srcIP := net.ParseIP(policy.ID)
		if srcIP == nil {
			return fmt.Errorf("invalid policy ID as source IP/CIDR: %s", policy.ID)
		}
		// Create a /32 network for single IP
		srcNet = &net.IPNet{
			IP:   srcIP,
			Mask: net.CIDRMask(32, 32),
		}
	} else {
		srcNet = ipnet
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

	// Set up new routes
	for _, provider := range providers {
		logrus.Debugf("Setting up provider: %s", provider.Name)
		if err := m.SetupProvider(provider); err != nil {
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

// calculatePriority calculates the priority based on CIDR specificity
// More specific CIDRs get lower priority numbers (higher priority)
// calculatePriority calculates the priority based on CIDR specificity
// More specific CIDRs get lower priority numbers (higher priority)
// /32 = 32 bits = priority 2000
// /31 = 31 bits = priority 2001
// /30 = 30 bits = priority 2002
// /29 = 29 bits = priority 2003
// /28 = 28 bits = priority 2004
// /27 = 27 bits = priority 2005
// /26 = 26 bits = priority 2006
// /25 = 25 bits = priority 2007
// /24 = 24 bits = priority 2008
// /23 = 23 bits = priority 2009
// /22 = 22 bits = priority 2010
// /21 = 21 bits = priority 2011
// /20 = 20 bits = priority 2012
// /19 = 19 bits = priority 2013
// /18 = 18 bits = priority 2014
// /17 = 17 bits = priority 2015
// /16 = 16 bits = priority 2016
// /15 = 15 bits = priority 2017
// /14 = 14 bits = priority 2018
// /13 = 13 bits = priority 2019
// /12 = 12 bits = priority 2020
// /11 = 11 bits = priority 2021
// /10 = 10 bits = priority 2022
// /9 = 9 bits = priority 2023
// /8 = 8 bits = priority 2024
// /7 = 7 bits = priority 2025
// /6 = 6 bits = priority 2026
// /5 = 5 bits = priority 2027
// /4 = 4 bits = priority 2028
// /3 = 3 bits = priority 2029
// /2 = 2 bits = priority 2030
// /1 = 1 bit = priority 2031
// /0 = 0 bits = priority 2032
func calculatePriority(srcNet *net.IPNet) int {
	ones, _ := srcNet.Mask.Size()
	specificity := ones // Number of network bits
	return 2000 + (32 - specificity)
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

	// Look for any rule with our source network
	lines := strings.Split(ruleOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse line format: "100: from 192.168.2.25 lookup 99"
		// The rule output shows IP without CIDR suffix, so we need to match just the IP part
		srcIP := srcNet.IP.String()
		if strings.Contains(line, fmt.Sprintf("from %s", srcIP)) {
			// Extract priority and table from the rule
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				priorityStr := strings.TrimSuffix(parts[0], ":")
				tableStr := parts[len(parts)-1]

				priority, _ := strconv.Atoi(priorityStr)
				table, _ := strconv.Atoi(tableStr)

				logrus.Debugf("Found existing rule: %s (priority: %d, table: %d)", line, priority, table)
				return true, priority, table
			}
		}
	}

	logrus.Debugf("No existing rule found for source %s", srcNet.String())
	return false, 0, 0
}

// removeAllRulesForSource removes all routing rules for a given source network
func (m *Manager) removeAllRulesForSource(srcNet *net.IPNet) error {
	srcIP := srcNet.IP.String()
	removedCount := 0
	maxAttempts := 10 // Prevent infinite loops

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Get current rules
		cmd := exec.Command("ip", "rule", "show")
		output, err := cmd.CombinedOutput()
		if err != nil {
			logrus.Warnf("Failed to check existing rules: %v", err)
			return err
		}

		ruleOutput := string(output)
		lines := strings.Split(ruleOutput, "\n")
		foundRule := false

		// Look for rules with our source network
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Check if this rule is for our specific source IP
			if strings.Contains(line, fmt.Sprintf("from %s", srcIP)) {
				// Extract priority from the rule
				parts := strings.Fields(line)
				if len(parts) >= 4 {
					priorityStr := strings.TrimSuffix(parts[0], ":")
					priority, _ := strconv.Atoi(priorityStr)

					logrus.Infof("Removing rule for source %s: %s (priority: %d)", srcIP, line, priority)

					// Remove the rule by source IP/CIDR instead of priority
					// This is safer as it only removes rules for this specific source
					cmd := exec.Command("ip", "rule", "del", "from", srcNet.String())
					if err := cmd.Run(); err != nil {
						logrus.Warnf("Failed to remove rule: %v", err)
					} else {
						removedCount++
						foundRule = true
						break // Remove one rule at a time
					}
				}
			}
		}

		// If no rule was found or removed, we're done
		if !foundRule {
			break
		}
	}

	if removedCount > 0 {
		logrus.Infof("Removed %d rules for source %s", removedCount, srcIP)
	}

	return nil
}

// removeRoutingRule removes a routing rule for a given source network
func (m *Manager) removeRoutingRule(srcNet *net.IPNet) error {
	exists, priority, _ := m.checkRoutingRuleExists(srcNet)
	if !exists {
		logrus.Debugf("No rule to remove for source %s", srcNet.String())
		return nil
	}

	cmd := exec.Command("ip", "rule", "del", "priority", strconv.Itoa(priority))
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("Failed to remove routing rule: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to remove routing rule: %v", err)
	}

	logrus.Infof("Removed routing rule for source %s (priority: %d)", srcNet.String(), priority)

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

	// Create a set of active policy source networks
	activeSources := make(map[string]bool)
	for _, policy := range activePolicies {
		// Parse policy ID as source IP/CIDR
		var srcNet *net.IPNet
		_, ipnet, err := net.ParseCIDR(policy.ID)
		if err != nil {
			// Try as single IP
			srcIP := net.ParseIP(policy.ID)
			if srcIP == nil {
				logrus.Warnf("Invalid policy ID as source IP/CIDR: %s", policy.ID)
				continue
			}
			// Create a /32 network for single IP
			srcNet = &net.IPNet{
				IP:   srcIP,
				Mask: net.CIDRMask(32, 32),
			}
		} else {
			srcNet = ipnet
		}
		activeSources[srcNet.IP.String()] = true
	}

	// Parse rules and remove those that don't correspond to active policies
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract priority to check if it's in our managed range (100-132)
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		priorityStr := strings.TrimSuffix(parts[0], ":")
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			continue // Skip lines that don't have valid priority
		}

		// Only manage rules in our priority range (2000-2032)
		if priority < 2000 || priority > 2032 {
			continue // Skip rules outside our managed range
		}

		// Skip default rules that might be in our range
		if strings.HasPrefix(line, "0:") || strings.HasPrefix(line, "32766:") || strings.HasPrefix(line, "32767:") {
			continue
		}

		// Parse line format: "100: from 192.168.2.25 lookup 99"
		if strings.Contains(line, "from") && strings.Contains(line, "lookup") {
			// Extract source IP from the rule
			srcIP := ""
			for i, part := range parts {
				if part == "from" && i+1 < len(parts) {
					srcIP = parts[i+1]
					break
				}
			}

			if srcIP != "" {
				// Check if this source IP matches any active policy
				// We need to check both the exact match and the IP part (for CIDR rules)
				found := false
				if activeSources[srcIP] {
					found = true
				} else {
					// For CIDR rules, also check the IP part without CIDR
					// e.g., if rule shows "192.168.2.0/25", also check "192.168.2.0"
					if strings.Contains(srcIP, "/") {
						ipPart := strings.Split(srcIP, "/")[0]
						if activeSources[ipPart] {
							found = true
						}
					}
				}

				if !found {
					// This rule is for a policy that no longer exists
					logrus.Infof("Removing stale rule for inactive policy: %s (priority: %d)", line, priority)

					cmd := exec.Command("ip", "rule", "del", "priority", strconv.Itoa(priority))
					if err := cmd.Run(); err != nil {
						logrus.Warnf("Failed to remove stale rule: %v", err)
					}
				}
			}
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

	// Parse all rules and group by source IP (only for our managed priority range 2000-2032)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract priority to check if it's in our managed range (2000-2032)
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		priorityStr := strings.TrimSuffix(parts[0], ":")
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			continue // Skip lines that don't have valid priority
		}

		// Only process rules in our managed range (2000-2032)
		if priority < 2000 || priority > 2032 {
			continue
		}

		// Extract source IP from the rule
		if strings.Contains(line, "from") && strings.Contains(line, "lookup") {
			for i, part := range parts {
				if part == "from" && i+1 < len(parts) {
					srcIP := parts[i+1]
					sourceRules[srcIP] = append(sourceRules[srcIP], line)
					break
				}
			}
		}
	}

	// Remove duplicate rules, keeping only the first one for each source IP
	removedCount := 0
	for srcIP, rules := range sourceRules {
		if len(rules) > 1 {
			logrus.Infof("Found %d duplicate rules for source %s, keeping first one", len(rules), srcIP)

			// Keep the first rule, remove the rest
			for i := 1; i < len(rules); i++ {
				rule := rules[i]
				parts := strings.Fields(rule)
				if len(parts) >= 1 {
					priorityStr := strings.TrimSuffix(parts[0], ":")
					priority, _ := strconv.Atoi(priorityStr)

					logrus.Infof("Removing duplicate rule: %s (priority: %d)", rule, priority)

					cmd := exec.Command("ip", "rule", "del", "priority", strconv.Itoa(priority))
					if err := cmd.Run(); err != nil {
						logrus.Warnf("Failed to remove duplicate rule: %v", err)
					} else {
						removedCount++
					}
				}
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

// CleanupAllRules removes all routing rules managed by this application (priority 2000-2032)
func (m *Manager) CleanupAllRules() error {
	logrus.Info("Cleaning up all routing rules (priority 2000-2032)")

	// Get all current routing rules
	cmd := exec.Command("ip", "rule", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("Failed to get current rules for cleanup: %v", err)
		return err
	}

	// Parse rules and remove those in our managed range
	lines := strings.Split(string(output), "\n")
	removedCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract priority to check if it's in our managed range (2000-2032)
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		priorityStr := strings.TrimSuffix(parts[0], ":")
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			continue // Skip lines that don't have valid priority
		}

		// Only remove rules in our managed range (2000-2032)
		if priority >= 2000 && priority <= 2032 {
			logrus.Infof("Removing rule during cleanup: %s (priority: %d)", line, priority)

			cmd := exec.Command("ip", "rule", "del", "priority", strconv.Itoa(priority))
			if err := cmd.Run(); err != nil {
				logrus.Warnf("Failed to remove rule during cleanup: %v", err)
			} else {
				removedCount++
			}
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

	// Track source IPs and their rules (only for our managed priority range 2000-2032)
	sourceRules := make(map[string][]string)
	lines := strings.Split(string(output), "\n")

	// Parse all rules and group by source IP
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract priority to check if it's in our managed range (2000-2032)
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		priorityStr := strings.TrimSuffix(parts[0], ":")
		priority, err := strconv.Atoi(priorityStr)
		if err != nil {
			continue // Skip lines that don't have valid priority
		}

		// Only process rules in our managed range (2000-2032)
		if priority < 2000 || priority > 2032 {
			continue
		}

		// Extract source IP from the rule
		if strings.Contains(line, "from") && strings.Contains(line, "lookup") {
			for i, part := range parts {
				if part == "from" && i+1 < len(parts) {
					srcIP := parts[i+1]
					// Ignore 'from all' system rules
					if srcIP == "all" {
						break
					}
					sourceRules[srcIP] = append(sourceRules[srcIP], line)
					break
				}
			}
		}
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
