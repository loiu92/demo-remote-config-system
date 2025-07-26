package handlers

import (
	"net/http"
	"strconv"
	"time"

	"remote-config-system/internal/models"
	"remote-config-system/internal/services"

	"github.com/gin-gonic/gin"
)

// ManagementHandler handles management API endpoints
type ManagementHandler struct {
	configService *services.ConfigService
}

// NewManagementHandler creates a new management handler
func NewManagementHandler(configService *services.ConfigService) *ManagementHandler {
	return &ManagementHandler{
		configService: configService,
	}
}

// Organization Management Endpoints

// ListOrganizations handles GET /admin/orgs
func (h *ManagementHandler) ListOrganizations(c *gin.Context) {
	params := models.DefaultPaginationParams()
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "invalid_parameters",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	response, err := h.configService.ListOrganizations(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:     "list_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetOrganization handles GET /admin/orgs/:org
func (h *ManagementHandler) GetOrganization(c *gin.Context) {
	orgSlug := c.Param("org")

	org, err := h.configService.GetOrganization(orgSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:     "not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, org)
}

// CreateOrganization handles POST /admin/orgs
func (h *ManagementHandler) CreateOrganization(c *gin.Context) {
	var req models.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "invalid_request",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	org, err := h.configService.CreateOrganization(&req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "organization with slug '"+req.Slug+"' already exists" {
			statusCode = http.StatusConflict
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "creation_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusCreated, org)
}

// UpdateOrganization handles PUT /admin/orgs/:org
func (h *ManagementHandler) UpdateOrganization(c *gin.Context) {
	orgSlug := c.Param("org")

	var req models.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "invalid_request",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	org, err := h.configService.UpdateOrganization(orgSlug, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "organization not found: "+orgSlug {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "update_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, org)
}

// DeleteOrganization handles DELETE /admin/orgs/:org
func (h *ManagementHandler) DeleteOrganization(c *gin.Context) {
	orgSlug := c.Param("org")

	err := h.configService.DeleteOrganization(orgSlug)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "organization not found: "+orgSlug {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "deletion_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Application Management Endpoints

// ListApplications handles GET /admin/orgs/:org/apps
func (h *ManagementHandler) ListApplications(c *gin.Context) {
	orgSlug := c.Param("org")

	params := models.DefaultPaginationParams()
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "invalid_parameters",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	response, err := h.configService.ListApplications(orgSlug, params)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "organization not found: "+orgSlug {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "list_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetApplication handles GET /admin/orgs/:org/apps/:app
func (h *ManagementHandler) GetApplication(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")

	app, err := h.configService.GetApplication(orgSlug, appSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:     "not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, app)
}

// CreateApplication handles POST /admin/orgs/:org/apps
func (h *ManagementHandler) CreateApplication(c *gin.Context) {
	orgSlug := c.Param("org")

	var req models.CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "invalid_request",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	app, err := h.configService.CreateApplication(orgSlug, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "organization not found: "+orgSlug {
			statusCode = http.StatusNotFound
		} else if err.Error() == "application with slug '"+req.Slug+"' already exists in organization '"+orgSlug+"'" {
			statusCode = http.StatusConflict
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "creation_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusCreated, app)
}

// UpdateApplication handles PUT /admin/orgs/:org/apps/:app
func (h *ManagementHandler) UpdateApplication(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")

	var req models.UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "invalid_request",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	app, err := h.configService.UpdateApplication(orgSlug, appSlug, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "application not found: "+appSlug {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "update_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, app)
}

// DeleteApplication handles DELETE /admin/orgs/:org/apps/:app
func (h *ManagementHandler) DeleteApplication(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")

	err := h.configService.DeleteApplication(orgSlug, appSlug)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "application not found: "+appSlug {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "deletion_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Environment Management Endpoints

// ListEnvironments handles GET /admin/orgs/:org/apps/:app/envs
func (h *ManagementHandler) ListEnvironments(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")

	params := models.DefaultPaginationParams()
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "invalid_parameters",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	response, err := h.configService.ListEnvironments(orgSlug, appSlug, params)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "application not found: "+appSlug {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "list_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetEnvironment handles GET /admin/orgs/:org/apps/:app/envs/:env
func (h *ManagementHandler) GetEnvironment(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")

	env, err := h.configService.GetEnvironment(orgSlug, appSlug, envSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:     "not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, env)
}

// CreateEnvironment handles POST /admin/orgs/:org/apps/:app/envs
func (h *ManagementHandler) CreateEnvironment(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")

	var req models.CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "invalid_request",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	env, err := h.configService.CreateEnvironment(orgSlug, appSlug, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "application not found: "+appSlug {
			statusCode = http.StatusNotFound
		} else if err.Error() == "environment with slug '"+req.Slug+"' already exists in application '"+appSlug+"'" {
			statusCode = http.StatusConflict
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "creation_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusCreated, env)
}

// UpdateEnvironment handles PUT /admin/orgs/:org/apps/:app/envs/:env
func (h *ManagementHandler) UpdateEnvironment(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")

	var req models.UpdateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "invalid_request",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	env, err := h.configService.UpdateEnvironment(orgSlug, appSlug, envSlug, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "environment not found: "+envSlug {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "update_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, env)
}

// DeleteEnvironment handles DELETE /admin/orgs/:org/apps/:app/envs/:env
func (h *ManagementHandler) DeleteEnvironment(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")

	err := h.configService.DeleteEnvironment(orgSlug, appSlug, envSlug)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "environment not found: "+envSlug {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "deletion_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Cache Management Endpoints

// GetCacheStats handles GET /admin/cache/stats
func (h *ManagementHandler) GetCacheStats(c *gin.Context) {
	stats, err := h.configService.GetCacheStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:     "cache_stats_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// WarmCache handles POST /admin/cache/warm
func (h *ManagementHandler) WarmCache(c *gin.Context) {
	err := h.configService.WarmCache()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:     "cache_warm_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Cache warming completed successfully",
		"timestamp": time.Now(),
	})
}

// ClearCache handles DELETE /admin/cache
func (h *ManagementHandler) ClearCache(c *gin.Context) {
	err := h.configService.ClearCache()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:     "cache_clear_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Cache cleared successfully",
		"timestamp": time.Now(),
	})
}
