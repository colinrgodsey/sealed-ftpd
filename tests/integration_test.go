package tests

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/textproto" // Added for FTP errors
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/colinrgodsey/sealed-ftpd/pkg/db"
	"github.com/colinrgodsey/sealed-ftpd/pkg/vfs"

	"log/slog"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/hashicorp/go-multierror" // Added for multierror handling
	"github.com/jlaffaye/ftp"
)

// checkDbFile directly queries the database for file info
func checkDbFile(t *testing.T, dbConn *sql.DB, path string, expectedSize int, expectedModTime time.Time) {
	var size int
	var modTimeStr string
	var content sql.NullString // Use NullString to check if content is NULL

	row := dbConn.QueryRow("SELECT size, mod_time, content FROM files WHERE path = ?", path)
	err := row.Scan(&size, &modTimeStr, &content)
	if err != nil {
		t.Errorf("checkDbFile: Failed to query file %s from DB: %v", path, err)
		return
	}

	if size != expectedSize {
		t.Errorf("checkDbFile: File size mismatch for %s. Expected %d, got %d", path, expectedSize, size)
	}

	// Parse the stored time from DB
	parsedModTime, err := time.Parse(time.RFC3339, modTimeStr)
	if err != nil {
		parsedModTime, err = time.Parse("2006-01-02 15:04:05", modTimeStr)
		if err != nil {
			t.Errorf("checkDbFile: Failed to parse mod_time '%s' for %s: %v", modTimeStr, path, err)
			return
		}
	}

	// Compare year, month, day, hour, minute, second, nanosecond
	// Ignore nanoseconds for comparison if not consistently stored
	if !parsedModTime.Truncate(time.Second).Equal(expectedModTime.Truncate(time.Second)) {
		t.Errorf("checkDbFile: ModTime mismatch for %s. Expected %s, got %s", path, expectedModTime.String(), parsedModTime.String())
	}
}

// setupServer starts the FTP server in a goroutine and returns its address and the DB connection
func setupServer(t *testing.T, dbPath string) (addr string, dbConn *sql.DB, cleanup func()) {
	// Initialize the database
	sqliteDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Find an available port dynamically for ListenAddr
	// We create and close a listener just to get a free port.
	tempListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := tempListener.Addr().(*net.TCPAddr).Port
	tempListener.Close() // Close the temporary listener

	listenAddr := fmt.Sprintf("127.0.0.1:%d", port)
	connectionTimeout := 5 * time.Second

	mainDriver := vfs.NewMainDriver(sqliteDB, 30000, 30009, listenAddr, connectionTimeout) // Pass listenAddr and timeout

	ftpServer := ftpserver.NewFtpServer(mainDriver)

	// Use slog for logging
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Use Debug level for tests
	}))
	ftpServer.Logger = slogLogger

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := ftpServer.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			t.Errorf("FTP server ListenAndServe failed: %v", err)
		}
	}()

	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)

	return listenAddr, sqliteDB, func() {
		ftpServer.Stop()
		wg.Wait() // Wait for the server goroutine to finish
		sqliteDB.Close()
		os.Remove(dbPath) // Clean up database file
	}
}

