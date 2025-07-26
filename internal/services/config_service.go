package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"remote-config-system/internal/cache"
	"remote-config-system/internal/db"
	"remote-config-system/internal/models"
)

// ConfigService handles configuration business logic
type ConfigService struct {
	repos *db.Repositories
	cache *cache.RedisClient
}

// NewConfigService creates a new configuration service
func NewConfigService(repos *db.Repositories, cacheClient *cache.RedisClient) *ConfigService {
	return &ConfigService{
		repos: repos,
		cache: cacheClient,
	}
}

// GetConfiguration retrieves the active configuration for an environment
func (s *ConfigService) GetConfiguration(orgSlug, appSlug, envSlug string) (*models.ConfigResponse, error) {
	// Try to get from cache first
	if s.cache != nil {
		cacheKey := cache.GenerateConfigKey(orgSlug, appSlug, envSlug)
		if cachedData, err := s.cache.GetConfig(cacheKey); err == nil && cachedData != nil {
			var response models.ConfigResponse
			if err := json.Unmarshal(cachedData, &response); err == nil {
				log.Printf("Cache hit for config: %s", cacheKey)
				return &response, nil
			}
			log.Printf("Failed to unmarshal cached config: %v", err)
		}
	}

	// Get the environment with all relationships
	env, err := s.repos.Environments.GetBySlug(orgSlug, appSlug, envSlug)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Get the active configuration version
	configVersion, err := s.repos.ConfigVersions.GetActiveByEnvironment(env.ID)
	if err != nil {
		return nil, fmt.Errorf("no active configuration found: %w", err)
	}

	// Build the response
	response := &models.ConfigResponse{
		Organization: env.Application.Organization.Slug,
		Application:  env.Application.Slug,
		Environment:  env.Slug,
		Version:      configVersion.Version,
		Config:       configVersion.ConfigJSON,
		UpdatedAt:    configVersion.CreatedAt,
	}

	// Cache the response
	if s.cache != nil {
		cacheKey := cache.GenerateConfigKey(orgSlug, appSlug, envSlug)
		if err := s.cache.SetConfig(cacheKey, response); err != nil {
			log.Printf("Failed to cache config: %v", err)
		} else {
			log.Printf("Cached config: %s", cacheKey)
		}
	}

	return response, nil
}

// GetConfigurationByAPIKey retrieves configuration using API key authentication
func (s *ConfigService) GetConfigurationByAPIKey(apiKey, envSlug string) (*models.ConfigResponse, error) {
	// Try to get from cache first
	if s.cache != nil {
		cacheKey := cache.GenerateAPIKeyConfigKey(apiKey, envSlug)
		if cachedData, err := s.cache.GetConfig(cacheKey); err == nil && cachedData != nil {
			var response models.ConfigResponse
			if err := json.Unmarshal(cachedData, &response); err == nil {
				log.Printf("Cache hit for API key config: %s", cacheKey)
				return &response, nil
			}
			log.Printf("Failed to unmarshal cached API key config: %v", err)
		}
	}

	// Get the application by API key
	app, err := s.repos.Applications.GetByAPIKey(apiKey)
	if err != nil {
		return nil, fmt.Errorf("invalid API key: %w", err)
	}

	// Get the environment
	env, err := s.repos.Environments.GetBySlug(app.Organization.Slug, app.Slug, envSlug)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Get the active configuration version
	configVersion, err := s.repos.ConfigVersions.GetActiveByEnvironment(env.ID)
	if err != nil {
		return nil, fmt.Errorf("no active configuration found: %w", err)
	}

	// Build the response
	response := &models.ConfigResponse{
		Organization: app.Organization.Slug,
		Application:  app.Slug,
		Environment:  env.Slug,
		Version:      configVersion.Version,
		Config:       configVersion.ConfigJSON,
		UpdatedAt:    configVersion.CreatedAt,
	}

	// Cache the response
	if s.cache != nil {
		cacheKey := cache.GenerateAPIKeyConfigKey(apiKey, envSlug)
		if err := s.cache.SetConfig(cacheKey, response); err != nil {
			log.Printf("Failed to cache API key config: %v", err)
		} else {
			log.Printf("Cached API key config: %s", cacheKey)
		}
	}

	return response, nil
}

