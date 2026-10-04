// Package db opens the SQLite database.
package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite"
)

// Open opens the database file at path. Pragmas are set in the DSN so that
// every connection gets them, as docs/SPEC.md section 2 requires.
func Open(path string) (*sql.DB, error) {
	return open("file:" + (&url.URL{Path: path}).EscapedPath())
}

// OpenMemory opens a private in-memory database for tests.
func OpenMemory() (*sql.DB, error) {
	return open("file:mem-" + rand.Text() + "?mode=memory&cache=shared")
}

func open(dsn string) (*sql.DB, error) {
	sep := "?"
	if u, err := url.Parse(dsn); err == nil && u.RawQuery != "" {
		sep = "&"
	}
	dsn += sep + "_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"

	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// One user, one writer: a single connection serialises every transaction,
	// which keeps revision numbers in commit order and rules out SQLITE_BUSY.
	d.SetMaxOpenConns(1)
	d.SetConnMaxIdleTime(0)
	d.SetConnMaxLifetime(0)
	if err := d.Ping(); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	return d, nil
}
