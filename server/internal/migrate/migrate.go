// Package migrate applies the numbered SQL files in server/migrations.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

type migration struct {
	version int
	name    string
}

// Run applies every migration newer than the recorded version, each in its
// own transaction together with its schema_migrations row.
func Run(ctx context.Context, d *sql.DB, files fs.FS, now func() time.Time) error {
	list, err := discover(files)
	if err != nil {
		return err
	}

	current, err := currentVersion(ctx, d)
	if err != nil {
		return err
	}

	for _, m := range list {
		if m.version <= current {
			continue
		}
		body, err := fs.ReadFile(files, m.name)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.name, err)
		}
		if err := apply(ctx, d, m, string(body), now().UnixMilli()); err != nil {
			return err
		}
	}
	return nil
}

func apply(ctx context.Context, d *sql.DB, m migration, body string, at int64) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, body); err != nil {
		return fmt.Errorf("migration %s: %w", m.name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`, m.version, at); err != nil {
		return fmt.Errorf("record migration %s: %w", m.name, err)
	}
	return tx.Commit()
}

func discover(files fs.FS) ([]migration, error) {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, err
	}
	seen := map[int]string{}
	var list []migration
	for _, name := range names {
		prefix, _, ok := strings.Cut(name, "_")
		v, err := strconv.Atoi(prefix)
		if !ok || err != nil || v <= 0 {
			return nil, fmt.Errorf("migration %q must be named NNNN_description.sql", name)
		}
		if other, dup := seen[v]; dup {
			return nil, fmt.Errorf("migrations %q and %q share version %d", other, name, v)
		}
		seen[v] = name
		list = append(list, migration{version: v, name: name})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].version < list[j].version })
	return list, nil
}

// currentVersion is 0 on a new database. 0001 creates schema_migrations
// itself, so its absence means nothing has run.
func currentVersion(ctx context.Context, d *sql.DB) (int, error) {
	var n int
	err := d.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&n)
	if err != nil || n == 0 {
		return 0, err
	}
	var v sql.NullInt64
	if err := d.QueryRowContext(ctx, `SELECT max(version) FROM schema_migrations`).Scan(&v); err != nil {
		return 0, err
	}
	return int(v.Int64), nil
}
