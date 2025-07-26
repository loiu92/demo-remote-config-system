package db

import (
	"database/sql"
	"fmt"

	"remote-config-system/internal/models"

	"github.com/google/uuid"
)

// ConfigVersionRepository handles database operations for configuration versions
type ConfigVersionRepository struct {
	db *DB
}

// NewConfigVersionRepository creates a new config version repository
func NewConfigVersionRepository(db *DB) *ConfigVersionRepository {
	return &ConfigVersionRepository{db: db}
}

// GetActiveByEnvironment retrieves the active configuration for an environment
func (r *ConfigVersionRepository) GetActiveByEnvironment(envID uuid.UUID) (*models.ConfigVersion, error) {
	query := `
		SELECT cv.id, cv.env_id, cv.version, cv.config_json, cv.is_active, cv.created_at, cv.created_by,
		       e.id, e.app_id, e.name, e.slug, e.created_at, e.updated_at,
		       a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM config_versions cv
		JOIN environments e ON cv.env_id = e.id
		JOIN applications a ON e.app_id = a.id
		JOIN organizations o ON a.org_id = o.id
		WHERE cv.env_id = $1 AND cv.is_active = TRUE
	`

	var cv models.ConfigVersion
	var env models.Environment
	var app models.Application
	var org models.Organization

	err := r.db.QueryRow(query, envID).Scan(
		&cv.ID, &cv.EnvID, &cv.Version, &cv.ConfigJSON, &cv.IsActive, &cv.CreatedAt, &cv.CreatedBy,
		&env.ID, &env.AppID, &env.Name, &env.Slug, &env.CreatedAt, &env.UpdatedAt,
		&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active configuration found for environment: %s", envID)
		}
		return nil, fmt.Errorf("failed to get active configuration: %w", err)
	}

	app.Organization = &org
	env.Application = &app
	cv.Environment = &env
	return &cv, nil
}

// GetByVersion retrieves a specific version of configuration for an environment
func (r *ConfigVersionRepository) GetByVersion(envID uuid.UUID, version int) (*models.ConfigVersion, error) {
	query := `
		SELECT cv.id, cv.env_id, cv.version, cv.config_json, cv.is_active, cv.created_at, cv.created_by,
		       e.id, e.app_id, e.name, e.slug, e.created_at, e.updated_at,
		       a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM config_versions cv
		JOIN environments e ON cv.env_id = e.id
		JOIN applications a ON e.app_id = a.id
		JOIN organizations o ON a.org_id = o.id
		WHERE cv.env_id = $1 AND cv.version = $2
	`

	var cv models.ConfigVersion
	var env models.Environment
	var app models.Application
	var org models.Organization

	err := r.db.QueryRow(query, envID, version).Scan(
		&cv.ID, &cv.EnvID, &cv.Version, &cv.ConfigJSON, &cv.IsActive, &cv.CreatedAt, &cv.CreatedBy,
		&env.ID, &env.AppID, &env.Name, &env.Slug, &env.CreatedAt, &env.UpdatedAt,
		&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("configuration version not found: env=%s, version=%d", envID, version)
		}
		return nil, fmt.Errorf("failed to get configuration version: %w", err)
	}

	app.Organization = &org
	env.Application = &app
	cv.Environment = &env
	return &cv, nil
}

// ListByEnvironment retrieves all configuration versions for an environment
func (r *ConfigVersionRepository) ListByEnvironment(envID uuid.UUID, params models.PaginationParams) ([]models.ConfigVersion, int, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM config_versions WHERE env_id = $1"
	var totalCount int
	err := r.db.QueryRow(countQuery, envID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get config versions count: %w", err)
	}

	// Get paginated results
	query := `
		SELECT cv.id, cv.env_id, cv.version, cv.config_json, cv.is_active, cv.created_at, cv.created_by
		FROM config_versions cv
		WHERE cv.env_id = $1
		ORDER BY cv.version DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, envID, params.PageSize, params.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list config versions: %w", err)
	}
	defer rows.Close()

	var versions []models.ConfigVersion
	for rows.Next() {
		var cv models.ConfigVersion
		err := rows.Scan(
			&cv.ID, &cv.EnvID, &cv.Version, &cv.ConfigJSON, &cv.IsActive, &cv.CreatedAt, &cv.CreatedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan config version: %w", err)
		}
		versions = append(versions, cv)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating config versions: %w", err)
	}

	return versions, totalCount, nil
}

// GetNextVersion returns the next version number for an environment
func (r *ConfigVersionRepository) GetNextVersion(envID uuid.UUID) (int, error) {
	query := "SELECT COALESCE(MAX(version), 0) + 1 FROM config_versions WHERE env_id = $1"

	var nextVersion int
	err := r.db.QueryRow(query, envID).Scan(&nextVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to get next version: %w", err)
	}

	return nextVersion, nil
}

// Create creates a new configuration version
func (r *ConfigVersionRepository) Create(cv *models.ConfigVersion) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Get next version if not set
	if cv.Version == 0 {
		nextVersion, err := r.GetNextVersion(cv.EnvID)
		if err != nil {
			return err
		}
		cv.Version = nextVersion
	}

	// Create the new version
	query := `
		INSERT INTO config_versions (id, env_id, version, config_json, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`

	if cv.ID == uuid.Nil {
		cv.ID = uuid.New()
	}

	err = tx.QueryRow(query, cv.ID, cv.EnvID, cv.Version, cv.ConfigJSON, cv.IsActive, cv.CreatedBy).Scan(&cv.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create config version: %w", err)
	}

	return tx.Commit()
}

// SetActive sets a configuration version as active (deactivating others)
func (r *ConfigVersionRepository) SetActive(envID uuid.UUID, version int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Deactivate all versions for this environment
	_, err = tx.Exec("UPDATE config_versions SET is_active = FALSE WHERE env_id = $1", envID)
	if err != nil {
		return fmt.Errorf("failed to deactivate config versions: %w", err)
	}

	// Activate the specified version
	result, err := tx.Exec("UPDATE config_versions SET is_active = TRUE WHERE env_id = $1 AND version = $2", envID, version)
	if err != nil {
		return fmt.Errorf("failed to activate config version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("configuration version not found: env=%s, version=%d", envID, version)
	}

	return tx.Commit()
}

// Delete deletes a configuration version (only if not active)
func (r *ConfigVersionRepository) Delete(envID uuid.UUID, version int) error {
	query := "DELETE FROM config_versions WHERE env_id = $1 AND version = $2 AND is_active = FALSE"

	result, err := r.db.Exec(query, envID, version)
	if err != nil {
		return fmt.Errorf("failed to delete config version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("configuration version not found or is active: env=%s, version=%d", envID, version)
	}

	return nil
}
