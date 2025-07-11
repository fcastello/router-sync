package api

import (
	"fmt"
	"net/http"
	"time"

	"router-sync/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateProviderRequest represents a request to create a provider
// The provider ID will be set to the name field
type CreateProviderRequest struct {
	Name        string `json:"name" binding:"required" example:"Telecom"`
	Interface   string `json:"interface" binding:"required" example:"eth0"`
	TableID     int    `json:"table_id" binding:"required,min=1" example:"100"`
	Gateway     string `json:"gateway" binding:"required" example:"192.168.1.1"`
	Description string `json:"description" example:"Primary internet connection"`
}

// UpdateProviderRequest represents a request to update a provider
// If the name is changed, the provider ID will also be updated to match the new name
type UpdateProviderRequest struct {
	Name        string `json:"name" binding:"required" example:"Telecom"`
	Interface   string `json:"interface" binding:"required" example:"eth0"`
	TableID     int    `json:"table_id" binding:"required,min=1" example:"100"`
	Gateway     string `json:"gateway" binding:"required" example:"192.168.1.1"`
	Description string `json:"description" example:"Primary internet connection"`
}

// CreatePolicyRequest represents a request to create a policy
// The source_ip will be used as the policy ID for routing
type CreatePolicyRequest struct {
	Name        string `json:"name" binding:"required" example:"Home Network"`
	SourceIP    string `json:"source_ip" binding:"required" example:"192.168.1.100"`
	ProviderID  string `json:"provider_id" binding:"required" example:"provider-123"`
	Description string `json:"description" example:"Route home network through primary provider"`
	Enabled     bool   `json:"enabled" example:"true"`
}

// UpdatePolicyRequest represents a request to update a policy
// The source_ip will be used as the policy ID for routing
type UpdatePolicyRequest struct {
	Name        string `json:"name" binding:"required" example:"Home Network"`
	SourceIP    string `json:"source_ip" binding:"required" example:"192.168.1.100"`
	ProviderID  string `json:"provider_id" binding:"required" example:"provider-123"`
	Description string `json:"description" example:"Route home network through primary provider"`
	Enabled     bool   `json:"enabled" example:"true"`
}

// listProviders lists all internet providers
// @Summary List providers
// @Description Get all internet providers
// @Tags providers
// @Accept json
// @Produce json
// @Success 200 {array} models.InternetProvider
// @Router /api/v1/providers [get]
func (s *Server) listProviders(c *gin.Context) {
	providers, err := s.natsClient.ListProviders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list providers",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, providers)
}

// createProvider creates a new internet provider
// @Summary Create provider
// @Description Create a new internet provider. The provider ID will be set to the name field.
// @Tags providers
// @Accept json
// @Produce json
// @Param provider body CreateProviderRequest true "Provider information"
// @Success 201 {object} models.InternetProvider
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{} "Provider with same name already exists"
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/providers [post]
func (s *Server) createProvider(c *gin.Context) {
	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Check if a provider with the same name already exists
	existingProvider, err := s.natsClient.GetProvider(req.Name)
	if err == nil && existingProvider != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Provider already exists",
			"details": fmt.Sprintf("A provider with name '%s' already exists", req.Name),
		})
		return
	}

	now := time.Now()
	provider := &models.InternetProvider{
		ID:          req.Name,
		Name:        req.Name,
		Interface:   req.Interface,
		TableID:     req.TableID,
		Gateway:     req.Gateway,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := provider.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	if err := s.natsClient.StoreProvider(provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create provider",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// getProvider gets a specific internet provider
// @Summary Get provider
// @Description Get a specific internet provider by ID
// @Tags providers
// @Accept json
// @Produce json
// @Param id path string true "Provider ID"
// @Success 200 {object} models.InternetProvider
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/providers/{id} [get]
func (s *Server) getProvider(c *gin.Context) {
	id := c.Param("id")

	provider, err := s.natsClient.GetProvider(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Provider not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// updateProvider updates an existing internet provider
// @Summary Update provider
// @Description Update an existing internet provider. If the name is changed, the provider ID will also be updated to match the new name.
// @Tags providers
// @Accept json
// @Produce json
// @Param id path string true "Provider ID"
// @Param provider body UpdateProviderRequest true "Provider information"
// @Success 200 {object} models.InternetProvider
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{} "Provider with new name already exists"
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/providers/{id} [put]
func (s *Server) updateProvider(c *gin.Context) {
	id := c.Param("id")

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get existing provider
	existing, err := s.natsClient.GetProvider(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Provider not found",
			"details": err.Error(),
		})
		return
	}

	// If the name is being changed, check for conflicts and handle ID change
	if existing.Name != req.Name {
		// Check if a provider with the new name already exists
		conflictingProvider, err := s.natsClient.GetProvider(req.Name)
		if err == nil && conflictingProvider != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "Provider name conflict",
				"details": fmt.Sprintf("A provider with name '%s' already exists", req.Name),
			})
			return
		}

		// Delete the old provider (with old ID)
		if err := s.natsClient.DeleteProvider(existing.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update provider",
				"details": "Failed to delete old provider record",
			})
			return
		}

		// Create new provider with new ID
		existing.ID = req.Name
		existing.Name = req.Name
		existing.Interface = req.Interface
		existing.TableID = req.TableID
		existing.Gateway = req.Gateway
		existing.Description = req.Description
		existing.UpdatedAt = time.Now()
	} else {
		// Update fields without changing ID
		existing.Name = req.Name
		existing.Interface = req.Interface
		existing.TableID = req.TableID
		existing.Gateway = req.Gateway
		existing.Description = req.Description
		existing.UpdatedAt = time.Now()
	}

	if err := existing.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	if err := s.natsClient.StoreProvider(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update provider",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// deleteProvider deletes an internet provider
// @Summary Delete provider
// @Description Delete an internet provider
// @Tags providers
// @Accept json
// @Produce json
// @Param id path string true "Provider ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/providers/{id} [delete]
func (s *Server) deleteProvider(c *gin.Context) {
	id := c.Param("id")

	if err := s.natsClient.DeleteProvider(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete provider",
			"details": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// listPolicies lists all routing policies
// @Summary List policies
// @Description Get all routing policies
// @Tags policies
// @Accept json
// @Produce json
// @Success 200 {array} models.RoutingPolicy
// @Router /api/v1/policies [get]
func (s *Server) listPolicies(c *gin.Context) {
	policies, err := s.natsClient.ListPolicies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list policies",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, policies)
}

// createPolicy creates a new routing policy
// @Summary Create policy
// @Description Create a new routing policy
// @Tags policies
// @Accept json
// @Produce json
// @Param policy body CreatePolicyRequest true "Policy information"
// @Success 201 {object} models.RoutingPolicy
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/policies [post]
func (s *Server) createPolicy(c *gin.Context) {
	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	now := time.Now()
	policy := &models.RoutingPolicy{
		ID:          req.SourceIP,
		Name:        req.Name,
		ProviderID:  req.ProviderID,
		Description: req.Description,
		Enabled:     req.Enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := policy.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Verify provider exists
	if _, err := s.natsClient.GetProvider(req.ProviderID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Provider not found",
			"details": "The specified provider ID does not exist",
		})
		return
	}

	if err := s.natsClient.StorePolicy(policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create policy",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// getPolicy gets a specific routing policy
// @Summary Get policy
// @Description Get a specific routing policy by ID. For CIDR-based IDs, use underscore instead of slash (e.g., 192.168.2.0_25 for 192.168.2.0/25)
// @Tags policies
// @Accept json
// @Produce json
// @Param id path string true "Policy ID (use underscore for CIDR, e.g., 192.168.2.0_25)"
// @Success 200 {object} models.RoutingPolicy
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/policies/{id} [get]
func (s *Server) getPolicy(c *gin.Context) {
	id := c.Param("id")

	policy, err := s.natsClient.GetPolicy(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Policy not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// updatePolicy updates an existing routing policy
// @Summary Update policy
// @Description Update an existing routing policy. For CIDR-based IDs, use underscore instead of slash (e.g., 192.168.2.0_25 for 192.168.2.0/25)
// @Tags policies
// @Accept json
// @Produce json
// @Param id path string true "Policy ID (use underscore for CIDR, e.g., 192.168.2.0_25)"
// @Param policy body UpdatePolicyRequest true "Policy information"
// @Success 200 {object} models.RoutingPolicy
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/policies/{id} [put]
func (s *Server) updatePolicy(c *gin.Context) {
	id := c.Param("id")

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get existing policy
	existing, err := s.natsClient.GetPolicy(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Policy not found",
			"details": err.Error(),
		})
		return
	}

	// Update fields
	existing.Name = req.Name
	existing.ID = req.SourceIP
	existing.ProviderID = req.ProviderID
	existing.Description = req.Description
	existing.Enabled = req.Enabled
	existing.UpdatedAt = time.Now()

	if err := existing.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Verify provider exists
	if _, err := s.natsClient.GetProvider(req.ProviderID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Provider not found",
			"details": "The specified provider ID does not exist",
		})
		return
	}

	if err := s.natsClient.StorePolicy(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update policy",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// deletePolicy deletes a routing policy
// @Summary Delete policy
// @Description Delete a routing policy. For CIDR-based IDs, use underscore instead of slash (e.g., 192.168.2.0_25 for 192.168.2.0/25)
// @Tags policies
// @Accept json
// @Produce json
// @Param id path string true "Policy ID (use underscore for CIDR, e.g., 192.168.2.0_25)"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/policies/{id} [delete]
func (s *Server) deletePolicy(c *gin.Context) {
	id := c.Param("id")

	if err := s.natsClient.DeletePolicy(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete policy",
			"details": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}
