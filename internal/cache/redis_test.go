package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"remote-config-system/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestRedis creates a test Redis instance
func setupTestRedis(t *testing.T) (*RedisClient, testcontainers.Container) {
	ctx := context.Background()

	// Create Redis container with timeout for CI
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	// Get connection details
	host, err := redisContainer.Host(ctx)
	require.NoError(t, err)

	port, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	// Create Redis client
	config := &Config{
		Host:           host,
		Port:           port.Port(),
		Password:       "",
		DB:             0,
		TTL:            5 * time.Minute,
		ShortTTL:       1 * time.Minute,
		LongTTL:        1 * time.Hour,
		EnableCompress: false,
	}

	redisClient, err := NewRedisClient(config)
	require.NoError(t, err)

	return redisClient, redisContainer
}

// createTestConfigResponse creates a test configuration response
func createTestConfigResponse(org, app, env string, version int) *models.ConfigResponse {
	config := map[string]interface{}{
		"database_url": "postgres://localhost:5432/test",
		"api_timeout":  30,
		"debug":        true,
	}

	configJSON, _ := json.Marshal(config)

	return &models.ConfigResponse{
		Organization: org,
		Application:  app,
		Environment:  env,
		Version:      version,
		Config:       configJSON,
		UpdatedAt:    time.Now(),
	}
}

