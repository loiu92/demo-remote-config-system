package sse

import (
	"context"
	"testing"
	"time"

	"remote-config-system/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSEService_NewSSEService(t *testing.T) {
	service := NewSSEService()
	
	assert.NotNil(t, service)
	assert.NotNil(t, service.clients)
	assert.NotNil(t, service.broadcast)
	assert.NotNil(t, service.register)
	assert.NotNil(t, service.unregister)
}

func TestSSEService_RegisterClient(t *testing.T) {
	service := NewSSEService()
	
	// Create a test client
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	client := &Client{
		ID:           uuid.New().String(),
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "prod",
		Channel:      make(chan models.SSEMessage, 10),
		Context:      ctx,
		Cancel:       cancel,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
	}

	// Register client
	service.RegisterClient(client)

	// Give some time for the goroutine to process
	time.Sleep(100 * time.Millisecond)

	// Check that client is registered
	stats := service.GetStats()
	assert.Equal(t, 1, stats.ActiveConnections)
}

func TestSSEService_UnregisterClient(t *testing.T) {
	service := NewSSEService()
	
	// Create and register a test client
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	client := &Client{
		ID:           uuid.New().String(),
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "prod",
		Channel:      make(chan models.SSEMessage, 10),
		Context:      ctx,
		Cancel:       cancel,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
	}

	service.RegisterClient(client)
	time.Sleep(100 * time.Millisecond)

	// Verify client is registered
	stats := service.GetStats()
	assert.Equal(t, 1, stats.ActiveConnections)

	// Unregister client
	service.UnregisterClient(client)
	time.Sleep(100 * time.Millisecond)

	// Verify client is unregistered
	stats = service.GetStats()
	assert.Equal(t, 0, stats.ActiveConnections)
}

func TestSSEService_BroadcastConfigUpdate(t *testing.T) {
	service := NewSSEService()
	
	// Create test clients
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	
	client1 := &Client{
		ID:           uuid.New().String(),
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "prod",
		Channel:      make(chan models.SSEMessage, 10),
		Context:      ctx1,
		Cancel:       cancel1,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
	}
	
	client2 := &Client{
		ID:           uuid.New().String(),
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "staging", // Different environment
		Channel:      make(chan models.SSEMessage, 10),
		Context:      ctx2,
		Cancel:       cancel2,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
	}

	// Register clients
	service.RegisterClient(client1)
	service.RegisterClient(client2)
	time.Sleep(100 * time.Millisecond)

	// Consume welcome messages
	<-client1.Channel // consume welcome message
	<-client2.Channel // consume welcome message

	// Create test config response
	configResponse := &models.ConfigResponse{
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "prod",
		Version:      2,
		Config:       []byte(`{"key": "value"}`),
		UpdatedAt:    time.Now(),
	}

	// Broadcast config update
	updateEvent := models.ConfigUpdateEvent{
		Organization: configResponse.Organization,
		Application:  configResponse.Application,
		Environment:  configResponse.Environment,
		Version:      configResponse.Version,
		Config:       configResponse.Config,
		Action:       "update",
		UpdatedAt:    configResponse.UpdatedAt,
	}
	service.BroadcastConfigUpdate(updateEvent)
	time.Sleep(100 * time.Millisecond)

	// Check that only client1 (prod environment) received the message
	select {
	case msg := <-client1.Channel:
		assert.Equal(t, "config_update", msg.Event)
		
		// Verify the data structure
		data, ok := msg.Data.(models.ConfigUpdateEvent)
		require.True(t, ok)
		assert.Equal(t, "test-org", data.Organization)
		assert.Equal(t, "test-app", data.Application)
		assert.Equal(t, "prod", data.Environment)
		assert.Equal(t, 2, data.Version)
		assert.Equal(t, "update", data.Action)
	default:
		t.Fatal("Client1 should have received a message")
	}

	// Check that client2 (staging environment) did not receive the message
	select {
	case <-client2.Channel:
		t.Fatal("Client2 should not have received a message")
	default:
		// Expected - no message for different environment
	}
}

