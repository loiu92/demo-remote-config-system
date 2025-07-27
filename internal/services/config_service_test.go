package services

import (
	"encoding/json"
	"testing"

	"remote-config-system/internal/models"
	"remote-config-system/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfigService_GetConfiguration(t *testing.T) {
	suite := testutil.SetupTestSuite(t)
	defer suite.Cleanup(t)

	// Create test data
	org := suite.CreateTestOrganization(t, "Test Org", "test-org")
	app := suite.CreateTestApplication(t, org.ID, "Test App", "test-app", "test-api-key")
	env := suite.CreateTestEnvironment(t, app.ID, "Production", "prod")

	// Create a configuration version
	configData := map[string]interface{}{
		"database_url": "postgres://localhost:5432/prod",
		"api_timeout":  30,
		"debug":        false,
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

	// Create service
	mockSSE := testutil.NewMockSSEService()
	service := NewConfigService(suite.Repos, suite.Redis.Client, mockSSE)

	t.Run("successful retrieval", func(t *testing.T) {
		config, err := service.GetConfiguration("test-org", "test-app", "prod")
		
		require.NoError(t, err)
		assert.Equal(t, "test-org", config.Organization)
		assert.Equal(t, "test-app", config.Application)
		assert.Equal(t, "prod", config.Environment)
		assert.Equal(t, 1, config.Version)
		assert.JSONEq(t, string(configJSON), string(config.Config))
	})

	t.Run("environment not found", func(t *testing.T) {
		_, err := service.GetConfiguration("test-org", "test-app", "nonexistent")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment not found")
	})

	t.Run("cache hit", func(t *testing.T) {
		// First call to populate cache
		config1, err := service.GetConfiguration("test-org", "test-app", "prod")
		require.NoError(t, err)

		// Second call should hit cache
		config2, err := service.GetConfiguration("test-org", "test-app", "prod")
		require.NoError(t, err)

		assert.Equal(t, config1.Version, config2.Version)
		assert.Equal(t, config1.Organization, config2.Organization)
	})
}

func TestConfigService_GetConfigurationByAPIKey(t *testing.T) {
	suite := testutil.SetupTestSuite(t)
	defer suite.Cleanup(t)

	// Create test data
	org := suite.CreateTestOrganization(t, "Test Org", "test-org")
	app := suite.CreateTestApplication(t, org.ID, "Test App", "test-app", "test-api-key-123")
	env := suite.CreateTestEnvironment(t, app.ID, "Production", "prod")

	// Create a configuration version
	configData := map[string]interface{}{
		"database_url": "postgres://localhost:5432/prod",
		"api_timeout":  30,
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

	// Create service
	mockSSE := testutil.NewMockSSEService()
	service := NewConfigService(suite.Repos, suite.Redis.Client, mockSSE)

	t.Run("successful retrieval with API key", func(t *testing.T) {
		config, err := service.GetConfigurationByAPIKey("test-api-key-123", "prod")
		
		require.NoError(t, err)
		assert.Equal(t, "test-org", config.Organization)
		assert.Equal(t, "test-app", config.Application)
		assert.Equal(t, "prod", config.Environment)
		assert.Equal(t, 1, config.Version)
	})

	t.Run("invalid API key", func(t *testing.T) {
		_, err := service.GetConfigurationByAPIKey("invalid-key", "prod")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
	})

	t.Run("environment not found", func(t *testing.T) {
		_, err := service.GetConfigurationByAPIKey("test-api-key-123", "nonexistent")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment not found")
	})
}

func TestConfigService_UpdateConfiguration(t *testing.T) {
	suite := testutil.SetupTestSuite(t)
	defer suite.Cleanup(t)

	// Create test data
	org := suite.CreateTestOrganization(t, "Test Org", "test-org")
	app := suite.CreateTestApplication(t, org.ID, "Test App", "test-app", "test-api-key")
	env := suite.CreateTestEnvironment(t, app.ID, "Production", "prod")

	// Create initial configuration version
	initialConfig := map[string]interface{}{
		"database_url": "postgres://localhost:5432/prod",
		"api_timeout":  30,
	}
	initialConfigJSON, _ := json.Marshal(initialConfig)

	configVersion := &models.ConfigVersion{
		ID:         uuid.New(),
		EnvID:      env.ID,
		Version:    1,
		ConfigJSON: initialConfigJSON,
		IsActive:   true,
		CreatedBy:  stringPtr("admin"),
	}
	err := suite.Repos.ConfigVersions.Create(configVersion)
	require.NoError(t, err)

	// Create service with mock SSE
	mockSSE := testutil.NewMockSSEService()
	mockSSE.On("BroadcastConfigUpdate", mock.AnythingOfType("models.ConfigUpdateEvent")).Return()
	
	service := NewConfigService(suite.Repos, suite.Redis.Client, mockSSE)

	t.Run("successful update", func(t *testing.T) {
		updateConfig := map[string]interface{}{
			"database_url": "postgres://localhost:5432/prod_updated",
			"api_timeout":  60,
			"debug":        true,
		}
		updateConfigJSON, _ := json.Marshal(updateConfig)

		req := &models.CreateConfigRequest{
			Config:    updateConfigJSON,
			CreatedBy: stringPtr("user1"),
		}

		config, err := service.UpdateConfiguration("test-org", "test-app", "prod", req)
		
		require.NoError(t, err)
		assert.Equal(t, "test-org", config.Organization)
		assert.Equal(t, "test-app", config.Application)
		assert.Equal(t, "prod", config.Environment)
		assert.Equal(t, 2, config.Version) // Should increment version
		assert.JSONEq(t, string(updateConfigJSON), string(config.Config))

		// Verify SSE broadcast was called
		mockSSE.AssertExpectations(t)
	})

	t.Run("environment not found", func(t *testing.T) {
		req := &models.CreateConfigRequest{
			Config:    json.RawMessage(`{"test": "value"}`),
			CreatedBy: stringPtr("user1"),
		}

		_, err := service.UpdateConfiguration("test-org", "test-app", "nonexistent", req)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment not found")
	})
}

func TestConfigService_ValidateAPIKey(t *testing.T) {
	suite := testutil.SetupTestSuite(t)
	defer suite.Cleanup(t)

	// Create test data
	org := suite.CreateTestOrganization(t, "Test Org", "test-org")
	app := suite.CreateTestApplication(t, org.ID, "Test App", "test-app", "valid-api-key-123")

	// Create service
	mockSSE := testutil.NewMockSSEService()
	service := NewConfigService(suite.Repos, suite.Redis.Client, mockSSE)

	t.Run("valid API key", func(t *testing.T) {
		application, err := service.ValidateAPIKey("valid-api-key-123")
		
		require.NoError(t, err)
		assert.Equal(t, app.ID, application.ID)
		assert.Equal(t, "test-app", application.Slug)
		assert.Equal(t, "test-org", application.Organization.Slug)
	})

	t.Run("invalid API key", func(t *testing.T) {
		_, err := service.ValidateAPIKey("invalid-key")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
	})
}

func TestConfigService_HealthCheck(t *testing.T) {
	suite := testutil.SetupTestSuite(t)
	defer suite.Cleanup(t)

	// Create service
	mockSSE := testutil.NewMockSSEService()
	service := NewConfigService(suite.Repos, suite.Redis.Client, mockSSE)

	t.Run("health check with cache", func(t *testing.T) {
		health := service.HealthCheck()
		
		assert.Equal(t, "connected", health["database"])
		assert.Equal(t, "connected", health["cache"])
	})

	t.Run("health check without cache", func(t *testing.T) {
		serviceNoCache := NewConfigService(suite.Repos, nil, mockSSE)
		health := serviceNoCache.HealthCheck()
		
		assert.Equal(t, "connected", health["database"])
		assert.Equal(t, "disabled", health["cache"])
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}