func TestIntegration(t *testing.T) {
	dbPath := t.TempDir() + "/test-integration.db"
	serverAddr, dbConn, cleanup := setupServer(t, dbPath)
	defer cleanup()

	// Connect to the FTP server
	c, err := ftp.Dial(serverAddr, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("FTP dial failed: %v", err)
	}
	defer c.Quit()

	// Login (no auth required)
	err = c.Login("anonymous", "password")
	if err != nil {
		t.Fatalf("FTP login failed: %v", err)
	}

	// Test PWD
	pwd, err := c.CurrentDir()
	if err != nil {
		t.Errorf("PWD failed: %v", err)
	}
	if pwd != "/" {
		t.Errorf("Expected PWD to be '/', got '%s'", pwd)
	}

	// Test MKD
	err = c.MakeDir("/testdir")
	if err != nil {
		t.Errorf("MKD failed: %v", err)
	}

	// Test CWD
	err = c.ChangeDir("/testdir")
	if err != nil {
		t.Errorf("CWD failed: %v", err)
	}
	pwd, err = c.CurrentDir()
	if err != nil {
		t.Errorf("PWD after CWD failed: %v", err)
	}
	if pwd != "/testdir" {
		t.Errorf("Expected PWD to be '/testdir', got '%s'", pwd)
	}
	err = c.ChangeDir("/") // Go back to root
	if err != nil {
		t.Errorf("CWD back to root failed: %v", err)
	}

	// Test STOR (upload)
	uploadContent := "Hello, FTP Mimic!"
	uploadTime := time.Now() // Capture time before upload
	err = c.Stor("upload.txt", strings.NewReader(uploadContent))
	if err != nil {
		t.Fatalf("STOR failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond) // Give server time to finalize
	// Verify content and size in DB
	checkDbFile(t, dbConn, "/upload.txt", len(uploadContent), uploadTime)

	// Test LIST/NLST
	entries, err := c.List("")
	if err != nil {
		t.Errorf("LIST failed: %v", err)
	}
	foundUpload := false
	foundTestdir := false
	for _, entry := range entries {
		if entry.Name == "upload.txt" && entry.Type == ftp.EntryTypeFile {
			foundUpload = true
		}
		if entry.Name == "testdir" && entry.Type == ftp.EntryTypeFolder {
			foundTestdir = true
		}
	}
	if !foundUpload {
		t.Error("upload.txt not found in LIST")
	}
	if !foundTestdir {
		t.Error("testdir not found in LIST")
	}

	// Test RETR (download)
	r, err := c.Retr("upload.txt")
	if err != nil {
		t.Fatalf("RETR failed: %v", err)
	}
	downloadedContent, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Reading downloaded content failed: %v", err)
	}
	r.Close() // Explicitly close to read the pending '226' response
	if string(downloadedContent) != uploadContent {
		t.Errorf("Downloaded content mismatch. Expected '%s', got '%s'", uploadContent, downloadedContent)
	}

	// Test SIZE
	size, err := c.FileSize("upload.txt")
	if err != nil {
		t.Errorf("SIZE failed: %v", err)
	}
	if int(size) != len(uploadContent) {
		t.Errorf("File SIZE mismatch. Expected %d, got %d", len(uploadContent), size)
	}

	// Test MDTM
	modTime, err := c.GetTime("upload.txt")
	if err != nil {
		t.Errorf("MDTM failed: %v", err)
	}
	if modTime.IsZero() { // GetTime returns time.Time
		t.Errorf("MDTM returned zero time")
	}

	// Test APPE (append)
	appendContent := " Appended content."
	err = c.Append("upload.txt", strings.NewReader(appendContent))
	if err != nil {
		t.Fatalf("APPE failed: %v", err)
	}
	// Verify appended content
	r, err = c.Retr("upload.txt")
	if err != nil {
		t.Fatalf("RETR after APPE failed: %v", err)
	}
	fullContent, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Reading full content after APPE failed: %v", err)
	}
	r.Close() // Explicitly close to read the pending '226' response
	expectedFullContent := uploadContent + appendContent
	if string(fullContent) != expectedFullContent {
		t.Errorf("Appended content mismatch. Expected '%s', got '%s'", expectedFullContent, fullContent)
	}

	// Test RNFR/RNTO
	err = c.Rename("upload.txt", "renamed.txt")
	if err != nil {
		t.Fatalf("RNFR/RNTO failed: %v", err)
	}
	_, err = c.FileSize("upload.txt")
	if err == nil {
		t.Error("Old file 'upload.txt' still exists after rename")
	}
	renamedSize, err := c.FileSize("renamed.txt")
	if err != nil || int(renamedSize) != len(expectedFullContent) {
		t.Errorf("Renamed file 'renamed.txt' not found or size mismatch")
	}

	// Test DELE
	err = c.Delete("renamed.txt")
	if err != nil {
		t.Fatalf("DELE failed: %v", err)
	}
	_, err = c.FileSize("renamed.txt")
	if err == nil {
		t.Error("Deleted file 'renamed.txt' still exists")
	}

	// Test RMD
	err = c.RemoveDir("/testdir")
	if err != nil {
		t.Fatalf("RMD failed: %v", err)
	}
	// Verify it's gone
	entries, err = c.List("")
	if err != nil {
		t.Errorf("LIST failed after RMD: %v", err)
	}
	for _, entry := range entries {
		if entry.Name == "testdir" {
			t.Error("testdir still found after RMD")
		}
	}
}

func TestLargeFileUploadExceedsLimit(t *testing.T) {
	dbPath := t.TempDir() + "/test-large-upload.db"
	serverAddr, dbConn, cleanup := setupServer(t, dbPath)
	defer cleanup()

	c, err := ftp.Dial(serverAddr, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("FTP dial failed: %v", err)
	}
	defer c.Quit()

	err = c.Login("anonymous", "password")
	if err != nil {
		t.Fatalf("FTP login failed: %v", err)
	}

	// Create content slightly larger than MaxFileSize (10MB)
	largeContent := make([]byte, vfs.MaxFileSize+1)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	err = c.Stor("largefile.txt", bytes.NewReader(largeContent))
	if err == nil {
		t.Error("Expected STOR of large file to fail, but it succeeded")
	} else {
		multiErr, isMultiErr := err.(*multierror.Error)
		if isMultiErr {
			found552 := false
			for _, wrappedErr := range multiErr.Errors {
				ftpErr, ok := wrappedErr.(*textproto.Error)
				if ok && ftpErr.Code == 552 {
					found552 = true
					break
				}
			}
			if !found552 {
				t.Errorf("Expected FTP error code 552 within multierror, got %v", multiErr)
			}
		} else {
			ftpErr, ok := err.(*textproto.Error)
			if !ok {
				t.Errorf("Expected a textproto.Error, got %T: %v", err, err)
			} else if ftpErr.Code != 552 {
				t.Errorf("Expected FTP error code 552, got %d: %v", ftpErr.Code, ftpErr.Msg)
			}
		}
	}
	// We verify that the file does not exist or is empty on the server.
	// Check database directly for largefile.txt to ensure it's not created
	var count int
	err = dbConn.QueryRow("SELECT COUNT(*) FROM files WHERE path = ?", "/largefile.txt").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query DB for largefile.txt: %v", err)
	}
	if count > 0 {
		t.Error("Large file was created in DB despite exceeding size limit")
	}
}
