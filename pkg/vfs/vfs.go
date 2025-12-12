package vfs

import (
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt" 
	"io"
	"log/slog" // Added for logging
	"os"
	"path/filepath"
	"strings"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"
)

var vfsLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))

const (
	MaxFileSize = 10 * 1024 * 1024 // 10MB
)

// MainDriver implements ftpserver.MainDriver
type MainDriver struct {
	db                *sql.DB
	passiveStart      int
	passiveEnd        int
	listenAddr        string
	connectionTimeout time.Duration
}

// NewMainDriver creates a new MainDriver
func NewMainDriver(db *sql.DB, passiveStart, passiveEnd int, listenAddr string, connectionTimeout time.Duration) *MainDriver {
	return &MainDriver{
		db:                db,
		passiveStart:      passiveStart,
		passiveEnd:        passiveEnd,
		listenAddr:        listenAddr,
		connectionTimeout: connectionTimeout,
	}
}

// GetSettings returns the server settings
func (d *MainDriver) GetSettings() (*ftpserver.Settings, error) {
	return &ftpserver.Settings{
		ListenAddr:               d.listenAddr,
		ConnectionTimeout:        int(d.connectionTimeout.Seconds()),
		PassiveTransferPortRange: ftpserver.PortRange{Start: d.passiveStart, End: d.passiveEnd},
	}, nil
}

// ClientConnected is called when a client connects
func (d *MainDriver) ClientConnected(cc ftpserver.ClientContext) (string, error) {
	return "Welcome to SQLite FTP Mimic", nil
}

// ClientDisconnected is called when a client disconnects
func (d *MainDriver) ClientDisconnected(cc ftpserver.ClientContext) {
}

// AuthUser authenticates the user and returns a ClientDriver (filesystem)
func (d *MainDriver) AuthUser(cc ftpserver.ClientContext, user, pass string) (ftpserver.ClientDriver, error) {
	// No authentication required as per requirements
	return &SQLiteFs{db: d.db}, nil
}

// GetTLSConfig returns the TLS configuration
func (d *MainDriver) GetTLSConfig() (*tls.Config, error) {
	return nil, nil
}

// SQLiteFs implements ftpserver.ClientDriver (which embeds afero.Fs)
type SQLiteFs struct {
	db *sql.DB
}

func (fs *SQLiteFs) Create(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (fs *SQLiteFs) Mkdir(name string, perm os.FileMode) error {
	name = normalizePath(name)
	if name == "/" {
		return os.ErrInvalid
	}

	parentPath := filepath.Dir(name)
	baseName := filepath.Base(name)

	// Check parent
	var parentIsDir bool
	err := fs.db.QueryRow("SELECT is_dir FROM files WHERE path = ?", parentPath).Scan(&parentIsDir)
	if err == sql.ErrNoRows {
		return os.ErrNotExist
	} else if err != nil {
		return err
	}
	if !parentIsDir {
		return os.ErrExist // Parent is a file
	}

	// Check existence
	var count int
	err = fs.db.QueryRow("SELECT COUNT(*) FROM files WHERE path = ?", name).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return os.ErrExist
	}

	_, err = fs.db.Exec(`
		INSERT INTO files (path, parent_path, name, is_dir, size, mod_time)
		VALUES (?, ?, ?, 1, 0, ?)
	`, name, parentPath, baseName, time.Now().Format(time.RFC3339))
	return err
}

