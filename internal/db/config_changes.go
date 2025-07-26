package db

import (
	"database/sql"
	"fmt"

	"remote-config-system/internal/models"

	"github.com/google/uuid"
)

// ConfigChangeRepository handles database operations for configuration changes
type ConfigChangeRepository struct {
	db *DB
}

// NewConfigChangeRepository creates a new config change repository
func NewConfigChangeRepository(db *DB) *ConfigChangeRepository {
	return &ConfigChangeRepository{db: db}
}

// ListByEnvironment retrieves all configuration changes for an environment
func (r *ConfigChangeRepository) ListByEnvironment(envID uuid.UUID, params models.PaginationParams) ([]models.ConfigChange, int, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM config_changes WHERE env_id = $1"
	var totalCount int
	err := r.db.QueryRow(countQuery, envID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get config changes count: %w", err)
	}

	// Get paginated results
	query := `
		SELECT cc.id, cc.env_id, cc.version_from, cc.version_to, cc.action, cc.created_at, cc.created_by,
		       e.id, e.app_id, e.name, e.slug, e.created_at, e.updated_at,
		       a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM config_changes cc
		JOIN environments e ON cc.env_id = e.id
		JOIN applications a ON e.app_id = a.id
		JOIN organizations o ON a.org_id = o.id
		WHERE cc.env_id = $1
		ORDER BY cc.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, envID, params.PageSize, params.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list config changes: %w", err)
	}
	defer rows.Close()

	var changes []models.ConfigChange
	for rows.Next() {
		var cc models.ConfigChange
		var env models.Environment
		var app models.Application
		var org models.Organization

		err := rows.Scan(
			&cc.ID, &cc.EnvID, &cc.VersionFrom, &cc.VersionTo, &cc.Action, &cc.CreatedAt, &cc.CreatedBy,
			&env.ID, &env.AppID, &env.Name, &env.Slug, &env.CreatedAt, &env.UpdatedAt,
			&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
			&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan config change: %w", err)
		}

		app.Organization = &org
		env.Application = &app
		cc.Environment = &env
		changes = append(changes, cc)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating config changes: %w", err)
	}

	return changes, totalCount, nil
}

// ListRecent retrieves recent configuration changes across all environments
func (r *ConfigChangeRepository) ListRecent(limit int) ([]models.ConfigChange, error) {
	query := `
		SELECT cc.id, cc.env_id, cc.version_from, cc.version_to, cc.action, cc.created_at, cc.created_by,
		       e.id, e.app_id, e.name, e.slug, e.created_at, e.updated_at,
		       a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM config_changes cc
		JOIN environments e ON cc.env_id = e.id
		JOIN applications a ON e.app_id = a.id
		JOIN organizations o ON a.org_id = o.id
		ORDER BY cc.created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent config changes: %w", err)
	}
	defer rows.Close()

	var changes []models.ConfigChange
	for rows.Next() {
		var cc models.ConfigChange
		var env models.Environment
		var app models.Application
		var org models.Organization

		err := rows.Scan(
			&cc.ID, &cc.EnvID, &cc.VersionFrom, &cc.VersionTo, &cc.Action, &cc.CreatedAt, &cc.CreatedBy,
			&env.ID, &env.AppID, &env.Name, &env.Slug, &env.CreatedAt, &env.UpdatedAt,
			&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
			&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config change: %w", err)
		}

		app.Organization = &org
		env.Application = &app
		cc.Environment = &env
		changes = append(changes, cc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating config changes: %w", err)
	}

	return changes, nil
}

// Create creates a new configuration change log entry
func (r *ConfigChangeRepository) Create(cc *models.ConfigChange) error {
	query := `
		INSERT INTO config_changes (id, env_id, version_from, version_to, action, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`

	if cc.ID == uuid.Nil {
		cc.ID = uuid.New()
	}

	err := r.db.QueryRow(query, cc.ID, cc.EnvID, cc.VersionFrom, cc.VersionTo, cc.Action, cc.CreatedBy).Scan(&cc.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create config change: %w", err)
	}

	return nil
}

// GetByID retrieves a configuration change by its ID
func (r *ConfigChangeRepository) GetByID(id uuid.UUID) (*models.ConfigChange, error) {
	query := `
		SELECT cc.id, cc.env_id, cc.version_from, cc.version_to, cc.action, cc.created_at, cc.created_by,
		       e.id, e.app_id, e.name, e.slug, e.created_at, e.updated_at,
		       a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM config_changes cc
		JOIN environments e ON cc.env_id = e.id
		JOIN applications a ON e.app_id = a.id
		JOIN organizations o ON a.org_id = o.id
		WHERE cc.id = $1
	`

	var cc models.ConfigChange
	var env models.Environment
	var app models.Application
	var org models.Organization

	err := r.db.QueryRow(query, id).Scan(
		&cc.ID, &cc.EnvID, &cc.VersionFrom, &cc.VersionTo, &cc.Action, &cc.CreatedAt, &cc.CreatedBy,
		&env.ID, &env.AppID, &env.Name, &env.Slug, &env.CreatedAt, &env.UpdatedAt,
		&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("config change not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get config change: %w", err)
	}

	app.Organization = &org
	env.Application = &app
	cc.Environment = &env
	return &cc, nil
}

// Delete deletes a configuration change log entry
func (r *ConfigChangeRepository) Delete(id uuid.UUID) error {
	query := "DELETE FROM config_changes WHERE id = $1"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete config change: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("config change not found: %s", id)
	}

	return nil
}

// GetStats returns statistics about configuration changes
func (r *ConfigChangeRepository) GetStats() (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_changes,
			COUNT(CASE WHEN action = 'create' THEN 1 END) as creates,
			COUNT(CASE WHEN action = 'update' THEN 1 END) as updates,
			COUNT(CASE WHEN action = 'rollback' THEN 1 END) as rollbacks,
			COUNT(CASE WHEN created_at >= NOW() - INTERVAL '24 hours' THEN 1 END) as changes_last_24h,
			COUNT(CASE WHEN created_at >= NOW() - INTERVAL '7 days' THEN 1 END) as changes_last_7d
		FROM config_changes
	`

	var stats struct {
		TotalChanges    int `db:"total_changes"`
		Creates         int `db:"creates"`
		Updates         int `db:"updates"`
		Rollbacks       int `db:"rollbacks"`
		ChangesLast24h  int `db:"changes_last_24h"`
		ChangesLast7d   int `db:"changes_last_7d"`
	}

	err := r.db.QueryRow(query).Scan(
		&stats.TotalChanges,
		&stats.Creates,
		&stats.Updates,
		&stats.Rollbacks,
		&stats.ChangesLast24h,
		&stats.ChangesLast7d,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get config change stats: %w", err)
	}

	return map[string]interface{}{
		"total_changes":     stats.TotalChanges,
		"creates":           stats.Creates,
		"updates":           stats.Updates,
		"rollbacks":         stats.Rollbacks,
		"changes_last_24h":  stats.ChangesLast24h,
		"changes_last_7d":   stats.ChangesLast7d,
	}, nil
}
