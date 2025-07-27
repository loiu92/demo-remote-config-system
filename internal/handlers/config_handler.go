package handlers

import (
	"net/http"
	"strconv"
	"time"

	"remote-config-system/internal/models"
	"remote-config-system/internal/services"

	"github.com/gin-gonic/gin"
)

// ConfigHandler handles configuration-related HTTP requests
type ConfigHandler struct {
	configService services.ConfigServiceInterface
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(configService services.ConfigServiceInterface) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

// GetConfig handles GET /api/config/:org/:app/:env
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")

	config, err := h.configService.GetConfiguration(orgSlug, appSlug, envSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:     "not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	// Set cache headers
	c.Header("Cache-Control", "public, max-age=300") // 5 minutes
	c.Header("ETag", `"`+strconv.Itoa(config.Version)+`"`)

	// Check if client has the latest version
	if match := c.GetHeader("If-None-Match"); match != "" {
		if match == `"`+strconv.Itoa(config.Version)+`"` {
			c.Status(http.StatusNotModified)
			return
		}
	}

	c.JSON(http.StatusOK, config)
}

// GetConfigByAPIKey handles GET /api/config/:env with API key authentication
func (h *ConfigHandler) GetConfigByAPIKey(c *gin.Context) {
	envSlug := c.Param("env")

	// Get API key from context (set by middleware)
	apiKey, exists := c.Get("api_key")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:     "unauthorized",
			Message:   "API key is required",
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	config, err := h.configService.GetConfigurationByAPIKey(apiKey.(string), envSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:     "not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	// Set cache headers
	c.Header("Cache-Control", "public, max-age=300") // 5 minutes
	c.Header("ETag", `"`+strconv.Itoa(config.Version)+`"`)

	// Check if client has the latest version
	if match := c.GetHeader("If-None-Match"); match != "" {
		if match == `"`+strconv.Itoa(config.Version)+`"` {
			c.Status(http.StatusNotModified)
			return
		}
	}

	c.JSON(http.StatusOK, config)
}

// UpdateConfig handles PUT /admin/orgs/:org/apps/:app/envs/:env
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")

	var req models.CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "bad_request",
			Message:   "Invalid request body: " + err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	config, err := h.configService.UpdateConfiguration(orgSlug, appSlug, envSlug, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "environment not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "invalid JSON configuration" {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "update_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, config)
}

// RollbackConfig handles POST /admin/orgs/:org/apps/:app/envs/:env/rollback
func (h *ConfigHandler) RollbackConfig(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")

	var req models.RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "bad_request",
			Message:   "Invalid request body: " + err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	config, err := h.configService.RollbackConfiguration(orgSlug, appSlug, envSlug, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "environment not found" || err.Error() == "target version not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "rollback_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetConfigHistory handles GET /admin/orgs/:org/apps/:app/envs/:env/history
func (h *ConfigHandler) GetConfigHistory(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")

	// Parse pagination parameters
	params := models.DefaultPaginationParams()
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
			params.PageSize = ps
		}
	}

	history, err := h.configService.GetConfigurationHistory(orgSlug, appSlug, envSlug, params)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "environment not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "history_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetConfigVersion handles GET /admin/orgs/:org/apps/:app/envs/:env/history/:version
func (h *ConfigHandler) GetConfigVersion(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")
	versionStr := c.Param("version")

	// Parse version parameter
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:     "bad_request",
			Message:   "Invalid version parameter: " + err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	config, err := h.configService.GetConfigurationVersion(orgSlug, appSlug, envSlug, version)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "environment not found" || err.Error() == "configuration version not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "version_not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	// Set cache headers for historical versions (longer cache time since they don't change)
	c.Header("Cache-Control", "public, max-age=3600") // 1 hour
	c.Header("ETag", `"`+strconv.Itoa(config.Version)+`"`)

	// Check if client has the version cached
	if match := c.GetHeader("If-None-Match"); match != "" {
		if match == `"`+strconv.Itoa(config.Version)+`"` {
			c.Status(http.StatusNotModified)
			return
		}
	}

	c.JSON(http.StatusOK, config)
}

// GetConfigChanges handles GET /admin/orgs/:org/apps/:app/envs/:env/changes
func (h *ConfigHandler) GetConfigChanges(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")

	// Parse pagination parameters
	params := models.DefaultPaginationParams()
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
			params.PageSize = ps
		}
	}

	changes, err := h.configService.GetConfigurationChanges(orgSlug, appSlug, envSlug, params)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "environment not found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:     "changes_failed",
			Message:   err.Error(),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	c.JSON(http.StatusOK, changes)
}

// HealthCheck handles GET /health
func (h *ConfigHandler) HealthCheck(c *gin.Context) {
	services := h.configService.HealthCheck()

	c.JSON(http.StatusOK, models.HealthResponse{
		Status:    "ok",
		Message:   "Remote Config System is running",
		Timestamp: time.Now(),
		Services:  services,
	})
}
