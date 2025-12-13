package tests

import (
	"database/sql"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ftp-mimic/pkg/db"
	"ftp-mimic/pkg/vfs"

	ftpserver "github.com/fclairamb/ftpserverlib"
	golog_slog "github.com/fclairamb/go-log/slog"
	"github.com/jlaffaye/ftp"
	"log/slog"
)

// setupStressServer starts the FTP server for stress testing.
// It uses a larger passive port range and specific tuning.
func setupStressServer(t *testing.T, dbPath string) (addr string, dbConn *sql.DB, cleanup func()) {
	// Initialize the database with potentially higher connection limits if needed.
	// We'll rely on the default InitDB tuning for now, which has 100 max open conns.
	sqliteDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Use a fixed port if possible to avoid port exhaustion during rapid creates, 
	// but dynamic is safer for CI/test environments.
	tempListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := tempListener.Addr().(*net.TCPAddr).Port
	tempListener.Close()

	listenAddr := fmt.Sprintf("127.0.0.1:%d", port)
	connectionTimeout := 30 * time.Second // Increased timeout for stress

	// Wider passive port range to accommodate many concurrent data transfers
	mainDriver := vfs.NewMainDriver(sqliteDB, 40000, 50000, listenAddr, connectionTimeout)

	ftpServer := ftpserver.NewFtpServer(mainDriver)

	// Suppress logging during stress test to avoid IO bottleneck and huge logs
	// Or keep it at ERROR level
	slogLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	ftpServer.Logger = golog_slog.NewWrap(slogLogger)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := ftpServer.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			t.Errorf("FTP server ListenAndServe failed: %v", err)
		}
	}()

	time.Sleep(200 * time.Millisecond)

	return listenAddr, sqliteDB, func() {
		ftpServer.Stop()
		wg.Wait()
		sqliteDB.Close()
		os.Remove(dbPath)
	}
}

func TestStressConcurrency(t *testing.T) {
	// Configuration
	// We will attempt 1000 if the environment allows.
	// NOTE: 1000 concurrent FTP clients means 1000 TCP control connections + 1000 potential data connections.
	// This requires ulimit -n to be at least 2048 + overhead.
	// We'll set it to 1000 and handle connection errors gracefully if they are OS limits.
	
	targetUsers := 1000
	
	dbPath := t.TempDir() + "/stress-test.db"
	serverAddr, _, cleanup := setupStressServer(t, dbPath)
	defer cleanup()

	var successCount int64
	var errorCount int64
	var wg sync.WaitGroup

	// Random source for each goroutine to avoid lock contention on global rand
	
	start := time.Now()

	for i := 0; i < targetUsers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Stagger starts slightly to avoid thundering herd on Dial
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			// 1. Connect
			c, err := ftp.Dial(serverAddr, ftp.DialWithTimeout(10*time.Second))
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				// t.Logf("User %d: Dial failed: %v", id, err)
				return
			}
			defer c.Quit()

			// 2. Login
			if err := c.Login("anonymous", "stress"); err != nil {
				atomic.AddInt64(&errorCount, 1)
				// t.Logf("User %d: Login failed: %v", id, err)
				return
			}

			// 3. Operations
			// Upload a unique file
			filename := "file_" + strconv.Itoa(id) + ".txt"
			content := "content_" + strconv.Itoa(id)
			if err := c.Stor(filename, strings.NewReader(content)); err != nil {
				atomic.AddInt64(&errorCount, 1)
				return
			}

			// List
			if _, err := c.List(""); err != nil {
				atomic.AddInt64(&errorCount, 1)
				return
			}

			// Download back
			r, err := c.Retr(filename)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				return
			}
			data, err := io.ReadAll(r)
			r.Close() // Important: Close data connection
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				return
			}
			if string(data) != content {
				atomic.AddInt64(&errorCount, 1)
				return
			}

			atomic.AddInt64(&successCount, 1)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Stress Test Finished in %v", duration)
	t.Logf("Total Users: %d", targetUsers)
	t.Logf("Successful: %d", successCount)
	t.Logf("Failed: %d", errorCount)

	if successCount < int64(targetUsers)*90/100 { // Expect at least 90% success
		t.Errorf("Too many failures. Success: %d, Failed: %d", successCount, errorCount)
	}
}
