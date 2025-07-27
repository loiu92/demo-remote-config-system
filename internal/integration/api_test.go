package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"remote-config-system/internal/cache"
	"remote-config-system/internal/handlers"
	"remote-config-system/internal/middleware"
	"remote-config-system/internal/models"
	"remote-config-system/internal/services"
	"remote-config-system/internal/sse"
	"remote-config-system/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IntegrationTestSuite provides a complete integration test environment
type IntegrationTestSuite struct {
	*testutil.TestSuite
	Router         *gin.Engine
	ConfigService  *services.ConfigService
	ConfigHandler  *handlers.ConfigHandler
	ManagementHandler *handlers.ManagementHandler
	SSEHandler     *handlers.SSEHandler
}

// SetupIntegrationTest creates a complete integration test environment
func SetupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	gin.SetMode(gin.TestMode)
	
	// Setup base test suite
	testSuite := testutil.SetupTestSuite(t)
	
	// Initialize SSE service
	sseService := sse.NewSSEService()
	
	// Initialize services
	configService := services.NewConfigService(testSuite.Repos, testSuite.Redis.Client, sseService)
	
	// Initialize handlers
	configHandler := handlers.NewConfigHandler(configService)
	managementHandler := handlers.NewManagementHandler(configService)
	sseHandler := handlers.NewSSEHandler(configService, sseService)
	
	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(configService)
	
	// Setup router
	router := gin.New()
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.ErrorHandler())
	
	// Health check endpoint
	router.GET("/health", configHandler.HealthCheck)
	
	// Public configuration endpoints
	publicAPI := router.Group("/config")
	{
		publicAPI.GET("/:org/:app/:env", configHandler.GetConfig)
	}
	
	// API endpoints with authentication
	apiV1 := router.Group("/api")
	apiV1.Use(authMiddleware.APIKeyAuth())
	{
		apiV1.GET("/config/:env", configHandler.GetConfigByAPIKey)
	}
	
	// Management endpoints
	adminAPI := router.Group("/admin")
	{
		adminAPI.GET("/orgs", managementHandler.ListOrganizations)
		adminAPI.POST("/orgs", managementHandler.CreateOrganization)
		adminAPI.GET("/orgs/:org/apps", managementHandler.ListApplications)
		adminAPI.POST("/orgs/:org/apps", managementHandler.CreateApplication)
		adminAPI.GET("/orgs/:org/apps/:app/envs", managementHandler.ListEnvironments)
		adminAPI.POST("/orgs/:org/apps/:app/envs", managementHandler.CreateEnvironment)
		adminAPI.PUT("/orgs/:org/apps/:app/envs/:env/config", configHandler.UpdateConfig)
	}
	
	return &IntegrationTestSuite{
		TestSuite:         testSuite,
		Router:            router,
		ConfigService:     configService,
		ConfigHandler:     configHandler,
		ManagementHandler: managementHandler,
		SSEHandler:        sseHandler,
	}
}

func TestIntegration_FullConfigurationWorkflow(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.Cleanup(t)
	
	t.Run("complete configuration management workflow", func(t *testing.T) {
		// Step 1: Create organization
		orgReq := &models.CreateOrganizationRequest{
			Name: "Test Company",
			Slug: "testcompany",
		}
		
		orgBody, _ := json.Marshal(orgReq)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/admin/orgs", bytes.NewBuffer(orgBody))
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)
		
		require.Equal(t, http.StatusCreated, w.Code)
		
		var org models.Organization
		err := json.Unmarshal(w.Body.Bytes(), &org)
		require.NoError(t, err)
		assert.Equal(t, "Test Company", org.Name)
		assert.Equal(t, "testcompany", org.Slug)
		
		// Step 2: Create application
		appReq := &models.CreateApplicationRequest{
			Name:   "Test Application",
			Slug:   "testapp",
			APIKey: "testapikey123",
		}
		
		appBody, _ := json.Marshal(appReq)
		w = httptest.NewRecorder()
		req = httptest.NewRequest("POST", "/admin/orgs/testcompany/apps", bytes.NewBuffer(appBody))
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)
		
		require.Equal(t, http.StatusCreated, w.Code)
		
		var app models.Application
		err = json.Unmarshal(w.Body.Bytes(), &app)
		require.NoError(t, err)
		assert.Equal(t, "Test Application", app.Name)
		assert.Equal(t, "testapp", app.Slug)
		assert.Equal(t, "testapikey123", app.APIKey)
		
		// Step 3: Create environment
		envReq := &models.CreateEnvironmentRequest{
			Name: "Production",
			Slug: "prod",
		}
		
		envBody, _ := json.Marshal(envReq)
		w = httptest.NewRecorder()
		req = httptest.NewRequest("POST", "/admin/orgs/testcompany/apps/testapp/envs", bytes.NewBuffer(envBody))
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)
		
		require.Equal(t, http.StatusCreated, w.Code)
		
		var env models.Environment
		err = json.Unmarshal(w.Body.Bytes(), &env)
		require.NoError(t, err)
		assert.Equal(t, "Production", env.Name)
		assert.Equal(t, "prod", env.Slug)
		
		// Step 4: Update configuration
		configData := map[string]interface{}{
			"database_url": "postgres://localhost:5432/prod",
			"api_timeout":  30,
			"debug":        false,
		}
		configJSON, _ := json.Marshal(configData)
		
		updateReq := &models.CreateConfigRequest{
			Config:    configJSON,
			CreatedBy: stringPtr("admin"),
		}
		
		updateBody, _ := json.Marshal(updateReq)
		w = httptest.NewRecorder()
		req = httptest.NewRequest("PUT", "/admin/orgs/testcompany/apps/testapp/envs/prod/config", bytes.NewBuffer(updateBody))
		req.Header.Set("Content-Type", "application/json")
		suite.Router.ServeHTTP(w, req)
		
		require.Equal(t, http.StatusOK, w.Code)
		
		var configResponse models.ConfigResponse
		err = json.Unmarshal(w.Body.Bytes(), &configResponse)
		require.NoError(t, err)
		assert.Equal(t, "testcompany", configResponse.Organization)
		assert.Equal(t, "testapp", configResponse.Application)
		assert.Equal(t, "prod", configResponse.Environment)
		assert.Equal(t, 1, configResponse.Version)
		
		// Step 5: Retrieve configuration via public API
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/config/testcompany/testapp/prod", nil)
		suite.Router.ServeHTTP(w, req)
		
		require.Equal(t, http.StatusOK, w.Code)
		
		var publicConfig models.ConfigResponse
		err = json.Unmarshal(w.Body.Bytes(), &publicConfig)
		require.NoError(t, err)
		assert.Equal(t, configResponse.Version, publicConfig.Version)
		assert.JSONEq(t, string(configResponse.Config), string(publicConfig.Config))
		
		// Step 6: Retrieve configuration via API key
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/api/config/prod", nil)
		req.Header.Set("X-API-Key", "testapikey123")
		suite.Router.ServeHTTP(w, req)
		
		require.Equal(t, http.StatusOK, w.Code)
		
		var apiKeyConfig models.ConfigResponse
		err = json.Unmarshal(w.Body.Bytes(), &apiKeyConfig)
		require.NoError(t, err)
		assert.Equal(t, configResponse.Version, apiKeyConfig.Version)
		assert.JSONEq(t, string(configResponse.Config), string(apiKeyConfig.Config))
	})
}

