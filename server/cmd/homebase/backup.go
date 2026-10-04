package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"homebase/internal/db"
	"homebase/internal/push"
)

// backup copies everything a restore needs into dir:
//
//	dir/homebase-YYYYMMDD-HHMMSS.db   a consistent copy, made with VACUUM INTO
//	dir/files/ab/cd/<sha256>          uploaded files, shared between backups
//	dir/vapid-private.pem             the push key; without it every device
//	                                  must turn reminders on again
//
// It is safe while the server runs: VACUUM INTO reads a snapshot, and stored
// files never change once written.
func backup(ctx context.Context, dataDir, dir string, now time.Time, out io.Writer) error {
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
