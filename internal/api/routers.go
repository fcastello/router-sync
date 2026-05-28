package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// listRouters returns all routers reporting state in the router-sync-state bucket.
// @Summary List routers
// @Description List all routers reporting state via NATS, including last heartbeat age.
// @Tags routers
// @Produce json
// @Success 200 {array} models.RouterState
// @Router /api/v1/routers [get]
func (s *Server) listRouters(c *gin.Context) {
	states, err := s.natsClient.ListRouterStates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list router states",
			"details": err.Error(),
		})
		return
	}

	now := time.Now().UTC()
	out := make([]gin.H, 0, len(states))
	for _, st := range states {
		age := now.Sub(st.LastSeen).Seconds()
		online := age < 30
		s.stateAgeSeconds.WithLabelValues(st.Hostname).Set(age)
		out = append(out, gin.H{
			"hostname":      st.Hostname,
			"agent_version": st.AgentVersion,
			"log_level":     st.LogLevel,
			"last_seen":     st.LastSeen,
			"age_seconds":   age,
			"online":        online,
			"interfaces":    st.Interfaces,
			"tables":        st.Tables,
			"rules":         st.Rules,
		})
	}
	s.routersKnown.Set(float64(len(states)))
	c.JSON(http.StatusOK, out)
}

// getRouter returns the full state for a single router.
// @Summary Get router state
// @Tags routers
// @Produce json
// @Param hostname path string true "Router hostname"
// @Success 200 {object} models.RouterState
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/routers/{hostname} [get]
func (s *Server) getRouter(c *gin.Context) {
	hostname := c.Param("hostname")
	state, err := s.natsClient.GetRouterState(hostname)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Router not found",
			"details": err.Error(),
		})
		return
	}

	now := time.Now().UTC()
	age := now.Sub(state.LastSeen).Seconds()
	c.JSON(http.StatusOK, gin.H{
		"hostname":      state.Hostname,
		"agent_version": state.AgentVersion,
		"log_level":     state.LogLevel,
		"last_seen":     state.LastSeen,
		"age_seconds":   age,
		"online":        age < 30,
		"interfaces":    state.Interfaces,
		"tables":        state.Tables,
		"rules":         state.Rules,
	})
}

// getRouterInterfaces returns just the interfaces for the named router.
// @Summary Get router interfaces
// @Tags routers
// @Produce json
// @Param hostname path string true "Router hostname"
// @Success 200 {array} models.Interface
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/routers/{hostname}/interfaces [get]
func (s *Server) getRouterInterfaces(c *gin.Context) {
	hostname := c.Param("hostname")
	state, err := s.natsClient.GetRouterState(hostname)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Router not found",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, state.Interfaces)
}

// getRouterRoutes returns the routing tables for the named router.
// @Summary Get router routes
// @Tags routers
// @Produce json
// @Param hostname path string true "Router hostname"
// @Success 200 {array} models.RoutingTable
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/routers/{hostname}/routes [get]
func (s *Server) getRouterRoutes(c *gin.Context) {
	hostname := c.Param("hostname")
	state, err := s.natsClient.GetRouterState(hostname)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Router not found",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, state.Tables)
}

// getRouterRules returns the ip rules for the named router.
// @Summary Get router rules
// @Tags routers
// @Produce json
// @Param hostname path string true "Router hostname"
// @Success 200 {array} models.IPRule
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/routers/{hostname}/rules [get]
func (s *Server) getRouterRules(c *gin.Context) {
	hostname := c.Param("hostname")
	state, err := s.natsClient.GetRouterState(hostname)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Router not found",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, state.Rules)
}
