// Package files stores uploaded files by their SHA-256, never under a name
// the user chose: data/files/ab/cd/<sha256>.
package files

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"homebase/internal/apperr"
)

// MaxSize is the largest file accepted (docs/SPEC.md section 1).
const MaxSize = 25 << 20

// Blobs is the folder of stored files.
type Blobs struct {
	dir string
}

// New returns Blobs rooted at dataDir/files, creating it if needed.
func New(dataDir string) (*Blobs, error) {
	dir := filepath.Join(dataDir, "files")
	if err := os.MkdirAll(filepath.Join(dir, "tmp"), 0o700); err != nil {
		return nil, fmt.Errorf("create files folder: %w", err)
	}
	return &Blobs{dir: dir}, nil
}

// Saved describes a stored file.
type Saved struct {
	SHA  string
	Size int64
	Mime string
}

// Save streams r to disk, hashing as it goes. A file that is already stored
// is not written twice.
func (b *Blobs) Save(r io.Reader) (Saved, error) {
	tmp, err := os.CreateTemp(filepath.Join(b.dir, "tmp"), "upload-*")
	if err != nil {
		return Saved{}, err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	h := sha256.New()
	head := &headBuffer{limit: 512}
	n, err := io.Copy(io.MultiWriter(tmp, h, head), io.LimitReader(r, MaxSize+1))
	if err != nil {
		return Saved{}, err
	}
	if n > MaxSize {
		return Saved{}, apperr.New(apperr.TooLarge, "Files can be at most 25 MB.")
	}
	if n == 0 {
		return Saved{}, apperr.New(apperr.Invalid, "The file is empty.")
	}
	if err := tmp.Sync(); err != nil {
		return Saved{}, err
	}

	s := Saved{SHA: hex.EncodeToString(h.Sum(nil)), Size: n, Mime: sniff(head.buf)}
	dst := b.path(s.SHA)
	if _, err := os.Stat(dst); err == nil {
		return s, nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return Saved{}, err
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return Saved{}, err
	}
	return s, nil
}

// Open opens a stored file. It returns fs.ErrNotExist for an unknown or
// cleaned-up file.
func (b *Blobs) Open(sha string) (*os.File, error) {
	if !ValidSHA(sha) {
		return nil, fs.ErrNotExist
	}
	return os.Open(b.path(sha))
}

// Remove deletes a stored file and reports whether there was one to delete.
func (b *Blobs) Remove(sha string) (bool, error) {
	if !ValidSHA(sha) {
		return false, nil
	}
	err := os.Remove(b.path(sha))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (b *Blobs) path(sha string) string {
	return filepath.Join(b.dir, sha[:2], sha[2:4], sha)
}

// ValidSHA reports whether s is a lower-case hex SHA-256.
func ValidSHA(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// sniff decides the type from the content, never from the client's claim.
func sniff(head []byte) string {
	mime := http.DetectContentType(head)
	if i := strings.IndexByte(mime, ';'); i >= 0 && !strings.HasPrefix(mime, "text/") {
		mime = mime[:i]
	}
	return mime
}

// Inline reports whether a type may be shown in the browser rather than
// downloaded: raster images and PDFs. SVG is excluded because it can carry
// script.
func Inline(mime string) bool {
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp", "application/pdf":
		return true
	}
	return false
}

type headBuffer struct {
	buf   []byte
	limit int
}

func (h *headBuffer) Write(p []byte) (int, error) {
	if room := h.limit - len(h.buf); room > 0 {
		h.buf = append(h.buf, p[:min(room, len(p))]...)
	}
	return len(p), nil
}