// UpdateConfiguration creates a new configuration version and sets it as active
func (s *ConfigService) UpdateConfiguration(orgSlug, appSlug, envSlug string, req *models.CreateConfigRequest) (*models.ConfigResponse, error) {
	// Get the environment
	env, err := s.repos.Environments.GetBySlug(orgSlug, appSlug, envSlug)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Validate JSON
	var configData interface{}
	if err := json.Unmarshal(req.Config, &configData); err != nil {
		return nil, fmt.Errorf("invalid JSON configuration: %w", err)
	}

	// Get the current active version (if any) for change logging
	var currentVersion *int
	if activeConfig, err := s.repos.ConfigVersions.GetActiveByEnvironment(env.ID); err == nil {
		currentVersion = &activeConfig.Version
	}

	// Create new configuration version
	newVersion := &models.ConfigVersion{
		EnvID:      env.ID,
		ConfigJSON: req.Config,
		IsActive:   true,
		CreatedBy:  req.CreatedBy,
	}

	if err := s.repos.ConfigVersions.Create(newVersion); err != nil {
		return nil, fmt.Errorf("failed to create configuration version: %w", err)
	}

	// Log the change
	change := &models.ConfigChange{
		EnvID:       env.ID,
		VersionFrom: currentVersion,
		VersionTo:   newVersion.Version,
		Action:      "update",
		CreatedBy:   req.CreatedBy,
	}

	if err := s.repos.ConfigChanges.Create(change); err != nil {
		log.Printf("Failed to log configuration change: %v", err)
	}

	// Invalidate cache for this configuration
	if s.cache != nil {
		// Invalidate both regular and API key caches
		configKey := cache.GenerateConfigKey(env.Application.Organization.Slug, env.Application.Slug, env.Slug)
		if err := s.cache.DeleteConfig(configKey); err != nil {
			log.Printf("Failed to invalidate config cache: %v", err)
		}

		// Invalidate API key cache pattern for this environment
		pattern := fmt.Sprintf("config:api:*:%s", env.Slug)
		if err := s.cache.InvalidatePattern(pattern); err != nil {
			log.Printf("Failed to invalidate API key cache pattern: %v", err)
		}

		log.Printf("Invalidated cache for config update: %s", configKey)
	}

	// Build the response
	response := &models.ConfigResponse{
		Organization: env.Application.Organization.Slug,
		Application:  env.Application.Slug,
		Environment:  env.Slug,
		Version:      newVersion.Version,
		Config:       newVersion.ConfigJSON,
		UpdatedAt:    newVersion.CreatedAt,
	}

	return response, nil
}

