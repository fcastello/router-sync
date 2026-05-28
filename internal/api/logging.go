package api

import (
	"net/http"

	"router-sync/internal/logging"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LogLevelResponse describes the current runtime log level.
type LogLevelResponse struct {
	Level  string   `json:"level"`
	Levels []string `json:"levels"`
}

// SetLogLevelRequest updates the runtime log level.
type SetLogLevelRequest struct {
	Level string `json:"level" binding:"required" example:"debug"`
}

// getLogLevel returns the current log level.
// @Summary Get log level
// @Description Get the current runtime logging verbosity
// @Tags logging
// @Produce json
// @Success 200 {object} LogLevelResponse
// @Router /api/v1/logging/level [get]
func (s *Server) getLogLevel(c *gin.Context) {
	c.JSON(http.StatusOK, LogLevelResponse{
		Level:  logging.GetLevelName(),
		Levels: logging.LevelNames(),
	})
}

// setLogLevel changes the runtime log level without restarting the service.
// @Summary Set log level
// @Description Change runtime logging verbosity (trace, debug, info, warn, error, fatal, panic)
// @Tags logging
// @Accept json
// @Produce json
// @Param body body SetLogLevelRequest true "Log level"
// @Success 200 {object} LogLevelResponse
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/logging/level [put]
func (s *Server) setLogLevel(c *gin.Context) {
	var req SetLogLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	level, err := logging.ParseLevel(req.Level)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid log level",
			"details": err.Error(),
		})
		return
	}

	prev := logging.GetLevelName()
	logging.SetLevel(level)
	logrus.Infof("Log level changed from %s to %s via API", prev, level.String())

	c.JSON(http.StatusOK, LogLevelResponse{
		Level:  logging.GetLevelName(),
		Levels: logging.LevelNames(),
	})
}
