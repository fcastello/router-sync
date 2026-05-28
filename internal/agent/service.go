// Package agent runs on every router (NET_ADMIN, host network) and is the only
// component that mutates the kernel routing state. It watches providers + policies
// in NATS, applies them locally, and heartbeats its router state back to NATS.
package agent

import (
	"context"
	"sync"
	"time"

	"router-sync/internal/config"
	"router-sync/internal/logging"
	"router-sync/internal/models"
	"router-sync/internal/nats"
	"router-sync/internal/router"
	"router-sync/internal/state"

	natsio "github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Service is the agent: it watches providers/policies and applies them locally,
// while publishing its own RouterState heartbeat to NATS every interval.
type Service struct {
	natsClient    *nats.Client
	routerManager *router.Manager
	collector     *state.Collector
	cfg           config.Config
	hostname      string
	agentVersion  string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	providers map[string]*models.InternetProvider
	policies  map[string]*models.RoutingPolicy
	cacheMu   sync.RWMutex

	syncTotal           prometheus.Counter
	syncDuration        prometheus.Histogram
	rulesTotal          prometheus.Gauge
	routesTotal         *prometheus.GaugeVec
	statePublishTotal   prometheus.Counter
	statePublishErrors  prometheus.Counter
	conntrackClearedTot prometheus.Counter
}

// NewService creates a new agent service. The Prometheus registry is owned by main;
// pass an already-registered Registerer (or nil to register against the default).
func NewService(natsClient *nats.Client, routerManager *router.Manager, cfg config.Config, agentVersion string, reg prometheus.Registerer) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Service{
		natsClient:    natsClient,
		routerManager: routerManager,
		collector:     state.NewCollector(cfg.Agent.Hostname),
		cfg:           cfg,
		hostname:      cfg.Agent.Hostname,
		agentVersion:  agentVersion,
		ctx:           ctx,
		cancel:        cancel,
		providers:     make(map[string]*models.InternetProvider),
		policies:      make(map[string]*models.RoutingPolicy),
	}

	s.syncTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "agent_sync_total",
		Help: "Number of full sync runs performed by the agent.",
	})
	s.syncDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "agent_sync_duration_seconds",
		Help:    "Duration of a full sync run.",
		Buckets: prometheus.DefBuckets,
	})
	s.rulesTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_rules_total",
		Help: "Number of ip rules currently installed by the agent.",
	})
	s.routesTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "agent_routes_total",
		Help: "Number of routes per routing table.",
	}, []string{"table"})
	s.statePublishTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "agent_state_publish_total",
		Help: "Number of router state heartbeats published.",
	})
	s.statePublishErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "agent_state_publish_errors_total",
		Help: "Number of failed router state heartbeats.",
	})
	s.conntrackClearedTot = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "agent_conntrack_cleared_total",
		Help: "Number of conntrack flush invocations issued by the agent.",
	})

	if reg != nil {
		reg.MustRegister(
			s.syncTotal,
			s.syncDuration,
			s.rulesTotal,
			s.routesTotal,
			s.statePublishTotal,
			s.statePublishErrors,
			s.conntrackClearedTot,
		)
	}

	return s
}

// Start launches the watchers and the heartbeat loop.
func (s *Service) Start() error {
	logrus.Infof("Starting agent service on host %q (version %s)", s.hostname, s.agentVersion)

	// Install the priority-10 "lookup main + suppress_prefixlength 0" rule
	// so local LAN traffic always resolves via the main table while only
	// default-route traffic falls through to the per-source policy rules.
	if err := s.routerManager.EnsureSuppressDefaultRule(); err != nil {
		logrus.Errorf("Failed to install suppress-default rule: %v", err)
	}

	if err := s.performFullSync(); err != nil {
		logrus.Errorf("Initial sync failed: %v", err)
	}

	s.wg.Add(1)
	go s.periodicSync()

	s.wg.Add(1)
	go s.watchProviders()

	s.wg.Add(1)
	go s.watchPolicies()

	s.wg.Add(1)
	go s.publishStateLoop()

	s.wg.Add(1)
	go s.watchLogLevel()

	logrus.Info("Agent service started")
	return nil
}

// Stop terminates all goroutines and waits for them to exit.
func (s *Service) Stop() error {
	logrus.Info("Stopping agent service")
	s.cancel()
	s.wg.Wait()
	logrus.Info("Agent service stopped")
	return nil
}

// periodicSync re-applies provider+policy state every config.Sync.Interval.
func (s *Service) periodicSync() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cfg.Sync.Interval)
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

func (s *Service) performFullSync() error {
	start := time.Now()
	defer func() {
		s.syncTotal.Inc()
		s.syncDuration.Observe(time.Since(start).Seconds())
	}()

	logrus.Debug("Performing full synchronization")

	providers, err := s.natsClient.ListProviders()
	if err != nil {
		logrus.Errorf("Failed to list providers: %v", err)
		return err
	}

	policies, err := s.natsClient.ListPolicies()
	if err != nil {
		logrus.Errorf("Failed to list policies: %v", err)
		return err
	}

	s.cacheMu.Lock()
	s.providers = make(map[string]*models.InternetProvider, len(providers))
	for _, provider := range providers {
		s.providers[provider.ID] = provider
	}
	s.policies = make(map[string]*models.RoutingPolicy, len(policies))
	for _, policy := range policies {
		s.policies[policy.ID] = policy
	}
	s.cacheMu.Unlock()

	s.refreshTableNames()

	logrus.Info("SYNC START")
	if err := s.routerManager.SyncProviders(providers); err != nil {
		logrus.Errorf("Failed to sync providers: %v", err)
	}
	if err := s.routerManager.SyncPolicies(policies, providers); err != nil {
		logrus.Errorf("Failed to sync policies: %v", err)
	}
	logrus.Info("SYNC FINISHED")
	return nil
}

