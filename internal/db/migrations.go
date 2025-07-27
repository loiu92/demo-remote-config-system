package db

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version  string
	Filename string
	SQL      string
}

// MigrationRunner handles database migrations
type MigrationRunner struct {
	db            *DB
	migrationsDir string
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *DB, migrationsDir string) *MigrationRunner {
	return &MigrationRunner{
		db:            db,
		migrationsDir: migrationsDir,
	}
}

// CreateMigrationsTable creates the migrations tracking table
func (mr *MigrationRunner) CreateMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	
	_, err := mr.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	return nil
}

// GetAppliedMigrations returns a list of applied migration versions
func (mr *MigrationRunner) GetAppliedMigrations() (map[string]bool, error) {
	query := "SELECT version FROM schema_migrations"
	rows, err := mr.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}

	return applied, nil
}

// LoadMigrations loads all migration files from the migrations directory
func (mr *MigrationRunner) LoadMigrations() ([]Migration, error) {
	files, err := os.ReadDir(mr.migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Extract version from filename (e.g., "001_initial.sql" -> "001")
		version := strings.Split(file.Name(), "_")[0]
		
		// Read the SQL content
		sqlPath := filepath.Join(mr.migrationsDir, file.Name())
		sqlBytes, err := os.ReadFile(sqlPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Filename: file.Name(),
			SQL:      string(sqlBytes),
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// RunMigrations executes all pending migrations
func (mr *MigrationRunner) RunMigrations() error {
	// Create migrations table if it doesn't exist
	if err := mr.CreateMigrationsTable(); err != nil {
		return err
	}

	// Get applied migrations
	applied, err := mr.GetAppliedMigrations()
	if err != nil {
		return err
	}

	// Load all migrations
	migrations, err := mr.LoadMigrations()
	if err != nil {
		return err
	}

	// Run pending migrations
	for _, migration := range migrations {
		if applied[migration.Version] {
			log.Printf("Migration %s already applied, skipping", migration.Filename)
			continue
		}

		log.Printf("Running migration %s", migration.Filename)
		
		// Start transaction
		tx, err := mr.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction for migration %s: %w", migration.Filename, err)
		}

		// Execute migration SQL
		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", migration.Filename, err)
		}

		// Record migration as applied
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migration.Filename, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Filename, err)
		}

		log.Printf("Successfully applied migration %s", migration.Filename)
	}

	return nil
}

// GetMigrationStatus returns the status of all migrations
func (mr *MigrationRunner) GetMigrationStatus() ([]map[string]interface{}, error) {
	applied, err := mr.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	migrations, err := mr.LoadMigrations()
	if err != nil {
		return nil, err
	}

	var status []map[string]interface{}
	for _, migration := range migrations {
		status = append(status, map[string]interface{}{
			"version":  migration.Version,
			"filename": migration.Filename,
			"applied":  applied[migration.Version],
		})
	}

	return status, nil
}