func TestIntegration_ErrorHandling(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.Cleanup(t)
	
	t.Run("404 for non-existent configuration", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/config/non-existent/app/env", nil)
		suite.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code) // 404 because environment not found

		var errorResponse models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "not_found", errorResponse.Error)
	})
	
	t.Run("401 for invalid API key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/config/prod", nil)
		req.Header.Set("X-API-Key", "invalid-key")
		suite.Router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var errorResponse models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", errorResponse.Error)
	})
	
	t.Run("401 for missing API key", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/config/prod", nil)
		suite.Router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var errorResponse models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", errorResponse.Error)
		assert.Equal(t, "API key is required", errorResponse.Message)
	})
}

func TestIntegration_HealthCheck(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.Cleanup(t)
	
	t.Run("health check returns service status", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		suite.Router.ServeHTTP(w, req)
		
		require.Equal(t, http.StatusOK, w.Code)
		
		var healthResponse models.HealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &healthResponse)
		require.NoError(t, err)
		
		assert.Equal(t, "ok", healthResponse.Status)
		assert.Equal(t, "Remote Config System is running", healthResponse.Message)
		assert.Equal(t, "connected", healthResponse.Services["database"])
		assert.Equal(t, "connected", healthResponse.Services["cache"])
	})
}

func TestIntegration_CacheIntegration(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.Cleanup(t)
	
	// Create test data
	org := suite.CreateTestOrganization(t, "Cache Test Org", "cache-test-org")
	app := suite.CreateTestApplication(t, org.ID, "Cache Test App", "cache-test-app", "cache-api-key")
	env := suite.CreateTestEnvironment(t, app.ID, "Production", "prod")
	
	// Create configuration
	configData := map[string]interface{}{
		"cache_test": true,
		"value":      42,
	}
	configJSON, _ := json.Marshal(configData)
	
	configVersion := &models.ConfigVersion{
		ID:         uuid.New(),
		EnvID:      env.ID,
		Version:    1,
		ConfigJSON: configJSON,
		IsActive:   true,
		CreatedBy:  stringPtr("admin"),
	}
	err := suite.Repos.ConfigVersions.Create(configVersion)
	require.NoError(t, err)
	
	t.Run("configuration is cached after first request", func(t *testing.T) {
		// First request - should populate cache
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/config/cache-test-org/cache-test-app/prod", nil)
		suite.Router.ServeHTTP(w1, req1)
		
		require.Equal(t, http.StatusOK, w1.Code)
		
		// Second request - should hit cache
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/config/cache-test-org/cache-test-app/prod", nil)
		suite.Router.ServeHTTP(w2, req2)
		
		require.Equal(t, http.StatusOK, w2.Code)
		
		// Both responses should be identical
		assert.Equal(t, w1.Body.String(), w2.Body.String())
		
		// Verify cache key exists
		cacheKey := cache.GenerateConfigKey("cache-test-org", "cache-test-app", "prod")
		cachedData, err := suite.Redis.Client.GetConfig(cacheKey)
		require.NoError(t, err)
		assert.NotNil(t, cachedData)
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
