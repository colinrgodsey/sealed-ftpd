package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the database schema and ensures the root directory exists.
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := CreateSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// CreateSchema creates the database schema.
func CreateSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT UNIQUE NOT NULL,
		parent_path TEXT NOT NULL,
		name TEXT NOT NULL,
		is_dir BOOLEAN NOT NULL DEFAULT 0,
		size INTEGER NOT NULL DEFAULT 0,
		mod_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		content BLOB
	);
	
	CREATE INDEX IF NOT EXISTS idx_parent_path ON files(parent_path);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Ensure root directory exists
	return EnsureRoot(db)
}

// EnsureRoot ensures the root directory '/' exists in the database.
func EnsureRoot(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM files WHERE path = '/'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for root directory: %w", err)
	}

	if count == 0 {
		_, err = db.Exec(`
			INSERT INTO files (path, parent_path, name, is_dir, size, mod_time)
			VALUES ('/', '', '/', 1, 0, ?)
		`, time.Now().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("failed to insert root directory: %w", err)
		}
	}

	return nil
}
