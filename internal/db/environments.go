package db

import (
	"database/sql"
	"fmt"

	"remote-config-system/internal/models"

	"github.com/google/uuid"
)

// EnvironmentRepository handles database operations for environments
type EnvironmentRepository struct {
	db *DB
}

// NewEnvironmentRepository creates a new environment repository
func NewEnvironmentRepository(db *DB) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

// GetBySlug retrieves an environment by organization, application, and environment slugs
func (r *EnvironmentRepository) GetBySlug(orgSlug, appSlug, envSlug string) (*models.Environment, error) {
	query := `
		SELECT e.id, e.app_id, e.name, e.slug, e.created_at, e.updated_at,
		       a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM environments e
		JOIN applications a ON e.app_id = a.id
		JOIN organizations o ON a.org_id = o.id
		WHERE o.slug = $1 AND a.slug = $2 AND e.slug = $3
	`

	var env models.Environment
	var app models.Application
	var org models.Organization

	err := r.db.QueryRow(query, orgSlug, appSlug, envSlug).Scan(
		&env.ID, &env.AppID, &env.Name, &env.Slug, &env.CreatedAt, &env.UpdatedAt,
		&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("environment not found: %s/%s/%s", orgSlug, appSlug, envSlug)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	app.Organization = &org
	env.Application = &app
	return &env, nil
}

// GetByID retrieves an environment by its ID
func (r *EnvironmentRepository) GetByID(id uuid.UUID) (*models.Environment, error) {
	query := `
		SELECT e.id, e.app_id, e.name, e.slug, e.created_at, e.updated_at,
		       a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM environments e
		JOIN applications a ON e.app_id = a.id
		JOIN organizations o ON a.org_id = o.id
		WHERE e.id = $1
	`

	var env models.Environment
	var app models.Application
	var org models.Organization

	err := r.db.QueryRow(query, id).Scan(
		&env.ID, &env.AppID, &env.Name, &env.Slug, &env.CreatedAt, &env.UpdatedAt,
		&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("environment not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	app.Organization = &org
	env.Application = &app
	return &env, nil
}

// ListByApplication retrieves all environments for an application
func (r *EnvironmentRepository) ListByApplication(appID uuid.UUID, params models.PaginationParams) ([]models.Environment, int, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM environments WHERE app_id = $1"
	var totalCount int
	err := r.db.QueryRow(countQuery, appID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get environments count: %w", err)
	}

	// Get paginated results
	query := `
		SELECT e.id, e.app_id, e.name, e.slug, e.created_at, e.updated_at,
		       a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM environments e
		JOIN applications a ON e.app_id = a.id
		JOIN organizations o ON a.org_id = o.id
		WHERE e.app_id = $1
		ORDER BY e.name
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, appID, params.PageSize, params.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list environments: %w", err)
	}
	defer rows.Close()

	var environments []models.Environment
	for rows.Next() {
		var env models.Environment
		var app models.Application
		var org models.Organization

		err := rows.Scan(
			&env.ID, &env.AppID, &env.Name, &env.Slug, &env.CreatedAt, &env.UpdatedAt,
			&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
			&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan environment: %w", err)
		}

		app.Organization = &org
		env.Application = &app
		environments = append(environments, env)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating environments: %w", err)
	}

	return environments, totalCount, nil
}

// Create creates a new environment
func (r *EnvironmentRepository) Create(env *models.Environment) error {
	query := `
		INSERT INTO environments (id, app_id, name, slug)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`

	if env.ID == uuid.Nil {
		env.ID = uuid.New()
	}

	err := r.db.QueryRow(query, env.ID, env.AppID, env.Name, env.Slug).Scan(
		&env.CreatedAt,
		&env.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create environment: %w", err)
	}

	return nil
}

// Update updates an existing environment
func (r *EnvironmentRepository) Update(env *models.Environment) error {
	query := `
		UPDATE environments
		SET name = $2, slug = $3
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(query, env.ID, env.Name, env.Slug).Scan(&env.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("environment not found: %s", env.ID)
		}
		return fmt.Errorf("failed to update environment: %w", err)
	}

	return nil
}

// Delete deletes an environment
func (r *EnvironmentRepository) Delete(id uuid.UUID) error {
	query := "DELETE FROM environments WHERE id = $1"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("environment not found: %s", id)
	}

	return nil
}

// Exists checks if an environment exists by application and slug
func (r *EnvironmentRepository) Exists(appID uuid.UUID, slug string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM environments WHERE app_id = $1 AND slug = $2)"

	var exists bool
	err := r.db.QueryRow(query, appID, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check environment existence: %w", err)
	}

	return exists, nil
}
