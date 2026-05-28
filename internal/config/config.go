package config

import (
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	LogLevel logrus.Level `yaml:"log_level"`
	NATS     NATSConfig   `yaml:"nats"`
	API      APIConfig    `yaml:"api"`
	Sync     SyncConfig   `yaml:"sync"`
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

// Load loads configuration from file and applies environment overrides.
//
// Environment variables (optional):
//   - ROUTER_SYNC_LOG_LEVEL
//   - ROUTER_SYNC_API_ADDRESS
//   - ROUTER_SYNC_NATS_URL (comma-separated for multiple URLs)
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
}

func applyEnvOverrides(config *Config) {
	if v := os.Getenv("ROUTER_SYNC_LOG_LEVEL"); v != "" {
		if level, err := logrus.ParseLevel(v); err == nil {
			config.LogLevel = level
		}
	}
	if v := os.Getenv("ROUTER_SYNC_API_ADDRESS"); v != "" {
		config.API.Address = v
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
