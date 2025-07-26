package cache

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheStats holds cache performance statistics
type CacheStats struct {
	Hits        int64 `json:"hits"`
	Misses      int64 `json:"misses"`
	Sets        int64 `json:"sets"`
	Deletes     int64 `json:"deletes"`
	Errors      int64 `json:"errors"`
	TotalKeys   int64 `json:"total_keys"`
}

// GetHitRatio returns the cache hit ratio as a percentage
func (s *CacheStats) GetHitRatio() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total) * 100
}

// RedisClient wraps the Redis client with configuration caching functionality
type RedisClient struct {
	client       *redis.Client
	ttl          time.Duration
	shortTTL     time.Duration // For frequently changing data
	longTTL      time.Duration // For rarely changing data
	stats        *CacheStats
	enableCompress bool
}

// Config holds Redis configuration
type Config struct {
	Host           string
	Port           string
	Password       string
	DB             int
	TTL            time.Duration // Default TTL
	ShortTTL       time.Duration // For frequently changing data
	LongTTL        time.Duration // For rarely changing data
	EnableCompress bool          // Enable compression for large values
}

// NewConfig creates a new Redis configuration from environment variables
func NewConfig() *Config {
	// Default TTL values
	ttl := 300 * time.Second      // Default 5 minutes
	shortTTL := 60 * time.Second  // 1 minute for frequently changing data
	longTTL := 3600 * time.Second // 1 hour for rarely changing data

	// Parse TTL from environment
	if ttlStr := os.Getenv("CACHE_TTL"); ttlStr != "" {
		if parsedTTL, err := strconv.Atoi(ttlStr); err == nil {
			ttl = time.Duration(parsedTTL) * time.Second
		}
	}

	if shortTTLStr := os.Getenv("CACHE_SHORT_TTL"); shortTTLStr != "" {
		if parsedTTL, err := strconv.Atoi(shortTTLStr); err == nil {
			shortTTL = time.Duration(parsedTTL) * time.Second
		}
	}

	if longTTLStr := os.Getenv("CACHE_LONG_TTL"); longTTLStr != "" {
		if parsedTTL, err := strconv.Atoi(longTTLStr); err == nil {
			longTTL = time.Duration(parsedTTL) * time.Second
		}
	}

	db := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if parsedDB, err := strconv.Atoi(dbStr); err == nil {
			db = parsedDB
		}
	}

	enableCompress := getEnv("CACHE_ENABLE_COMPRESSION", "false") == "true"

	return &Config{
		Host:           getEnv("REDIS_HOST", "localhost"),
		Port:           getEnv("REDIS_PORT", "6379"),
		Password:       getEnv("REDIS_PASSWORD", ""),
		DB:             db,
		TTL:            ttl,
		ShortTTL:       shortTTL,
		LongTTL:        longTTL,
		EnableCompress: enableCompress,
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
		client:       rdb,
		ttl:          config.TTL,
		shortTTL:     config.ShortTTL,
		longTTL:      config.LongTTL,
		enableCompress: config.EnableCompress,
		stats:        &CacheStats{},
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
			atomic.AddInt64(&r.stats.Misses, 1)
			return nil, nil // Cache miss
		}
		atomic.AddInt64(&r.stats.Errors, 1)
		return nil, fmt.Errorf("failed to get config from cache: %w", err)
	}

	atomic.AddInt64(&r.stats.Hits, 1)

	// Handle compression
	data := []byte(val)
	if r.enableCompress && strings.HasPrefix(key, "compressed:") {
		decompressed, err := r.decompress(data)
		if err != nil {
			atomic.AddInt64(&r.stats.Errors, 1)
			return nil, fmt.Errorf("failed to decompress cached data: %w", err)
		}
		data = decompressed
	}

	return data, nil
}

// SetConfig stores a configuration in cache with default TTL
func (r *RedisClient) SetConfig(key string, config interface{}) error {
	return r.SetConfigWithTTL(key, config, r.ttl)
}

