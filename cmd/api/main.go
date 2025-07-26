package main

import (
	"log"
	"os"

	"remote-config-system/internal/db"
	"remote-config-system/internal/handlers"
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

	// Initialize repositories
	repos := db.NewRepositories(database)

	// Initialize services
	configService := services.NewConfigService(repos)

	// Initialize handlers
	configHandler := handlers.NewConfigHandler(configService)

	// Initialize Gin router
	r := gin.Default()

	// Add middleware (commented out for debugging)
	// r.Use(middleware.CORS())
	// r.Use(middleware.RequestLogger())
	// r.Use(middleware.ErrorHandler())
	// r.Use(middleware.RateLimiter())

	// Health check endpoint
	r.GET("/health", configHandler.HealthCheck)

	// Configuration endpoint (no auth for now)
	r.GET("/config/:org/:app/:env", configHandler.GetConfig)

	log.Printf("Starting server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
