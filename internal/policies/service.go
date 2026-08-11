package policies

import (
	"fmt"
	"time"

	"router-sync/internal/models"
)

// CreateRequest is the input for creating a routing policy.
type CreateRequest struct {
	Name        string
	SourceIP    string
	ProviderID  string
	Description string
	Tags        []string
	Enabled     bool
	Favorite    bool
}

// UpdateRequest is the input for replacing a routing policy.
type UpdateRequest struct {
	Name        string
	SourceIP    string
	ProviderID  string
	Description string
	Tags        []string
	Enabled     bool
	Favorite    bool
}

// Store is the minimal NATS surface needed for policy CRUD.
type Store interface {
	ListPolicies() ([]*models.RoutingPolicy, error)
	GetPolicy(id string) (*models.RoutingPolicy, error)
	StorePolicy(policy *models.RoutingPolicy) error
	DeletePolicy(id string) error
	GetProvider(id string) (*models.InternetProvider, error)
}

// Service performs routing policy CRUD against NATS (shared by REST and MCP).
type Service struct {
	store Store
}

// NewService creates a policy service backed by the given store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns all routing policies.
func (s *Service) List() ([]*models.RoutingPolicy, error) {
	return s.store.ListPolicies()
}

// Get returns a policy by ID (CIDR IDs use underscore in URLs, e.g. 192.168.1.0_24).
func (s *Service) Get(id string) (*models.RoutingPolicy, error) {
	return s.store.GetPolicy(id)
}

// Create stores a new policy.
func (s *Service) Create(req CreateRequest) (*models.RoutingPolicy, error) {
	now := time.Now()
	policy := &models.RoutingPolicy{
		ID:          req.SourceIP,
		Name:        req.Name,
		ProviderID:  req.ProviderID,
		Description: req.Description,
		Tags:        models.NormalizeTags(req.Tags),
		Enabled:     req.Enabled,
		Favorite:    req.Favorite,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}
	if _, err := s.store.GetProvider(req.ProviderID); err != nil {
		return nil, fmt.Errorf("provider not found: %s", req.ProviderID)
	}
	if err := s.store.StorePolicy(policy); err != nil {
		return nil, err
	}
	return policy, nil
}

// Update replaces an existing policy (id is the current policy key in NATS).
func (s *Service) Update(id string, req UpdateRequest) (*models.RoutingPolicy, error) {
	existing, err := s.store.GetPolicy(id)
	if err != nil {
		return nil, err
	}
	existing.Name = req.Name
	existing.ID = req.SourceIP
	existing.ProviderID = req.ProviderID
	existing.Description = req.Description
	existing.Tags = models.NormalizeTags(req.Tags)
	existing.Enabled = req.Enabled
	existing.Favorite = req.Favorite
	existing.UpdatedAt = time.Now()

	if err := existing.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}
	if _, err := s.store.GetProvider(req.ProviderID); err != nil {
		return nil, fmt.Errorf("provider not found: %s", req.ProviderID)
	}
	if err := s.store.StorePolicy(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// Delete removes a policy by ID.
func (s *Service) Delete(id string) error {
	return s.store.DeletePolicy(id)
}
