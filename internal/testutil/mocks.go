package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"remote-config-system/internal/cache"
	"remote-config-system/internal/db"
	"remote-config-system/internal/models"
	"remote-config-system/internal/sse"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestSuite provides a complete test environment with database and cache
type TestSuite struct {
	DB              *db.DB
	Redis           *TestRedisClient
	Repos           *db.Repositories
	PostgresContainer *postgres.PostgresContainer
	RedisContainer    *redis.RedisContainer
	ctx             context.Context
}

// TestRedisClient wraps the cache client for testing
type TestRedisClient struct {
	Client *cache.RedisClient
	Container *redis.RedisContainer
}

// SetupTestSuite creates a complete test environment with real database and cache
func SetupTestSuite(t *testing.T) *TestSuite {
	ctx := context.Background()

	// Check if we're running in CI environment (GitHub Actions provides services)
	isCI := os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"

	var database *db.DB
	var redisClient *cache.RedisClient
	var postgresContainer *postgres.PostgresContainer
	var redisContainer *redis.RedisContainer

	if isCI {
		// Use provided services in CI
		dbConfig := &db.Config{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "test_remote_config"),
			SSLMode:  "disable",
		}

		var err error
		database, err = db.Connect(dbConfig)
		require.NoError(t, err)

		// Run migrations
		migrationsDir := findMigrationsDir()
		migrationRunner := db.NewMigrationRunner(database, migrationsDir)
		err = migrationRunner.RunMigrations()
		require.NoError(t, err)

		// Connect to Redis
		cacheConfig := &cache.Config{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
			TTL:      5 * time.Minute,
			ShortTTL: 1 * time.Minute,
			LongTTL:  10 * time.Minute,
		}

		redisClient, err = cache.NewRedisClient(cacheConfig)
		require.NoError(t, err)
	} else {
		// Use testcontainers for local development
		var err error

		// Start PostgreSQL container
		postgresContainer, err = postgres.RunContainer(ctx,
			testcontainers.WithImage("postgres:15-alpine"),
			postgres.WithDatabase("test_remote_config"),
			postgres.WithUsername("test"),
			postgres.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second)),
		)
		require.NoError(t, err)

		// Get PostgreSQL connection details
		host, err := postgresContainer.Host(ctx)
		require.NoError(t, err)
		port, err := postgresContainer.MappedPort(ctx, "5432")
		require.NoError(t, err)

		// Connect to PostgreSQL
		dbConfig := &db.Config{
			Host:     host,
			Port:     port.Port(),
			User:     "test",
			Password: "test",
			DBName:   "test_remote_config",
			SSLMode:  "disable",
		}

		database, err = db.Connect(dbConfig)
		require.NoError(t, err)

		// Run migrations
		migrationsDir := findMigrationsDir()
		migrationRunner := db.NewMigrationRunner(database, migrationsDir)
		err = migrationRunner.RunMigrations()
		require.NoError(t, err)

		// Start Redis container
		redisContainer, err = redis.RunContainer(ctx,
			testcontainers.WithImage("redis:7-alpine"),
			testcontainers.WithWaitStrategy(wait.ForLog("Ready to accept connections")),
		)
		require.NoError(t, err)

		// Get Redis connection details
		redisHost, err := redisContainer.Host(ctx)
		require.NoError(t, err)
		redisPort, err := redisContainer.MappedPort(ctx, "6379")
		require.NoError(t, err)

		// Connect to Redis
		cacheConfig := &cache.Config{
			Host:     redisHost,
			Port:     redisPort.Port(),
			Password: "",
			DB:       0,
			TTL:      5 * time.Minute,
			ShortTTL: 1 * time.Minute,
			LongTTL:  10 * time.Minute,
		}

		redisClient, err = cache.NewRedisClient(cacheConfig)
		require.NoError(t, err)
	}

	// Create repositories
	repos := db.NewRepositories(database)

	return &TestSuite{
		DB:                database,
		Redis:             &TestRedisClient{Client: redisClient, Container: redisContainer},
		Repos:             repos,
		PostgresContainer: postgresContainer,
		RedisContainer:    redisContainer,
		ctx:               ctx,
	}
}

