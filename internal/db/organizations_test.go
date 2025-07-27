package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"remote-config-system/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB creates a test database instance
func setupTestDB(t *testing.T) (*Repositories, testcontainers.Container) {
	ctx := context.Background()

	// Create PostgreSQL container with longer timeout for CI
	postgresContainer, err := postgres.RunContainer(ctx,
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

	testDB := &DB{DB: sqlDB}
	repos := NewRepositories(testDB)

	return repos, postgresContainer
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

func TestOrganizationRepository_Create(t *testing.T) {
	repos, container := setupTestDB(t)
	defer container.Terminate(context.Background())

	repo := repos.Organizations

	t.Run("create organization successfully", func(t *testing.T) {
		org := &models.Organization{
			Name: "Test Organization",
			Slug: "test-org",
		}

		err := repo.Create(org)
		require.NoError(t, err)

		// Verify the organization was created with proper fields
		assert.NotEqual(t, uuid.Nil, org.ID)
		assert.False(t, org.CreatedAt.IsZero())
		assert.False(t, org.UpdatedAt.IsZero())
		assert.Equal(t, "Test Organization", org.Name)
		assert.Equal(t, "test-org", org.Slug)
	})

	t.Run("create organization with duplicate slug fails", func(t *testing.T) {
		// Create first organization
		org1 := &models.Organization{
			Name: "First Organization",
			Slug: "duplicate-slug",
		}
		err := repo.Create(org1)
		require.NoError(t, err)

		// Try to create second organization with same slug
		org2 := &models.Organization{
			Name: "Second Organization",
			Slug: "duplicate-slug",
		}
		err = repo.Create(org2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key value")
	})

	t.Run("create organization with pre-set ID", func(t *testing.T) {
		presetID := uuid.New()
		org := &models.Organization{
			ID:   presetID,
			Name: "Preset ID Organization",
			Slug: "preset-id-org",
		}

		err := repo.Create(org)
		require.NoError(t, err)

		// Verify the preset ID was used
		assert.Equal(t, presetID, org.ID)
	})
}

func TestOrganizationRepository_GetBySlug(t *testing.T) {
	repos, container := setupTestDB(t)
	defer container.Terminate(context.Background())

	repo := repos.Organizations

	// Create test organization
	testOrg := &models.Organization{
		Name: "Test Organization",
		Slug: "test-org",
	}
	err := repo.Create(testOrg)
	require.NoError(t, err)

	t.Run("get existing organization by slug", func(t *testing.T) {
		org, err := repo.GetBySlug("test-org")
		require.NoError(t, err)

		assert.Equal(t, testOrg.ID, org.ID)
		assert.Equal(t, testOrg.Name, org.Name)
		assert.Equal(t, testOrg.Slug, org.Slug)
		assert.False(t, org.CreatedAt.IsZero())
		assert.False(t, org.UpdatedAt.IsZero())
	})

	t.Run("get non-existent organization by slug", func(t *testing.T) {
		_, err := repo.GetBySlug("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization not found")
	})
}

func TestOrganizationRepository_GetByID(t *testing.T) {
	repos, container := setupTestDB(t)
	defer container.Terminate(context.Background())

	repo := repos.Organizations

	// Create test organization
	testOrg := &models.Organization{
		Name: "Test Organization",
		Slug: "test-org",
	}
	err := repo.Create(testOrg)
	require.NoError(t, err)

	t.Run("get existing organization by ID", func(t *testing.T) {
		org, err := repo.GetByID(testOrg.ID)
		require.NoError(t, err)

		assert.Equal(t, testOrg.ID, org.ID)
		assert.Equal(t, testOrg.Name, org.Name)
		assert.Equal(t, testOrg.Slug, org.Slug)
	})

	t.Run("get non-existent organization by ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.GetByID(nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization not found")
	})
}

func TestOrganizationRepository_List(t *testing.T) {
	repos, container := setupTestDB(t)
	defer container.Terminate(context.Background())

	repo := repos.Organizations

	// Create multiple test organizations
	org1 := &models.Organization{Name: "Organization A", Slug: "org-a"}
	org2 := &models.Organization{Name: "Organization B", Slug: "org-b"}
	org3 := &models.Organization{Name: "Organization C", Slug: "org-c"}

	err := repo.Create(org1)
	require.NoError(t, err)
	err = repo.Create(org2)
	require.NoError(t, err)
	err = repo.Create(org3)
	require.NoError(t, err)

	t.Run("list all organizations", func(t *testing.T) {
		orgs, total, err := repo.List(models.PaginationParams{
			Page:     1,
			PageSize: 100,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 3)

		// Should have at least our 3 test organizations
		assert.GreaterOrEqual(t, len(orgs), 3)

		// Check that our test organizations are in the list
		orgIDs := make(map[uuid.UUID]bool)
		for _, org := range orgs {
			orgIDs[org.ID] = true
		}

		assert.True(t, orgIDs[org1.ID])
		assert.True(t, orgIDs[org2.ID])
		assert.True(t, orgIDs[org3.ID])
	})
}

func TestOrganizationRepository_Update(t *testing.T) {
	repos, container := setupTestDB(t)
	defer container.Terminate(context.Background())

	repo := repos.Organizations

	// Create test organization
	testOrg := &models.Organization{
		Name: "Original Name",
		Slug: "original-slug",
	}
	err := repo.Create(testOrg)
	require.NoError(t, err)

	t.Run("update organization successfully", func(t *testing.T) {
		// Update the organization
		testOrg.Name = "Updated Name"

		err := repo.Update(testOrg)
		require.NoError(t, err)

		// Verify the update
		updatedOrg, err := repo.GetByID(testOrg.ID)
		require.NoError(t, err)

		assert.Equal(t, "Updated Name", updatedOrg.Name)
		assert.Equal(t, "original-slug", updatedOrg.Slug) // Slug should remain unchanged
		assert.True(t, updatedOrg.UpdatedAt.After(updatedOrg.CreatedAt) || updatedOrg.UpdatedAt.Equal(updatedOrg.CreatedAt))
	})

	t.Run("update non-existent organization", func(t *testing.T) {
		nonExistentOrg := &models.Organization{
			ID:   uuid.New(),
			Name: "Non-existent",
			Slug: "non-existent",
		}

		err := repo.Update(nonExistentOrg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization not found")
	})
}

func TestOrganizationRepository_Delete(t *testing.T) {
	repos, container := setupTestDB(t)
	defer container.Terminate(context.Background())

	repo := repos.Organizations

	// Create test organization
	testOrg := &models.Organization{
		Name: "To Be Deleted",
		Slug: "to-be-deleted",
	}
	err := repo.Create(testOrg)
	require.NoError(t, err)

	t.Run("delete existing organization", func(t *testing.T) {
		err := repo.Delete(testOrg.ID)
		require.NoError(t, err)

		// Verify the organization is deleted
		_, err = repo.GetByID(testOrg.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization not found")
	})

	t.Run("delete non-existent organization", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization not found")
	})
}

func TestOrganizationRepository_Exists(t *testing.T) {
	repos, container := setupTestDB(t)
	defer container.Terminate(context.Background())

	repo := repos.Organizations

	// Create test organization
	testOrg := &models.Organization{
		Name: "Existing Organization",
		Slug: "existing-org",
	}
	err := repo.Create(testOrg)
	require.NoError(t, err)

	t.Run("check existing organization", func(t *testing.T) {
		exists, err := repo.Exists("existing-org")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("check non-existent organization", func(t *testing.T) {
		exists, err := repo.Exists("non-existent-org")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	// Clean up
	_ = testOrg
}

func TestOrganizationRepository_GetWithApplications(t *testing.T) {
	repos, container := setupTestDB(t)
	defer container.Terminate(context.Background())

	orgRepo := repos.Organizations
	appRepo := repos.Applications

	// Create test organization
	testOrg := &models.Organization{
		Name: "Org with Apps",
		Slug: "org-with-apps",
	}
	err := orgRepo.Create(testOrg)
	require.NoError(t, err)

	// Create test applications
	app1 := &models.Application{
		OrgID:  testOrg.ID,
		Name:   "App 1",
		Slug:   "app-1",
		APIKey: "api-key-1",
	}
	app2 := &models.Application{
		OrgID:  testOrg.ID,
		Name:   "App 2",
		Slug:   "app-2",
		APIKey: "api-key-2",
	}
	err = appRepo.Create(app1)
	require.NoError(t, err)
	err = appRepo.Create(app2)
	require.NoError(t, err)

	t.Run("get organization with applications", func(t *testing.T) {
		// This test assumes there's a method to get organization with applications
		// If not implemented, we can test the basic functionality
		org, err := orgRepo.GetByID(testOrg.ID)
		require.NoError(t, err)

		// Get applications for this organization
		apps, total, err := appRepo.ListByOrganization(testOrg.ID, models.PaginationParams{
			Page:     1,
			PageSize: 100,
		})
		require.NoError(t, err)

		assert.Equal(t, testOrg.ID, org.ID)
		assert.GreaterOrEqual(t, len(apps), 2)
		assert.GreaterOrEqual(t, total, 2)

		// Verify our test applications are included
		appIDs := make(map[uuid.UUID]bool)
		for _, app := range apps {
			appIDs[app.ID] = true
		}

		assert.True(t, appIDs[app1.ID])
		assert.True(t, appIDs[app2.ID])
	})
}
