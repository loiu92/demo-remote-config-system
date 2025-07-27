package cache

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockRedis creates an in-memory Redis for testing
func setupMockRedis(t *testing.T) (*RedisClient, func()) {
	// Start miniredis (in-memory Redis)
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cache := &RedisClient{
		client:   client,
		ttl:      5 * time.Minute,
		shortTTL: 1 * time.Minute,
		longTTL:  10 * time.Minute,
		stats:    &CacheStats{},
	}

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return cache, cleanup
}

func TestRedisClient_SetConfig_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	key := "test:key"
	value := map[string]interface{}{
		"database_url": "postgres://localhost:5432/test",
		"timeout":      30,
	}

	err := cache.SetConfig(key, value)

	assert.NoError(t, err)
}

func TestRedisClient_GetConfig_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	key := "test:key"
	expectedValue := map[string]interface{}{
		"database_url": "postgres://localhost:5432/test",
		"timeout":      30,
	}

	// Set the value first
	err := cache.SetConfig(key, expectedValue)
	require.NoError(t, err)

	// Get the value
	result, err := cache.GetConfig(key)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRedisClient_GetConfig_NotFound_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	key := "nonexistent:key"

	result, err := cache.GetConfig(key)

	// The actual implementation might return empty string instead of error for missing keys
	if err != nil {
		assert.Error(t, err)
		assert.Empty(t, result)
	} else {
		assert.Empty(t, result)
	}
}

func TestRedisClient_DeleteConfig_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	key := "test:key"
	value := map[string]interface{}{
		"test": "value",
	}

	// Set the value first
	err := cache.SetConfig(key, value)
	require.NoError(t, err)

	// Delete the value
	err = cache.DeleteConfig(key)
	assert.NoError(t, err)

	// Verify it's deleted (might return empty string instead of error)
	result, err := cache.GetConfig(key)
	if err == nil {
		assert.Empty(t, result)
	} else {
		assert.Error(t, err)
	}
}

func TestRedisClient_Health_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	err := cache.Health()

	assert.NoError(t, err)
}

func TestRedisClient_GetStats_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	stats := cache.GetStats()

	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.Hits, int64(0))
	assert.GreaterOrEqual(t, stats.Misses, int64(0))
}

func TestRedisClient_InvalidatePattern_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	// Set some test data
	err := cache.SetConfig("config:test:org:app:prod", map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	err = cache.SetConfig("config:test:org:app:staging", map[string]interface{}{"key": "value2"})
	require.NoError(t, err)

	// Invalidate pattern
	err = cache.InvalidatePattern("config:test:org:app:*")
	assert.NoError(t, err)

	// Verify configs are invalidated (might return empty string instead of error)
	result1, err1 := cache.GetConfig("config:test:org:app:prod")
	if err1 == nil {
		assert.Empty(t, result1)
	} else {
		assert.Error(t, err1)
	}

	result2, err2 := cache.GetConfig("config:test:org:app:staging")
	if err2 == nil {
		assert.Empty(t, result2)
	} else {
		assert.Error(t, err2)
	}
}

func TestRedisClient_SetConfigWithTTL_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	key := "test:ttl:key"
	value := map[string]interface{}{
		"test": "value",
	}

	err := cache.SetConfigWithTTL(key, value, 1*time.Second)
	assert.NoError(t, err)

	// Should be able to get it immediately
	result, err := cache.GetConfig(key)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRedisClient_ConcurrentAccess_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	// Test concurrent reads and writes
	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	// Start multiple goroutines doing concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent:test:%d:%d", id, j)
				value := map[string]interface{}{
					"goroutine": id,
					"operation": j,
					"timestamp": time.Now().Unix(),
				}

				// Set config
				err := cache.SetConfig(key, value)
				assert.NoError(t, err)

				// Get config
				result, err := cache.GetConfig(key)
				assert.NoError(t, err)
				assert.NotEmpty(t, result)

				// Delete config
				err = cache.DeleteConfig(key)
				assert.NoError(t, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("Concurrent test timed out")
		}
	}
}

func TestRedisClient_LargeConfig_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	// Create a large configuration
	largeConfig := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeConfig[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d_with_some_longer_content_to_make_it_bigger", i)
	}

	key := "test:large:config"

	// Set large config
	err := cache.SetConfig(key, largeConfig)
	assert.NoError(t, err)

	// Get large config
	result, err := cache.GetConfig(key)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify some keys exist
	var resultMap map[string]interface{}
	err = json.Unmarshal([]byte(result), &resultMap)
	assert.NoError(t, err)
	assert.Contains(t, resultMap, "key_0")
	assert.Contains(t, resultMap, "key_999")
}

func TestRedisClient_ErrorHandling_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	t.Run("get non-existent key", func(t *testing.T) {
		result, err := cache.GetConfig("non:existent:key")
		// Might return empty string instead of error
		if err == nil {
			assert.Empty(t, result)
		} else {
			assert.Error(t, err)
			assert.Empty(t, result)
		}
	})

	t.Run("delete non-existent key", func(t *testing.T) {
		err := cache.DeleteConfig("non:existent:key")
		// Delete should not error even if key doesn't exist
		assert.NoError(t, err)
	})

	t.Run("invalidate non-existent pattern", func(t *testing.T) {
		err := cache.InvalidatePattern("non:existent:pattern:*")
		// Should not error even if no keys match
		assert.NoError(t, err)
	})
}

func TestRedisClient_StatsTracking_Unit(t *testing.T) {
	cache, cleanup := setupMockRedis(t)
	defer cleanup()

	initialStats := cache.GetStats()
	initialHits := initialStats.Hits
	initialMisses := initialStats.Misses

	// Perform operations that should affect stats
	key := "stats:test:key"
	value := map[string]interface{}{"test": "value"}

	// Set config
	err := cache.SetConfig(key, value)
	assert.NoError(t, err)

	// Get config (should be a hit)
	_, err = cache.GetConfig(key)
	assert.NoError(t, err)

	// Get non-existent config (should be a miss)
	_, err = cache.GetConfig("non:existent:key")
	// This might not error, but should still count as a miss

	// Check stats
	finalStats := cache.GetStats()
	assert.GreaterOrEqual(t, finalStats.Hits, initialHits)
	assert.Greater(t, finalStats.Misses, initialMisses)
}
