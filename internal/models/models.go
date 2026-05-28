package models

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// InternetProvider represents an internet service provider.
//
// Interfaces maps a router hostname to the interface name used on that router
// (e.g. {"r1":"enp1s0","r2":"enp2s0"}). All routers use the same TableID and Gateway.
// Interface is deprecated and kept only for backward compatibility with existing
// records — it is auto-migrated into Interfaces on the next write.
type InternetProvider struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Interfaces  map[string]string `json:"interfaces,omitempty" yaml:"interfaces,omitempty"`
	Interface   string            `json:"interface,omitempty" yaml:"interface,omitempty"` // deprecated
	TableID     int               `json:"table_id" yaml:"table_id"`
	Gateway     string            `json:"gateway" yaml:"gateway"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Generation  uint64            `json:"generation" yaml:"generation"`
	WriterID    string            `json:"writer_id" yaml:"writer_id"`
	CreatedAt   time.Time         `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" yaml:"updated_at"`
}

// InterfaceForHost returns the interface name to use on the given router.
// Falls back to the legacy Interface field if no per-router mapping exists.
func (p *InternetProvider) InterfaceForHost(hostname string) string {
	if p.Interfaces != nil {
		if iface, ok := p.Interfaces[hostname]; ok && iface != "" {
			return iface
		}
	}
	return p.Interface
}

// HasInterfaceForHost returns true if the provider has an interface assigned for the host.
func (p *InternetProvider) HasInterfaceForHost(hostname string) bool {
	return p.InterfaceForHost(hostname) != ""
}

// RoutingPolicy represents a routing policy where the policy ID is used as the source IP
type RoutingPolicy struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	ProviderID  string    `json:"provider_id" yaml:"provider_id"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled     bool      `json:"enabled" yaml:"enabled"`
	Generation  uint64    `json:"generation" yaml:"generation"`
	WriterID    string    `json:"writer_id" yaml:"writer_id"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
}

// RouterState is the per-router heartbeat snapshot stored in the router-sync-state KV bucket.
type RouterState struct {
	Hostname     string         `json:"hostname"`
	AgentVersion string         `json:"agent_version"`
	LogLevel     string         `json:"log_level"`
	LastSeen     time.Time      `json:"last_seen"`
	Interfaces   []Interface    `json:"interfaces"`
	Tables       []RoutingTable `json:"tables"`
	Rules        []IPRule       `json:"rules"`
}

// Interface is a snapshot of a single network interface on a router.
type Interface struct {
	Name      string   `json:"name"`
	MAC       string   `json:"mac,omitempty"`
	MTU       int      `json:"mtu"`
	Up        bool     `json:"up"`
	Addresses []string `json:"addresses"`
}

// RoutingTable contains the routes installed in a kernel routing table.
type RoutingTable struct {
	ID     int     `json:"id"`
	Name   string  `json:"name,omitempty"`
	Routes []Route `json:"routes"`
}

// Route is a single routing table entry.
type Route struct {
	Dst       string `json:"dst"`     // "default" or CIDR
	Gateway   string `json:"gateway,omitempty"`
	Interface string `json:"interface,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Metric    int    `json:"metric,omitempty"`
}

// IPRule is a single `ip rule` entry.
type IPRule struct {
	Priority int    `json:"priority"`
	From     string `json:"from"`
	Table    int    `json:"table"`
	TableName string `json:"table_name,omitempty"`
}

// Validate validates the InternetProvider
func (p *InternetProvider) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("provider ID is required")
	}
	if p.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if len(p.Interfaces) == 0 && p.Interface == "" {
		return fmt.Errorf("provider requires at least one interface (interfaces map or legacy interface)")
	}
	if p.TableID <= 0 {
		return fmt.Errorf("provider table ID must be greater than 0")
	}
	if p.Gateway == "" {
		return fmt.Errorf("provider gateway is required")
	}

	if net.ParseIP(p.Gateway) == nil {
		return fmt.Errorf("invalid gateway IP address: %s", p.Gateway)
	}

	return nil
}

// Validate validates the RoutingPolicy
func (p *RoutingPolicy) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("policy ID is required")
	}
	if p.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if p.ProviderID == "" {
		return fmt.Errorf("provider ID is required")
	}

	_, _, err := net.ParseCIDR(p.ID)
	if err != nil {
		if net.ParseIP(p.ID) == nil {
			return fmt.Errorf("policy ID must be a valid IP address or CIDR notation: %s", p.ID)
		}
	}

	return nil
}

// ToJSON converts the model to JSON
func (p *InternetProvider) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// ToJSON converts the model to JSON
func (p *RoutingPolicy) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// FromJSON populates the model from JSON
func (p *InternetProvider) FromJSON(data []byte) error {
	return json.Unmarshal(data, p)
}

// FromJSON populates the model from JSON
func (p *RoutingPolicy) FromJSON(data []byte) error {
	return json.Unmarshal(data, p)
}

// ToJSON converts the RouterState to JSON.
func (r *RouterState) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON populates RouterState from JSON.
func (r *RouterState) FromJSON(data []byte) error {
	return json.Unmarshal(data, r)
}
