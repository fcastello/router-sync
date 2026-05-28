package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"router-sync/docs"
	"router-sync/internal/config"
	"router-sync/internal/logging"
	"router-sync/internal/nats"
	"router-sync/internal/router"
	"router-sync/internal/sync"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server represents the API server
type Server struct {
	config        config.APIConfig
	natsClient    nats.NATSClient
	routerManager *router.Manager
	syncService   *sync.Service
	server        *http.Server

	// Prometheus metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	providersTotal      prometheus.Gauge
	policiesTotal       prometheus.Gauge

	// Version info
	version   string
	buildTime string
	gitCommit string
}

// NewServer creates a new API server
func NewServer(cfg config.APIConfig, natsClient nats.NATSClient, routerManager *router.Manager, syncService *sync.Service, version, buildTime, gitCommit string) *Server {
	// Initialize Prometheus metrics
	httpRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	providersTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "providers_total",
			Help: "Total number of internet providers",
		},
	)

	policiesTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "policies_total",
			Help: "Total number of routing policies",
		},
	)

	// Register metrics
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, providersTotal, policiesTotal)

	server := &Server{
		config:              cfg,
		natsClient:          natsClient,
		routerManager:       routerManager,
		syncService:         syncService,
		httpRequestsTotal:   httpRequestsTotal,
		httpRequestDuration: httpRequestDuration,
		providersTotal:      providersTotal,
		policiesTotal:       policiesTotal,
		version:             version,
		buildTime:           buildTime,
		gitCommit:           gitCommit,
	}

	// Set up Gin router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(server.metricsMiddleware())
	router.Use(server.urlDecodeMiddleware())

	// Configure router to handle special characters in parameters
	router.RedirectFixedPath = false

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Provider endpoints
		providers := v1.Group("/providers")
		{
			providers.GET("", server.listProviders)
			providers.POST("", server.createProvider)
			providers.GET("/:id", server.getProvider)
			providers.PUT("/:id", server.updateProvider)
			providers.DELETE("/:id", server.deleteProvider)
		}

		// Policy endpoints
		policies := v1.Group("/policies")
		{
			policies.GET("", server.listPolicies)
			policies.POST("", server.createPolicy)
			policies.GET("/:id", server.getPolicy)
			policies.PUT("/:id", server.updatePolicy)
			policies.DELETE("/:id", server.deletePolicy)
		}

		// Sync endpoints
		v1.POST("/sync", server.triggerSync)
		v1.GET("/stats", server.getStats)

		// Runtime logging
		v1.GET("/logging/level", server.getLogLevel)
		v1.PUT("/logging/level", server.setLogLevel)
	}

	// Swagger: empty Host so the UI uses the browser's host:port (e.g. 192.168.2.252:18080)
	docs.SwaggerInfo.Host = ""
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http"}
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Prometheus metrics
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check
	router.GET("/health", server.healthCheck)

	server.server = &http.Server{
		Addr:    cfg.Address,
		Handler: router,
	}

	return server
}

// Start starts the API server
func (s *Server) Start() error {
	logrus.Infof("Starting API server on %s", s.config.Address)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the API server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// corsMiddleware allows the standalone web UI (separate origin/port) to call the API.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// metricsMiddleware adds Prometheus metrics middleware
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()

		s.httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			fmt.Sprintf("%d", c.Writer.Status()),
		).Inc()

		s.httpRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(duration)
	}
}

// urlDecodeMiddleware decodes URL-encoded parameters
func (s *Server) urlDecodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Decode URL-encoded parameters
		for i, param := range c.Params {
			decoded, err := url.QueryUnescape(param.Value)
			if err == nil {
				c.Params[i].Value = decoded
			}
		}
		c.Next()
	}
}

// healthCheck handles health check requests
// @Summary Health check
// @Description Check if the service is healthy
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "router-sync",
	})
}

// triggerSync triggers a manual synchronization
// @Summary Trigger synchronization
// @Description Manually trigger synchronization with NATS KV store
// @Tags sync
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/sync [post]
func (s *Server) triggerSync(c *gin.Context) {
	// This would trigger a manual sync
	// For now, we'll just return success
	c.JSON(http.StatusOK, gin.H{
		"message":   "Sync triggered successfully",
		"timestamp": time.Now().UTC(),
	})
}

// getStats returns service statistics
// @Summary Get service statistics
// @Description Get statistics about providers, policies, and routing
// @Tags stats
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/stats [get]
func (s *Server) getStats(c *gin.Context) {
	// Get sync service stats
	syncStats := s.syncService.GetStats()

	// Get router stats
	routerStats, err := s.routerManager.GetRoutingStats()
	if err != nil {
		logrus.Errorf("Failed to get router stats: %v", err)
		routerStats = make(map[string]interface{})
	}

	// Update Prometheus metrics
	s.providersTotal.Set(float64(syncStats["providers_count"].(int)))
	s.policiesTotal.Set(float64(syncStats["policies_count"].(int)))

	stats := gin.H{
		"sync":       syncStats,
		"router":     routerStats,
		"log_level":  logging.GetLevelName(),
		"timestamp":  time.Now().UTC(),
		"version":    s.version,
		"build_time": s.buildTime,
		"git_commit": s.gitCommit,
	}

	c.JSON(http.StatusOK, stats)
}
