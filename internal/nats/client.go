package nats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"router-sync/internal/config"
	"router-sync/internal/models"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

// NATSClient defines the interface for NATS operations used by the API
// This allows for mocking in tests.
type NATSClient interface {
	StoreProvider(provider *models.InternetProvider) error
	GetProvider(id string) (*models.InternetProvider, error)
	ListProviders() ([]*models.InternetProvider, error)
	DeleteProvider(id string) error

	StorePolicy(policy *models.RoutingPolicy) error
	GetPolicy(id string) (*models.RoutingPolicy, error)
	ListPolicies() ([]*models.RoutingPolicy, error)
	DeletePolicy(id string) error

	Close()
}

// Client represents a NATS client with key-value store capabilities
type Client struct {
	conn *nats.Conn
	js   nats.JetStreamContext
	kv   nats.KeyValue
}

// sanitizeKey sanitizes a key to be compatible with NATS key-value store
func sanitizeKey(key string) string {
	// NATS keys should only contain alphanumeric characters, dots, and underscores
	// Replace all invalid characters with underscores
	var result strings.Builder

	for _, char := range key {
		switch {
		case (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9'):
			// Alphanumeric characters are valid
			result.WriteRune(char)
		case char == '.' || char == '_':
			// Dots and underscores are valid
			result.WriteRune(char)
		default:
			// Replace all other characters with underscore
			result.WriteRune('_')
		}
	}

	sanitized := result.String()
	logrus.Debugf("Sanitized key: '%s' -> '%s'", key, sanitized)
	return sanitized
}

// NewClient creates a new NATS client
func NewClient(cfg config.NATSConfig) (*Client, error) {
	opts := []nats.Option{
		nats.Name(cfg.ClientID),
		nats.Timeout(10 * time.Second),
		nats.ReconnectWait(1 * time.Second),
		nats.MaxReconnects(5),
	}

	if cfg.Username != "" && cfg.Password != "" {
		opts = append(opts, nats.UserInfo(cfg.Username, cfg.Password))
	}

	if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	}

	conn, err := nats.Connect(cfg.URLs[0], opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Create or get the key-value store
	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket: "router-sync",
		TTL:    0, // No TTL for persistence
	})
	if err != nil {
		// Try to get existing bucket
		kv, err = js.KeyValue("router-sync")
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create/get key-value store: %w", err)
		}
	}

	client := &Client{
		conn: conn,
		js:   js,
		kv:   kv,
	}

	// Test the key-value store
	if err := client.testKeyValueStore(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("key-value store test failed: %w", err)
	}

	logrus.Info("Connected to NATS server")
	return client, nil
}

// Close closes the NATS connection
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// StoreProvider stores an internet provider in the key-value store
func (c *Client) StoreProvider(provider *models.InternetProvider) error {
	data, err := provider.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal provider: %w", err)
	}

	key := fmt.Sprintf("providers.%s", sanitizeKey(provider.ID))
	logrus.Debugf("Storing provider with key: %s (original ID: %s)", key, provider.ID)

	_, err = c.kv.Put(key, data)
	if err != nil {
		return fmt.Errorf("failed to store provider: %w", err)
	}

	logrus.Debugf("Stored provider %s", provider.ID)
	return nil
}

// GetProvider retrieves an internet provider from the key-value store
func (c *Client) GetProvider(id string) (*models.InternetProvider, error) {
	// Try with sanitized key first
	key := fmt.Sprintf("providers.%s", sanitizeKey(id))
	entry, err := c.kv.Get(key)
	if err != nil {
		// If that fails, try with the original ID (in case it was stored before sanitization)
		key = fmt.Sprintf("providers.%s", id)
		entry, err = c.kv.Get(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get provider: %w", err)
		}
	}

	var provider models.InternetProvider
	if err := provider.FromJSON(entry.Value()); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider: %w", err)
	}

	return &provider, nil
}

// ListProviders retrieves all internet providers from the key-value store
func (c *Client) ListProviders() ([]*models.InternetProvider, error) {
	keys, err := c.kv.Keys()
	if err != nil {
		// Check if the error is due to no keys found (empty bucket)
		if strings.Contains(err.Error(), "no keys found") {
			logrus.Debug("No providers found in key-value store")
			return []*models.InternetProvider{}, nil
		}
		return nil, fmt.Errorf("failed to list provider keys: %w", err)
	}

	var providers []*models.InternetProvider
	for _, key := range keys {
		if len(key) > 10 && key[:10] == "providers." {
			// Extract the ID from the key (remove "providers." prefix)
			providerID := key[10:]

			// Since we can't reliably reverse the sanitization (multiple chars could map to '_'),
			// we'll try to get the provider using the sanitized ID first
			provider, err := c.GetProvider(providerID)
			if err != nil {
				logrus.Warnf("Failed to get provider with sanitized ID %s: %v", providerID, err)
				continue
			}
			providers = append(providers, provider)
		}
	}

	return providers, nil
}

// DeleteProvider deletes an internet provider from the key-value store
func (c *Client) DeleteProvider(id string) error {
	// Try with sanitized key first
	key := fmt.Sprintf("providers.%s", sanitizeKey(id))
	err := c.kv.Delete(key)
	if err != nil {
		// If that fails, try with the original ID (in case it was stored before sanitization)
		key = fmt.Sprintf("providers.%s", id)
		err = c.kv.Delete(key)
		if err != nil {
			return fmt.Errorf("failed to delete provider: %w", err)
		}
	}

	logrus.Debugf("Deleted provider %s", id)
	return nil
}

