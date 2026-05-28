package nats

import (
	"context"
	"errors"
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

	StoreRouterState(state *models.RouterState) error
	GetRouterState(hostname string) (*models.RouterState, error)
	ListRouterStates() ([]*models.RouterState, error)
	DeleteRouterState(hostname string) error

	SetServiceLogLevel(serviceID, level string) error
	GetServiceLogLevel(serviceID string) (string, error)
	ListServiceLogLevels() (map[string]string, error)

	Close()
}

// Bucket names
const (
	bucketCore    = "router-sync"
	bucketState   = "router-sync-state"
	bucketLogging = "router-sync-logging"

	stateTTL = 60 * time.Second
)

// Client represents a NATS client with key-value store capabilities
type Client struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	kv         nats.KeyValue
	kvState    nats.KeyValue
	kvLogging  nats.KeyValue
	writerID   string
}

// sanitizeKey sanitizes a key to be compatible with NATS key-value store
func sanitizeKey(key string) string {
	var result strings.Builder

	for _, char := range key {
		switch {
		case (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9'):
			result.WriteRune(char)
		case char == '.' || char == '_' || char == '-':
			result.WriteRune(char)
		default:
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
		nats.MaxReconnects(-1),
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

	kv, err := ensureBucket(js, bucketCore, 0)
	if err != nil {
		conn.Close()
		return nil, err
	}

	kvState, err := ensureBucket(js, bucketState, stateTTL)
	if err != nil {
		conn.Close()
		return nil, err
	}

	kvLogging, err := ensureBucket(js, bucketLogging, 0)
	if err != nil {
		conn.Close()
		return nil, err
	}

	writerID := cfg.WriterID
	if writerID == "" {
		writerID = cfg.ClientID
	}

	client := &Client{
		conn:      conn,
		js:        js,
		kv:        kv,
		kvState:   kvState,
		kvLogging: kvLogging,
		writerID:  writerID,
	}

	if err := client.testKeyValueStore(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("key-value store test failed: %w", err)
	}

	logrus.Info("Connected to NATS server")
	return client, nil
}

// ensureBucket creates a bucket if missing or returns the existing one.
func ensureBucket(js nats.JetStreamContext, name string, ttl time.Duration) (nats.KeyValue, error) {
	cfg := &nats.KeyValueConfig{Bucket: name, TTL: ttl}
	kv, err := js.CreateKeyValue(cfg)
	if err == nil {
		return kv, nil
	}
	kv, err = js.KeyValue(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create/get %s bucket: %w", name, err)
	}
	return kv, nil
}

// Close closes the NATS connection
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// WriterID returns the writer identity used for active/active conflict resolution.
func (c *Client) WriterID() string {
	return c.writerID
}

// StoreProvider stores an internet provider in the key-value store using revision CAS.
func (c *Client) StoreProvider(provider *models.InternetProvider) error {
	key := fmt.Sprintf("providers.%s", sanitizeKey(provider.ID))
	logrus.Debugf("Storing provider with key: %s (original ID: %s)", key, provider.ID)

	return c.storeWithCAS(c.kv, key, func(existing []byte) ([]byte, error) {
		var prev *models.InternetProvider
		if len(existing) > 0 {
			var parsed models.InternetProvider
			if err := parsed.FromJSON(existing); err != nil {
				return nil, fmt.Errorf("failed to unmarshal existing provider: %w", err)
			}
			prev = &parsed
		}
		PrepareProviderWrite(provider, prev, c.writerID)
		return provider.ToJSON()
	})
}

// GetProvider retrieves an internet provider from the key-value store
func (c *Client) GetProvider(id string) (*models.InternetProvider, error) {
	key := fmt.Sprintf("providers.%s", sanitizeKey(id))
	entry, err := c.kv.Get(key)
	if err != nil {
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
		if strings.Contains(err.Error(), "no keys found") {
			logrus.Debug("No providers found in key-value store")
			return []*models.InternetProvider{}, nil
		}
		return nil, fmt.Errorf("failed to list provider keys: %w", err)
	}

	var providers []*models.InternetProvider
	for _, key := range keys {
		if len(key) > 10 && key[:10] == "providers." {
			providerID := key[10:]
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
	key := fmt.Sprintf("providers.%s", sanitizeKey(id))
	err := c.kv.Delete(key)
	if err != nil {
		key = fmt.Sprintf("providers.%s", id)
		err = c.kv.Delete(key)
		if err != nil {
			return fmt.Errorf("failed to delete provider: %w", err)
		}
	}

	logrus.Debugf("Deleted provider %s", id)
	return nil
}

// StorePolicy stores a routing policy in the key-value store using revision CAS.
func (c *Client) StorePolicy(policy *models.RoutingPolicy) error {
	key := fmt.Sprintf("policies.%s", sanitizeKey(policy.ID))

	return c.storeWithCAS(c.kv, key, func(existing []byte) ([]byte, error) {
		var prev *models.RoutingPolicy
		if len(existing) > 0 {
			var parsed models.RoutingPolicy
			if err := parsed.FromJSON(existing); err != nil {
				return nil, fmt.Errorf("failed to unmarshal existing policy: %w", err)
			}
			prev = &parsed
		}
		PreparePolicyWrite(policy, prev, c.writerID)
		return policy.ToJSON()
	})
}

const maxCASRetries = 5

// storeWithCAS writes a KV entry using JetStream revision compare-and-swap.
func (c *Client) storeWithCAS(kv nats.KeyValue, key string, build func(existing []byte) ([]byte, error)) error {
	var lastErr error
	for attempt := 0; attempt < maxCASRetries; attempt++ {
		entry, err := kv.Get(key)
		if err != nil && !errors.Is(err, nats.ErrKeyNotFound) {
			return fmt.Errorf("failed to read key %s: %w", key, err)
		}

		var existing []byte
		var revision uint64
		if err == nil {
			existing = entry.Value()
			revision = entry.Revision()
		}

		data, err := build(existing)
		if err != nil {
			return err
		}

		if revision == 0 {
			_, err = kv.Create(key, data)
		} else {
			_, err = kv.Update(key, data, revision)
		}
		if err == nil {
			logrus.Debugf("Stored key %s (attempt %d)", key, attempt+1)
			return nil
		}
		if errors.Is(err, nats.ErrKeyExists) || errors.Is(err, nats.ErrKeyNotFound) {
			lastErr = err
			continue
		}
		return fmt.Errorf("failed to store key %s: %w", key, err)
	}
	if lastErr != nil {
		return fmt.Errorf("failed to store key %s after %d attempts: %w", key, maxCASRetries, lastErr)
	}
	return fmt.Errorf("failed to store key %s after %d attempts", key, maxCASRetries)
}

// GetPolicy retrieves a routing policy from the key-value store
func (c *Client) GetPolicy(id string) (*models.RoutingPolicy, error) {
	key := fmt.Sprintf("policies.%s", sanitizeKey(id))
	entry, err := c.kv.Get(key)
	if err != nil {
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
		if strings.Contains(err.Error(), "no keys found") {
			logrus.Debug("No policies found in key-value store")
			return []*models.RoutingPolicy{}, nil
		}
		return nil, fmt.Errorf("failed to list policy keys: %w", err)
	}

	var policies []*models.RoutingPolicy
	for _, key := range keys {
		if len(key) > 9 && key[:9] == "policies." {
			policyID := key[9:]
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
	key := fmt.Sprintf("policies.%s", sanitizeKey(id))
	err := c.kv.Delete(key)
	if err != nil {
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
	// "providers.>" matches every key under the "providers." prefix, including
	// multi-token IDs. Plain "providers.*" only matches single-token IDs, which
	// would silently drop providers whose name contains a dot.
	watcher, err := c.kv.Watch("providers.>")
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
	// "policies.>" matches every key under the "policies." prefix, including
	// IPv4 IDs like 192.168.2.25 that contain dots. Plain "policies.*" matches
	// only single-token IDs, silently dropping every IP-keyed policy.
	watcher, err := c.kv.Watch("policies.>")
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

// StoreRouterState stores a router state heartbeat. Uses simple Put because state
// is TTL'd and overwritten by the same writer every interval.
func (c *Client) StoreRouterState(state *models.RouterState) error {
	if state.Hostname == "" {
		return fmt.Errorf("router state hostname is required")
	}
	state.LastSeen = time.Now().UTC()
	data, err := state.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal router state: %w", err)
	}
	key := fmt.Sprintf("router.%s", sanitizeKey(state.Hostname))
	if _, err := c.kvState.Put(key, data); err != nil {
		return fmt.Errorf("failed to store router state for %s: %w", state.Hostname, err)
	}
	return nil
}

// GetRouterState fetches a single router state.
func (c *Client) GetRouterState(hostname string) (*models.RouterState, error) {
	key := fmt.Sprintf("router.%s", sanitizeKey(hostname))
	entry, err := c.kvState.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get router state %s: %w", hostname, err)
	}
	var state models.RouterState
	if err := state.FromJSON(entry.Value()); err != nil {
		return nil, fmt.Errorf("failed to unmarshal router state: %w", err)
	}
	return &state, nil
}

// ListRouterStates returns all known router states (alive within the TTL window).
func (c *Client) ListRouterStates() ([]*models.RouterState, error) {
	keys, err := c.kvState.Keys()
	if err != nil {
		if strings.Contains(err.Error(), "no keys found") {
			return []*models.RouterState{}, nil
		}
		return nil, fmt.Errorf("failed to list router state keys: %w", err)
	}

	var states []*models.RouterState
	for _, key := range keys {
		if !strings.HasPrefix(key, "router.") {
			continue
		}
		entry, err := c.kvState.Get(key)
		if err != nil {
			logrus.Warnf("Failed to get router state %s: %v", key, err)
			continue
		}
		var state models.RouterState
		if err := state.FromJSON(entry.Value()); err != nil {
			logrus.Warnf("Failed to unmarshal router state %s: %v", key, err)
			continue
		}
		states = append(states, &state)
	}
	return states, nil
}

// DeleteRouterState removes a router state entry.
func (c *Client) DeleteRouterState(hostname string) error {
	key := fmt.Sprintf("router.%s", sanitizeKey(hostname))
	if err := c.kvState.Delete(key); err != nil {
		return fmt.Errorf("failed to delete router state %s: %w", hostname, err)
	}
	return nil
}

// SetServiceLogLevel persists a runtime log level for a service ID
// (e.g. "api", "agent.r1"). Consumers watch level.<service_id> to apply changes.
func (c *Client) SetServiceLogLevel(serviceID, level string) error {
	key := fmt.Sprintf("level.%s", sanitizeKey(serviceID))
	if _, err := c.kvLogging.Put(key, []byte(level)); err != nil {
		return fmt.Errorf("failed to set log level for %s: %w", serviceID, err)
	}
	return nil
}

// GetServiceLogLevel returns the persisted log level for a service ID.
func (c *Client) GetServiceLogLevel(serviceID string) (string, error) {
	key := fmt.Sprintf("level.%s", sanitizeKey(serviceID))
	entry, err := c.kvLogging.Get(key)
	if err != nil {
		return "", fmt.Errorf("failed to get log level for %s: %w", serviceID, err)
	}
	return string(entry.Value()), nil
}

// ListServiceLogLevels returns the persisted log levels for every known service.
func (c *Client) ListServiceLogLevels() (map[string]string, error) {
	keys, err := c.kvLogging.Keys()
	if err != nil {
		if strings.Contains(err.Error(), "no keys found") {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("failed to list log level keys: %w", err)
	}
	out := make(map[string]string)
	for _, key := range keys {
		if !strings.HasPrefix(key, "level.") {
			continue
		}
		serviceID := key[len("level."):]
		entry, err := c.kvLogging.Get(key)
		if err != nil {
			continue
		}
		out[serviceID] = string(entry.Value())
	}
	return out, nil
}

// WatchServiceLogLevel watches a single service's log level for runtime changes.
// callback receives the level string on every update.
func (c *Client) WatchServiceLogLevel(ctx context.Context, serviceID string, callback func(level string)) error {
	key := fmt.Sprintf("level.%s", sanitizeKey(serviceID))
	watcher, err := c.kvLogging.Watch(key)
	if err != nil {
		return fmt.Errorf("failed to watch log level for %s: %w", serviceID, err)
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
			if update.Operation() == nats.KeyValueDelete {
				continue
			}
			callback(string(update.Value()))
		}
	}
}

// WatchRouterStates watches for router state changes. Useful for API gauges and UI live updates.
func (c *Client) WatchRouterStates(ctx context.Context, callback func(*models.RouterState, nats.KeyValueOp)) error {
	// Match every "router.<hostname>" key, including hostnames with dots.
	watcher, err := c.kvState.Watch("router.>")
	if err != nil {
		return fmt.Errorf("failed to create router state watcher: %w", err)
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
			if update.Operation() == nats.KeyValueDelete || update.Operation() == nats.KeyValuePurge {
				callback(nil, update.Operation())
				continue
			}
			var state models.RouterState
			if err := state.FromJSON(update.Value()); err != nil {
				logrus.Warnf("Failed to unmarshal router state update: %v", err)
				continue
			}
			callback(&state, update.Operation())
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