func (fs *SQLiteFs) MkdirAll(path string, perm os.FileMode) error {
	path = normalizePath(path)
	parts := strings.Split(path, "/")
	currentPath := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		currentPath += "/" + part
		if currentPath == "/" {
			continue
		}
		err := fs.Mkdir(currentPath, perm)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func (fs *SQLiteFs) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *SQLiteFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	name = normalizePath(name)
	
	var fileInfo FileInfo
	var modTimeStr string
	var content []byte

	row := fs.db.QueryRow(`
		SELECT name, size, is_dir, mod_time, path, content
		FROM files
		WHERE path = ?
	`, name)

	err := row.Scan(&fileInfo.name, &fileInfo.size, &fileInfo.isDir, &modTimeStr, &fileInfo.path, &content)
	
	// Handle creation
	if err == sql.ErrNoRows {
		if flag&os.O_CREATE != 0 {
			parentPath := filepath.Dir(name)
			baseName := filepath.Base(name)
			
			// Check parent exists
			var parentIsDir bool
			err := fs.db.QueryRow("SELECT is_dir FROM files WHERE path = ?", parentPath).Scan(&parentIsDir)
			if err != nil {
				return nil, os.ErrNotExist
			}
			if !parentIsDir {
				return nil, os.ErrNotExist
			}

			// Insert empty file placeholder
			now := time.Now()
			_, err = fs.db.Exec(`
				INSERT INTO files (path, parent_path, name, is_dir, size, mod_time, content)
				VALUES (?, ?, ?, 0, 0, ?, NULL)
			`, name, parentPath, baseName, now.Format(time.RFC3339))
			if err != nil {
				return nil, err
			}
			
			return &SqliteFile{
				path:    name,
				fs:      fs,
				content: []byte{},
				flag:    flag,
				modTime: now,
			}, nil
		}
		return nil, os.ErrNotExist
	} else if err != nil {
		return nil, err
	}

	if fileInfo.isDir {
		t, _ := time.Parse(time.RFC3339, modTimeStr)
		return &SqliteFile{
			path:    name,
			fs:      fs,
			isDir:   true,
			modTime: t,
		}, nil
	}

	// Existing file
	t, err := time.Parse(time.RFC3339, modTimeStr)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05", modTimeStr)
	}

	f := &SqliteFile{
		path:    name,
		fs:      fs,
		flag:    flag,
		modTime: t,
	}

	// Handle flags
	if flag&os.O_TRUNC != 0 {
		f.content = []byte{}
	} else {
		f.content = content
	}

	if flag&os.O_APPEND != 0 {
		f.pos = int64(len(f.content))
	}

	return f, nil
}

func (fs *SQLiteFs) Remove(name string) error {
	name = normalizePath(name)
	if name == "/" {
		return os.ErrInvalid
	}

	// Check if directory is empty
	var isDir bool
	err := fs.db.QueryRow("SELECT is_dir FROM files WHERE path = ?", name).Scan(&isDir)
	if err == sql.ErrNoRows {
		return os.ErrNotExist
	} else if err != nil {
		return err
	}

	if isDir {
		var count int
		err := fs.db.QueryRow("SELECT COUNT(*) FROM files WHERE parent_path = ?", name).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			// Directory not empty
			return &os.PathError{Op: "remove", Path: name, Err: errors.New("directory not empty")} 
		}
	}

	_, err = fs.db.Exec("DELETE FROM files WHERE path = ?", name)
	return err
}

func (fs *SQLiteFs) RemoveAll(path string) error {
	return fs.Remove(path)
}

