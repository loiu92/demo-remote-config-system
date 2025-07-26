package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis client with configuration caching functionality
type RedisClient struct {
	client *redis.Client
	ttl    time.Duration
}

// Config holds Redis configuration
type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
	TTL      time.Duration
}

// NewConfig creates a new Redis configuration from environment variables
func NewConfig() *Config {
	ttl := 300 * time.Second // Default 5 minutes
	if ttlStr := os.Getenv("CACHE_TTL"); ttlStr != "" {
		if parsedTTL, err := strconv.Atoi(ttlStr); err == nil {
			ttl = time.Duration(parsedTTL) * time.Second
		}
	}

	db := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if parsedDB, err := strconv.Atoi(dbStr); err == nil {
			db = parsedDB
		}
	}

	return &Config{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       db,
		TTL:      ttl,
	}
}

// NewRedisClient creates a new Redis client
func NewRedisClient(config *Config) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Successfully connected to Redis %s:%s", config.Host, config.Port)

	return &RedisClient{
		client: rdb,
		ttl:    config.TTL,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Health checks if Redis is healthy
func (r *RedisClient) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	return nil
}

// GetConfig retrieves a configuration from cache
func (r *RedisClient) GetConfig(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get config from cache: %w", err)
	}

	return []byte(val), nil
}

// SetConfig stores a configuration in cache
func (r *RedisClient) SetConfig(key string, config interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set config in cache: %w", err)
	}

	return nil
}

// DeleteConfig removes a configuration from cache
func (r *RedisClient) DeleteConfig(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete config from cache: %w", err)
	}

	return nil
}

// InvalidatePattern removes all configurations matching a pattern
func (r *RedisClient) InvalidatePattern(pattern string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		return nil
	}

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete keys for pattern %s: %w", pattern, err)
	}

	log.Printf("Invalidated %d cache entries for pattern: %s", len(keys), pattern)
	return nil
}

// GenerateConfigKey generates a cache key for configuration
func GenerateConfigKey(orgSlug, appSlug, envSlug string) string {
	return fmt.Sprintf("config:%s:%s:%s", orgSlug, appSlug, envSlug)
}

// GenerateAPIKeyConfigKey generates a cache key for API key-based configuration
func GenerateAPIKeyConfigKey(apiKey, envSlug string) string {
	return fmt.Sprintf("config:api:%s:%s", apiKey, envSlug)
}

// GenerateInvalidationPattern generates a pattern for cache invalidation
func GenerateInvalidationPattern(orgSlug, appSlug, envSlug string) string {
	return fmt.Sprintf("config:*:%s:%s:%s", orgSlug, appSlug, envSlug)
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
