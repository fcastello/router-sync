package api

import (
	"router-sync/internal/nats"

	"github.com/sirupsen/logrus"
)

// MigrateProviderInterfaces runs the one-shot migration described in the
// microservice-split plan: for any provider that still has the legacy
// `Interface` field set with an empty `Interfaces` map, populate the map for
// each currently-known router hostname.
//
// If no routers are reporting state yet, the migrator falls back to "r2" as
// the default writer hostname (matches the historic deployment).
func MigrateProviderInterfaces(client *nats.Client) error {
	providers, err := client.ListProviders()
	if err != nil {
		return err
	}

	states, _ := client.ListRouterStates()
	hosts := make([]string, 0, len(states))
	for _, st := range states {
		if st.Hostname != "" {
			hosts = append(hosts, st.Hostname)
		}
	}
	if len(hosts) == 0 {
		hosts = []string{"r2"}
	}

	migrated := 0
	for _, p := range providers {
		if len(p.Interfaces) > 0 || p.Interface == "" {
			continue
		}
		ifaces := make(map[string]string, len(hosts))
		for _, h := range hosts {
			ifaces[h] = p.Interface
		}
		p.Interfaces = ifaces
		if err := client.StoreProvider(p); err != nil {
			logrus.Warnf("Failed to migrate provider %s: %v", p.Name, err)
			continue
		}
		migrated++
		logrus.Infof("Migrated provider %s: Interfaces=%v", p.Name, ifaces)
	}

	if migrated > 0 {
		logrus.Infof("Provider migration done: %d providers updated", migrated)
	}
	return nil
}
