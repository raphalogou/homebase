// Package webui serves the built web app embedded in the binary.
//
// `make build` copies web/dist into ./dist before compiling. Without that step
// only the placeholder is embedded and every page answers with a hint.
package webui

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// Handler serves the embedded build.
func Handler() http.Handler {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		// The embed directive guarantees the directory exists.
		panic(err)
	}
	return New(dist)
}

// New serves files from dist and falls back to index.html for any other path,
// so client-side routes such as /plan work on reload.
func New(dist fs.FS) http.Handler {
	files := http.FileServerFS(dist)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

		if name != "" && name != "index.html" && fileExists(dist, name) {
			// Vite puts a content hash in every file name under assets/.
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(w, r)
			return
		}

		if path.Ext(name) != "" && name != "index.html" {
			http.NotFound(w, r)
			return
		}

		index, err := fs.ReadFile(dist, "index.html")
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, "The web app is not built into this binary. Run make build, or use make dev.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	})
}

func fileExists(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && !info.IsDir()
}
