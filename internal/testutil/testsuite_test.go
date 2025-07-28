package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupTestSuite tests the basic setup of the test suite
func TestSetupTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupTestSuite(t)
	defer suite.Cleanup(t)

	// Test database connection
	require.NotNil(t, suite.DB)
	err := suite.DB.Ping()
	assert.NoError(t, err)

	// Test Redis connection
	require.NotNil(t, suite.Redis)
	require.NotNil(t, suite.Redis.Client)
	err = suite.Redis.Client.Health()
	assert.NoError(t, err)

	// Test repositories
	require.NotNil(t, suite.Repos)
	require.NotNil(t, suite.Repos.Organizations)
	require.NotNil(t, suite.Repos.Applications)
	require.NotNil(t, suite.Repos.Environments)
	require.NotNil(t, suite.Repos.ConfigVersions)
	require.NotNil(t, suite.Repos.ConfigChanges)
}

// TestCreateTestData tests the helper methods for creating test data
func TestCreateTestData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	suite := SetupTestSuite(t)
	defer suite.Cleanup(t)

	// Create test organization
	org := suite.CreateTestOrganization(t, "Test Org", "test-org")
	assert.NotNil(t, org)
	assert.Equal(t, "Test Org", org.Name)
	assert.Equal(t, "test-org", org.Slug)

	// Create test application
	app := suite.CreateTestApplication(t, org.ID, "Test App", "test-app", "test-api-key")
	assert.NotNil(t, app)
	assert.Equal(t, "Test App", app.Name)
	assert.Equal(t, "test-app", app.Slug)
	assert.Equal(t, "test-api-key", app.APIKey)
	assert.Equal(t, org.ID, app.OrgID)

	// Create test environment
	env := suite.CreateTestEnvironment(t, app.ID, "Test Env", "test-env")
	assert.NotNil(t, env)
	assert.Equal(t, "Test Env", env.Name)
	assert.Equal(t, "test-env", env.Slug)
	assert.Equal(t, app.ID, env.AppID)
}