func (fs *SQLiteFs) Rename(oldname, newname string) error {
	oldname = normalizePath(oldname)
	newname = normalizePath(newname)

	if oldname == "/" || newname == "/" {
		return os.ErrInvalid
	}
	
	// Check old exists
	var oldIsDir bool
	err := fs.db.QueryRow("SELECT is_dir FROM files WHERE path = ?", oldname).Scan(&oldIsDir)
	if err == sql.ErrNoRows {
		return os.ErrNotExist
	}

	// Check new does not exist
	var count int
	fs.db.QueryRow("SELECT COUNT(*) FROM files WHERE path = ?", newname).Scan(&count)
	if count > 0 {
		return os.ErrExist
	}

	tx, err := fs.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	newParent := filepath.Dir(newname)
	newNameBase := filepath.Base(newname)

	_, err = tx.Exec("UPDATE files SET path = ?, parent_path = ?, name = ? WHERE path = ?", newname, newParent, newNameBase, oldname)
	if err != nil {
		return err
	}
	
	// Update children if dir
	if oldIsDir {
		_, err = tx.Exec("UPDATE files SET parent_path = ? WHERE parent_path = ?", newname, oldname)
		if err != nil {
			return err
		}
		// Shallow path update for children
		_, err = tx.Exec("UPDATE files SET path = ? || SUBSTR(path, LENGTH(?)+1) WHERE path LIKE ? || '/%'", newname, oldname, oldname)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (fs *SQLiteFs) Stat(name string) (os.FileInfo, error) {
	name = normalizePath(name)
	
	var fileInfo FileInfo
	var modTimeStr string

	row := fs.db.QueryRow(`
		SELECT name, size, is_dir, mod_time, path
		FROM files
		WHERE path = ?
	`, name)

	err := row.Scan(&fileInfo.name, &fileInfo.size, &fileInfo.isDir, &modTimeStr, &fileInfo.path)
	if err == sql.ErrNoRows {
		vfsLogger.Debug("SQLiteFs.Stat: file not found", "path", name)
		return nil, os.ErrNotExist
	} else if err != nil {
		vfsLogger.Error("SQLiteFs.Stat: failed to query file", "path", name, "error", err)
		return nil, err
	}

	vfsLogger.Debug("SQLiteFs.Stat: raw modTimeStr", "path", name, "modTimeStr", modTimeStr)
	fileInfo.modTime, _ = time.Parse(time.RFC3339, modTimeStr)
	if fileInfo.modTime.IsZero() {
		vfsLogger.Debug("SQLiteFs.Stat: RFC3339 parse failed, trying YYYY-MM-DD HH:MM:SS", "path", name, "modTimeStr", modTimeStr)
		fileInfo.modTime, _ = time.Parse("2006-01-02 15:04:05", modTimeStr)
	}
	vfsLogger.Debug("SQLiteFs.Stat: parsed modTime", "path", name, "modTime", fileInfo.modTime)

	return &fileInfo, nil
}

func (fs *SQLiteFs) Name() string {
	return "sqlite-vfs"
}

func (fs *SQLiteFs) Chmod(name string, mode os.FileMode) error {
	return nil // No-op
}

func (fs *SQLiteFs) Chown(name string, uid, gid int) error {
	return nil // No-op
}

func (fs *SQLiteFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	name = normalizePath(name)
	_, err := fs.db.Exec("UPDATE files SET mod_time = ? WHERE path = ?", mtime, name)
	return err
}

// SqliteFile implements afero.File
type SqliteFile struct {
	path    string
	fs      *SQLiteFs
	content []byte
	pos     int64
	flag    int
	isDir   bool
	modTime time.Time
}

func (f *SqliteFile) Close() error {
	if f.isDir {
		return nil
	}
	if f.flag&os.O_WRONLY != 0 || f.flag&os.O_RDWR != 0 || f.flag&os.O_APPEND != 0 || f.flag&os.O_CREATE != 0 {
		vfsLogger.Debug("SqliteFile.Close called (writing)", "path", f.path, "len_content_before_update", len(f.content))
		_, err := f.fs.db.Exec("UPDATE files SET content = ?, size = ?, mod_time = ? WHERE path = ?", f.content, len(f.content), time.Now(), f.path)
		if err != nil {
            vfsLogger.Error("Failed to update file content on close", "path", f.path, "error", err)
            return fmt.Errorf("failed to update file %s: %w", f.path, err)
        }
		vfsLogger.Debug("SqliteFile.Close success", "path", f.path, "size", len(f.content))
		return nil
	}
	return nil
}

func (f *SqliteFile) Read(p []byte) (n int, err error) {
	if f.isDir {
		return 0, os.ErrInvalid
	}
	if f.pos >= int64(len(f.content)) {
		return 0, io.EOF
	}
	n = copy(p, f.content[f.pos:])
	f.pos += int64(n)
	return n, nil
}

func (f *SqliteFile) ReadAt(p []byte, off int64) (n int, err error) {
	if f.isDir {
		return 0, os.ErrInvalid
	}
	if off >= int64(len(f.content)) {
		return 0, io.EOF
	}
	n = copy(p, f.content[off:])
	return n, nil
}

func (f *SqliteFile) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = f.pos + offset
	case io.SeekEnd:
		abs = int64(len(f.content)) + offset
	default:
		return 0, errors.New("invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("negative position")
	}
	f.pos = abs
	return abs, nil
}

func (f *SqliteFile) Write(p []byte) (n int, err error) {
	if f.isDir {
		return 0, os.ErrInvalid
	}
	
	if int64(len(f.content)) + int64(len(p)) > MaxFileSize {
		vfsLogger.Warn("SqliteFile.Write: write would exceed MaxFileSize, deleting file", "path", f.path, "current_len", len(f.content), "write_len", len(p), "max_size", MaxFileSize)
		_, deleteErr := f.fs.db.Exec("DELETE FROM files WHERE path = ?", f.path)
		if deleteErr != nil {
			vfsLogger.Error("Failed to delete oversized file on write", "path", f.path, "error", deleteErr)
		}
		return 0, ftpserver.ErrStorageExceeded
	}

	if f.pos >= int64(len(f.content)) {
		f.content = append(f.content, p...)
		n = len(p)
		f.pos += int64(n)
	} else {
		n = copy(f.content[f.pos:], p)
		if n < len(p) { // If p is larger than remaining space, append the rest
			f.content = append(f.content, p[n:]...)
		}
		f.pos += int64(len(p))
	}
	return len(p), nil
}

func (f *SqliteFile) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, errors.New("WriteAt not supported")
}

