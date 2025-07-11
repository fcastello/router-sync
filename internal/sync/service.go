package sync

import (
	"context"
	"sync"
	"time"

	"router-sync/internal/config"
	"router-sync/internal/models"
	"router-sync/internal/nats"
	"router-sync/internal/router"

	natsio "github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

// Service handles synchronization between NATS KV store and router configuration
type Service struct {
	natsClient    *nats.Client
	routerManager *router.Manager
	config        config.SyncConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Cache for current state
	providers map[string]*models.InternetProvider
	policies  map[string]*models.RoutingPolicy
	cacheMu   sync.RWMutex
}

// NewService creates a new sync service
func NewService(natsClient *nats.Client, routerManager *router.Manager, config config.SyncConfig) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		natsClient:    natsClient,
		routerManager: routerManager,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		providers:     make(map[string]*models.InternetProvider),
		policies:      make(map[string]*models.RoutingPolicy),
	}
}

// Start starts the sync service
func (s *Service) Start() error {
	logrus.Info("Starting sync service")

	// Initial sync
	if err := s.performFullSync(); err != nil {
		logrus.Errorf("Initial sync failed: %v", err)
	}

	// Start periodic sync
	s.wg.Add(1)
	go s.periodicSync()

	// Start watchers
	s.wg.Add(1)
	go s.watchProviders()

	s.wg.Add(1)
	go s.watchPolicies()

	logrus.Info("Sync service started")
	return nil
}

// Stop stops the sync service
func (s *Service) Stop() error {
	logrus.Info("Stopping sync service")

	s.cancel()
	s.wg.Wait()

	logrus.Info("Sync service stopped")
	return nil
}

// periodicSync performs periodic synchronization
func (s *Service) periodicSync() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.performFullSync(); err != nil {
				logrus.Errorf("Periodic sync failed: %v", err)
			}
		}
	}
}

// performFullSync performs a full synchronization
func (s *Service) performFullSync() error {
	logrus.Debug("Performing full synchronization")

	// Get all providers from NATS
	providers, err := s.natsClient.ListProviders()
	if err != nil {
		logrus.Errorf("Failed to list providers: %v", err)
		return err
	}
	logrus.Debugf("Loaded %d providers from NATS", len(providers))

	// Get all policies from NATS
	policies, err := s.natsClient.ListPolicies()
	if err != nil {
		logrus.Errorf("Failed to list policies: %v", err)
		return err
	}
	logrus.Debugf("Loaded %d policies from NATS", len(policies))

	// Update cache
	s.cacheMu.Lock()
	s.providers = make(map[string]*models.InternetProvider)
	for _, provider := range providers {
		s.providers[provider.ID] = provider
		logrus.Debugf("Cached provider: %s (ID: %s)", provider.Name, provider.ID)
	}

	s.policies = make(map[string]*models.RoutingPolicy)
	for _, policy := range policies {
		s.policies[policy.ID] = policy
		logrus.Debugf("Cached policy: %s (ID: %s, ProviderID: %s)", policy.Name, policy.ID, policy.ProviderID)
	}
	s.cacheMu.Unlock()

	// Only sync policies, skip provider sync
	logrus.Info("SYNC START")
	logrus.Debugf("About to call SyncPolicies with %d policies and %d providers", len(policies), len(providers))
	if err := s.routerManager.SyncPolicies(policies, providers); err != nil {
		logrus.Errorf("Failed to sync policies: %v", err)
	}

	logrus.Info("SYNC FINISHED")
	return nil
}

// watchProviders watches for provider changes
func (s *Service) watchProviders() {
	defer s.wg.Done()

	err := s.natsClient.WatchProviders(s.ctx, func(provider *models.InternetProvider, op natsio.KeyValueOp) {
		s.cacheMu.Lock()
		defer s.cacheMu.Unlock()

		switch op {
		case natsio.KeyValuePut:
			if provider != nil {
				s.providers[provider.ID] = provider
				logrus.Infof("Provider updated: %s", provider.Name)
				// Skip provider sync - only cache the provider
			}
		case natsio.KeyValueDelete:
			if provider != nil {
				delete(s.providers, provider.ID)
				logrus.Infof("Provider deleted: %s", provider.Name)
				// Skip provider sync - only remove from cache
			}
		}
	})

	if err != nil {
		logrus.Errorf("Provider watcher error: %v", err)
	}
}

// watchPolicies watches for policy changes
func (s *Service) watchPolicies() {
	defer s.wg.Done()

	err := s.natsClient.WatchPolicies(s.ctx, func(policy *models.RoutingPolicy, op natsio.KeyValueOp) {
		s.cacheMu.Lock()
		defer s.cacheMu.Unlock()

		switch op {
		case natsio.KeyValuePut:
			if policy != nil {
				s.policies[policy.ID] = policy
				logrus.Infof("Policy updated: %s", policy.Name)

				// Get the provider
				provider, exists := s.providers[policy.ProviderID]
				if !exists {
					logrus.Warnf("Provider %s not found for policy %s", policy.ProviderID, policy.Name)
					return
				}

				// Apply the change to router
				if err := s.routerManager.SetupPolicy(policy, provider); err != nil {
					logrus.Errorf("Failed to set up policy %s: %v", policy.Name, err)
				}
			}
		case natsio.KeyValueDelete:
			if policy != nil {
				delete(s.policies, policy.ID)
				logrus.Infof("Policy deleted: %s", policy.Name)

				// Get the provider
				provider, exists := s.providers[policy.ProviderID]
				if !exists {
					logrus.Warnf("Provider %s not found for policy %s", policy.ProviderID, policy.Name)
					return
				}

				// Remove from router
				if err := s.routerManager.RemovePolicy(policy, provider); err != nil {
					logrus.Errorf("Failed to remove policy %s: %v", policy.Name, err)
				}
			}
		}
	})

	if err != nil {
		logrus.Errorf("Policy watcher error: %v", err)
	}
}

// GetStats returns synchronization statistics
func (s *Service) GetStats() map[string]interface{} {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	stats := make(map[string]interface{})
	stats["providers_count"] = len(s.providers)
	stats["policies_count"] = len(s.policies)
	stats["sync_interval"] = s.config.Interval.String()

	// Count policies per provider
	policiesPerProvider := make(map[string]int)
	for _, policy := range s.policies {
		policiesPerProvider[policy.ProviderID]++
	}
	stats["policies_per_provider"] = policiesPerProvider

	return stats
}
