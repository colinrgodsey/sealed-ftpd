package db

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

func TestInitDB(t *testing.T) {
	// Create a temporary file for the database
	tmpfile, err := os.CreateTemp("", "testdb-*.sqlite")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	dbPath := tmpfile.Name()
	tmpfile.Close()
	defer os.Remove(dbPath)

	// Initialize the database
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// Verify root directory exists
	checkRootExists(t, db)

	// Verify we can insert a file
	checkInsertFile(t, db)
}

func checkRootExists(t *testing.T, db *sql.DB) {
	var path string
	var isDir bool
	err := db.QueryRow("SELECT path, is_dir FROM files WHERE path = '/'").Scan(&path, &isDir)
	if err != nil {
		t.Fatalf("Failed to query root directory: %v", err)
	}

	if path != "/" {
		t.Errorf("Expected path '/', got '%s'", path)
	}
	if !isDir {
		t.Errorf("Expected is_dir to be true for root")
	}
}

func checkInsertFile(t *testing.T, db *sql.DB) {
	res, err := db.Exec(`
		INSERT INTO files (path, parent_path, name, is_dir, size, mod_time, content)
		VALUES ('/test.txt', '/', 'test.txt', 0, 12, ?, ?)
	`, time.Now(), []byte("hello world"))
	if err != nil {
		t.Fatalf("Failed to insert test file: %v", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}

	var content []byte
	err = db.QueryRow("SELECT content FROM files WHERE path = '/test.txt'").Scan(&content)
	if err != nil {
		t.Fatalf("Failed to query test file content: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("Expected content 'hello world', got '%s'", string(content))
	}
}