// Cleanup cleans up the test environment
func (ts *TestSuite) Cleanup(t *testing.T) {
	if ts.DB != nil {
		ts.DB.Close()
	}
	if ts.Redis != nil && ts.Redis.Client != nil {
		ts.Redis.Client.Close()
	}
	if ts.PostgresContainer != nil {
		if err := ts.PostgresContainer.Terminate(ts.ctx); err != nil {
			log.Printf("Failed to terminate PostgreSQL container: %v", err)
		}
	}
	if ts.RedisContainer != nil {
		if err := ts.RedisContainer.Terminate(ts.ctx); err != nil {
			log.Printf("Failed to terminate Redis container: %v", err)
		}
	}
}

// CreateTestOrganization creates a test organization in the database
func (ts *TestSuite) CreateTestOrganization(t *testing.T, name, slug string) *models.Organization {
	org := &models.Organization{
		ID:   uuid.New(),
		Name: name,
		Slug: slug,
	}
	err := ts.Repos.Organizations.Create(org)
	require.NoError(t, err)
	return org
}

// CreateTestApplication creates a test application in the database
func (ts *TestSuite) CreateTestApplication(t *testing.T, orgID uuid.UUID, name, slug, apiKey string) *models.Application {
	app := &models.Application{
		ID:     uuid.New(),
		OrgID:  orgID,
		Name:   name,
		Slug:   slug,
		APIKey: apiKey,
	}
	err := ts.Repos.Applications.Create(app)
	require.NoError(t, err)
	return app
}

// CreateTestEnvironment creates a test environment in the database
func (ts *TestSuite) CreateTestEnvironment(t *testing.T, appID uuid.UUID, name, slug string) *models.Environment {
	env := &models.Environment{
		ID:    uuid.New(),
		AppID: appID,
		Name:  name,
		Slug:  slug,
	}
	err := ts.Repos.Environments.Create(env)
	require.NoError(t, err)
	return env
}

// MockConfigService is a mock implementation of the ConfigService
type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) GetConfiguration(orgSlug, appSlug, envSlug string) (*models.ConfigResponse, error) {
	args := m.Called(orgSlug, appSlug, envSlug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigResponse), args.Error(1)
}

func (m *MockConfigService) GetConfigurationByAPIKey(apiKey, envSlug string) (*models.ConfigResponse, error) {
	args := m.Called(apiKey, envSlug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigResponse), args.Error(1)
}

func (m *MockConfigService) UpdateConfiguration(orgSlug, appSlug, envSlug string, req *models.CreateConfigRequest) (*models.ConfigResponse, error) {
	args := m.Called(orgSlug, appSlug, envSlug, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigResponse), args.Error(1)
}

func (m *MockConfigService) ValidateAPIKey(apiKey string) (*models.Application, error) {
	args := m.Called(apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Application), args.Error(1)
}

func (m *MockConfigService) HealthCheck() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

func (m *MockConfigService) CreateOrganization(req *models.CreateOrganizationRequest) (*models.Organization, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockConfigService) CreateApplication(orgSlug string, req *models.CreateApplicationRequest) (*models.Application, error) {
	args := m.Called(orgSlug, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Application), args.Error(1)
}

func (m *MockConfigService) CreateEnvironment(orgSlug, appSlug string, req *models.CreateEnvironmentRequest) (*models.Environment, error) {
	args := m.Called(orgSlug, appSlug, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Environment), args.Error(1)
}

// Additional methods for ConfigServiceInterface
func (m *MockConfigService) RollbackConfiguration(orgSlug, appSlug, envSlug string, req *models.RollbackRequest) (*models.ConfigResponse, error) {
	args := m.Called(orgSlug, appSlug, envSlug, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigResponse), args.Error(1)
}

func (m *MockConfigService) GetConfigurationHistory(orgSlug, appSlug, envSlug string, params models.PaginationParams) (*models.PaginatedResponse, error) {
	args := m.Called(orgSlug, appSlug, envSlug, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedResponse), args.Error(1)
}

func (m *MockConfigService) GetConfigurationVersion(orgSlug, appSlug, envSlug string, version int) (*models.ConfigResponse, error) {
	args := m.Called(orgSlug, appSlug, envSlug, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigResponse), args.Error(1)
}

func (m *MockConfigService) GetConfigurationChanges(orgSlug, appSlug, envSlug string, params models.PaginationParams) (*models.PaginatedResponse, error) {
	args := m.Called(orgSlug, appSlug, envSlug, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedResponse), args.Error(1)
}

// MockSSEService is a mock implementation of the SSE service
type MockSSEService struct {
	mock.Mock
	clients map[string]*sse.Client
	mu      sync.RWMutex
}

// Ensure MockSSEService implements SSEServiceInterface
var _ sse.SSEServiceInterface = (*MockSSEService)(nil)

