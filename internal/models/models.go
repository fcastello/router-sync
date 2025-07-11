package models

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// InternetProvider represents an internet service provider
type InternetProvider struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	Interface   string    `json:"interface" yaml:"interface"`
	TableID     int       `json:"table_id" yaml:"table_id"`
	Gateway     string    `json:"gateway" yaml:"gateway"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
}

// RoutingPolicy represents a routing policy where the policy ID is used as the source IP
type RoutingPolicy struct {
	ID          string    `json:"id" yaml:"id"`
	Name        string    `json:"name" yaml:"name"`
	ProviderID  string    `json:"provider_id" yaml:"provider_id"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled     bool      `json:"enabled" yaml:"enabled"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
}

// Validate validates the InternetProvider
func (p *InternetProvider) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("provider ID is required")
	}
	if p.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if p.Interface == "" {
		return fmt.Errorf("provider interface is required")
	}
	if p.TableID <= 0 {
		return fmt.Errorf("provider table ID must be greater than 0")
	}
	if p.Gateway == "" {
		return fmt.Errorf("provider gateway is required")
	}

	// Validate gateway IP
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

	// Validate that policy ID is a valid IP address or CIDR notation
	_, _, err := net.ParseCIDR(p.ID)
	if err != nil {
		// Try as single IP
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
