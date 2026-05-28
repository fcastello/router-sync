// Package main is the single binary that runs both the API service and the
// per-router Agent service, selected by `--mode={api|agent}`.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"router-sync/internal/agent"
	"router-sync/internal/api"
	"router-sync/internal/config"
	"router-sync/internal/logging"
	"router-sync/internal/metrics"
	"router-sync/internal/nats"
	"router-sync/internal/router"

	_ "router-sync/docs" // register Swagger doc.json

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// @title Router Sync API
// @version 1.0
// @description Router synchronization service for managing internet providers and routing policies
// @host localhost:18080
// @BasePath /
func main() {
	var (
		configPath string
		modeFlag   string
	)
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.StringVar(&modeFlag, "mode", "", "Runtime mode: api or agent (overrides config.mode)")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	if modeFlag != "" {
		cfg.Mode = config.Mode(strings.ToLower(strings.TrimSpace(modeFlag)))
	}
	if cfg.Mode == "" {
		cfg.Mode = config.ModeAPI
	}

	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	switch cfg.Mode {
	case config.ModeAPI:
		runAPI(cfg)
	case config.ModeAgent:
		runAgent(cfg)
	default:
		logrus.Fatalf("Unknown mode %q (expected: api or agent)", cfg.Mode)
	}
}

func runAPI(cfg *config.Config) {
	logging.Init(cfg.LogLevel, "api")
	logrus.Infof("Starting router-sync API (version %s, build %s, commit %s)", Version, BuildTime, GitCommit)

	natsClient, err := nats.NewClient(cfg.NATS)
	if err != nil {
		logrus.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	if err := api.MigrateProviderInterfaces(natsClient); err != nil {
		logrus.Warnf("Provider interface migration failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api.WatchOwnLogLevel(ctx, natsClient)

	apiServer := api.NewServer(cfg.API, natsClient, Version, BuildTime, GitCommit)

	go func() {
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start API server: %v", err)
		}
	}()

	awaitShutdown(func(ctx context.Context) {
		if err := apiServer.Shutdown(ctx); err != nil {
			logrus.Errorf("Error during API server shutdown: %v", err)
		}
	})
}

func runAgent(cfg *config.Config) {
	hostname := cfg.Agent.Hostname
	if hostname == "" {
		if hn, err := os.Hostname(); err == nil {
			hostname = hn
			cfg.Agent.Hostname = hn
		}
	}

	serviceID := "agent." + hostname
	logging.Init(cfg.LogLevel, serviceID)
	logrus.Infof("Starting router-sync agent on host %q (version %s, build %s, commit %s)", hostname, Version, BuildTime, GitCommit)

	natsClient, err := nats.NewClient(cfg.NATS)
	if err != nil {
		logrus.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	routerManager, err := router.NewManager(hostname)
	if err != nil {
		logrus.Fatalf("Failed to initialize router manager: %v", err)
	}

	reg := metrics.NewRegistry()
	agentSvc := agent.NewService(natsClient, routerManager, *cfg, Version, reg)

	go func() {
		if err := agentSvc.Start(); err != nil {
			logrus.Errorf("Agent service error: %v", err)
		}
	}()

	httpServer := newAgentHTTPServer(cfg.Agent.MetricsAddress, reg, hostname)
	go func() {
		logrus.Infof("Starting agent HTTP listener on %s", cfg.Agent.MetricsAddress)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("Agent HTTP listener error: %v", err)
		}
	}()

	awaitShutdown(func(ctx context.Context) {
		if err := httpServer.Shutdown(ctx); err != nil {
			logrus.Errorf("Error during agent HTTP shutdown: %v", err)
		}
		if err := agentSvc.Stop(); err != nil {
			logrus.Errorf("Error during agent service shutdown: %v", err)
		}
		if err := routerManager.CleanupAllRules(); err != nil {
			logrus.Errorf("Error during routing rules cleanup: %v", err)
		}
		if err := routerManager.RemoveSuppressDefaultRule(); err != nil {
			logrus.Errorf("Error during suppress-default rule cleanup: %v", err)
		}
	})
}

func newAgentHTTPServer(addr string, reg *prometheus.Registry, hostname string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy","service":"router-sync-agent","hostname":%q,"timestamp":%q}`,
			hostname, time.Now().UTC().Format(time.RFC3339))
	})
	mux.Handle("/metrics", metrics.HandlerFor(reg))
	return &http.Server{Addr: addr, Handler: mux}
}

func awaitShutdown(shutdown func(context.Context)) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	shutdown(ctx)
	logrus.Info("Stopped")
}