// StorePolicy stores a routing policy in the key-value store
func (c *Client) StorePolicy(policy *models.RoutingPolicy) error {
	data, err := policy.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	key := fmt.Sprintf("policies.%s", sanitizeKey(policy.ID))
	_, err = c.kv.Put(key, data)
	if err != nil {
		return fmt.Errorf("failed to store policy: %w", err)
	}

	logrus.Debugf("Stored policy %s", policy.ID)
	return nil
}

// GetPolicy retrieves a routing policy from the key-value store
func (c *Client) GetPolicy(id string) (*models.RoutingPolicy, error) {
	// Try with sanitized key first
	key := fmt.Sprintf("policies.%s", sanitizeKey(id))
	entry, err := c.kv.Get(key)
	if err != nil {
		// If that fails, try with the original ID (in case it was stored before sanitization)
		key = fmt.Sprintf("policies.%s", id)
		entry, err = c.kv.Get(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get policy: %w", err)
		}
	}

	var policy models.RoutingPolicy
	if err := policy.FromJSON(entry.Value()); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	return &policy, nil
}

// ListPolicies retrieves all routing policies from the key-value store
func (c *Client) ListPolicies() ([]*models.RoutingPolicy, error) {
	keys, err := c.kv.Keys()
	if err != nil {
		// Check if the error is due to no keys found (empty bucket)
		if strings.Contains(err.Error(), "no keys found") {
			logrus.Debug("No policies found in key-value store")
			return []*models.RoutingPolicy{}, nil
		}
		return nil, fmt.Errorf("failed to list policy keys: %w", err)
	}

	var policies []*models.RoutingPolicy
	for _, key := range keys {
		if len(key) > 9 && key[:9] == "policies." {
			// Extract the ID from the key (remove "policies." prefix)
			policyID := key[9:]

			// Since we can't reliably reverse the sanitization (multiple chars could map to '_'),
			// we'll try to get the policy using the sanitized ID first
			policy, err := c.GetPolicy(policyID)
			if err != nil {
				logrus.Warnf("Failed to get policy with sanitized ID %s: %v", policyID, err)
				continue
			}
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

// DeletePolicy deletes a routing policy from the key-value store
func (c *Client) DeletePolicy(id string) error {
	// Try with sanitized key first
	key := fmt.Sprintf("policies.%s", sanitizeKey(id))
	err := c.kv.Delete(key)
	if err != nil {
		// If that fails, try with the original ID (in case it was stored before sanitization)
		key = fmt.Sprintf("policies.%s", id)
		err = c.kv.Delete(key)
		if err != nil {
			return fmt.Errorf("failed to delete policy: %w", err)
		}
	}

	logrus.Debugf("Deleted policy %s", id)
	return nil
}

// WatchProviders watches for changes to providers
func (c *Client) WatchProviders(ctx context.Context, callback func(*models.InternetProvider, nats.KeyValueOp)) error {
	watcher, err := c.kv.Watch("providers.*")
	if err != nil {
		return fmt.Errorf("failed to create provider watcher: %w", err)
	}
	defer func() { _ = watcher.Stop() }()

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-watcher.Updates():
			if update == nil {
				continue
			}

			if len(update.Key()) > 10 && update.Key()[:10] == "providers." {
				if update.Operation() == nats.KeyValueDelete {
					// Handle deletion
					callback(nil, update.Operation())
					continue
				}

				var provider models.InternetProvider
				if err := provider.FromJSON(update.Value()); err != nil {
					logrus.Warnf("Failed to unmarshal provider update: %v", err)
					continue
				}
				callback(&provider, update.Operation())
			}
		}
	}
}

// WatchPolicies watches for changes to policies
func (c *Client) WatchPolicies(ctx context.Context, callback func(*models.RoutingPolicy, nats.KeyValueOp)) error {
	watcher, err := c.kv.Watch("policies.*")
	if err != nil {
		return fmt.Errorf("failed to create policy watcher: %w", err)
	}
	defer func() { _ = watcher.Stop() }()

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-watcher.Updates():
			if update == nil {
				continue
			}

			if len(update.Key()) > 9 && update.Key()[:9] == "policies." {
				if update.Operation() == nats.KeyValueDelete {
					// Handle deletion
					callback(nil, update.Operation())
					continue
				}

				var policy models.RoutingPolicy
				if err := policy.FromJSON(update.Value()); err != nil {
					logrus.Warnf("Failed to unmarshal policy update: %v", err)
					continue
				}
				callback(&policy, update.Operation())
			}
		}
	}
}

// testKeyValueStore tests if the key-value store is working properly
func (c *Client) testKeyValueStore() error {
	testKey := "test_simple_key"
	testValue := []byte("test_value")

	_, err := c.kv.Put(testKey, testValue)
	if err != nil {
		return fmt.Errorf("failed to put test key: %w", err)
	}

	entry, err := c.kv.Get(testKey)
	if err != nil {
		return fmt.Errorf("failed to get test key: %w", err)
	}

	if string(entry.Value()) != "test_value" {
		return fmt.Errorf("test value mismatch")
	}

	err = c.kv.Delete(testKey)
	if err != nil {
		return fmt.Errorf("failed to delete test key: %w", err)
	}

	logrus.Debug("Key-value store test passed")
	return nil
}
