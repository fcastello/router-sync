package api

import (
	"net/http"
	"sort"
	"time"

	"router-sync/internal/logging"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LogLevelResponse describes the current runtime log level for a service.
type LogLevelResponse struct {
	ServiceID string   `json:"service_id"`
	Level     string   `json:"level"`
	Levels    []string `json:"levels"`
}

// LogLevelsResponse describes log levels for all known services.
type LogLevelsResponse struct {
	Services map[string]ServiceLevel `json:"services"`
	Levels   []string                `json:"levels"`
}

// ServiceLevel pairs a service id with its level and a hint about its source.
type ServiceLevel struct {
	Level   string `json:"level"`
	Source  string `json:"source,omitempty"`  // "live" or "persisted"
	Online  bool   `json:"online,omitempty"`
}

// SetLogLevelRequest updates the runtime log level for a service.
type SetLogLevelRequest struct {
	Level string `json:"level" binding:"required" example:"debug"`
}

// getOwnLogLevel returns the API's current log level (back-compat for old clients).
// @Summary Get API log level
// @Tags logging
// @Produce json
// @Success 200 {object} LogLevelResponse
// @Router /api/v1/logging/level [get]
func (s *Server) getOwnLogLevel(c *gin.Context) {
	c.JSON(http.StatusOK, LogLevelResponse{
		ServiceID: logging.ServiceID(),
		Level:     logging.GetLevelName(),
		Levels:    logging.LevelNames(),
	})
}

// setOwnLogLevel changes the API's log level (back-compat for old clients).
// @Summary Set API log level
// @Tags logging
// @Accept json
// @Produce json
// @Param body body SetLogLevelRequest true "Log level"
// @Success 200 {object} LogLevelResponse
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/logging/level [put]
func (s *Server) setOwnLogLevel(c *gin.Context) {
	var req SetLogLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	level, err := logging.ParseLevel(req.Level)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log level", "details": err.Error()})
		return
	}

	prev := logging.GetLevelName()
	logging.SetLevel(level)
	if sid := logging.ServiceID(); sid != "" {
		if err := s.natsClient.SetServiceLogLevel(sid, level.String()); err != nil {
			logrus.Warnf("Failed to persist log level for %s: %v", sid, err)
		}
	}
	s.logLevelSetTotal.Inc()
	logrus.Infof("Log level changed from %s to %s via API", prev, level.String())

	c.JSON(http.StatusOK, LogLevelResponse{
		ServiceID: logging.ServiceID(),
		Level:     logging.GetLevelName(),
		Levels:    logging.LevelNames(),
	})
}

// getLogLevelByService returns the persisted log level for a named service.
// service_id is "api" or "agent.<hostname>".
// @Summary Get log level by service
// @Tags logging
// @Produce json
// @Param service_id path string true "Service ID (e.g. api, agent.r1)"
// @Success 200 {object} LogLevelResponse
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/logging/level/{service_id} [get]
func (s *Server) getLogLevelByService(c *gin.Context) {
	serviceID := c.Param("service_id")

	if serviceID == logging.ServiceID() {
		c.JSON(http.StatusOK, LogLevelResponse{
			ServiceID: serviceID,
			Level:     logging.GetLevelName(),
			Levels:    logging.LevelNames(),
		})
		return
	}

	level, err := s.natsClient.GetServiceLogLevel(serviceID)
	if err != nil || level == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Service log level not found",
			"details": "no persisted level for " + serviceID,
		})
		return
	}
	c.JSON(http.StatusOK, LogLevelResponse{
		ServiceID: serviceID,
		Level:     level,
		Levels:    logging.LevelNames(),
	})
}

// setLogLevelByService persists a log level under level.<service_id>. The owning
// service (which watches its own key) applies it. The API also applies it locally
// when service_id matches the API's own service ID.
// @Summary Set log level by service
// @Tags logging
// @Accept json
// @Produce json
// @Param service_id path string true "Service ID (e.g. api, agent.r1)"
// @Param body body SetLogLevelRequest true "Log level"
// @Success 200 {object} LogLevelResponse
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/logging/level/{service_id} [put]
func (s *Server) setLogLevelByService(c *gin.Context) {
	serviceID := c.Param("service_id")
	var req SetLogLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	level, err := logging.ParseLevel(req.Level)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log level", "details": err.Error()})
		return
	}

	if err := s.natsClient.SetServiceLogLevel(serviceID, level.String()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist log level", "details": err.Error()})
		return
	}

	if serviceID == logging.ServiceID() {
		prev := logging.GetLevelName()
		logging.SetLevel(level)
		logrus.Infof("Log level changed from %s to %s via API", prev, level.String())
	}
	s.logLevelSetTotal.Inc()

	c.JSON(http.StatusOK, LogLevelResponse{
		ServiceID: serviceID,
		Level:     level.String(),
		Levels:    logging.LevelNames(),
	})
}

// listLogLevels merges persisted log levels with live agent state to surface
// per-service log levels (api, agent.r1, agent.r2, ...).
// @Summary List log levels
// @Tags logging
// @Produce json
// @Success 200 {object} LogLevelsResponse
// @Router /api/v1/logging/levels [get]
func (s *Server) listLogLevels(c *gin.Context) {
	persisted, err := s.natsClient.ListServiceLogLevels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list log levels",
			"details": err.Error(),
		})
		return
	}

	services := make(map[string]ServiceLevel)

	if apiID := logging.ServiceID(); apiID != "" {
		services[apiID] = ServiceLevel{
			Level:  logging.GetLevelName(),
			Source: "live",
			Online: true,
		}
	}

	states, err := s.natsClient.ListRouterStates()
	if err == nil {
		now := time.Now().UTC()
		for _, st := range states {
			id := "agent." + st.Hostname
			level := st.LogLevel
			if level == "" {
				level = persisted[id]
			}
			services[id] = ServiceLevel{
				Level:  level,
				Source: "live",
				Online: now.Sub(st.LastSeen).Seconds() < 30,
			}
		}
	}

	for id, lvl := range persisted {
		if _, ok := services[id]; ok {
			continue
		}
		services[id] = ServiceLevel{Level: lvl, Source: "persisted"}
	}

	keys := make([]string, 0, len(services))
	for k := range services {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]ServiceLevel, len(services))
	for _, k := range keys {
		ordered[k] = services[k]
	}

	c.JSON(http.StatusOK, LogLevelsResponse{
		Services: ordered,
		Levels:   logging.LevelNames(),
	})
}
