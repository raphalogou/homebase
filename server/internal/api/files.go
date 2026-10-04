package api

import (
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"strings"
	"time"

	"homebase/internal/apperr"
	"homebase/internal/files"
)

// uploadChecks is writeChecks for the one endpoint that takes multipart.
func (s *server) uploadChecks(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Homebase") != "1" {
			s.writeError(w, r, apperr.New(apperr.Invalid, "Missing X-Homebase header."))
			return
		}
		ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if ct != "multipart/form-data" {
			s.writeError(w, r, apperr.New(apperr.Invalid, "Content-Type must be multipart/form-data."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleUpload streams the file part to disk while hashing it, so a 25 MB
// upload never sits in memory. Fields may come before or after the file.
func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, files.MaxSize+1<<20)
	mr, err := r.MultipartReader()
	if err != nil {
		s.writeError(w, r, apperr.New(apperr.Invalid, "Request body is not multipart."))
		return
	}

	var saved *files.Saved
	var ownerKind, ownerID, name, fileName string
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.writeError(w, r, uploadError(err))
			return
		}
		switch part.FormName() {
		case "file":
			if saved != nil {
				s.writeError(w, r, apperr.New(apperr.Invalid, "Send one file per upload."))
				return
			}
			f, err := s.Blobs.Save(part)
			if err != nil {
				s.writeError(w, r, uploadError(err))
				return
			}
			saved, fileName = &f, part.FileName()
		case "ownerKind", "ownerId", "name":
			v, err := io.ReadAll(io.LimitReader(part, 1024))
			if err != nil {
				s.writeError(w, r, uploadError(err))
				return
			}
			switch part.FormName() {
			case "ownerKind":
				ownerKind = string(v)
			case "ownerId":
				ownerID = string(v)
			default:
				name = string(v)
			}
		}
		_ = part.Close()
	}
	if saved == nil {
		s.writeError(w, r, apperr.New(apperr.Invalid, "No file in the upload."))
		return
	}
	if name == "" {
		name = fileName
	}

	att, err := s.Sync.AttachFile(r.Context(), ownerKind, ownerID, name, *saved)
	if err != nil {
		// Keep the bytes only if another attachment already uses them.
		if _, ferr := s.Sync.File(r.Context(), saved.SHA); ferr != nil {
			_, _ = s.Blobs.Remove(saved.SHA)
		}
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusCreated, att)
}

func uploadError(err error) error {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return apperr.New(apperr.TooLarge, "Files can be at most 25 MB.")
	}
	if _, ok := apperr.As(err); ok {
		return err
	}
	return apperr.New(apperr.Invalid, "The upload was cut short.")
}

// handleFile serves a stored file. Images and PDFs show inline; everything
// else downloads. Either way the browser may not sniff the type, and the
// sandbox policy keeps any script inside from running.
func (s *server) handleFile(w http.ResponseWriter, r *http.Request) {
	sha := r.PathValue("sha")
	if !files.ValidSHA(sha) {
		s.writeError(w, r, apperr.New(apperr.NotFound, "No such file."))
		return
	}
	meta, err := s.Sync.File(r.Context(), sha)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	f, err := s.Blobs.Open(sha)
	if errors.Is(err, fs.ErrNotExist) {
		s.writeError(w, r, apperr.New(apperr.NotFound, "This file was removed after it went unused."))
		return
	}
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	defer f.Close()

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" || len(name) > 300 || strings.ContainsAny(name, "\r\n\"/\\") {
		name = "file"
	}
	disposition := "attachment"
	ctype := "application/octet-stream"
	if files.Inline(meta.Mime) {
		disposition, ctype = "inline", meta.Mime
	}

	h := w.Header()
	h.Set("Content-Type", ctype)
	h.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": name}))
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	// The URL is the content's hash, so it never changes.
	h.Set("Cache-Control", "private, max-age=31536000, immutable")
	h.Set("ETag", `"`+sha+`"`)
	http.ServeContent(w, r, "", time.UnixMilli(meta.CreatedAt), f)
}
