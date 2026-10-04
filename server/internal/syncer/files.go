package syncer

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"homebase/internal/apperr"
	"homebase/internal/files"
	"homebase/internal/store"
)

// OrphanAge is how long a file nobody uses is kept before its bytes go.
const OrphanAge = 7 * 24 * time.Hour

// AttachFile records an uploaded file and creates the attachment that points
// to it, in one transaction. Exactly one owner is required, and it must be a
// live row.
func (s *Syncer) AttachFile(ctx context.Context, ownerKind, ownerID, name string, f files.Saved) (store.Attachment, error) {
	if !ValidID(ownerID) {
		return store.Attachment{}, invalid("ownerId must be a ULID.")
	}
	name = cleanName(name)

	var att store.Attachment
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		now := s.now()
		at := now.UnixMilli()

		att = store.Attachment{ID: NewID(now), Kind: "file", Name: name, FileSHA: &f.SHA, CreatedAt: at, UpdatedAt: at}
		var deleted *int64
		var err error
		switch ownerKind {
		case string(store.OwnerGoal):
			var g store.Goal
			g, err = tx.Goal(ownerID)
			deleted, att.GoalID = g.DeletedAt, &ownerID
		case string(store.OwnerProject):
			var p store.Project
			p, err = tx.Project(ownerID)
			deleted, att.ProjectID = p.DeletedAt, &ownerID
		case string(store.OwnerTask):
			var t store.Task
			t, err = tx.Task(ownerID)
			deleted, att.TaskID = t.DeletedAt, &ownerID
		default:
			return invalid("ownerKind must be goal, project or task.")
		}
		if errors.Is(err, store.ErrNotFound) || (err == nil && deleted != nil) {
			return apperr.New(apperr.NotFound, "The goal, project or task for this file does not exist.")
		}
		if err != nil {
			return err
		}

		if err := tx.InsertFile(store.File{SHA: f.SHA, Mime: f.Mime, Size: f.Size, CreatedAt: at}); err != nil {
			return err
		}
		if att.Rev, err = tx.NextRev(); err != nil {
			return err
		}
		return tx.PutAttachment(att)
	})
	return att, err
}

// File returns the stored type and size of a file.
func (s *Syncer) File(ctx context.Context, sha string) (store.File, error) {
	var f store.File
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		var err error
		f, err = tx.File(sha)
		if errors.Is(err, store.ErrNotFound) {
			return apperr.New(apperr.NotFound, "No such file.")
		}
		return err
	})
	return f, err
}

// OrphanFiles lists files no attachment has used for OrphanAge. Their rows
// stay, because tombstoned attachments still refer to them; only the bytes
// are removed, and uploading the same file again restores them.
func (s *Syncer) OrphanFiles(ctx context.Context) ([]string, error) {
	var list []string
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		var err error
		list, err = tx.OrphanFiles(s.now().Add(-OrphanAge).UnixMilli())
		return err
	})
	return list, err
}

// cleanName keeps a display name safe to show and to put in a header:
// no path, no control characters, at most 300 characters.
func cleanName(name string) string {
	name = strings.ToValidUTF8(name, "")
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > MaxName {
		name = string([]rune(name)[:MaxName])
	}
	if name == "" {
		name = "File"
	}
	return name
}
