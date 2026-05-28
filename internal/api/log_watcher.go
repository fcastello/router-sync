package api

import (
	"context"

	"router-sync/internal/logging"
	"router-sync/internal/nats"

	"github.com/sirupsen/logrus"
)

// WatchOwnLogLevel applies persisted log level changes to the API service in
// real-time so the UI can flip the API's verbosity without restarting.
func WatchOwnLogLevel(ctx context.Context, client *nats.Client) {
	sid := logging.ServiceID()
	if sid == "" {
		return
	}

	if current, err := client.GetServiceLogLevel(sid); err == nil && current != "" {
		if lvl, err := logging.ParseLevel(current); err == nil {
			logging.SetLevel(lvl)
			logrus.Infof("Applied persisted log level %s for %s", lvl.String(), sid)
		}
	}

	go func() {
		err := client.WatchServiceLogLevel(ctx, sid, func(level string) {
			lvl, err := logging.ParseLevel(level)
			if err != nil {
				logrus.Warnf("Invalid log level %q from NATS: %v", level, err)
				return
			}
			prev := logging.GetLevelName()
			logging.SetLevel(lvl)
			logrus.Infof("Log level changed from %s to %s via NATS for %s", prev, lvl.String(), sid)
		})
		if err != nil {
			logrus.Warnf("Log level watcher (api) error: %v", err)
		}
	}()
}
