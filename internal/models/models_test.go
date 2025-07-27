package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganization_Validation(t *testing.T) {
	t.Run("valid organization", func(t *testing.T) {
		org := Organization{
			ID:        uuid.New(),
			Name:      "Test Organization",
			Slug:      "test-org",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.NotEmpty(t, org.ID)
		assert.Equal(t, "Test Organization", org.Name)
		assert.Equal(t, "test-org", org.Slug)
		assert.False(t, org.CreatedAt.IsZero())
		assert.False(t, org.UpdatedAt.IsZero())
	})

	t.Run("organization JSON marshaling", func(t *testing.T) {
		org := Organization{
			ID:        uuid.New(),
			Name:      "Test Organization",
			Slug:      "test-org",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		data, err := json.Marshal(org)
		require.NoError(t, err)
		assert.Contains(t, string(data), "Test Organization")
		assert.Contains(t, string(data), "test-org")

		var unmarshaled Organization
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, org.Name, unmarshaled.Name)
		assert.Equal(t, org.Slug, unmarshaled.Slug)
	})
}

func TestApplication_Validation(t *testing.T) {
	t.Run("valid application", func(t *testing.T) {
		app := Application{
			ID:        uuid.New(),
			Name:      "Test App",
			Slug:      "test-app",
			OrgID:     uuid.New(),
			APIKey:    "test-api-key-123",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.NotEmpty(t, app.ID)
		assert.Equal(t, "Test App", app.Name)
		assert.Equal(t, "test-app", app.Slug)
		assert.NotEmpty(t, app.OrgID)
		assert.Equal(t, "test-api-key-123", app.APIKey)
	})

	t.Run("application with organization", func(t *testing.T) {
		app := Application{
			ID:   uuid.New(),
			Name: "Test App",
			Slug: "test-app",
			Organization: &Organization{
				ID:   uuid.New(),
				Name: "Test Org",
				Slug: "test-org",
			},
		}

		assert.NotNil(t, app.Organization)
		assert.Equal(t, "Test Org", app.Organization.Name)
		assert.Equal(t, "test-org", app.Organization.Slug)
	})
}

func TestEnvironment_Validation(t *testing.T) {
	t.Run("valid environment", func(t *testing.T) {
		env := Environment{
			ID:        uuid.New(),
			Name:      "Production",
			Slug:      "prod",
			AppID:     uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.NotEmpty(t, env.ID)
		assert.Equal(t, "Production", env.Name)
		assert.Equal(t, "prod", env.Slug)
		assert.NotEmpty(t, env.AppID)
	})
}

func TestConfigVersion_JSONHandling(t *testing.T) {
	t.Run("valid config version", func(t *testing.T) {
		configData := map[string]interface{}{
			"database_url": "postgres://localhost:5432/test",
			"timeout":      30,
			"debug":        true,
		}

		jsonData, err := json.Marshal(configData)
		require.NoError(t, err)

		version := ConfigVersion{
			ID:         uuid.New(),
			EnvID:      uuid.New(),
			Version:    1,
			ConfigJSON: json.RawMessage(jsonData),
			IsActive:   true,
			CreatedAt:  time.Now(),
		}

		assert.NotEmpty(t, version.ID)
		assert.NotEmpty(t, version.EnvID)
		assert.Equal(t, 1, version.Version)
		assert.True(t, version.IsActive)
		assert.NotEmpty(t, version.ConfigJSON)

		// Test unmarshaling configuration
		var config map[string]interface{}
		err = json.Unmarshal(version.ConfigJSON, &config)
		require.NoError(t, err)
		assert.Equal(t, "postgres://localhost:5432/test", config["database_url"])
		assert.Equal(t, float64(30), config["timeout"]) // JSON numbers become float64
		assert.Equal(t, true, config["debug"])
	})
}

func TestConfigResponse_Structure(t *testing.T) {
	t.Run("valid config response", func(t *testing.T) {
		configData := map[string]interface{}{
			"api_url": "https://api.example.com",
			"timeout": 30,
		}
		jsonData, _ := json.Marshal(configData)

		response := ConfigResponse{
			Organization: "test-org",
			Application:  "test-app",
			Environment:  "prod",
			Version:      1,
			Config:       json.RawMessage(jsonData),
			UpdatedAt:    time.Now(),
		}

		assert.Equal(t, "test-org", response.Organization)
		assert.Equal(t, "test-app", response.Application)
		assert.Equal(t, "prod", response.Environment)
		assert.Equal(t, 1, response.Version)
		assert.NotEmpty(t, response.Config)
	})

	t.Run("config response JSON serialization", func(t *testing.T) {
		configData := map[string]interface{}{
			"feature_flags": map[string]bool{
				"new_ui":       true,
				"beta_feature": false,
			},
		}
		jsonData, _ := json.Marshal(configData)

		response := ConfigResponse{
			Organization: "test-org",
			Application:  "test-app",
			Environment:  "prod",
			Version:      1,
			Config:       json.RawMessage(jsonData),
			UpdatedAt:    time.Now(),
		}

		data, err := json.Marshal(response)
		require.NoError(t, err)

		var unmarshaled ConfigResponse
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, response.Organization, unmarshaled.Organization)
		assert.Equal(t, response.Application, unmarshaled.Application)
		assert.Equal(t, response.Environment, unmarshaled.Environment)
		assert.Equal(t, response.Version, unmarshaled.Version)
	})
}

func TestPaginationParams_Validation(t *testing.T) {
	t.Run("valid pagination params", func(t *testing.T) {
		params := PaginationParams{
			Page:     1,
			PageSize: 10,
		}

		assert.Equal(t, 1, params.Page)
		assert.Equal(t, 10, params.PageSize)
	})

	t.Run("default pagination params", func(t *testing.T) {
		params := DefaultPaginationParams()

		assert.Equal(t, 1, params.Page)
		assert.Equal(t, 20, params.PageSize)
	})

	t.Run("pagination offset calculation", func(t *testing.T) {
		params := PaginationParams{
			Page:     3,
			PageSize: 10,
		}

		offset := params.Offset()
		assert.Equal(t, 20, offset) // (3-1) * 10 = 20
	})
}

func TestErrorResponse_Structure(t *testing.T) {
	t.Run("simple error response", func(t *testing.T) {
		errResp := ErrorResponse{
			Error:     "validation_failed",
			Message:   "Invalid input provided",
			Timestamp: time.Now(),
			Path:      "/api/test",
		}

		assert.Equal(t, "validation_failed", errResp.Error)
		assert.Equal(t, "Invalid input provided", errResp.Message)
		assert.Equal(t, "/api/test", errResp.Path)
		assert.False(t, errResp.Timestamp.IsZero())
	})
}

func TestHealthResponse_Structure(t *testing.T) {
	t.Run("healthy response", func(t *testing.T) {
		health := HealthResponse{
			Status:    "ok",
			Message:   "All systems operational",
			Timestamp: time.Now(),
			Services: map[string]string{
				"database": "healthy",
				"redis":    "healthy",
			},
		}

		assert.Equal(t, "ok", health.Status)
		assert.Equal(t, "All systems operational", health.Message)
		assert.Equal(t, "healthy", health.Services["database"])
		assert.Equal(t, "healthy", health.Services["redis"])
		assert.False(t, health.Timestamp.IsZero())
	})
}

func TestSSEMessage_Structure(t *testing.T) {
	t.Run("config update message", func(t *testing.T) {
		msg := SSEMessage{
			Event: "config_update",
			Data: map[string]interface{}{
				"organization": "test-org",
				"application":  "test-app",
				"environment":  "prod",
				"version":      2,
			},
		}

		assert.Equal(t, "config_update", msg.Event)
		data := msg.Data.(map[string]interface{})
		assert.Equal(t, "test-org", data["organization"])
		assert.Equal(t, 2, data["version"])
	})

	t.Run("custom event message", func(t *testing.T) {
		msg := SSEMessage{
			Event: "maintenance",
			Data: map[string]interface{}{
				"message":    "System maintenance scheduled",
				"start_time": "2024-01-01T00:00:00Z",
				"duration":   "2 hours",
			},
		}

		assert.Equal(t, "maintenance", msg.Event)
		data := msg.Data.(map[string]interface{})
		assert.Equal(t, "System maintenance scheduled", data["message"])
		assert.Equal(t, "2 hours", data["duration"])
	})
}
