package api

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	// SQLite3 database/sql driver.
	_ "github.com/mattn/go-sqlite3"
)

// Core represents the central manager of application activity.
type Core struct {
	Config Config
	db     *sql.DB
	quit   chan struct{}
}

// Config represents the core configuration parameters.
type Config struct {
	Addr           string
	Token          []byte
	DataSourceName string
	UploadDir      string
	MaxUploadSize  int64
}

// NewCore returns a new *Core.
func NewCore(config Config) (*Core, error) {
	_, _, err := net.SplitHostPort(config.Addr)
	if err != nil {
		return nil, errors.New("fu: addr must be a tcp network address for server mode")
	}
	db, err := sql.Open("sqlite3", config.DataSourceName)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id           INTEGER PRIMARY KEY,
			name         TEXT NOT NULL UNIQUE,
			created_at   DATETIME NOT NULL,
			expires_at   DATETIME NOT NULL
		);
	`)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(config.UploadDir, 0750)
	if err != nil {
		return nil, err
	}
	c := &Core{
		Config: config,
		db:     db,
	}
	return c, nil
}

// UploadForm represents a file upload form.
type UploadForm struct {
	Duration time.Duration
	File     io.Reader
	Ext      string
}

// Put uploads a new file and returns the metadata.
func (c *Core) Put(form UploadForm) (*File, error) {
	f := &File{CreatedAt: time.Now().UTC()}
	f.ExpiresAt = f.CreatedAt.Add(form.Duration)
	f.setName(form.Ext)
	tx, err := c.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	query := "INSERT INTO files (name, created_at, expires_at) VALUES (?, ?, ?)"
	res, err := c.db.Exec(query, f.Name, f.CreatedAt, f.ExpiresAt)
	if err != nil {
		return nil, err
	}
	f.ID, err = res.LastInsertId()
	if err != nil {
		return nil, err
	}
	name := c.getFilePath(f)
	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	_, err = io.Copy(file, form.File)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Run runs the background jobs.
func (c *Core) Run() {
	for {
		select {
		case <-time.After(time.Minute):
			_, err := c.deleteExpired()
			if err != nil {
				log.Printf("deleteExpired err=%q", err)
			}
		case <-c.quit:
			return
		}
	}
}

// getExpired returns a slice of expired files.
func (c *Core) getExpired() ([]*File, error) {
	var files []*File
	rows, err := c.db.Query("SELECT id, name, created_at, expires_at FROM files WHERE expires_at < ?", time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		f := &File{}
		err = rows.Scan(&f.ID, &f.Name, &f.CreatedAt, &f.ExpiresAt)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return files, nil
}

// deleteExpired deletes expired files from the file system and the database.
func (c *Core) deleteExpired() (int, error) {
	var vals []interface{}
	files, err := c.getExpired()
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		name := c.getFilePath(f)
		err = os.Remove(name)
		if err != nil && !os.IsNotExist(err) {
			log.Printf("deleteExpired err=%q\n", err)
			continue
		}
		vals = append(vals, f.ID)
	}
	n := len(vals)
	if n == 0 {
		return 0, nil
	}
	args := strings.Join(strings.Split(strings.Repeat("?", n), ""), ", ")
	query := fmt.Sprintf("DELETE FROM files WHERE id IN (%s)", args)
	_, err = c.db.Exec(query, vals...)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Close cleans up for safe shutdown.
// Close implements the io.Closer interface.
func (c *Core) Close() error {
	close(c.quit)
	return c.db.Close()
}

// getFilePath returns the full path to the file.
func (c *Core) getFilePath(f *File) string {
	return filepath.Join(c.Config.UploadDir, f.Name)
}