func (f *SqliteFile) Name() string {
	return filepath.Base(f.path)
}

func (f *SqliteFile) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isDir {
		return nil, os.ErrInvalid
	}

	rows, err := f.fs.db.Query("SELECT name, size, is_dir, mod_time, path FROM files WHERE parent_path = ? ORDER BY name", f.path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var infos []os.FileInfo
	for rows.Next() {
		var fi FileInfo
		var modTimeStr string
		err := rows.Scan(&fi.name, &fi.size, &fi.isDir, &modTimeStr, &fi.path)
		if err != nil {
			return nil, err
		}
		fi.modTime, _ = time.Parse(time.RFC3339, modTimeStr)
		if fi.modTime.IsZero() {
			fi.modTime, _ = time.Parse("2006-01-02 15:04:05", modTimeStr)
		}
		infos = append(infos, &fi)
		
		if count > 0 && len(infos) >= count {
			break
		}
	}
	return infos, nil
}

func (f *SqliteFile) Readdirnames(n int) ([]string, error) {
	infos, err := f.Readdir(n)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.Name()
	}
	return names, nil
}

func (f *SqliteFile) Stat() (os.FileInfo, error) {
	return f.fs.Stat(f.path)
}

func (f *SqliteFile) Sync() error {
	return nil 
}

func (f *SqliteFile) Truncate(size int64) error {
	if size < 0 {
		return os.ErrInvalid
	}
	if size > int64(len(f.content)) {
		diff := size - int64(len(f.content))
		f.content = append(f.content, make([]byte, diff)...)
	} else {
		f.content = f.content[:size]
	}
	return nil
}

func (f *SqliteFile) WriteString(s string) (ret int, err error) {
	return f.Write([]byte(s))
}

// FileInfo struct
type FileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
	path    string
}

func (fi *FileInfo) Name() string       { return fi.name }
func (fi *FileInfo) Size() int64        { return fi.size }
func (fi *FileInfo) Mode() os.FileMode  { 
	if fi.isDir { 
		return os.ModeDir | 0755 
	} 
	return 0644 
}
func (fi *FileInfo) ModTime() time.Time { return fi.modTime }
func (fi *FileInfo) IsDir() bool        { return fi.isDir }
func (fi *FileInfo) Sys() interface{}   { return nil }

func normalizePath(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return filepath.Clean(p)
}
