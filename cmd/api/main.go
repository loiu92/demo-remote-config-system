package main

import (
	"log"
	"os"

	"remote-config-system/internal/cache"
	"remote-config-system/internal/db"
	"remote-config-system/internal/handlers"
	"remote-config-system/internal/middleware"
	"remote-config-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Set default port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database connection
	dbConfig := db.NewConfig()
	log.Printf("Connecting to database with config: %+v", dbConfig)
	database, err := db.Connect(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()
	log.Println("Successfully connected to database")

	// Initialize Redis cache
	cacheConfig := cache.NewConfig()
	log.Printf("Connecting to Redis with config: %+v", cacheConfig)
	redisClient, err := cache.NewRedisClient(cacheConfig)
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		log.Println("Continuing without cache...")
		redisClient = nil
	} else {
		defer redisClient.Close()
		log.Println("Successfully connected to Redis")
	}

	// Initialize repositories
	repos := db.NewRepositories(database)

	// Initialize services
	configService := services.NewConfigService(repos, redisClient)

	// Initialize handlers
	configHandler := handlers.NewConfigHandler(configService)
	managementHandler := handlers.NewManagementHandler(configService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(configService)

	// Initialize Gin router
	r := gin.Default()

	// Add global middleware
	r.Use(middleware.CORS())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.RateLimiter())

	// Health check endpoint
	r.GET("/health", configHandler.HealthCheck)

	// Public configuration endpoints (no authentication required)
	publicAPI := r.Group("/config")
	{
		publicAPI.GET("/:org/:app/:env", configHandler.GetConfig)
	}

	// API endpoints with authentication
	apiV1 := r.Group("/api")
	apiV1.Use(authMiddleware.APIKeyAuth())
	{
		// Configuration endpoints for applications
		apiV1.GET("/config/:env", configHandler.GetConfigByAPIKey)
	}

	// Admin endpoints with optional authentication
	adminAPI := r.Group("/admin")
	adminAPI.Use(authMiddleware.OptionalAPIKeyAuth())
	{
		// Organization management
		adminAPI.GET("/orgs", managementHandler.ListOrganizations)
		adminAPI.POST("/orgs", managementHandler.CreateOrganization)

		orgs := adminAPI.Group("/orgs/:org")
		{
			orgs.GET("", managementHandler.GetOrganization)
			orgs.PUT("", managementHandler.UpdateOrganization)
			orgs.DELETE("", managementHandler.DeleteOrganization)

			// Application management
			orgs.GET("/apps", managementHandler.ListApplications)
			orgs.POST("/apps", managementHandler.CreateApplication)

			apps := orgs.Group("/apps/:app")
			{
				apps.GET("", managementHandler.GetApplication)
				apps.PUT("", managementHandler.UpdateApplication)
				apps.DELETE("", managementHandler.DeleteApplication)

				// Environment management
				apps.GET("/envs", managementHandler.ListEnvironments)
				apps.POST("/envs", managementHandler.CreateEnvironment)

				envs := apps.Group("/envs/:env")
				{
					envs.GET("", managementHandler.GetEnvironment)
					envs.PUT("", managementHandler.UpdateEnvironment)
					envs.DELETE("", managementHandler.DeleteEnvironment)

					// Configuration management
					envs.PUT("/config", configHandler.UpdateConfig)
					envs.GET("/history", configHandler.GetConfigHistory)
					envs.GET("/changes", configHandler.GetConfigChanges)
					envs.POST("/rollback", configHandler.RollbackConfig)
				}
			}
		}
	}

	log.Printf("Starting server on port %s", port)
	log.Println("Available endpoints:")
	log.Println("  GET  /health                                         - Health check")
	log.Println("  GET  /config/:org/:app/:env                          - Get config (public)")
	log.Println("  GET  /api/config/:env                                - Get config (API key required)")
	log.Println("")
	log.Println("Management API:")
	log.Println("  GET    /admin/orgs                                   - List organizations")
	log.Println("  POST   /admin/orgs                                   - Create organization")
	log.Println("  GET    /admin/orgs/:org                              - Get organization")
	log.Println("  PUT    /admin/orgs/:org                              - Update organization")
	log.Println("  DELETE /admin/orgs/:org                              - Delete organization")
	log.Println("  GET    /admin/orgs/:org/apps                         - List applications")
	log.Println("  POST   /admin/orgs/:org/apps                         - Create application")
	log.Println("  GET    /admin/orgs/:org/apps/:app                    - Get application")
	log.Println("  PUT    /admin/orgs/:org/apps/:app                    - Update application")
	log.Println("  DELETE /admin/orgs/:org/apps/:app                    - Delete application")
	log.Println("  GET    /admin/orgs/:org/apps/:app/envs               - List environments")
	log.Println("  POST   /admin/orgs/:org/apps/:app/envs               - Create environment")
	log.Println("  GET    /admin/orgs/:org/apps/:app/envs/:env          - Get environment")
	log.Println("  PUT    /admin/orgs/:org/apps/:app/envs/:env          - Update environment")
	log.Println("  DELETE /admin/orgs/:org/apps/:app/envs/:env          - Delete environment")
	log.Println("")
	log.Println("Configuration API:")
	log.Println("  PUT    /admin/orgs/:org/apps/:app/envs/:env/config   - Update config")
	log.Println("  GET    /admin/orgs/:org/apps/:app/envs/:env/history  - Get config history")
	log.Println("  GET    /admin/orgs/:org/apps/:app/envs/:env/changes  - Get config changes")
	log.Println("  POST   /admin/orgs/:org/apps/:app/envs/:env/rollback - Rollback config")

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
