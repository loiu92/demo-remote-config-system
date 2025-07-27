package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"remote-config-system/internal/cache"
	"remote-config-system/internal/db"
	"remote-config-system/internal/models"
	"remote-config-system/internal/sse"
)

// ConfigService handles configuration business logic
type ConfigService struct {
	repos      *db.Repositories
	cache      *cache.RedisClient
	sseService *sse.SSEService
}

// NewConfigService creates a new configuration service
func NewConfigService(repos *db.Repositories, cacheClient *cache.RedisClient, sseService *sse.SSEService) *ConfigService {
	return &ConfigService{
		repos:      repos,
		cache:      cacheClient,
		sseService: sseService,
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

	// Cache the response with appropriate TTL
	if s.cache != nil {
		cacheKey := cache.GenerateConfigKey(orgSlug, appSlug, envSlug)
		// Use default TTL for configuration data
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
	if err := s.InvalidateEnvironmentCache(env.Application.Organization.Slug, env.Application.Slug, env.Slug); err != nil {
		log.Printf("Failed to invalidate environment cache: %v", err)
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

	// Broadcast SSE event for configuration update
	if s.sseService != nil {
		updateEvent := models.ConfigUpdateEvent{
			Organization: response.Organization,
			Application:  response.Application,
			Environment:  response.Environment,
			Version:      response.Version,
			Config:       response.Config,
			Action:       "update",
			UpdatedAt:    response.UpdatedAt,
		}
		s.sseService.BroadcastConfigUpdate(updateEvent)
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
	if err := s.InvalidateEnvironmentCache(env.Application.Organization.Slug, env.Application.Slug, env.Slug); err != nil {
		log.Printf("Failed to invalidate environment cache: %v", err)
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

	// Broadcast SSE event for configuration rollback
	if s.sseService != nil {
		rollbackEvent := models.ConfigUpdateEvent{
			Organization: response.Organization,
			Application:  response.Application,
			Environment:  response.Environment,
			Version:      response.Version,
			Config:       response.Config,
			Action:       "rollback",
			UpdatedAt:    response.UpdatedAt,
		}
		s.sseService.BroadcastConfigUpdate(rollbackEvent)
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

// GetConfigurationVersion retrieves a specific version of configuration for an environment
func (s *ConfigService) GetConfigurationVersion(orgSlug, appSlug, envSlug string, version int) (*models.ConfigResponse, error) {
	// Get the environment
	env, err := s.repos.Environments.GetBySlug(orgSlug, appSlug, envSlug)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Get the specific configuration version
	configVersion, err := s.repos.ConfigVersions.GetByVersion(env.ID, version)
	if err != nil {
		return nil, fmt.Errorf("configuration version not found: %w", err)
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

	return response, nil
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

// Organization Management Methods

// ListOrganizations retrieves all organizations with pagination
func (s *ConfigService) ListOrganizations(params models.PaginationParams) (*models.PaginatedResponse, error) {
	orgs, totalCount, err := s.repos.Organizations.List(params)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	response := models.NewPaginatedResponse(orgs, params.Page, params.PageSize, totalCount)
	return &response, nil
}

// GetOrganization retrieves an organization by slug
func (s *ConfigService) GetOrganization(slug string) (*models.Organization, error) {
	org, err := s.repos.Organizations.GetBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}
	return org, nil
}

// CreateOrganization creates a new organization
func (s *ConfigService) CreateOrganization(req *models.CreateOrganizationRequest) (*models.Organization, error) {
	// Check if organization with this slug already exists
	if _, err := s.repos.Organizations.GetBySlug(req.Slug); err == nil {
		return nil, fmt.Errorf("organization with slug '%s' already exists", req.Slug)
	}

	org := &models.Organization{
		Name: req.Name,
		Slug: req.Slug,
	}

	if err := s.repos.Organizations.Create(org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return org, nil
}

// UpdateOrganization updates an existing organization
func (s *ConfigService) UpdateOrganization(slug string, req *models.UpdateOrganizationRequest) (*models.Organization, error) {
	org, err := s.repos.Organizations.GetBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	org.Name = req.Name

	if err := s.repos.Organizations.Update(org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return org, nil
}

// DeleteOrganization deletes an organization
func (s *ConfigService) DeleteOrganization(slug string) error {
	org, err := s.repos.Organizations.GetBySlug(slug)
	if err != nil {
		return fmt.Errorf("organization not found: %w", err)
	}

	if err := s.repos.Organizations.Delete(org.ID); err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return nil
}

// Application Management Methods

// ListApplications retrieves all applications for an organization with pagination
func (s *ConfigService) ListApplications(orgSlug string, params models.PaginationParams) (*models.PaginatedResponse, error) {
	org, err := s.repos.Organizations.GetBySlug(orgSlug)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	apps, totalCount, err := s.repos.Applications.ListByOrganization(org.ID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	response := models.NewPaginatedResponse(apps, params.Page, params.PageSize, totalCount)
	return &response, nil
}

// GetApplication retrieves an application by organization and application slug
func (s *ConfigService) GetApplication(orgSlug, appSlug string) (*models.Application, error) {
	app, err := s.repos.Applications.GetBySlug(orgSlug, appSlug)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}
	return app, nil
}

// CreateApplication creates a new application
func (s *ConfigService) CreateApplication(orgSlug string, req *models.CreateApplicationRequest) (*models.Application, error) {
	org, err := s.repos.Organizations.GetBySlug(orgSlug)
	if err != nil {
		return nil, fmt.Errorf("organization not found: %w", err)
	}

	// Check if application with this slug already exists in the organization
	if exists, err := s.repos.Applications.Exists(org.ID, req.Slug); err != nil {
		return nil, fmt.Errorf("failed to check application existence: %w", err)
	} else if exists {
		return nil, fmt.Errorf("application with slug '%s' already exists in organization '%s'", req.Slug, orgSlug)
	}

	// Generate API key if not provided
	apiKey := req.APIKey
	if apiKey == "" {
		apiKey = generateAPIKey()
	}

	app := &models.Application{
		OrgID:  org.ID,
		Name:   req.Name,
		Slug:   req.Slug,
		APIKey: apiKey,
	}

	if err := s.repos.Applications.Create(app); err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	// Load the organization relationship
	app.Organization = org

	return app, nil
}

// UpdateApplication updates an existing application
func (s *ConfigService) UpdateApplication(orgSlug, appSlug string, req *models.UpdateApplicationRequest) (*models.Application, error) {
	app, err := s.repos.Applications.GetBySlug(orgSlug, appSlug)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	app.Name = req.Name

	if err := s.repos.Applications.Update(app); err != nil {
		return nil, fmt.Errorf("failed to update application: %w", err)
	}

	return app, nil
}

// DeleteApplication deletes an application
func (s *ConfigService) DeleteApplication(orgSlug, appSlug string) error {
	app, err := s.repos.Applications.GetBySlug(orgSlug, appSlug)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	if err := s.repos.Applications.Delete(app.ID); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	return nil
}

// Environment Management Methods

// ListEnvironments retrieves all environments for an application with pagination
func (s *ConfigService) ListEnvironments(orgSlug, appSlug string, params models.PaginationParams) (*models.PaginatedResponse, error) {
	app, err := s.repos.Applications.GetBySlug(orgSlug, appSlug)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	envs, totalCount, err := s.repos.Environments.ListByApplication(app.ID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	response := models.NewPaginatedResponse(envs, params.Page, params.PageSize, totalCount)
	return &response, nil
}

// GetEnvironment retrieves an environment by organization, application, and environment slug
func (s *ConfigService) GetEnvironment(orgSlug, appSlug, envSlug string) (*models.Environment, error) {
	env, err := s.repos.Environments.GetBySlug(orgSlug, appSlug, envSlug)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}
	return env, nil
}

// CreateEnvironment creates a new environment
func (s *ConfigService) CreateEnvironment(orgSlug, appSlug string, req *models.CreateEnvironmentRequest) (*models.Environment, error) {
	app, err := s.repos.Applications.GetBySlug(orgSlug, appSlug)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	// Check if environment with this slug already exists in the application
	if exists, err := s.repos.Environments.Exists(app.ID, req.Slug); err != nil {
		return nil, fmt.Errorf("failed to check environment existence: %w", err)
	} else if exists {
		return nil, fmt.Errorf("environment with slug '%s' already exists in application '%s'", req.Slug, appSlug)
	}

	env := &models.Environment{
		AppID: app.ID,
		Name:  req.Name,
		Slug:  req.Slug,
	}

	if err := s.repos.Environments.Create(env); err != nil {
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}

	// Load the application relationship
	env.Application = app

	return env, nil
}

// UpdateEnvironment updates an existing environment
func (s *ConfigService) UpdateEnvironment(orgSlug, appSlug, envSlug string, req *models.UpdateEnvironmentRequest) (*models.Environment, error) {
	env, err := s.repos.Environments.GetBySlug(orgSlug, appSlug, envSlug)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	env.Name = req.Name

	if err := s.repos.Environments.Update(env); err != nil {
		return nil, fmt.Errorf("failed to update environment: %w", err)
	}

	return env, nil
}

// DeleteEnvironment deletes an environment
func (s *ConfigService) DeleteEnvironment(orgSlug, appSlug, envSlug string) error {
	env, err := s.repos.Environments.GetBySlug(orgSlug, appSlug, envSlug)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	if err := s.repos.Environments.Delete(env.ID); err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	return nil
}

// Cache Management Methods

// WarmCache preloads frequently accessed configurations into cache
func (s *ConfigService) WarmCache() error {
	if s.cache == nil {
		return fmt.Errorf("cache is not enabled")
	}

	log.Println("Starting cache warming...")

	// Get all environments with their active configurations
	// This is a simplified approach - in production you might want to be more selective
	params := models.PaginationParams{Page: 1, PageSize: 100}

	// Get organizations
	orgs, _, err := s.repos.Organizations.List(params)
	if err != nil {
		return fmt.Errorf("failed to get organizations for cache warming: %w", err)
	}

	configs := make(map[string]interface{})

	for _, org := range orgs {
		// Get applications for this organization
		apps, _, err := s.repos.Applications.ListByOrganization(org.ID, params)
		if err != nil {
			log.Printf("Failed to get applications for org %s: %v", org.Slug, err)
			continue
		}

		for _, app := range apps {
			// Get environments for this application
			envs, _, err := s.repos.Environments.ListByApplication(app.ID, params)
			if err != nil {
				log.Printf("Failed to get environments for app %s: %v", app.Slug, err)
				continue
			}

			for _, env := range envs {
				// Get active configuration for this environment
				configVersion, err := s.repos.ConfigVersions.GetActiveByEnvironment(env.ID)
				if err != nil {
					log.Printf("No active config for env %s/%s/%s: %v", org.Slug, app.Slug, env.Slug, err)
					continue
				}

				// Build cache entry
				response := &models.ConfigResponse{
					Organization: org.Slug,
					Application:  app.Slug,
					Environment:  env.Slug,
					Version:      configVersion.Version,
					Config:       configVersion.ConfigJSON,
					UpdatedAt:    configVersion.CreatedAt,
				}

				// Add to cache warming batch
				cacheKey := cache.GenerateConfigKey(org.Slug, app.Slug, env.Slug)
				configs[cacheKey] = response

				// Also add API key cache entry if available
				if app.APIKey != "" {
					apiCacheKey := cache.GenerateAPIKeyConfigKey(app.APIKey, env.Slug)
					configs[apiCacheKey] = response
				}
			}
		}
	}

	if len(configs) == 0 {
		log.Println("No configurations found for cache warming")
		return nil
	}

	// Warm the cache
	if err := s.cache.WarmCache(configs); err != nil {
		return fmt.Errorf("failed to warm cache: %w", err)
	}

	log.Printf("Cache warming completed: %d configurations loaded", len(configs))
	return nil
}

// GetCacheStats returns cache statistics
func (s *ConfigService) GetCacheStats() (map[string]interface{}, error) {
	if s.cache == nil {
		return map[string]interface{}{
			"enabled": false,
			"message": "Cache is not enabled",
		}, nil
	}

	info, err := s.cache.GetCacheInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}

	info["enabled"] = true
	return info, nil
}

// ClearCache clears all cached configurations
func (s *ConfigService) ClearCache() error {
	if s.cache == nil {
		return fmt.Errorf("cache is not enabled")
	}

	if err := s.cache.InvalidatePattern("config:*"); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// Reset statistics
	s.cache.ResetStats()
	log.Println("Cache cleared successfully")
	return nil
}

// InvalidateEnvironmentCache invalidates cache for a specific environment
func (s *ConfigService) InvalidateEnvironmentCache(orgSlug, appSlug, envSlug string) error {
	if s.cache == nil {
		return nil // No cache to invalidate
	}

	// Invalidate regular config cache
	configKey := cache.GenerateConfigKey(orgSlug, appSlug, envSlug)
	if err := s.cache.DeleteConfig(configKey); err != nil {
		log.Printf("Failed to invalidate config cache: %v", err)
	}

	// Invalidate API key cache pattern for this environment
	pattern := fmt.Sprintf("config:api:*:%s", envSlug)
	if err := s.cache.InvalidatePattern(pattern); err != nil {
		log.Printf("Failed to invalidate API key cache pattern: %v", err)
	}

	log.Printf("Invalidated cache for environment: %s/%s/%s", orgSlug, appSlug, envSlug)
	return nil
}

// generateAPIKey generates a random API key
func generateAPIKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based key if random generation fails
		return fmt.Sprintf("api_%d", time.Now().UnixNano())
	}
	return "api_" + hex.EncodeToString(bytes)
}
