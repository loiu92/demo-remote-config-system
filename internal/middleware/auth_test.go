package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"remote-config-system/internal/models"
	"remote-config-system/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_APIKeyAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid API key in Authorization header with Bearer prefix", func(t *testing.T) {
		// Setup mock service
		mockService := &testutil.MockConfigService{}
		app := testutil.CreateTestApplication(uuid.New(), "Test App", "test-app", "valid-api-key")
		
		mockService.On("ValidateAPIKey", "valid-api-key").Return(app, nil)

		// Create middleware
		authMiddleware := NewAuthMiddleware(mockService)

		// Create test context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer valid-api-key")

		// Create a test handler to verify the middleware passes through
		var handlerCalled bool
		testHandler := func(c *gin.Context) {
			handlerCalled = true
			
			// Verify that application and api_key are set in context
			appFromContext, exists := c.Get("application")
			assert.True(t, exists)
			assert.Equal(t, app, appFromContext)
			
			apiKeyFromContext, exists := c.Get("api_key")
			assert.True(t, exists)
			assert.Equal(t, "valid-api-key", apiKeyFromContext)
			
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}

		// Execute middleware and handler
		authMiddleware.APIKeyAuth()(c)
		if !c.IsAborted() {
			testHandler(c)
		}

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("valid API key in X-API-Key header", func(t *testing.T) {
		// Setup mock service
		mockService := &testutil.MockConfigService{}
		app := testutil.CreateTestApplication(uuid.New(), "Test App", "test-app", "valid-api-key")
		
		mockService.On("ValidateAPIKey", "valid-api-key").Return(app, nil)

		// Create middleware
		authMiddleware := NewAuthMiddleware(mockService)

		// Create test context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "valid-api-key")

		// Execute middleware
		authMiddleware.APIKeyAuth()(c)

		assert.False(t, c.IsAborted())
		mockService.AssertExpectations(t)
	})

	t.Run("valid API key in query parameter", func(t *testing.T) {
		// Setup mock service
		mockService := &testutil.MockConfigService{}
		app := testutil.CreateTestApplication(uuid.New(), "Test App", "test-app", "valid-api-key")
		
		mockService.On("ValidateAPIKey", "valid-api-key").Return(app, nil)

		// Create middleware
		authMiddleware := NewAuthMiddleware(mockService)

		// Create test context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test?api_key=valid-api-key", nil)

		// Execute middleware
		authMiddleware.APIKeyAuth()(c)

		assert.False(t, c.IsAborted())
		mockService.AssertExpectations(t)
	})

	t.Run("missing API key", func(t *testing.T) {
		// Setup mock service
		mockService := &testutil.MockConfigService{}

		// Create middleware
		authMiddleware := NewAuthMiddleware(mockService)

		// Create test context without API key
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		// Execute middleware
		authMiddleware.APIKeyAuth()(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Verify error response
		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", response.Error)
		assert.Equal(t, "API key is required", response.Message)
	})

	t.Run("invalid API key", func(t *testing.T) {
		// Setup mock service
		mockService := &testutil.MockConfigService{}
		mockService.On("ValidateAPIKey", "invalid-api-key").Return(nil, assert.AnError)

		// Create middleware
		authMiddleware := NewAuthMiddleware(mockService)

		// Create test context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "invalid-api-key")

		// Execute middleware
		authMiddleware.APIKeyAuth()(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Verify error response
		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", response.Error)
		assert.Equal(t, "Invalid API key", response.Message)

		mockService.AssertExpectations(t)
	})

	t.Run("API key with ApiKey prefix", func(t *testing.T) {
		// Setup mock service
		mockService := &testutil.MockConfigService{}
		app := testutil.CreateTestApplication(uuid.New(), "Test App", "test-app", "valid-api-key")
		
		mockService.On("ValidateAPIKey", "valid-api-key").Return(app, nil)

		// Create middleware
		authMiddleware := NewAuthMiddleware(mockService)

		// Create test context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "ApiKey valid-api-key")

		// Execute middleware
		authMiddleware.APIKeyAuth()(c)

		assert.False(t, c.IsAborted())
		mockService.AssertExpectations(t)
	})
}

func TestAuthMiddleware_OptionalAPIKeyAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("request without API key passes through", func(t *testing.T) {
		// Setup mock service
		mockService := &testutil.MockConfigService{}

		// Create middleware
		authMiddleware := NewAuthMiddleware(mockService)

		// Create test context without API key
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		// Create a test handler to verify the middleware passes through
		var handlerCalled bool
		testHandler := func(c *gin.Context) {
			handlerCalled = true
			
			// Verify that no application is set in context
			_, exists := c.Get("application")
			assert.False(t, exists)
			
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}

		// Execute middleware and handler
		authMiddleware.OptionalAPIKeyAuth()(c)
		if !c.IsAborted() {
			testHandler(c)
		}

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("request with valid API key sets context", func(t *testing.T) {
		// Setup mock service
		mockService := &testutil.MockConfigService{}
		app := testutil.CreateTestApplication(uuid.New(), "Test App", "test-app", "valid-api-key")
		
		mockService.On("ValidateAPIKey", "valid-api-key").Return(app, nil)

		// Create middleware
		authMiddleware := NewAuthMiddleware(mockService)

		// Create test context with API key
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "valid-api-key")

		// Create a test handler to verify the middleware sets context
		var handlerCalled bool
		testHandler := func(c *gin.Context) {
			handlerCalled = true
			
			// Verify that application and api_key are set in context
			appFromContext, exists := c.Get("application")
			assert.True(t, exists)
			assert.Equal(t, app, appFromContext)
			
			apiKeyFromContext, exists := c.Get("api_key")
			assert.True(t, exists)
			assert.Equal(t, "valid-api-key", apiKeyFromContext)
			
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}

		// Execute middleware and handler
		authMiddleware.OptionalAPIKeyAuth()(c)
		if !c.IsAborted() {
			testHandler(c)
		}

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("request with invalid API key returns error", func(t *testing.T) {
		// Setup mock service
		mockService := &testutil.MockConfigService{}
		mockService.On("ValidateAPIKey", "invalid-api-key").Return(nil, assert.AnError)

		// Create middleware
		authMiddleware := NewAuthMiddleware(mockService)

		// Create test context with invalid API key
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-API-Key", "invalid-api-key")

		// Execute middleware
		authMiddleware.OptionalAPIKeyAuth()(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Verify error response
		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", response.Error)
		assert.Equal(t, "Invalid API key", response.Message)

		mockService.AssertExpectations(t)
	})
}
