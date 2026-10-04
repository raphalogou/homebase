package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"homebase/internal/files"
)

type upload struct {
	ownerKind, ownerID, name string
	fileName                 string
	body                     io.Reader
	fileFirst                bool
	noHeader                 bool
}

func doUpload(h http.Handler, token string, u upload) *httptest.ResponseRecorder {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		fields := func() {
			_ = mw.WriteField("ownerKind", u.ownerKind)
			_ = mw.WriteField("ownerId", u.ownerID)
			if u.name != "" {
				_ = mw.WriteField("name", u.name)
			}
		}
		if !u.fileFirst {
			fields()
		}
		if u.body != nil {
			fw, _ := mw.CreateFormFile("file", u.fileName)
			_, _ = io.Copy(fw, u.body)
		}
		if u.fileFirst {
			fields()
		}
		_ = mw.Close()
		_ = pw.Close()
	}()
	r := httptest.NewRequest("POST", "/api/files", pr)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	if !u.noHeader {
		r.Header.Set("X-Homebase", "1")
	}
	r.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// seedOwners creates a project, a task and a deleted task through sync.
func seedOwners(t *testing.T, h http.Handler, token string) {
	t.Helper()
	now := time.Now().UnixMilli()
	body := fmt.Sprintf(`{"base":0,"ops":[
		{"op":"upsert","table":"projects","id":"01HZZZZZZZZZZZZZZZZZZZ0001","updatedAt":%d,"row":{"title":"Budgeting app","status":"open"}},
		{"op":"upsert","table":"tasks","id":"01HZZZZZZZZZZZZZZZZZZZ0002","updatedAt":%d,"row":{"title":"Import","status":"open"}},
		{"op":"upsert","table":"tasks","id":"01HZZZZZZZZZZZZZZZZZZZ0003","updatedAt":%d,"row":{"title":"Gone","status":"open"}},
		{"op":"delete","table":"tasks","id":"01HZZZZZZZZZZZZZZZZZZZ0003","updatedAt":%d}
	]}`, now, now, now, now+1)
	if rec := do(h, call{method: "POST", path: "/api/sync", body: body, cookie: token}); rec.Code != 200 {
		t.Fatalf("seed: %d %s", rec.Code, rec.Body)
	}
}

const project = "01HZZZZZZZZZZZZZZZZZZZ0001"

func TestUploadOwnerChecks(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	seedOwners(t, h, token)

	tests := []struct {
		name     string
		u        upload
		wantCode int
		wantErr  string
	}{
		{"project", upload{ownerKind: "project", ownerID: project}, 201, ""},
		{"task, file before fields", upload{ownerKind: "task", ownerID: "01HZZZZZZZZZZZZZZZZZZZ0002", fileFirst: true}, 201, ""},
		{"unknown kind", upload{ownerKind: "attachment", ownerID: project}, 400, "invalid"},
		{"kind does not match id", upload{ownerKind: "goal", ownerID: project}, 404, "not_found"},
		{"deleted owner", upload{ownerKind: "task", ownerID: "01HZZZZZZZZZZZZZZZZZZZ0003"}, 404, "not_found"},
		{"no owner", upload{}, 400, "invalid"},
		{"bad owner id", upload{ownerKind: "project", ownerID: "../x"}, 400, "invalid"},
		{"no X-Homebase", upload{ownerKind: "project", ownerID: project, noHeader: true}, 400, "invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.u.fileName, tt.u.body = "notes.txt", strings.NewReader("hello "+tt.name)
			rec := doUpload(h, token, tt.u)
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.wantCode, rec.Body)
			}
			if tt.wantErr != "" && errCode(t, rec) != tt.wantErr {
				t.Errorf("code = %s, want %s", errCode(t, rec), tt.wantErr)
			}
		})
	}

	rec := doUpload(h, token, upload{ownerKind: "project", ownerID: project})
	if rec.Code != 400 {
		t.Errorf("upload without a file part = %d, want 400", rec.Code)
	}
}

