package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"remote-config-system/internal/models"
)

// Client represents a connected SSE client
type Client struct {
	ID           string
	Organization string
	Application  string
	Environment  string
	Channel      chan models.SSEMessage
	Context      context.Context
	Cancel       context.CancelFunc
	ConnectedAt  time.Time
	LastPing     time.Time
}

// SSEService manages Server-Sent Events connections and broadcasting
type SSEService struct {
	clients    map[string]*Client
	clientsMux sync.RWMutex
	
	// Channel for broadcasting events to all clients
	broadcast chan BroadcastMessage
	
	// Channel for registering new clients
	register chan *Client
	
	// Channel for unregistering clients
	unregister chan *Client
	
	// Statistics
	stats SSEStats
}

// BroadcastMessage represents a message to be broadcasted
type BroadcastMessage struct {
	Organization string
	Application  string
	Environment  string
	Message      models.SSEMessage
}

// SSEStats holds SSE service statistics
type SSEStats struct {
	TotalConnections    int64     `json:"total_connections"`
	ActiveConnections   int       `json:"active_connections"`
	MessagesSent        int64     `json:"messages_sent"`
	ConnectionsDropped  int64     `json:"connections_dropped"`
	LastActivity        time.Time `json:"last_activity"`
}

// NewSSEService creates a new SSE service
func NewSSEService() *SSEService {
	service := &SSEService{
		clients:    make(map[string]*Client),
		broadcast:  make(chan BroadcastMessage, 1000),
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		stats: SSEStats{
			LastActivity: time.Now(),
		},
	}

	// Start the service in a goroutine
	go service.run()

	// Start periodic cleanup
	go service.periodicCleanup()

	return service
}

// run handles the main event loop for the SSE service
func (s *SSEService) run() {
	log.Println("SSE service started")

	for {
		select {
		case client := <-s.register:
			s.registerClient(client)

		case client := <-s.unregister:
			s.unregisterClient(client)

		case message := <-s.broadcast:
			s.broadcastMessage(message)
		}
	}
}

// registerClient adds a new client to the service
func (s *SSEService) registerClient(client *Client) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	s.clients[client.ID] = client
	s.stats.TotalConnections++
	s.stats.ActiveConnections = len(s.clients)
	s.stats.LastActivity = time.Now()

	log.Printf("SSE client registered: %s (%s/%s/%s)", 
		client.ID, client.Organization, client.Application, client.Environment)

	// Send welcome message
	welcomeMsg := models.SSEMessage{
		Event: "connected",
		Data: map[string]interface{}{
			"message":      "Connected to configuration updates",
			"client_id":    client.ID,
			"connected_at": client.ConnectedAt,
		},
	}

	select {
	case client.Channel <- welcomeMsg:
		s.stats.MessagesSent++
	default:
		log.Printf("Failed to send welcome message to client %s", client.ID)
	}
}

// unregisterClient removes a client from the service
func (s *SSEService) unregisterClient(client *Client) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if _, exists := s.clients[client.ID]; exists {
		delete(s.clients, client.ID)
		close(client.Channel)
		client.Cancel()
		s.stats.ConnectionsDropped++
		s.stats.ActiveConnections = len(s.clients)
		s.stats.LastActivity = time.Now()

		log.Printf("SSE client unregistered: %s", client.ID)
	}
}

// broadcastMessage sends a message to all matching clients
func (s *SSEService) broadcastMessage(message BroadcastMessage) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	sentCount := 0
	for _, client := range s.clients {
		// Check if client should receive this message
		if s.shouldReceiveMessage(client, message) {
			select {
			case client.Channel <- message.Message:
				sentCount++
				s.stats.MessagesSent++
			default:
				// Client channel is full, remove the client
				log.Printf("Client %s channel full, removing", client.ID)
				go func(c *Client) {
					s.unregister <- c
				}(client)
			}
		}
	}

	if sentCount > 0 {
		s.stats.LastActivity = time.Now()
		log.Printf("Broadcasted message to %d clients for %s/%s/%s", 
			sentCount, message.Organization, message.Application, message.Environment)
	}
}

// shouldReceiveMessage determines if a client should receive a specific message
func (s *SSEService) shouldReceiveMessage(client *Client, message BroadcastMessage) bool {
	// Match organization, application, and environment
	return client.Organization == message.Organization &&
		   client.Application == message.Application &&
		   client.Environment == message.Environment
}

// RegisterClient registers a new SSE client
func (s *SSEService) RegisterClient(client *Client) {
	s.register <- client
}

// UnregisterClient unregisters an SSE client
func (s *SSEService) UnregisterClient(client *Client) {
	s.unregister <- client
}

// BroadcastConfigUpdate broadcasts a configuration update to relevant clients
func (s *SSEService) BroadcastConfigUpdate(event models.ConfigUpdateEvent) {
	message := BroadcastMessage{
		Organization: event.Organization,
		Application:  event.Application,
		Environment:  event.Environment,
		Message: models.SSEMessage{
			Event: "config_update",
			Data:  event,
		},
	}

	select {
	case s.broadcast <- message:
		// Message queued successfully
	default:
		log.Printf("Broadcast channel full, dropping message for %s/%s/%s", 
			event.Organization, event.Application, event.Environment)
	}
}

// BroadcastCustomEvent broadcasts a custom event to clients
func (s *SSEService) BroadcastCustomEvent(org, app, env, eventType string, data interface{}) {
	message := BroadcastMessage{
		Organization: org,
		Application:  app,
		Environment:  env,
		Message: models.SSEMessage{
			Event: eventType,
			Data:  data,
		},
	}

	select {
	case s.broadcast <- message:
		// Message queued successfully
	default:
		log.Printf("Broadcast channel full, dropping custom event for %s/%s/%s", org, app, env)
	}
}

// GetStats returns current SSE service statistics
func (s *SSEService) GetStats() SSEStats {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	stats := s.stats
	stats.ActiveConnections = len(s.clients)
	return stats
}

// GetClients returns information about connected clients
func (s *SSEService) GetClients() []map[string]interface{} {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	clients := make([]map[string]interface{}, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, map[string]interface{}{
			"id":           client.ID,
			"organization": client.Organization,
			"application":  client.Application,
			"environment":  client.Environment,
			"connected_at": client.ConnectedAt,
			"last_ping":    client.LastPing,
		})
	}

	return clients
}

// periodicCleanup removes stale connections
func (s *SSEService) periodicCleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupStaleConnections()
	}
}

// cleanupStaleConnections removes connections that haven't been active
func (s *SSEService) cleanupStaleConnections() {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	now := time.Now()
	staleThreshold := 5 * time.Minute

	for id, client := range s.clients {
		if now.Sub(client.LastPing) > staleThreshold {
			log.Printf("Removing stale SSE client: %s", id)
			delete(s.clients, id)
			close(client.Channel)
			client.Cancel()
			s.stats.ConnectionsDropped++
		}
	}

	s.stats.ActiveConnections = len(s.clients)
}

// Ping updates the last ping time for a client
func (s *SSEService) Ping(clientID string) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if client, exists := s.clients[clientID]; exists {
		client.LastPing = time.Now()
	}
}
