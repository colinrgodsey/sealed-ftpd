package main

import (
	"flag"
	"fmt" // For Sprintf
	stdlog "log" // Alias standard log
	"log/slog"   // Standard library slog
	"os"

	"ftp-mimic/pkg/db"
	"ftp-mimic/pkg/vfs"

	ftpserver "github.com/fclairamb/ftpserverlib"
	golog_slog "github.com/fclairamb/go-log/slog" // Adapter for slog
)

func main() {
	passiveStart := flag.Int("passive-port-start", 20000, "Start of the passive port range")
	passiveEnd := flag.Int("passive-port-end", 20009, "End of the passive port range")
	flag.Parse()

	// Initialize the database
	dbPath := "./ftp-mimic.db" // Default DB file
	sqliteDB, err := db.InitDB(dbPath)
	if err != nil {
		stdlog.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqliteDB.Close()

	// Create our MainDriver
	mainDriver := vfs.NewMainDriver(sqliteDB, *passiveStart, *passiveEnd)

	// Create the FTP server
	ftpServer := ftpserver.NewFtpServer(mainDriver)

	// Create a standard slog logger
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Wrap the slog logger with the go-log adapter
	ftpServer.Logger = golog_slog.NewWrap(slogLogger)

	settings, err := mainDriver.GetSettings()
	if err != nil {
		stdlog.Fatalf("Failed to get server settings: %v", err)
	}

	// Cast PassiveTransferPortRange to PortRange to access Start and End
	portRange := settings.PassiveTransferPortRange.(ftpserver.PortRange)
	passivePorts := fmt.Sprintf("%d-%d", portRange.Start, portRange.End)
	stdlog.Printf("Starting FTP server on %s with passive ports %s...", settings.ListenAddr, passivePorts)
	if err := ftpServer.ListenAndServe(); err != nil {
		stdlog.Fatalf("FTP server failed: %v", err)
	}
}
