package db

import (
	"database/sql"
	"fmt"

	"remote-config-system/internal/models"

	"github.com/google/uuid"
)

// OrganizationRepository handles database operations for organizations
type OrganizationRepository struct {
	db *DB
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// GetBySlug retrieves an organization by its slug
func (r *OrganizationRepository) GetBySlug(slug string) (*models.Organization, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`

	var org models.Organization
	err := r.db.QueryRow(query, slug).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.CreatedAt,
		&org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("organization not found: %s", slug)
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &org, nil
}

// GetByID retrieves an organization by its ID
func (r *OrganizationRepository) GetByID(id uuid.UUID) (*models.Organization, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`

	var org models.Organization
	err := r.db.QueryRow(query, id).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.CreatedAt,
		&org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("organization not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &org, nil
}

// List retrieves all organizations with pagination
func (r *OrganizationRepository) List(params models.PaginationParams) ([]models.Organization, int, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM organizations"
	var totalCount int
	err := r.db.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get organizations count: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM organizations
		ORDER BY name
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, params.PageSize, params.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	var organizations []models.Organization
	for rows.Next() {
		var org models.Organization
		err := rows.Scan(
			&org.ID,
			&org.Name,
			&org.Slug,
			&org.CreatedAt,
			&org.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan organization: %w", err)
		}
		organizations = append(organizations, org)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating organizations: %w", err)
	}

	return organizations, totalCount, nil
}

// Create creates a new organization
func (r *OrganizationRepository) Create(org *models.Organization) error {
	query := `
		INSERT INTO organizations (id, name, slug)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at
	`

	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}

	err := r.db.QueryRow(query, org.ID, org.Name, org.Slug).Scan(
		&org.CreatedAt,
		&org.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	return nil
}

// Update updates an existing organization
func (r *OrganizationRepository) Update(org *models.Organization) error {
	query := `
		UPDATE organizations
		SET name = $2
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(query, org.ID, org.Name).Scan(&org.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("organization not found: %s", org.ID)
		}
		return fmt.Errorf("failed to update organization: %w", err)
	}

	return nil
}

// Delete deletes an organization
func (r *OrganizationRepository) Delete(id uuid.UUID) error {
	query := `DELETE FROM organizations WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("organization not found: %s", id)
	}

	return nil
}

// Update updates an existing organization
func (r *OrganizationRepository) Update(org *models.Organization) error {
	query := `
		UPDATE organizations
		SET name = $2, slug = $3
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(query, org.ID, org.Name, org.Slug).Scan(&org.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("organization not found: %s", org.ID)
		}
		return fmt.Errorf("failed to update organization: %w", err)
	}

	return nil
}

// Delete deletes an organization
func (r *OrganizationRepository) Delete(id uuid.UUID) error {
	query := "DELETE FROM organizations WHERE id = $1"

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("organization not found: %s", id)
	}

	return nil
}

// Exists checks if an organization exists by slug
func (r *OrganizationRepository) Exists(slug string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM organizations WHERE slug = $1)"

	var exists bool
	err := r.db.QueryRow(query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check organization existence: %w", err)
	}

	return exists, nil
}
