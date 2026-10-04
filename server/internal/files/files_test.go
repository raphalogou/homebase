package files

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"homebase/internal/apperr"
)

func TestSave(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 100)...)
	tests := []struct {
		name     string
		body     io.Reader
		wantMime string
		wantCode apperr.Code
	}{
		{"png sniffed from content", bytes.NewReader(png), "image/png", ""},
		{"pdf", strings.NewReader("%PDF-1.7\n..."), "application/pdf", ""},
		{"html stays text/html", strings.NewReader("<html><script>alert(1)</script>"), "text/html; charset=utf-8", ""},
		{"empty", strings.NewReader(""), "", apperr.Invalid},
		{"one byte too large", io.LimitReader(zeros{}, MaxSize+1), "", apperr.TooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := New(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			s, err := b.Save(tt.body)
			if tt.wantCode != "" {
				e, ok := apperr.As(err)
				if !ok || e.Code != tt.wantCode {
					t.Fatalf("err = %v, want %s", err, tt.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if s.Mime != tt.wantMime {
				t.Errorf("mime = %q, want %q", s.Mime, tt.wantMime)
			}
			f, err := b.Open(s.SHA)
			if err != nil {
				t.Fatal(err)
			}
			_ = f.Close()
		})
	}
}

func TestSaveExactlyMaxSize(t *testing.T) {
	b, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := b.Save(io.LimitReader(zeros{}, MaxSize))
	if err != nil || s.Size != MaxSize {
		t.Fatalf("Save(25 MB) = %+v, %v", s, err)
	}
}

func TestSameFileStoredOnce(t *testing.T) {
	dir := t.TempDir()
	b, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	a, err := b.Save(strings.NewReader("same bytes"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := b.Save(strings.NewReader("same bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if a.SHA != c.SHA {
		t.Fatalf("different hashes for the same bytes")
	}
	count := 0
	_ = filepath.WalkDir(filepath.Join(dir, "files"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			count++
		}
		return nil
	})
	if count != 1 {
		t.Errorf("files on disk = %d, want 1 (temp files cleaned, one blob)", count)
	}
	if got := filepath.Join(dir, "files", a.SHA[:2], a.SHA[2:4], a.SHA); !exists(got) {
		t.Errorf("blob not at %s", got)
	}
}

func TestOpenRejectsPaths(t *testing.T) {
	b, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, sha := range []string{"../../etc/passwd", "", strings.Repeat("A", 64)} {
		if _, err := b.Open(sha); err == nil {
			t.Errorf("Open(%q) succeeded", sha)
		}
	}
}

func TestInline(t *testing.T) {
	tests := []struct {
		mime string
		want bool
	}{
		{"image/png", true},
		{"application/pdf", true},
		{"image/svg+xml", false},
		{"text/html; charset=utf-8", false},
		{"application/octet-stream", false},
	}
	for _, tt := range tests {
		if got := Inline(tt.mime); got != tt.want {
			t.Errorf("Inline(%q) = %v, want %v", tt.mime, got, tt.want)
		}
	}
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
