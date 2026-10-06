// Package backup copies the database, files and push key somewhere a
// restore can use them, on demand (homebase backup) or daily from serve.
package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"homebase/internal/db"
	"homebase/internal/push"
)

// Run copies everything a restore needs into dir:
//
//	dir/homebase-YYYYMMDD-HHMMSS.db   a consistent copy, made with VACUUM INTO
//	dir/files/ab/cd/<sha256>          uploaded files, shared between backups
//	dir/vapid-private.pem             the push key; without it every device
//	                                  must turn reminders on again
//
// It is safe while the server runs: VACUUM INTO reads a snapshot, and stored
// files never change once written.
func Run(ctx context.Context, dataDir, dir string, now time.Time, out io.Writer) error {
	src := filepath.Join(dataDir, "homebase.db")
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("no database at %s: %w", src, err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	d, err := db.Open(src)
	if err != nil {
		return err
	}
	defer d.Close()
	dst := filepath.Join(dir, "homebase-"+now.UTC().Format("20060102-150405")+".db")
	if _, err := d.ExecContext(ctx, `VACUUM INTO ?`, dst); err != nil {
		return fmt.Errorf("copy database: %w", err)
	}
	if err := os.Chmod(dst, 0o600); err != nil {
		return err
	}

	copied, err := copyFiles(filepath.Join(dataDir, "files"), filepath.Join(dir, "files"))
	if err != nil {
		return err
	}
	if err := copyFile(filepath.Join(dataDir, push.KeyFile), filepath.Join(dir, push.KeyFile)); err != nil &&
		!errors.Is(err, fs.ErrNotExist) {
		return err
	}

	_, err = fmt.Fprintf(out, "Database copied to %s\n%d new files copied to %s\n", dst, copied, filepath.Join(dir, "files"))
	return err
}

// Loop backs up into dir every interval while on reports true. It checks each
// minute against the newest copy in dir rather than a clock time, so the
// first backup follows soon after it is switched on, and one missed while
// the server was off happens soon after it starts again.
func Loop(ctx context.Context, log *slog.Logger, on func(context.Context) (bool, error), dataDir, dir string, interval time.Duration) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		enabled, err := on(ctx)
		if err != nil && ctx.Err() == nil {
			log.Error("backup", "err", err)
		}
		if now := time.Now(); enabled && Due(dir, now, interval) {
			if err := Run(ctx, dataDir, dir, now, io.Discard); err != nil && ctx.Err() == nil {
				log.Error("backup", "dir", dir, "err", err)
			} else if err == nil {
				log.Info("backup", "dir", dir)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Due reports whether the newest database copy in dir is interval old or
// missing. A minute of slack keeps the once-a-minute check from drifting a
// minute later each time.
func Due(dir string, now time.Time, interval time.Duration) bool {
	last, ok := Last(dir)
	return !ok || now.Sub(last) >= interval-time.Minute
}

// Last is when the newest database copy in dir was made. The names sort by
// time, so the last match is the newest.
func Last(dir string) (time.Time, bool) {
	names, _ := filepath.Glob(filepath.Join(dir, "homebase-*.db"))
	if len(names) == 0 {
		return time.Time{}, false
	}
	t, err := time.Parse("homebase-20060102-150405.db", filepath.Base(slices.Max(names)))
	return t, err == nil
}

// RunAll backs up everyone: each planner in dataDir/users/<id> into
// dir/<id>, and the push key, which they share, into dir. An install from
// before several people still has its database at the top and is backed
// up as before.
func RunAll(ctx context.Context, dataDir, dir string, now time.Time, out io.Writer) error {
	if _, err := os.Stat(filepath.Join(dataDir, "homebase.db")); err == nil {
		return Run(ctx, dataDir, dir, now, out)
	}
	users, err := os.ReadDir(filepath.Join(dataDir, "users"))
	if err != nil {
		return fmt.Errorf("no planners in %s: %w", dataDir, err)
	}
	for _, u := range users {
		if !u.IsDir() {
			continue
		}
		if err := Run(ctx, filepath.Join(dataDir, "users", u.Name()), filepath.Join(dir, u.Name()), now, out); err != nil {
			return fmt.Errorf("%s: %w", u.Name(), err)
		}
	}
	err = copyFile(filepath.Join(dataDir, push.KeyFile), filepath.Join(dir, push.KeyFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// copyFiles copies stored files that the backup does not have yet. They are
// named by their hash, so one already there is the same file.
func copyFiles(from, to string) (int, error) {
	n := 0
	err := filepath.WalkDir(from, func(path string, e fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) && path == from {
			return filepath.SkipAll
		}
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		if e.IsDir() {
			if rel == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		dst := filepath.Join(to, rel)
		if _, err := os.Stat(dst); err == nil {
			return nil
		}
		if err := copyFile(path, dst); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	tmp := dst + ".part"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}
