//go:build !linux

package state

import "router-sync/internal/models"

// Collect returns a minimal RouterState snapshot on non-Linux platforms. The agent
// is only ever deployed on Linux routers; this stub exists so the project still
// builds and tests on macOS/Windows during development.
func (c *Collector) Collect() (*models.RouterState, error) {
	return &models.RouterState{Hostname: c.hostname}, nil
}