func TestSSEService_BroadcastCustomEvent(t *testing.T) {
	service := NewSSEService()
	
	// Create test client
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	client := &Client{
		ID:           uuid.New().String(),
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "prod",
		Channel:      make(chan models.SSEMessage, 10),
		Context:      ctx,
		Cancel:       cancel,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
	}

	// Register client
	service.RegisterClient(client)
	time.Sleep(100 * time.Millisecond)

	// Consume welcome message
	<-client.Channel

	// Broadcast custom event
	customData := map[string]interface{}{
		"message": "System maintenance scheduled",
		"time":    "2024-01-01T00:00:00Z",
	}
	
	service.BroadcastCustomEvent("test-org", "test-app", "prod", "maintenance", customData)
	time.Sleep(100 * time.Millisecond)

	// Check that client received the custom event
	select {
	case msg := <-client.Channel:
		assert.Equal(t, "maintenance", msg.Event)
		assert.Equal(t, customData, msg.Data)
	default:
		t.Fatal("Client should have received a custom event")
	}
}

func TestSSEService_Ping(t *testing.T) {
	service := NewSSEService()
	
	// Create test client
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	clientID := uuid.New().String()
	client := &Client{
		ID:           clientID,
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "prod",
		Channel:      make(chan models.SSEMessage, 10),
		Context:      ctx,
		Cancel:       cancel,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now().Add(-time.Minute), // Set old ping time
	}

	// Register client
	service.RegisterClient(client)
	time.Sleep(100 * time.Millisecond)

	// Get initial ping time
	initialPingTime := client.LastPing

	// Ping the client
	service.Ping(clientID)
	time.Sleep(100 * time.Millisecond)

	// Verify ping time was updated
	assert.True(t, client.LastPing.After(initialPingTime))
}

func TestSSEService_GetStats(t *testing.T) {
	service := NewSSEService()
	
	// Initial stats should be empty
	stats := service.GetStats()
	assert.Equal(t, 0, stats.ActiveConnections)
	assert.Equal(t, int64(0), stats.TotalConnections)
	assert.Equal(t, int64(0), stats.MessagesSent)

	// Create and register test clients
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	
	client1 := &Client{
		ID:           uuid.New().String(),
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "prod",
		Channel:      make(chan models.SSEMessage, 10),
		Context:      ctx1,
		Cancel:       cancel1,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
	}
	
	client2 := &Client{
		ID:           uuid.New().String(),
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "staging",
		Channel:      make(chan models.SSEMessage, 10),
		Context:      ctx2,
		Cancel:       cancel2,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
	}

	service.RegisterClient(client1)
	service.RegisterClient(client2)
	time.Sleep(100 * time.Millisecond)

	// Check stats after registering clients
	stats = service.GetStats()
	assert.Equal(t, 2, stats.ActiveConnections)
	assert.Equal(t, int64(2), stats.TotalConnections)

	// Send a broadcast message to increment message count
	configResponse := &models.ConfigResponse{
		Organization: "test-org",
		Application:  "test-app",
		Environment:  "prod",
		Version:      1,
		Config:       []byte(`{}`),
		UpdatedAt:    time.Now(),
	}
	
	updateEvent := models.ConfigUpdateEvent{
		Organization: configResponse.Organization,
		Application:  configResponse.Application,
		Environment:  configResponse.Environment,
		Version:      configResponse.Version,
		Config:       configResponse.Config,
		Action:       "update",
		UpdatedAt:    configResponse.UpdatedAt,
	}
	service.BroadcastConfigUpdate(updateEvent)
	time.Sleep(100 * time.Millisecond)

	// Check that message count increased
	stats = service.GetStats()
	assert.True(t, stats.MessagesSent > 0)
	assert.True(t, !stats.LastActivity.IsZero())
}
