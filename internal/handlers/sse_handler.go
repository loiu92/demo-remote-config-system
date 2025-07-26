package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"remote-config-system/internal/models"
	"remote-config-system/internal/services"
	"remote-config-system/internal/sse"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SSEHandler handles Server-Sent Events endpoints
type SSEHandler struct {
	configService *services.ConfigService
	sseService    *sse.SSEService
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(configService *services.ConfigService, sseService *sse.SSEService) *SSEHandler {
	return &SSEHandler{
		configService: configService,
		sseService:    sseService,
	}
}

// StreamConfigUpdates handles GET /api/events/:org/:app/:env
func (h *SSEHandler) StreamConfigUpdates(c *gin.Context) {
	orgSlug := c.Param("org")
	appSlug := c.Param("app")
	envSlug := c.Param("env")

	// Validate that the environment exists
	_, err := h.configService.GetEnvironment(orgSlug, appSlug, envSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:     "not_found",
			Message:   fmt.Sprintf("Environment %s/%s/%s not found", orgSlug, appSlug, envSlug),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Create client context
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Create SSE client
	client := &sse.Client{
		ID:           uuid.New().String(),
		Organization: orgSlug,
		Application:  appSlug,
		Environment:  envSlug,
		Channel:      make(chan models.SSEMessage, 100),
		Context:      ctx,
		Cancel:       cancel,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
	}

	// Register client with SSE service
	h.sseService.RegisterClient(client)
	defer h.sseService.UnregisterClient(client)

	// Send initial configuration
	if config, err := h.configService.GetConfiguration(orgSlug, appSlug, envSlug); err == nil {
		initialEvent := models.ConfigUpdateEvent{
			Organization: config.Organization,
			Application:  config.Application,
			Environment:  config.Environment,
			Version:      config.Version,
			Config:       config.Config,
			Action:       "initial",
			UpdatedAt:    config.UpdatedAt,
		}

		initialMsg := models.SSEMessage{
			Event: "initial_config",
			Data:  initialEvent,
		}

		select {
		case client.Channel <- initialMsg:
		default:
		}
	}

	// Handle client connection
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	for {
		select {
		case <-ctx.Done():
			return

		case message, ok := <-client.Channel:
			if !ok {
				return
			}

			// Update last ping
			h.sseService.Ping(client.ID)

			// Send SSE message
			if err := h.writeSSEMessage(c.Writer, message); err != nil {
				return
			}

		case <-time.After(30 * time.Second):
			// Send keep-alive ping
			pingMsg := models.SSEMessage{
				Event: "ping",
				Data: map[string]interface{}{
					"timestamp": time.Now(),
				},
			}

			if err := h.writeSSEMessage(c.Writer, pingMsg); err != nil {
				return
			}

			// Update last ping
			h.sseService.Ping(client.ID)
		}
	}
}

// StreamConfigUpdatesWithAPIKey handles GET /api/events/:env with API key authentication
func (h *SSEHandler) StreamConfigUpdatesWithAPIKey(c *gin.Context) {
	envSlug := c.Param("env")

	// Get API key from context (set by middleware)
	apiKey, exists := c.Get("api_key")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:     "unauthorized",
			Message:   "API key is required",
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	// Get application info from API key
	app, err := h.configService.ValidateAPIKey(apiKey.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:     "unauthorized",
			Message:   "Invalid API key",
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	// Validate that the environment exists
	_, err = h.configService.GetEnvironment(app.Organization.Slug, app.Slug, envSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:     "not_found",
			Message:   fmt.Sprintf("Environment %s not found", envSlug),
			Timestamp: time.Now(),
			Path:      c.Request.URL.Path,
		})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Create client context
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Create SSE client
	client := &sse.Client{
		ID:           uuid.New().String(),
		Organization: app.Organization.Slug,
		Application:  app.Slug,
		Environment:  envSlug,
		Channel:      make(chan models.SSEMessage, 100),
		Context:      ctx,
		Cancel:       cancel,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
	}

	// Register client with SSE service
	h.sseService.RegisterClient(client)
	defer h.sseService.UnregisterClient(client)

	// Send initial configuration
	if config, err := h.configService.GetConfigurationByAPIKey(apiKey.(string), envSlug); err == nil {
		initialEvent := models.ConfigUpdateEvent{
			Organization: config.Organization,
			Application:  config.Application,
			Environment:  config.Environment,
			Version:      config.Version,
			Config:       config.Config,
			Action:       "initial",
			UpdatedAt:    config.UpdatedAt,
		}

		initialMsg := models.SSEMessage{
			Event: "initial_config",
			Data:  initialEvent,
		}

		select {
		case client.Channel <- initialMsg:
		default:
		}
	}

	// Handle client connection
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	for {
		select {
		case <-ctx.Done():
			return

		case message, ok := <-client.Channel:
			if !ok {
				return
			}

			// Update last ping
			h.sseService.Ping(client.ID)

			// Send SSE message
			if err := h.writeSSEMessage(c.Writer, message); err != nil {
				return
			}

		case <-time.After(30 * time.Second):
			// Send keep-alive ping
			pingMsg := models.SSEMessage{
				Event: "ping",
				Data: map[string]interface{}{
					"timestamp": time.Now(),
				},
			}

			if err := h.writeSSEMessage(c.Writer, pingMsg); err != nil {
				return
			}

			// Update last ping
			h.sseService.Ping(client.ID)
		}
	}
}

// GetSSEStats handles GET /admin/sse/stats
func (h *SSEHandler) GetSSEStats(c *gin.Context) {
	stats := h.sseService.GetStats()
	clients := h.sseService.GetClients()

	response := map[string]interface{}{
		"stats":   stats,
		"clients": clients,
	}

	c.JSON(http.StatusOK, response)
}

// writeSSEMessage writes an SSE message to the response writer
func (h *SSEHandler) writeSSEMessage(w gin.ResponseWriter, message models.SSEMessage) error {
	// Convert data to JSON
	dataJSON, err := json.Marshal(message.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE data: %w", err)
	}

	// Write SSE format
	if _, err := fmt.Fprintf(w, "event: %s\n", message.Event); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "data: %s\n\n", string(dataJSON)); err != nil {
		return err
	}

	// Flush the response
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}
