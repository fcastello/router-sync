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
	"router-sync/internal/metrics"
	"router-sync/internal/nats"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server represents the API server. It owns no kernel state; it only mediates
// between the UI and NATS (providers, policies, router state, log levels).
type Server struct {
	config     config.APIConfig
	natsClient nats.NATSClient
	server     *http.Server

	reg                 *prometheus.Registry
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	providersTotal      prometheus.Gauge
	policiesTotal       prometheus.Gauge
	routersKnown        prometheus.Gauge
	stateAgeSeconds     *prometheus.GaugeVec
	logLevelSetTotal    prometheus.Counter

	version   string
	buildTime string
	gitCommit string
}

// NewServer creates a new API server.
func NewServer(cfg config.APIConfig, natsClient nats.NATSClient, version, buildTime, gitCommit string) *Server {
	reg := metrics.NewRegistry()

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

	providersTotal := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "providers_total",
		Help: "Total number of internet providers",
	})

	policiesTotal := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "policies_total",
		Help: "Total number of routing policies",
	})

	routersKnown := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "routers_known",
		Help: "Number of routers reporting state to the API.",
	})

	stateAgeSeconds := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "router_state_age_seconds",
		Help: "Age of the latest router state heartbeat in seconds.",
	}, []string{"hostname"})

	logLevelSetTotal := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "log_level_set_total",
		Help: "Number of log level changes applied via the API.",
	})

	reg.MustRegister(httpRequestsTotal, httpRequestDuration, providersTotal, policiesTotal, routersKnown, stateAgeSeconds, logLevelSetTotal)

	server := &Server{
		config:              cfg,
		natsClient:          natsClient,
		reg:                 reg,
		httpRequestsTotal:   httpRequestsTotal,
		httpRequestDuration: httpRequestDuration,
		providersTotal:      providersTotal,
		policiesTotal:       policiesTotal,
		routersKnown:        routersKnown,
		stateAgeSeconds:     stateAgeSeconds,
		logLevelSetTotal:    logLevelSetTotal,
		version:             version,
		buildTime:           buildTime,
		gitCommit:           gitCommit,
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(server.metricsMiddleware())
	router.Use(server.urlDecodeMiddleware())

	router.RedirectFixedPath = false

	v1 := router.Group("/api/v1")
	{
		providers := v1.Group("/providers")
		{
			providers.GET("", server.listProviders)
			providers.POST("", server.createProvider)
			providers.GET("/:id", server.getProvider)
			providers.PUT("/:id", server.updateProvider)
			providers.DELETE("/:id", server.deleteProvider)
		}

		policies := v1.Group("/policies")
		{
			policies.GET("", server.listPolicies)
			policies.POST("", server.createPolicy)
			policies.GET("/:id", server.getPolicy)
			policies.PUT("/:id", server.updatePolicy)
			policies.DELETE("/:id", server.deletePolicy)
		}

		routers := v1.Group("/routers")
		{
			routers.GET("", server.listRouters)
			routers.GET("/:hostname", server.getRouter)
			routers.GET("/:hostname/interfaces", server.getRouterInterfaces)
			routers.GET("/:hostname/routes", server.getRouterRoutes)
			routers.GET("/:hostname/rules", server.getRouterRules)
		}

		logs := v1.Group("/logging")
		{
			logs.GET("/levels", server.listLogLevels)
			logs.GET("/level", server.getOwnLogLevel)
			logs.PUT("/level", server.setOwnLogLevel)
			logs.GET("/level/:service_id", server.getLogLevelByService)
			logs.PUT("/level/:service_id", server.setLogLevelByService)
		}

		v1.POST("/sync", server.triggerSync)
		v1.GET("/stats", server.getStats)
	}

	docs.SwaggerInfo.Host = ""
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http"}
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/metrics", gin.WrapH(metrics.HandlerFor(reg)))
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

func (s *Server) urlDecodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		"service":   "router-sync-api",
	})
}

// triggerSync is kept as a no-op for compatibility; agents perform sync.
// @Summary Trigger synchronization
// @Description Manually trigger synchronization. Agents perform the actual sync; this endpoint is a no-op in the split architecture.
// @Tags sync
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/sync [post]
func (s *Server) triggerSync(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "Agents continuously sync from NATS; this endpoint is a no-op.",
		"timestamp": time.Now().UTC(),
	})
}

// getStats returns aggregated service statistics
// @Summary Get service statistics
// @Description Get statistics about providers, policies, routers, and the API itself.
// @Tags stats
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/stats [get]
func (s *Server) getStats(c *gin.Context) {
	providers, _ := s.natsClient.ListProviders()
	policies, _ := s.natsClient.ListPolicies()
	states, _ := s.natsClient.ListRouterStates()

	providersCount := len(providers)
	policiesCount := len(policies)

	policiesPerProvider := make(map[string]int)
	for _, p := range policies {
		policiesPerProvider[p.ProviderID]++
	}

	routerInfos := make([]gin.H, 0, len(states))
	now := time.Now().UTC()
	for _, st := range states {
		age := now.Sub(st.LastSeen).Seconds()
		s.stateAgeSeconds.WithLabelValues(st.Hostname).Set(age)
		routerInfos = append(routerInfos, gin.H{
			"hostname":      st.Hostname,
			"agent_version": st.AgentVersion,
			"log_level":     st.LogLevel,
			"last_seen":     st.LastSeen,
			"age_seconds":   age,
		})
	}

	s.providersTotal.Set(float64(providersCount))
	s.policiesTotal.Set(float64(policiesCount))
	s.routersKnown.Set(float64(len(states)))

	stats := gin.H{
		"sync": gin.H{
			"providers_count":       providersCount,
			"policies_count":        policiesCount,
			"policies_per_provider": policiesPerProvider,
		},
		"routers":    routerInfos,
		"log_level":  logging.GetLevelName(),
		"timestamp":  time.Now().UTC(),
		"version":    s.version,
		"build_time": s.buildTime,
		"git_commit": s.gitCommit,
	}

	c.JSON(http.StatusOK, stats)
}