// SetConfigWithTTL stores a configuration in cache with custom TTL
func (r *RedisClient) SetConfigWithTTL(key string, config interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(config)
	if err != nil {
		atomic.AddInt64(&r.stats.Errors, 1)
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Handle compression for large data
	finalKey := key
	if r.enableCompress && len(data) > 1024 { // Compress if larger than 1KB
		compressed, err := r.compress(data)
		if err != nil {
			atomic.AddInt64(&r.stats.Errors, 1)
			return fmt.Errorf("failed to compress config: %w", err)
		}
		data = compressed
		finalKey = "compressed:" + key
	}

	if err := r.client.Set(ctx, finalKey, data, ttl).Err(); err != nil {
		atomic.AddInt64(&r.stats.Errors, 1)
		return fmt.Errorf("failed to set config in cache: %w", err)
	}

	atomic.AddInt64(&r.stats.Sets, 1)
	atomic.AddInt64(&r.stats.TotalKeys, 1)
	return nil
}

// DeleteConfig removes a configuration from cache
func (r *RedisClient) DeleteConfig(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Delete both regular and compressed versions
	keys := []string{key, "compressed:" + key}

	deleted, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		atomic.AddInt64(&r.stats.Errors, 1)
		return fmt.Errorf("failed to delete config from cache: %w", err)
	}

	if deleted > 0 {
		atomic.AddInt64(&r.stats.Deletes, 1)
		atomic.AddInt64(&r.stats.TotalKeys, -deleted)
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

// SetConfigShortTTL stores a configuration with short TTL (for frequently changing data)
func (r *RedisClient) SetConfigShortTTL(key string, config interface{}) error {
	return r.SetConfigWithTTL(key, config, r.shortTTL)
}

// SetConfigLongTTL stores a configuration with long TTL (for rarely changing data)
func (r *RedisClient) SetConfigLongTTL(key string, config interface{}) error {
	return r.SetConfigWithTTL(key, config, r.longTTL)
}

// WarmCache preloads frequently accessed configurations
func (r *RedisClient) WarmCache(configs map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipe := r.client.Pipeline()

	for key, config := range configs {
		data, err := json.Marshal(config)
		if err != nil {
			log.Printf("Failed to marshal config for cache warming: %v", err)
			continue
		}

		// Handle compression for large data
		finalKey := key
		if r.enableCompress && len(data) > 1024 {
			compressed, err := r.compress(data)
			if err != nil {
				log.Printf("Failed to compress config for cache warming: %v", err)
				continue
			}
			data = compressed
			finalKey = "compressed:" + key
		}

		pipe.Set(ctx, finalKey, data, r.ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		atomic.AddInt64(&r.stats.Errors, 1)
		return fmt.Errorf("failed to warm cache: %w", err)
	}

	atomic.AddInt64(&r.stats.Sets, int64(len(configs)))
	atomic.AddInt64(&r.stats.TotalKeys, int64(len(configs)))
	log.Printf("Warmed cache with %d configurations", len(configs))
	return nil
}

// GetCacheInfo returns information about cached keys
func (r *RedisClient) GetCacheInfo() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info := make(map[string]interface{})

	// Get total keys count
	keys, err := r.client.Keys(ctx, "config:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache info: %w", err)
	}

	info["total_keys"] = len(keys)
	info["stats"] = r.GetStats()

	// Get memory usage if available
	if memInfo, err := r.client.Info(ctx, "memory").Result(); err == nil {
		info["memory_info"] = memInfo
	}

	return info, nil
}

// GetStats returns current cache statistics
func (r *RedisClient) GetStats() *CacheStats {
	return &CacheStats{
		Hits:      atomic.LoadInt64(&r.stats.Hits),
		Misses:    atomic.LoadInt64(&r.stats.Misses),
		Sets:      atomic.LoadInt64(&r.stats.Sets),
		Deletes:   atomic.LoadInt64(&r.stats.Deletes),
		Errors:    atomic.LoadInt64(&r.stats.Errors),
		TotalKeys: atomic.LoadInt64(&r.stats.TotalKeys),
	}
}

// ResetStats resets cache statistics
func (r *RedisClient) ResetStats() {
	atomic.StoreInt64(&r.stats.Hits, 0)
	atomic.StoreInt64(&r.stats.Misses, 0)
	atomic.StoreInt64(&r.stats.Sets, 0)
	atomic.StoreInt64(&r.stats.Deletes, 0)
	atomic.StoreInt64(&r.stats.Errors, 0)
	atomic.StoreInt64(&r.stats.TotalKeys, 0)
}

// compress compresses data using gzip
func (r *RedisClient) compress(data []byte) ([]byte, error) {
	if !r.enableCompress {
		return data, nil
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decompress decompresses gzip data
func (r *RedisClient) decompress(data []byte) ([]byte, error) {
	if !r.enableCompress {
		return data, nil
	}

	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	return io.ReadAll(gz)
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
