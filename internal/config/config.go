package config

import (
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Mode selects the runtime role of the binary.
type Mode string

const (
	// ModeAPI runs only the HTTP API on the configured address.
	ModeAPI Mode = "api"
	// ModeAgent runs the router-local agent (NET_ADMIN) that applies policies
	// and reports state back to NATS.
	ModeAgent Mode = "agent"
)

// Config represents the application configuration
type Config struct {
	Mode     Mode         `yaml:"mode"`
	LogLevel logrus.Level `yaml:"log_level"`
	NATS     NATSConfig   `yaml:"nats"`
	API      APIConfig    `yaml:"api"`
	Sync     SyncConfig   `yaml:"sync"`
	Agent    AgentConfig  `yaml:"agent"`
}

// NATSConfig represents NATS connection configuration
type NATSConfig struct {
	URLs      []string `yaml:"urls"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
	Token     string   `yaml:"token"`
	ClusterID string   `yaml:"cluster_id"`
	ClientID  string   `yaml:"client_id"`
	WriterID  string   `yaml:"writer_id"`
}

// APIConfig represents API server configuration
type APIConfig struct {
	Address string `yaml:"address"`
}

// SyncConfig represents synchronization configuration
type SyncConfig struct {
	Interval time.Duration `yaml:"interval"`
}

// AgentConfig represents agent-mode configuration.
//
// Hostname identifies this agent inside NATS (defaults to os.Hostname()).
// MetricsAddress is the listener for /health and /metrics on the agent.
// StatePublishInterval is how often the agent publishes RouterState to NATS.
type AgentConfig struct {
	Hostname             string        `yaml:"hostname"`
	MetricsAddress       string        `yaml:"metrics_address"`
	StatePublishInterval time.Duration `yaml:"state_publish_interval"`
}

// Load loads configuration from file and applies environment overrides.
//
// Environment variables (optional):
//   - ROUTER_SYNC_MODE                  (api|agent)
//   - ROUTER_SYNC_LOG_LEVEL
//   - ROUTER_SYNC_API_ADDRESS
//   - ROUTER_SYNC_AGENT_HOSTNAME
//   - ROUTER_SYNC_AGENT_METRICS_ADDRESS
//   - ROUTER_SYNC_AGENT_STATE_INTERVAL  (Go duration: 5s, 1m...)
//   - ROUTER_SYNC_NATS_URL              (comma-separated for multiple URLs)
//   - ROUTER_SYNC_NATS_USERNAME
//   - ROUTER_SYNC_NATS_PASSWORD
//   - ROUTER_SYNC_NATS_TOKEN
//   - ROUTER_SYNC_NATS_CLIENT_ID
//   - ROUTER_SYNC_WRITER_ID
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	applyDefaults(&config)
	applyEnvOverrides(&config)

	return &config, nil
}

func applyDefaults(config *Config) {
	if config.Mode == "" {
		config.Mode = ModeAPI
	}
	if config.API.Address == "" {
		config.API.Address = ":18080"
	}
	if config.Sync.Interval == 0 {
		config.Sync.Interval = 30 * time.Second
	}
	if config.LogLevel == 0 {
		config.LogLevel = logrus.WarnLevel
	}
	if config.NATS.ClientID == "" {
		config.NATS.ClientID = "router-sync-client"
	}
	if config.NATS.WriterID == "" {
		config.NATS.WriterID = config.NATS.ClientID
	}
	if config.Agent.MetricsAddress == "" {
		config.Agent.MetricsAddress = ":18082"
	}
	if config.Agent.StatePublishInterval == 0 {
		config.Agent.StatePublishInterval = 5 * time.Second
	}
	if config.Agent.Hostname == "" {
		if hn, err := os.Hostname(); err == nil {
			config.Agent.Hostname = hn
		}
	}
}

func applyEnvOverrides(config *Config) {
	if v := os.Getenv("ROUTER_SYNC_MODE"); v != "" {
		config.Mode = Mode(strings.ToLower(strings.TrimSpace(v)))
	}
	if v := os.Getenv("ROUTER_SYNC_LOG_LEVEL"); v != "" {
		if level, err := logrus.ParseLevel(v); err == nil {
			config.LogLevel = level
		}
	}
	if v := os.Getenv("ROUTER_SYNC_API_ADDRESS"); v != "" {
		config.API.Address = v
	}
	if v := os.Getenv("ROUTER_SYNC_AGENT_HOSTNAME"); v != "" {
		config.Agent.Hostname = v
	}
	if v := os.Getenv("ROUTER_SYNC_AGENT_METRICS_ADDRESS"); v != "" {
		config.Agent.MetricsAddress = v
	}
	if v := os.Getenv("ROUTER_SYNC_AGENT_STATE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.Agent.StatePublishInterval = d
		}
	}
	if v := os.Getenv("ROUTER_SYNC_NATS_URL"); v != "" {
		parts := strings.Split(v, ",")
		urls := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				urls = append(urls, p)
			}
		}
		if len(urls) > 0 {
			config.NATS.URLs = urls
		}
	}
	if v := os.Getenv("ROUTER_SYNC_NATS_USERNAME"); v != "" {
		config.NATS.Username = v
	}
	if v := os.Getenv("ROUTER_SYNC_NATS_PASSWORD"); v != "" {
		config.NATS.Password = v
	}
	if v := os.Getenv("ROUTER_SYNC_NATS_TOKEN"); v != "" {
		config.NATS.Token = v
	}
	if v := os.Getenv("ROUTER_SYNC_NATS_CLIENT_ID"); v != "" {
		config.NATS.ClientID = v
	}
	if v := os.Getenv("ROUTER_SYNC_WRITER_ID"); v != "" {
		config.NATS.WriterID = v
	}
}