// RollbackConfiguration rolls back to a previous configuration version
func (s *ConfigService) RollbackConfiguration(orgSlug, appSlug, envSlug string, req *models.RollbackRequest) (*models.ConfigResponse, error) {
	// Get the environment
	env, err := s.repos.Environments.GetBySlug(orgSlug, appSlug, envSlug)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Get the current active version
	currentConfig, err := s.repos.ConfigVersions.GetActiveByEnvironment(env.ID)
	if err != nil {
		return nil, fmt.Errorf("no active configuration found: %w", err)
	}

	// Check if the target version exists
	targetConfig, err := s.repos.ConfigVersions.GetByVersion(env.ID, req.ToVersion)
	if err != nil {
		return nil, fmt.Errorf("target version not found: %w", err)
	}

	// Set the target version as active
	if err := s.repos.ConfigVersions.SetActive(env.ID, req.ToVersion); err != nil {
		return nil, fmt.Errorf("failed to rollback configuration: %w", err)
	}

	// Log the rollback
	change := &models.ConfigChange{
		EnvID:       env.ID,
		VersionFrom: &currentConfig.Version,
		VersionTo:   req.ToVersion,
		Action:      "rollback",
		CreatedBy:   req.CreatedBy,
	}

	if err := s.repos.ConfigChanges.Create(change); err != nil {
		log.Printf("Failed to log configuration rollback: %v", err)
	}

	// Invalidate cache for this configuration
	if s.cache != nil {
		// Invalidate both regular and API key caches
		configKey := cache.GenerateConfigKey(env.Application.Organization.Slug, env.Application.Slug, env.Slug)
		if err := s.cache.DeleteConfig(configKey); err != nil {
			log.Printf("Failed to invalidate config cache: %v", err)
		}

		// Invalidate API key cache pattern for this environment
		pattern := fmt.Sprintf("config:api:*:%s", env.Slug)
		if err := s.cache.InvalidatePattern(pattern); err != nil {
			log.Printf("Failed to invalidate API key cache pattern: %v", err)
		}

		log.Printf("Invalidated cache for config rollback: %s", configKey)
	}

	// Build the response
	response := &models.ConfigResponse{
		Organization: env.Application.Organization.Slug,
		Application:  env.Application.Slug,
		Environment:  env.Slug,
		Version:      targetConfig.Version,
		Config:       targetConfig.ConfigJSON,
		UpdatedAt:    time.Now(), // Use current time for rollback
	}

	return response, nil
}

// GetConfigurationHistory retrieves the version history for an environment
func (s *ConfigService) GetConfigurationHistory(orgSlug, appSlug, envSlug string, params models.PaginationParams) (*models.PaginatedResponse, error) {
	// Get the environment
	env, err := s.repos.Environments.GetBySlug(orgSlug, appSlug, envSlug)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Get configuration versions
	versions, totalCount, err := s.repos.ConfigVersions.ListByEnvironment(env.ID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration history: %w", err)
	}

	// Convert to response format
	var responseData []map[string]interface{}
	for _, version := range versions {
		responseData = append(responseData, map[string]interface{}{
			"version":    version.Version,
			"is_active":  version.IsActive,
			"created_at": version.CreatedAt,
			"created_by": version.CreatedBy,
		})
	}

	response := models.NewPaginatedResponse(responseData, params.Page, params.PageSize, totalCount)
	return &response, nil
}

// GetConfigurationChanges retrieves the change history for an environment
func (s *ConfigService) GetConfigurationChanges(orgSlug, appSlug, envSlug string, params models.PaginationParams) (*models.PaginatedResponse, error) {
	// Get the environment
	env, err := s.repos.Environments.GetBySlug(orgSlug, appSlug, envSlug)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Get configuration changes
	changes, totalCount, err := s.repos.ConfigChanges.ListByEnvironment(env.ID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration changes: %w", err)
	}

	// Convert to response format
	var responseData []map[string]interface{}
	for _, change := range changes {
		responseData = append(responseData, map[string]interface{}{
			"id":           change.ID,
			"version_from": change.VersionFrom,
			"version_to":   change.VersionTo,
			"action":       change.Action,
			"created_at":   change.CreatedAt,
			"created_by":   change.CreatedBy,
		})
	}

	response := models.NewPaginatedResponse(responseData, params.Page, params.PageSize, totalCount)
	return &response, nil
}

// ValidateAPIKey validates an API key and returns the associated application
func (s *ConfigService) ValidateAPIKey(apiKey string) (*models.Application, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	app, err := s.repos.Applications.GetByAPIKey(apiKey)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	return app, nil
}

// HealthCheck returns the health status of the service and its dependencies
func (s *ConfigService) HealthCheck() map[string]string {
	services := make(map[string]string)

	// Always assume database is connected since we got this far
	services["database"] = "connected"

	// Check cache health
	if s.cache != nil {
		if err := s.cache.Health(); err != nil {
			services["cache"] = "disconnected"
		} else {
			services["cache"] = "connected"
		}
	} else {
		services["cache"] = "disabled"
	}

	return services
}
