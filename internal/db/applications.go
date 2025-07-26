package db

import (
	"database/sql"
	"fmt"

	"remote-config-system/internal/models"

	"github.com/google/uuid"
)

// ApplicationRepository handles database operations for applications
type ApplicationRepository struct {
	db *DB
}

// NewApplicationRepository creates a new application repository
func NewApplicationRepository(db *DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

// GetBySlug retrieves an application by organization slug and application slug
func (r *ApplicationRepository) GetBySlug(orgSlug, appSlug string) (*models.Application, error) {
	query := `
		SELECT a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM applications a
		JOIN organizations o ON a.org_id = o.id
		WHERE o.slug = $1 AND a.slug = $2
	`

	var app models.Application
	var org models.Organization

	err := r.db.QueryRow(query, orgSlug, appSlug).Scan(
		&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("application not found: %s/%s", orgSlug, appSlug)
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	app.Organization = &org
	return &app, nil
}

// GetByAPIKey retrieves an application by its API key
func (r *ApplicationRepository) GetByAPIKey(apiKey string) (*models.Application, error) {
	query := `
		SELECT a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM applications a
		JOIN organizations o ON a.org_id = o.id
		WHERE a.api_key = $1
	`

	var app models.Application
	var org models.Organization

	err := r.db.QueryRow(query, apiKey).Scan(
		&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("application not found for API key")
		}
		return nil, fmt.Errorf("failed to get application by API key: %w", err)
	}

	app.Organization = &org
	return &app, nil
}

// GetByID retrieves an application by its ID
func (r *ApplicationRepository) GetByID(id uuid.UUID) (*models.Application, error) {
	query := `
		SELECT a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM applications a
		JOIN organizations o ON a.org_id = o.id
		WHERE a.id = $1
	`

	var app models.Application
	var org models.Organization

	err := r.db.QueryRow(query, id).Scan(
		&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("application not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	app.Organization = &org
	return &app, nil
}

// ListByOrganization retrieves all applications for an organization
func (r *ApplicationRepository) ListByOrganization(orgID uuid.UUID, params models.PaginationParams) ([]models.Application, int, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM applications WHERE org_id = $1"
	var totalCount int
	err := r.db.QueryRow(countQuery, orgID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get applications count: %w", err)
	}

	// Get paginated results
	query := `
		SELECT a.id, a.org_id, a.name, a.slug, a.api_key, a.created_at, a.updated_at,
		       o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM applications a
		JOIN organizations o ON a.org_id = o.id
		WHERE a.org_id = $1
		ORDER BY a.name
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, orgID, params.PageSize, params.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list applications: %w", err)
	}
	defer rows.Close()

	var applications []models.Application
	for rows.Next() {
		var app models.Application
		var org models.Organization

		err := rows.Scan(
			&app.ID, &app.OrgID, &app.Name, &app.Slug, &app.APIKey, &app.CreatedAt, &app.UpdatedAt,
			&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan application: %w", err)
		}

		app.Organization = &org
		applications = append(applications, app)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating applications: %w", err)
	}

	return applications, totalCount, nil
}

// Create creates a new application
func (r *ApplicationRepository) Create(app *models.Application) error {
	query := `
		INSERT INTO applications (id, org_id, name, slug, api_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`

	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}

	err := r.db.QueryRow(query, app.ID, app.OrgID, app.Name, app.Slug, app.APIKey).Scan(
		&app.CreatedAt,
		&app.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	return nil
}

// Update updates an existing application
func (r *ApplicationRepository) Update(app *models.Application) error {
	query := `
		UPDATE applications
		SET name = $2, slug = $3, api_key = $4
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(query, app.ID, app.Name, app.Slug, app.APIKey).Scan(&app.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("application not found: %s", app.ID)
		}
		return fmt.Errorf("failed to update application: %w", err)
	}

	return nil
}

// Delete deletes an application
func (r *ApplicationRepository) Delete(id uuid.UUID) error {
	query := "DELETE FROM applications WHERE id = $1"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("application not found: %s", id)
	}

	return nil
}

// Exists checks if an application exists by organization and slug
func (r *ApplicationRepository) Exists(orgID uuid.UUID, slug string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM applications WHERE org_id = $1 AND slug = $2)"

	var exists bool
	err := r.db.QueryRow(query, orgID, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check application existence: %w", err)
	}

	return exists, nil
}
