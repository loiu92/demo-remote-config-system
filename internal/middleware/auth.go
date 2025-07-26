package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"remote-config-system/internal/models"
	"remote-config-system/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware handles API key authentication
type AuthMiddleware struct {
	configService *services.ConfigService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(configService *services.ConfigService) *AuthMiddleware {
	return &AuthMiddleware{
		configService: configService,
	}
}

// APIKeyAuth middleware validates API key from header or query parameter
func (m *AuthMiddleware) APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get API key from Authorization header
		authHeader := c.GetHeader("Authorization")
		var apiKey string

		if authHeader != "" {
			// Support both "Bearer <key>" and "ApiKey <key>" formats
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			} else if strings.HasPrefix(authHeader, "ApiKey ") {
				apiKey = strings.TrimPrefix(authHeader, "ApiKey ")
			} else {
				apiKey = authHeader
			}
		}

		// Fallback to query parameter
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		// Fallback to X-API-Key header
		if apiKey == "" {
			apiKey = c.GetHeader("X-API-Key")
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:     "unauthorized",
				Message:   "API key is required",
				Timestamp: time.Now(),
				Path:      c.Request.URL.Path,
			})
			c.Abort()
			return
		}

		// Validate the API key
		app, err := m.configService.ValidateAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:     "unauthorized",
				Message:   "Invalid API key",
				Timestamp: time.Now(),
				Path:      c.Request.URL.Path,
			})
			c.Abort()
			return
		}

		// Store application info in context
		c.Set("application", app)
		c.Set("api_key", apiKey)

		c.Next()
	}
}

// OptionalAPIKeyAuth middleware validates API key if present but doesn't require it
func (m *AuthMiddleware) OptionalAPIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get API key from various sources
		authHeader := c.GetHeader("Authorization")
		var apiKey string

		if authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			} else if strings.HasPrefix(authHeader, "ApiKey ") {
				apiKey = strings.TrimPrefix(authHeader, "ApiKey ")
			} else {
				apiKey = authHeader
			}
		}

		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			apiKey = c.GetHeader("X-API-Key")
		}

		// If API key is provided, validate it
		if apiKey != "" {
			app, err := m.configService.ValidateAPIKey(apiKey)
			if err != nil {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:     "unauthorized",
					Message:   "Invalid API key",
					Timestamp: time.Now(),
					Path:      c.Request.URL.Path,
				})
				c.Abort()
				return
			}

			// Store application info in context
			c.Set("application", app)
			c.Set("api_key", apiKey)
		}

		c.Next()
	}
}

// CORS middleware handles Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow all origins for development (in production, you'd want to be more restrictive)
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestLogger middleware logs HTTP requests
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

// ErrorHandler middleware handles panics and errors
func ErrorHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:     "internal_server_error",
			Message:   "An internal server error occurred",
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

// RateLimiter middleware (basic implementation)
func RateLimiter() gin.HandlerFunc {
	// This is a simple in-memory rate limiter
	// In production, you'd want to use Redis or a more sophisticated solution
	return func(c *gin.Context) {
		// For now, just pass through
		// TODO: Implement proper rate limiting
		c.Next()
	}
}