func NewMockSSEService() *MockSSEService {
	return &MockSSEService{
		clients: make(map[string]*sse.Client),
	}
}

func (m *MockSSEService) RegisterClient(client *sse.Client) {
	m.Called(client)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[client.ID] = client
}

func (m *MockSSEService) UnregisterClient(client *sse.Client) {
	m.Called(client)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, client.ID)
}

func (m *MockSSEService) BroadcastConfigUpdate(event models.ConfigUpdateEvent) {
	m.Called(event)
}

func (m *MockSSEService) BroadcastCustomEvent(org, app, env, eventType string, data interface{}) {
	m.Called(org, app, env, eventType, data)
}

func (m *MockSSEService) GetStats() sse.SSEStats {
	args := m.Called()
	return args.Get(0).(sse.SSEStats)
}

func (m *MockSSEService) Ping(clientID string) {
	m.Called(clientID)
}

// MockCacheClient is a mock implementation of the Redis cache client
type MockCacheClient struct {
	mock.Mock
	data map[string][]byte
	mu   sync.RWMutex
}

func NewMockCacheClient() *MockCacheClient {
	return &MockCacheClient{
		data: make(map[string][]byte),
	}
}

func (m *MockCacheClient) GetConfig(key string) ([]byte, error) {
	args := m.Called(key)
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if data, exists := m.data[key]; exists {
		return data, nil
	}
	
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheClient) SetConfig(key string, config interface{}) error {
	args := m.Called(key, config)
	
	// Actually store the data for testing
	data, _ := json.Marshal(config)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = data
	
	return args.Error(0)
}

func (m *MockCacheClient) SetConfigWithTTL(key string, config interface{}, ttl time.Duration) error {
	args := m.Called(key, config, ttl)
	
	// Actually store the data for testing
	data, _ := json.Marshal(config)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = data
	
	return args.Error(0)
}

func (m *MockCacheClient) DeleteConfig(key string) error {
	args := m.Called(key)
	
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	
	return args.Error(0)
}

func (m *MockCacheClient) InvalidatePattern(pattern string) error {
	args := m.Called(pattern)
	return args.Error(0)
}

func (m *MockCacheClient) Health() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCacheClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Test data helpers

// CreateTestConfigResponse creates a test configuration response
func CreateTestConfigResponse(org, app, env string, version int) *models.ConfigResponse {
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

// CreateTestApplication creates a test application model
func CreateTestApplication(orgID uuid.UUID, name, slug, apiKey string) *models.Application {
	return &models.Application{
		ID:     uuid.New(),
		OrgID:  orgID,
		Name:   name,
		Slug:   slug,
		APIKey: apiKey,
		Organization: &models.Organization{
			ID:   orgID,
			Name: "Test Org",
			Slug: "test-org",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestOrganization creates a test organization model
func CreateTestOrganization(name, slug string) *models.Organization {
	return &models.Organization{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestEnvironment creates a test environment model
func CreateTestEnvironment(appID uuid.UUID, name, slug string) *models.Environment {
	return &models.Environment{
		ID:        uuid.New(),
		AppID:     appID,
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestUpdateConfigRequest creates a test update config request
func CreateTestUpdateConfigRequest(createdBy string) *models.CreateConfigRequest {
	config := map[string]interface{}{
		"database_url": "postgres://localhost:5432/updated",
		"api_timeout":  60,
		"debug":        false,
		"new_feature":  true,
	}

	configJSON, _ := json.Marshal(config)

	return &models.CreateConfigRequest{
		Config:    configJSON,
		CreatedBy: &createdBy,
	}
}

// AssertConfigEqual asserts that two configurations are equal
func AssertConfigEqual(t interface{}, expected, actual *models.ConfigResponse) {
	// This would use testify's assert package in real implementation
	// For now, we'll use a simple comparison
	if expected.Organization != actual.Organization ||
		expected.Application != actual.Application ||
		expected.Environment != actual.Environment ||
		expected.Version != actual.Version {
		panic(fmt.Sprintf("Config mismatch: expected %+v, got %+v", expected, actual))
	}
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// findMigrationsDir finds the migrations directory relative to the current working directory
func findMigrationsDir() string {
	// Try different possible paths
	possiblePaths := []string{
		"../../migrations",  // From internal/testutil or internal/integration
		"../migrations",     // From internal
		"./migrations",      // From root
		"migrations",        // From root (alternative)
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default fallback
	return "../../migrations"
}
