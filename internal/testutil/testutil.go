package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"remote-config-system/internal/cache"
	"remote-config-system/internal/db"
	"remote-config-system/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDatabase represents a test database instance
type TestDatabase struct {
	Container testcontainers.Container
	DB        *db.DB
	DSN       string
}

// TestRedis represents a test Redis instance
type TestRedis struct {
	Container testcontainers.Container
	Client    *cache.RedisClient
	Host      string
	Port      string
}

// TestSuite provides a complete test environment
type TestSuite struct {
	Database *TestDatabase
	Redis    *TestRedis
	Repos    *db.Repositories
}

// SetupTestDatabase creates a test PostgreSQL database using testcontainers
func SetupTestDatabase(t *testing.T) *TestDatabase {
	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("test_remote_config"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	// Get connection details
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=test_remote_config sslmode=disable",
		host, port.Port())

	// Connect to database
	sqlDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)

	// Test connection
	err = sqlDB.Ping()
	require.NoError(t, err)

	// Run migrations
	err = runMigrations(sqlDB)
	require.NoError(t, err)

	testDB := &db.DB{DB: sqlDB}

	return &TestDatabase{
		Container: postgresContainer,
		DB:        testDB,
		DSN:       dsn,
	}
}

// SetupTestRedis creates a test Redis instance using testcontainers
func SetupTestRedis(t *testing.T) *TestRedis {
	ctx := context.Background()

	// Create Redis container
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(wait.ForLog("Ready to accept connections")),
	)
	require.NoError(t, err)

	// Get connection details
	host, err := redisContainer.Host(ctx)
	require.NoError(t, err)

	port, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	// Create Redis client
	config := &cache.Config{
		Host:           host,
		Port:           port.Port(),
		Password:       "",
		DB:             0,
		TTL:            5 * time.Minute,
		ShortTTL:       1 * time.Minute,
		LongTTL:        1 * time.Hour,
		EnableCompress: false,
	}

	redisClient, err := cache.NewRedisClient(config)
	require.NoError(t, err)

	return &TestRedis{
		Container: redisContainer,
		Client:    redisClient,
		Host:      host,
		Port:      port.Port(),
	}
}

// SetupTestSuite creates a complete test environment
func SetupTestSuite(t *testing.T) *TestSuite {
	testDB := SetupTestDatabase(t)
	testRedis := SetupTestRedis(t)
	repos := db.NewRepositories(testDB.DB)

	return &TestSuite{
		Database: testDB,
		Redis:    testRedis,
		Repos:    repos,
	}
}

// Cleanup cleans up test resources
func (ts *TestSuite) Cleanup(t *testing.T) {
	ctx := context.Background()

	if ts.Redis != nil {
		if ts.Redis.Client != nil {
			ts.Redis.Client.Close()
		}
		if ts.Redis.Container != nil {
			err := ts.Redis.Container.Terminate(ctx)
			if err != nil {
				log.Printf("Failed to terminate Redis container: %v", err)
			}
		}
	}

	if ts.Database != nil {
		if ts.Database.DB != nil {
			ts.Database.DB.Close()
		}
		if ts.Database.Container != nil {
			err := ts.Database.Container.Terminate(ctx)
			if err != nil {
				log.Printf("Failed to terminate database container: %v", err)
			}
		}
	}
}

// CreateTestOrganization creates a test organization
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

// CreateTestApplication creates a test application
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

// CreateTestEnvironment creates a test environment
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

// runMigrations runs the database migrations for testing
func runMigrations(db *sql.DB) error {
	// Read and execute the migration file
	migrationSQL := `
		-- Organizations table
		CREATE TABLE IF NOT EXISTS organizations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(100) NOT NULL UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Applications table
		CREATE TABLE IF NOT EXISTS applications (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(100) NOT NULL,
			api_key VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(org_id, slug)
		);

		-- Environments table
		CREATE TABLE IF NOT EXISTS environments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			app_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(100) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(app_id, slug)
		);

		-- Configuration versions table
		CREATE TABLE IF NOT EXISTS config_versions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			env_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
			version INTEGER NOT NULL,
			config_json JSONB NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_by VARCHAR(255),
			UNIQUE(env_id, version)
		);

		-- Configuration changes table
		CREATE TABLE IF NOT EXISTS config_changes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			env_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
			version_from INTEGER,
			version_to INTEGER NOT NULL,
			action VARCHAR(50) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_by VARCHAR(255)
		);

		-- Indexes for better performance
		CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);
		CREATE INDEX IF NOT EXISTS idx_applications_org_id ON applications(org_id);
		CREATE INDEX IF NOT EXISTS idx_applications_api_key ON applications(api_key);
		CREATE INDEX IF NOT EXISTS idx_environments_app_id ON environments(app_id);
		CREATE INDEX IF NOT EXISTS idx_config_versions_env_id ON config_versions(env_id);
		CREATE INDEX IF NOT EXISTS idx_config_versions_active ON config_versions(env_id, is_active) WHERE is_active = true;
		CREATE INDEX IF NOT EXISTS idx_config_changes_env_id ON config_changes(env_id);
	`

	_, err := db.Exec(migrationSQL)
	return err
}
