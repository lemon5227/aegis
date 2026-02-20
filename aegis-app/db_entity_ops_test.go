package main

import (
	"path/filepath"
	"testing"
	"time"
)

func newEntityOpsTestApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("AEGIS_DEV_MODE", "1")
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "entity_ops.db"))
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

func TestEntityOpsLoggedForPostAndCommentLifecycle(t *testing.T) {
	app := newEntityOpsTestApp(t)
	author := "alice"
	postID := "post-entity-ops"
	commentID := "comment-entity-ops"

	_, err := app.insertMessage(ForumMessage{
		ID:        postID,
		Pubkey:    author,
		OpID:      postID + ":alice:10:create",
		Title:     "hello",
		Body:      "v1",
		Timestamp: 10,
		Lamport:   10,
		Zone:      "public",
		SubID:     defaultSubID,
	})
	if err != nil {
		t.Fatalf("insert post create: %v", err)
	}

	_, err = app.insertMessage(ForumMessage{
		ID:        postID,
		Pubkey:    author,
		OpID:      postID + ":alice:11:update",
		Title:     "hello-2",
		Body:      "v2",
		Timestamp: 11,
		Lamport:   11,
		Zone:      "public",
		SubID:     defaultSubID,
	})
	if err != nil {
		t.Fatalf("insert post update: %v", err)
	}

	_, err = app.insertComment(Comment{
		ID:        commentID,
		PostID:    postID,
		Pubkey:    author,
		OpID:      commentID + ":alice:12:create",
		Body:      "comment-v1",
		Timestamp: 12,
		Lamport:   12,
	})
	if err != nil {
		t.Fatalf("insert comment create: %v", err)
	}

	if _, err = app.deleteLocalCommentAsAuthor(author, commentID, 13, 13, commentID+":alice:13:delete"); err != nil {
		t.Fatalf("delete comment: %v", err)
	}
	if err = app.deleteLocalPostAsAuthor(author, postID, 14, 14, postID+":alice:14:delete"); err != nil {
		t.Fatalf("delete post: %v", err)
	}

	records, err := app.ListEntityOps("", "", 100)
	if err != nil {
		t.Fatalf("list entity ops: %v", err)
	}
	if len(records) < 5 {
		t.Fatalf("expected at least 5 entity ops, got %d", len(records))
	}
}

func TestRunTombstoneGCTwoStablePasses(t *testing.T) {
	app := newEntityOpsTestApp(t)
	author := "alice"
	postID := "post-gc"
	commentID := "comment-gc"

	_, err := app.insertMessage(ForumMessage{
		ID:        postID,
		Pubkey:    author,
		OpID:      postID + ":alice:10:create",
		Title:     "t",
		Body:      "b",
		Timestamp: 10,
		Lamport:   10,
		Zone:      "public",
		SubID:     defaultSubID,
	})
	if err != nil {
		t.Fatalf("insert post create: %v", err)
	}
	_, err = app.insertComment(Comment{
		ID:        commentID,
		PostID:    postID,
		Pubkey:    author,
		OpID:      commentID + ":alice:11:create",
		Body:      "c",
		Timestamp: 11,
		Lamport:   11,
	})
	if err != nil {
		t.Fatalf("insert comment create: %v", err)
	}

	if err = app.deleteLocalPostAsAuthor(author, postID, 12, 12, postID+":alice:12:delete"); err != nil {
		t.Fatalf("delete post: %v", err)
	}
	if _, err = app.deleteLocalCommentAsAuthor(author, commentID, 13, 13, commentID+":alice:13:delete"); err != nil {
		t.Fatalf("delete comment: %v", err)
	}

	oldTs := time.Now().Unix() - 40*24*3600
	if _, err = app.db.Exec(`UPDATE messages SET deleted_at_ts = ?, timestamp = ? WHERE id = ?;`, oldTs, oldTs, postID); err != nil {
		t.Fatalf("age post tombstone: %v", err)
	}
	if _, err = app.db.Exec(`UPDATE comments SET deleted_at = ?, timestamp = ? WHERE id = ?;`, oldTs, oldTs, commentID); err != nil {
		t.Fatalf("age comment tombstone: %v", err)
	}

	res1, err := app.RunTombstoneGC(30, 2, 100)
	if err != nil {
		t.Fatalf("run gc first pass: %v", err)
	}
	if res1.DeletedPosts != 0 || res1.DeletedComments != 0 {
		t.Fatalf("first pass should not delete, got posts=%d comments=%d", res1.DeletedPosts, res1.DeletedComments)
	}

	res2, err := app.RunTombstoneGC(30, 2, 100)
	if err != nil {
		t.Fatalf("run gc second pass: %v", err)
	}
	if res2.DeletedPosts != 1 || res2.DeletedComments != 1 {
		t.Fatalf("second pass should delete one post and one comment, got posts=%d comments=%d", res2.DeletedPosts, res2.DeletedComments)
	}
}
