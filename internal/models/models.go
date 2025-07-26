package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Organization represents an organization in the system
type Organization struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Application represents an application within an organization
type Application struct {
	ID        uuid.UUID `json:"id" db:"id"`
	OrgID     uuid.UUID `json:"org_id" db:"org_id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	APIKey    string    `json:"api_key" db:"api_key"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Relationships
	Organization *Organization `json:"organization,omitempty"`
}

// Environment represents an environment for an application
type Environment struct {
	ID        uuid.UUID `json:"id" db:"id"`
	AppID     uuid.UUID `json:"app_id" db:"app_id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Relationships
	Application *Application `json:"application,omitempty"`
}

// ConfigVersion represents a version of configuration for an environment
type ConfigVersion struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	EnvID      uuid.UUID       `json:"env_id" db:"env_id"`
	Version    int             `json:"version" db:"version"`
	ConfigJSON json.RawMessage `json:"config_json" db:"config_json"`
	IsActive   bool            `json:"is_active" db:"is_active"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	CreatedBy  *string         `json:"created_by" db:"created_by"`

	// Relationships
	Environment *Environment `json:"environment,omitempty"`
}

// ConfigChange represents a change log entry for configuration changes
type ConfigChange struct {
	ID          uuid.UUID `json:"id" db:"id"`
	EnvID       uuid.UUID `json:"env_id" db:"env_id"`
	VersionFrom *int      `json:"version_from" db:"version_from"`
	VersionTo   int       `json:"version_to" db:"version_to"`
	Action      string    `json:"action" db:"action"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CreatedBy   *string   `json:"created_by" db:"created_by"`

	// Relationships
	Environment *Environment `json:"environment,omitempty"`
}

// ConfigResponse represents the response structure for configuration API
type ConfigResponse struct {
	Organization string          `json:"organization"`
	Application  string          `json:"application"`
	Environment  string          `json:"environment"`
	Version      int             `json:"version"`
	Config       json.RawMessage `json:"config"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CreateConfigRequest represents a request to create/update configuration
type CreateConfigRequest struct {
	Config    json.RawMessage `json:"config" binding:"required"`
	CreatedBy *string         `json:"created_by"`
}

// RollbackRequest represents a request to rollback configuration
type RollbackRequest struct {
	ToVersion int     `json:"to_version" binding:"required"`
	CreatedBy *string `json:"created_by"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path,omitempty"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"page_size" binding:"min=1,max=100"`
}

// DefaultPaginationParams returns default pagination parameters
func DefaultPaginationParams() PaginationParams {
	return PaginationParams{
		Page:     1,
		PageSize: 20,
	}
}

// Offset calculates the offset for database queries
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int         `json:"total_count"`
	TotalPages int         `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, page, pageSize, totalCount int) PaginatedResponse {
	totalPages := (totalCount + pageSize - 1) / pageSize
	return PaginatedResponse{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}

// SSEMessage represents a Server-Sent Event message
type SSEMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// ConfigUpdateEvent represents a configuration update event
type ConfigUpdateEvent struct {
	Organization string          `json:"organization"`
	Application  string          `json:"application"`
	Environment  string          `json:"environment"`
	Version      int             `json:"version"`
	Config       json.RawMessage `json:"config"`
	Action       string          `json:"action"` // "update", "rollback"
	UpdatedAt    time.Time       `json:"updated_at"`
}
