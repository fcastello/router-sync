package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"router-sync/internal/api"
	"router-sync/internal/config"
	"router-sync/internal/nats"
	"router-sync/internal/router"
	"router-sync/internal/sync"

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
// @host localhost:8080
// @BasePath /api/v1
func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	logrus.SetLevel(cfg.LogLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logrus.Info("Starting Router Sync Service")

	// Initialize NATS connection
	natsClient, err := nats.NewClient(cfg.NATS)
	if err != nil {
		logrus.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// Initialize router manager
	routerManager, err := router.NewManager()
	if err != nil {
		logrus.Fatalf("Failed to initialize router manager: %v", err)
	}

	// Initialize sync service
	syncService := sync.NewService(natsClient, routerManager, cfg.Sync)

	// Initialize API server (pass version info)
	apiServer := api.NewServer(cfg.API, natsClient, routerManager, syncService, Version, BuildTime, GitCommit)

	// Start sync service
	go func() {
		if err := syncService.Start(); err != nil {
			logrus.Errorf("Sync service error: %v", err)
		}
	}()

	// Start API server
	go func() {
		logrus.Infof("Starting API server on %s", cfg.API.Address)
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start API server: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down Router Sync Service")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		logrus.Errorf("Error during API server shutdown: %v", err)
	}

	if err := syncService.Stop(); err != nil {
		logrus.Errorf("Error during sync service shutdown: %v", err)
	}

	// Clean up all routing rules managed by this application
	if err := routerManager.CleanupAllRules(); err != nil {
		logrus.Errorf("Error during routing rules cleanup: %v", err)
	}

	logrus.Info("Router Sync Service stopped")
}