func TestUploadSizeLimit(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	seedOwners(t, h, token)

	ok := doUpload(h, token, upload{ownerKind: "project", ownerID: project, fileName: "big.bin",
		body: bytes.NewReader(make([]byte, files.MaxSize))})
	if ok.Code != 201 {
		t.Fatalf("25 MB upload = %d: %s", ok.Code, ok.Body)
	}
	tooBig := doUpload(h, token, upload{ownerKind: "project", ownerID: project, fileName: "big.bin",
		body: bytes.NewReader(make([]byte, files.MaxSize+1))})
	if tooBig.Code != http.StatusRequestEntityTooLarge || errCode(t, tooBig) != "too_large" {
		t.Fatalf("25 MB + 1 byte = %d %s, want 413 too_large", tooBig.Code, tooBig.Body)
	}
}

func TestSameFileTwiceStoredOnce(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	seedOwners(t, h, token)

	var shas []string
	var ids []string
	for i := range 2 {
		rec := doUpload(h, token, upload{ownerKind: "project", ownerID: project,
			fileName: fmt.Sprintf("copy-%d.pdf", i), body: strings.NewReader("%PDF-1.7 same bytes")})
		if rec.Code != 201 {
			t.Fatalf("upload %d: %d %s", i, rec.Code, rec.Body)
		}
		var att struct {
			ID      string
			Name    string
			FileSha string
			Kind    string
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &att); err != nil {
			t.Fatal(err)
		}
		if att.Kind != "file" || att.Name != fmt.Sprintf("copy-%d.pdf", i) {
			t.Errorf("attachment = %+v", att)
		}
		shas = append(shas, att.FileSha)
		ids = append(ids, att.ID)
	}
	if shas[0] != shas[1] || ids[0] == ids[1] {
		t.Fatalf("want one file and two attachments, got shas %v ids %v", shas, ids)
	}

	// Both attachments arrive through sync like any other row.
	rec := do(h, call{method: "GET", path: "/api/sync?since=0", cookie: token})
	if n := strings.Count(rec.Body.String(), shas[0]); n != 2 {
		t.Errorf("sync shows the file %d times, want 2", n)
	}
}

func TestServingFiles(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	seedOwners(t, h, token)

	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1}, 64)...)
	tests := []struct {
		name            string
		body            []byte
		wantType        string
		wantDisposition string
	}{
		{"hostile html downloads", []byte("<html><script>alert(document.cookie)</script></html>"), "application/octet-stream", "attachment"},
		{"svg downloads", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), "application/octet-stream", "attachment"},
		{"png shows inline", png, "image/png", "inline"},
		{"pdf shows inline", []byte("%PDF-1.7\n%..."), "application/pdf", "inline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up := doUpload(h, token, upload{ownerKind: "project", ownerID: project, fileName: "x", body: bytes.NewReader(tt.body)})
			if up.Code != 201 {
				t.Fatalf("upload: %d %s", up.Code, up.Body)
			}
			var att struct{ FileSha string }
			_ = json.Unmarshal(up.Body.Bytes(), &att)

			rec := do(h, call{method: "GET", path: "/api/files/" + att.FileSha + "?name=report.html", cookie: token})
			if rec.Code != 200 {
				t.Fatalf("get: %d", rec.Code)
			}
			hdr := rec.Header()
			if hdr.Get("Content-Type") != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", hdr.Get("Content-Type"), tt.wantType)
			}
			if !strings.HasPrefix(hdr.Get("Content-Disposition"), tt.wantDisposition) {
				t.Errorf("Content-Disposition = %q, want %s", hdr.Get("Content-Disposition"), tt.wantDisposition)
			}
			if hdr.Get("X-Content-Type-Options") != "nosniff" || hdr.Get("Content-Security-Policy") != "default-src 'none'; sandbox" {
				t.Errorf("security headers: %v", hdr)
			}
			if !bytes.Equal(rec.Body.Bytes(), tt.body) {
				t.Error("body differs from upload")
			}
		})
	}

	if rec := do(h, call{method: "GET", path: "/api/files/" + strings.Repeat("a", 64)}); rec.Code != 401 {
		t.Errorf("without a session = %d, want 401", rec.Code)
	}
	if rec := do(h, call{method: "GET", path: "/api/files/" + strings.Repeat("a", 64), cookie: token}); rec.Code != 404 {
		t.Errorf("unknown file = %d, want 404", rec.Code)
	}
}
