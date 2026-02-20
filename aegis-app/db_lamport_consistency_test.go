package main

import (
	"path/filepath"
	"testing"
)

func newLamportTestApp(t *testing.T) *App {
	t.Helper()
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "lamport_consistency.db"))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("init database: %v", err)
	}
	t.Cleanup(func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	})
	return app
}

func TestPostDeleteRejectsStaleReplay(t *testing.T) {
	app := newLamportTestApp(t)

	postID := "post-stale-replay"
	author := "alice"
	_, err := app.insertMessage(ForumMessage{
		ID:        postID,
		Pubkey:    author,
		OpID:      "op-create-10",
		Title:     "hello",
		Body:      "body-v1",
		Timestamp: 10,
		Lamport:   10,
		Zone:      "public",
		SubID:     defaultSubID,
	})
	if err != nil {
		t.Fatalf("insert initial post: %v", err)
	}

	if err = app.deleteLocalPostAsAuthor(author, postID, 20, 20, "op-delete-20"); err != nil {
		t.Fatalf("delete post: %v", err)
	}

	_, err = app.insertMessage(ForumMessage{
		ID:        postID,
		Pubkey:    author,
		OpID:      "op-stale-15",
		Title:     "resurrect-attempt",
		Body:      "should-not-apply",
		Timestamp: 15,
		Lamport:   15,
		Zone:      "public",
		SubID:     defaultSubID,
	})
	if err != nil {
		t.Fatalf("stale replay should be no-op, got error: %v", err)
	}

	var visibility string
	var lamport int64
	if err = app.db.QueryRow(`SELECT visibility, lamport FROM messages WHERE id = ?;`, postID).Scan(&visibility, &lamport); err != nil {
		t.Fatalf("read post: %v", err)
	}
	if visibility != "deleted" {
		t.Fatalf("expected deleted visibility, got %q", visibility)
	}
	if lamport != 20 {
		t.Fatalf("expected lamport 20 after delete, got %d", lamport)
	}
}

func TestEqualLamportResolvesByOpID(t *testing.T) {
	app := newLamportTestApp(t)

	postID := "post-equal-lamport"
	author := "alice"
	_, err := app.insertMessage(ForumMessage{
		ID:        postID,
		Pubkey:    author,
		OpID:      "op-100-a",
		Title:     "v1",
		Body:      "body-a",
		Timestamp: 100,
		Lamport:   100,
		Zone:      "public",
		SubID:     defaultSubID,
	})
	if err != nil {
		t.Fatalf("insert v1: %v", err)
	}

	_, err = app.insertMessage(ForumMessage{
		ID:        postID,
		Pubkey:    author,
		OpID:      "op-100-z",
		Title:     "v2",
		Body:      "body-z",
		Timestamp: 101,
		Lamport:   100,
		Zone:      "public",
		SubID:     defaultSubID,
	})
	if err != nil {
		t.Fatalf("insert v2: %v", err)
	}

	var title string
	var opID string
	if err = app.db.QueryRow(`SELECT title, current_op_id FROM messages WHERE id = ?;`, postID).Scan(&title, &opID); err != nil {
		t.Fatalf("read post: %v", err)
	}
	if title != "v2" {
		t.Fatalf("expected newer op to win, got title=%q", title)
	}
	if opID != "op-100-z" {
		t.Fatalf("expected current_op_id op-100-z, got %q", opID)
	}
}

func TestDigestTombstonePreventsOfflineResurrection(t *testing.T) {
	app := newLamportTestApp(t)

	postID := "post-offline-replay"
	author := "alice"
	if _, err := app.upsertPublicPostIndexFromDigest(SyncPostDigest{
		ID:        postID,
		Pubkey:    author,
		OpID:      "op-del-50",
		OpType:    postOpTypeDelete,
		Deleted:   true,
		Lamport:   50,
		Timestamp: 50,
		SubID:     defaultSubID,
	}); err != nil {
		t.Fatalf("apply tombstone: %v", err)
	}

	if _, err := app.upsertPublicPostIndexFromDigest(SyncPostDigest{
		ID:         postID,
		Pubkey:     author,
		OpID:       "op-create-40",
		OpType:     postOpTypeCreate,
		Deleted:    false,
		Title:      "stale",
		ContentCID: "cid-40",
		Lamport:    40,
		Timestamp:  40,
		SubID:      defaultSubID,
	}); err != nil {
		t.Fatalf("apply stale digest: %v", err)
	}

	var visibility string
	var lamport int64
	if err := app.db.QueryRow(`SELECT visibility, lamport FROM messages WHERE id = ?;`, postID).Scan(&visibility, &lamport); err != nil {
		t.Fatalf("read post: %v", err)
	}
	if visibility != "deleted" {
		t.Fatalf("expected tombstone to remain, got visibility=%q", visibility)
	}
	if lamport != 50 {
		t.Fatalf("expected lamport 50, got %d", lamport)
	}
}