func (s *Service) refreshTableNames() {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	names := make(map[int]string, len(s.providers))
	for _, p := range s.providers {
		if p.TableID > 0 {
			names[p.TableID] = p.Name
		}
	}
	s.collector.SetTableNames(names)
}

func (s *Service) watchProviders() {
	defer s.wg.Done()

	err := s.natsClient.WatchProviders(s.ctx, func(provider *models.InternetProvider, op natsio.KeyValueOp) {
		s.cacheMu.Lock()

		switch op {
		case natsio.KeyValuePut:
			if provider != nil {
				s.providers[provider.ID] = provider
				logrus.Infof("Provider updated: %s", provider.Name)
				s.cacheMu.Unlock()
				if err := s.routerManager.SetupProvider(provider); err != nil {
					logrus.Errorf("Failed to set up provider %s: %v", provider.Name, err)
				}
				return
			}
		case natsio.KeyValueDelete:
			if provider != nil {
				delete(s.providers, provider.ID)
				logrus.Infof("Provider deleted: %s", provider.Name)
			}
		}
		s.cacheMu.Unlock()
	})

	if err != nil {
		logrus.Errorf("Provider watcher error: %v", err)
	}
}

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

				provider, exists := s.providers[policy.ProviderID]
				if !exists {
					logrus.Warnf("Provider %s not found for policy %s", policy.ProviderID, policy.Name)
					return
				}
				if err := s.routerManager.SetupPolicy(policy, provider); err != nil {
					logrus.Errorf("Failed to set up policy %s: %v", policy.Name, err)
				}
			}
		case natsio.KeyValueDelete:
			if policy != nil {
				delete(s.policies, policy.ID)
				logrus.Infof("Policy deleted: %s", policy.Name)

				provider, exists := s.providers[policy.ProviderID]
				if !exists {
					logrus.Warnf("Provider %s not found for policy %s", policy.ProviderID, policy.Name)
					return
				}
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

// publishStateLoop sends a RouterState heartbeat every Agent.StatePublishInterval.
func (s *Service) publishStateLoop() {
	defer s.wg.Done()

	if err := s.publishState(); err != nil {
		logrus.Warnf("Initial state publish failed: %v", err)
	}

	ticker := time.NewTicker(s.cfg.Agent.StatePublishInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			_ = s.natsClient.DeleteRouterState(s.hostname)
			return
		case <-ticker.C:
			if err := s.publishState(); err != nil {
				s.statePublishErrors.Inc()
				logrus.Warnf("State publish failed: %v", err)
			} else {
				s.statePublishTotal.Inc()
			}
		}
	}
}

func (s *Service) publishState() error {
	st, err := s.collector.Collect()
	if err != nil {
		return err
	}
	st.AgentVersion = s.agentVersion
	st.LogLevel = logging.GetLevelName()

	s.rulesTotal.Set(float64(len(st.Rules)))
	for _, t := range st.Tables {
		s.routesTotal.WithLabelValues(itoaTableLabel(t)).Set(float64(len(t.Routes)))
	}

	return s.natsClient.StoreRouterState(st)
}

func itoaTableLabel(t models.RoutingTable) string {
	if t.Name != "" {
		return t.Name
	}
	return tableIDLabel(t.ID)
}

func tableIDLabel(id int) string {
	switch id {
	case 254:
		return "main"
	case 253:
		return "default"
	case 255:
		return "local"
	}
	return "t" + itoa(id)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	bp := len(b)
	for i > 0 {
		bp--
		b[bp] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		bp--
		b[bp] = '-'
	}
	return string(b[bp:])
}

// watchLogLevel applies remote log level changes published in router-sync-logging
// under level.agent.<hostname>.
func (s *Service) watchLogLevel() {
	defer s.wg.Done()

	serviceID := logging.ServiceID()
	if serviceID == "" {
		logrus.Warn("Agent log level watcher disabled: no service ID configured")
		return
	}

	if current, err := s.natsClient.GetServiceLogLevel(serviceID); err == nil && current != "" {
		if lvl, err := logging.ParseLevel(current); err == nil {
			logging.SetLevel(lvl)
			logrus.Infof("Applied persisted log level %s for %s", lvl.String(), serviceID)
		}
	}

	err := s.natsClient.WatchServiceLogLevel(s.ctx, serviceID, func(level string) {
		lvl, err := logging.ParseLevel(level)
		if err != nil {
			logrus.Warnf("Invalid log level %q from NATS: %v", level, err)
			return
		}
		prev := logging.GetLevelName()
		logging.SetLevel(lvl)
		logrus.Infof("Log level changed from %s to %s via NATS for %s", prev, lvl.String(), serviceID)
	})
	if err != nil {
		logrus.Errorf("Log level watcher error: %v", err)
	}
}

// GetStats returns synchronization statistics (used for agent /health endpoint).
func (s *Service) GetStats() map[string]interface{} {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	stats := make(map[string]interface{})
	stats["providers_count"] = len(s.providers)
	stats["policies_count"] = len(s.policies)
	stats["sync_interval"] = s.cfg.Sync.Interval.String()
	stats["hostname"] = s.hostname

	policiesPerProvider := make(map[string]int)
	for _, policy := range s.policies {
		policiesPerProvider[policy.ProviderID]++
	}
	stats["policies_per_provider"] = policiesPerProvider

	return stats
}
