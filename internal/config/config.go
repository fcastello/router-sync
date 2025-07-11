package config

import (
	"os"
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
}

// APIConfig represents API server configuration
type APIConfig struct {
	Address string `yaml:"address"`
}

// SyncConfig represents synchronization configuration
type SyncConfig struct {
	Interval time.Duration `yaml:"interval"`
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set defaults
	if config.API.Address == "" {
		config.API.Address = ":8080"
	}
	if config.Sync.Interval == 0 {
		config.Sync.Interval = 30 * time.Second
	}
	if config.LogLevel == 0 {
		config.LogLevel = logrus.InfoLevel
	}

	return &config, nil
} 