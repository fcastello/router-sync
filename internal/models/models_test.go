package models

import (
	"testing"
	"time"
)

func TestInternetProvider_Validate(t *testing.T) {
	tests := []struct {
		name     string
		provider *InternetProvider
		wantErr  bool
	}{
		{
			name: "valid provider",
			provider: &InternetProvider{
				ID:        "test-1",
				Name:      "Test Provider",
				Interface: "eth0",
				TableID:   100,
				Gateway:   "192.168.1.1",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			provider: &InternetProvider{
				Name:      "Test Provider",
				Interface: "eth0",
				TableID:   100,
				Gateway:   "192.168.1.1",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			provider: &InternetProvider{
				ID:        "test-1",
				Interface: "eth0",
				TableID:   100,
				Gateway:   "192.168.1.1",
			},
			wantErr: true,
		},
		{
			name: "missing interface",
			provider: &InternetProvider{
				ID:      "test-1",
				Name:    "Test Provider",
				TableID: 100,
				Gateway: "192.168.1.1",
			},
			wantErr: true,
		},
		{
			name: "invalid table ID",
			provider: &InternetProvider{
				ID:        "test-1",
				Name:      "Test Provider",
				Interface: "eth0",
				TableID:   0,
				Gateway:   "192.168.1.1",
			},
			wantErr: true,
		},
		{
			name: "missing gateway",
			provider: &InternetProvider{
				ID:        "test-1",
				Name:      "Test Provider",
				Interface: "eth0",
				TableID:   100,
			},
			wantErr: true,
		},
		{
			name: "invalid gateway IP",
			provider: &InternetProvider{
				ID:        "test-1",
				Name:      "Test Provider",
				Interface: "eth0",
				TableID:   100,
				Gateway:   "invalid-ip",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("InternetProvider.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoutingPolicy_Validate(t *testing.T) {
	tests := []struct {
		name    string
		policy  *RoutingPolicy
		wantErr bool
	}{
		{
			name: "valid policy with single IP ID",
			policy: &RoutingPolicy{
				ID:         "192.168.1.100",
				Name:       "Test Policy",
				ProviderID: "provider-1",
			},
			wantErr: false,
		},
		{
			name: "valid policy with CIDR ID",
			policy: &RoutingPolicy{
				ID:         "192.168.1.0/24",
				Name:       "Test Policy",
				ProviderID: "provider-1",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			policy: &RoutingPolicy{
				Name:       "Test Policy",
				ProviderID: "provider-1",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			policy: &RoutingPolicy{
				ID:         "192.168.1.100",
				ProviderID: "provider-1",
			},
			wantErr: true,
		},
		{
			name: "missing provider ID",
			policy: &RoutingPolicy{
				ID:   "192.168.1.100",
				Name: "Test Policy",
			},
			wantErr: true,
		},
		{
			name: "invalid ID (not IP or CIDR)",
			policy: &RoutingPolicy{
				ID:         "invalid-id",
				Name:       "Test Policy",
				ProviderID: "provider-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RoutingPolicy.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInternetProvider_JSON(t *testing.T) {
	provider := &InternetProvider{
		ID:          "test-1",
		Name:        "Test Provider",
		Interface:   "eth0",
		TableID:     100,
		Gateway:     "192.168.1.1",
		Description: "Test description",
		CreatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Test ToJSON
	jsonData, err := provider.ToJSON()
	if err != nil {
		t.Errorf("ToJSON() error = %v", err)
		return
	}

	// Test FromJSON
	var newProvider InternetProvider
	err = newProvider.FromJSON(jsonData)
	if err != nil {
		t.Errorf("FromJSON() error = %v", err)
		return
	}

	// Compare fields
	if provider.ID != newProvider.ID {
		t.Errorf("ID mismatch: got %v, want %v", newProvider.ID, provider.ID)
	}
	if provider.Name != newProvider.Name {
		t.Errorf("Name mismatch: got %v, want %v", newProvider.Name, provider.Name)
	}
	if provider.Interface != newProvider.Interface {
		t.Errorf("Interface mismatch: got %v, want %v", newProvider.Interface, provider.Interface)
	}
	if provider.TableID != newProvider.TableID {
		t.Errorf("TableID mismatch: got %v, want %v", newProvider.TableID, provider.TableID)
	}
	if provider.Gateway != newProvider.Gateway {
		t.Errorf("Gateway mismatch: got %v, want %v", newProvider.Gateway, provider.Gateway)
	}
	if provider.Description != newProvider.Description {
		t.Errorf("Description mismatch: got %v, want %v", newProvider.Description, provider.Description)
	}
}

func TestRoutingPolicy_JSON(t *testing.T) {
	policy := &RoutingPolicy{
		ID:          "192.168.1.100",
		Name:        "Test Policy",
		ProviderID:  "provider-1",
		Description: "Test description",
		Enabled:     true,
		CreatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Test ToJSON
	jsonData, err := policy.ToJSON()
	if err != nil {
		t.Errorf("ToJSON() error = %v", err)
		return
	}

	// Test FromJSON
	var newPolicy RoutingPolicy
	err = newPolicy.FromJSON(jsonData)
	if err != nil {
		t.Errorf("FromJSON() error = %v", err)
		return
	}

	// Compare fields
	if policy.ID != newPolicy.ID {
		t.Errorf("ID mismatch: got %v, want %v", newPolicy.ID, policy.ID)
	}
	if policy.Name != newPolicy.Name {
		t.Errorf("Name mismatch: got %v, want %v", newPolicy.Name, policy.Name)
	}
	if policy.ProviderID != newPolicy.ProviderID {
		t.Errorf("ProviderID mismatch: got %v, want %v", newPolicy.ProviderID, policy.ProviderID)
	}
	if policy.Description != newPolicy.Description {
		t.Errorf("Description mismatch: got %v, want %v", newPolicy.Description, policy.Description)
	}
	if policy.Enabled != newPolicy.Enabled {
		t.Errorf("Enabled mismatch: got %v, want %v", newPolicy.Enabled, policy.Enabled)
	}
}