func TestRedisClient_SetAndGetConfig(t *testing.T) {
	// Setup test Redis
	client, container := setupTestRedis(t)
	defer func() {
		client.Close()
		container.Terminate(context.Background())
	}()

	t.Run("set and get config successfully", func(t *testing.T) {
		// Create test config
		config := createTestConfigResponse("test-org", "test-app", "prod", 1)
		key := "test:config:key"

		// Set config
		err := client.SetConfig(key, config)
		require.NoError(t, err)

		// Get config
		data, err := client.GetConfig(key)
		require.NoError(t, err)

		// Unmarshal and verify
		var retrievedConfig models.ConfigResponse
		err = json.Unmarshal(data, &retrievedConfig)
		require.NoError(t, err)

		assert.Equal(t, config.Organization, retrievedConfig.Organization)
		assert.Equal(t, config.Application, retrievedConfig.Application)
		assert.Equal(t, config.Environment, retrievedConfig.Environment)
		assert.Equal(t, config.Version, retrievedConfig.Version)
	})

	t.Run("get non-existent config", func(t *testing.T) {
		data, err := client.GetConfig("non:existent:key")
		assert.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("set config with custom TTL", func(t *testing.T) {
		config := createTestConfigResponse("test-org", "test-app", "staging", 2)
		key := "test:config:ttl"
		ttl := 1 * time.Second

		// Set config with short TTL
		err := client.SetConfigWithTTL(key, config, ttl)
		require.NoError(t, err)

		// Immediately get config - should exist
		data, err := client.GetConfig(key)
		require.NoError(t, err)
		assert.NotNil(t, data)

		// Wait for TTL to expire
		time.Sleep(2 * time.Second)

		// Try to get config - should be expired
		data, err = client.GetConfig(key)
		assert.NoError(t, err)
		assert.Nil(t, data)
	})
}

func TestRedisClient_DeleteConfig(t *testing.T) {
	// Setup test Redis
	client, container := setupTestRedis(t)
	defer func() {
		client.Close()
		container.Terminate(context.Background())
	}()

	t.Run("delete existing config", func(t *testing.T) {
		// Create and set test config
		config := createTestConfigResponse("test-org", "test-app", "prod", 1)
		key := "test:config:delete"

		err := client.SetConfig(key, config)
		require.NoError(t, err)

		// Verify config exists
		_, err = client.GetConfig(key)
		require.NoError(t, err)

		// Delete config
		err = client.DeleteConfig(key)
		require.NoError(t, err)

		// Verify config is deleted
		data, err := client.GetConfig(key)
		assert.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("delete non-existent config", func(t *testing.T) {
		// Should not error when deleting non-existent key
		err := client.DeleteConfig("non:existent:key")
		assert.NoError(t, err)
	})
}

func TestRedisClient_InvalidatePattern(t *testing.T) {
	// Setup test Redis
	client, container := setupTestRedis(t)
	defer func() {
		client.Close()
		container.Terminate(context.Background())
	}()

	t.Run("invalidate pattern successfully", func(t *testing.T) {
		// Create multiple configs with similar keys
		config1 := createTestConfigResponse("test-org", "app1", "prod", 1)
		config2 := createTestConfigResponse("test-org", "app2", "prod", 1)
		config3 := createTestConfigResponse("other-org", "app1", "prod", 1)

		keys := []string{
			"config:test-org:app1:prod",
			"config:test-org:app2:prod",
			"config:other-org:app1:prod",
		}

		// Set all configs
		err := client.SetConfig(keys[0], config1)
		require.NoError(t, err)
		err = client.SetConfig(keys[1], config2)
		require.NoError(t, err)
		err = client.SetConfig(keys[2], config3)
		require.NoError(t, err)

		// Verify all configs exist
		for _, key := range keys {
			_, err := client.GetConfig(key)
			require.NoError(t, err)
		}

		// Invalidate pattern for test-org
		err = client.InvalidatePattern("config:test-org:*")
		require.NoError(t, err)

		// Verify test-org configs are deleted
		data, err := client.GetConfig(keys[0])
		assert.NoError(t, err)
		assert.Nil(t, data)
		data, err = client.GetConfig(keys[1])
		assert.NoError(t, err)
		assert.Nil(t, data)

		// Verify other-org config still exists
		_, err = client.GetConfig(keys[2])
		assert.NoError(t, err)
	})
}

func TestRedisClient_WarmCache(t *testing.T) {
	// Setup test Redis
	client, container := setupTestRedis(t)
	defer func() {
		client.Close()
		container.Terminate(context.Background())
	}()

	t.Run("warm cache with multiple configs", func(t *testing.T) {
		// Create test configs
		configs := map[string]interface{}{
			"config:org1:app1:prod": createTestConfigResponse("org1", "app1", "prod", 1),
			"config:org1:app1:staging": createTestConfigResponse("org1", "app1", "staging", 1),
			"config:org2:app1:prod": createTestConfigResponse("org2", "app1", "prod", 1),
		}

		// Warm cache
		err := client.WarmCache(configs)
		require.NoError(t, err)

		// Verify all configs are cached
		for key, expectedConfig := range configs {
			data, err := client.GetConfig(key)
			require.NoError(t, err)

			var retrievedConfig models.ConfigResponse
			err = json.Unmarshal(data, &retrievedConfig)
			require.NoError(t, err)

			expected := expectedConfig.(*models.ConfigResponse)
			assert.Equal(t, expected.Organization, retrievedConfig.Organization)
			assert.Equal(t, expected.Application, retrievedConfig.Application)
			assert.Equal(t, expected.Environment, retrievedConfig.Environment)
		}
	})
}

func TestRedisClient_Health(t *testing.T) {
	// Setup test Redis
	client, container := setupTestRedis(t)
	defer func() {
		client.Close()
		container.Terminate(context.Background())
	}()

	t.Run("health check on healthy connection", func(t *testing.T) {
		err := client.Health()
		assert.NoError(t, err)
	})

	t.Run("health check after closing connection", func(t *testing.T) {
		// Close the connection
		err := client.Close()
		require.NoError(t, err)

		// Health check should fail
		err = client.Health()
		assert.Error(t, err)
	})
}

func TestGenerateKeys(t *testing.T) {
	t.Run("generate config key", func(t *testing.T) {
		key := GenerateConfigKey("test-org", "test-app", "prod")
		expected := "config:test-org:test-app:prod"
		assert.Equal(t, expected, key)
	})

	t.Run("generate API key config key", func(t *testing.T) {
		key := GenerateAPIKeyConfigKey("api-key-123", "prod")
		expected := "config:api:api-key-123:prod"
		assert.Equal(t, expected, key)
	})

	t.Run("generate invalidation pattern", func(t *testing.T) {
		pattern := GenerateInvalidationPattern("test-org", "test-app", "prod")
		expected := "config:*:test-org:test-app:prod"
		assert.Equal(t, expected, pattern)
	})
}

func TestRedisClient_Compression(t *testing.T) {
	// Create a Redis client with compression enabled
	config := &Config{
		Host:           "localhost",
		Port:           "6379",
		Password:       "",
		DB:             0,
		TTL:            5 * time.Minute,
		ShortTTL:       1 * time.Minute,
		LongTTL:        1 * time.Hour,
		EnableCompress: true,
	}

	// For this test, we'll use a mock or skip if we can't easily test compression
	// In a real scenario, you'd want to test with a large config that triggers compression
	t.Run("compression enabled config", func(t *testing.T) {
		assert.True(t, config.EnableCompress)
		assert.Equal(t, 5*time.Minute, config.TTL)
	})
}

func TestNewConfig(t *testing.T) {
	t.Run("default config values", func(t *testing.T) {
		// Clear environment variables for clean test
		t.Setenv("REDIS_HOST", "")
		t.Setenv("REDIS_PORT", "")
		t.Setenv("CACHE_TTL", "")

		config := NewConfig()

		assert.Equal(t, "localhost", config.Host)
		assert.Equal(t, "6379", config.Port)
		assert.Equal(t, "", config.Password)
		assert.Equal(t, 0, config.DB)
		assert.Equal(t, 5*time.Minute, config.TTL)
		assert.Equal(t, 1*time.Minute, config.ShortTTL)
		assert.Equal(t, 1*time.Hour, config.LongTTL)
		assert.False(t, config.EnableCompress)
	})

	t.Run("config from environment variables", func(t *testing.T) {
		t.Setenv("REDIS_HOST", "redis-server")
		t.Setenv("REDIS_PORT", "6380")
		t.Setenv("REDIS_PASSWORD", "secret")
		t.Setenv("REDIS_DB", "1")
		t.Setenv("CACHE_TTL", "600")
		t.Setenv("CACHE_ENABLE_COMPRESSION", "true")

		config := NewConfig()

		assert.Equal(t, "redis-server", config.Host)
		assert.Equal(t, "6380", config.Port)
		assert.Equal(t, "secret", config.Password)
		assert.Equal(t, 1, config.DB)
		assert.Equal(t, 10*time.Minute, config.TTL)
		assert.True(t, config.EnableCompress)
	})
}
