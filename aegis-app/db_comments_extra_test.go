package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// DownvoteComment
// -----------------------------------------------------------------------------

func TestDownvoteCommentSetsNegativeScore(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "to be downvoted")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	if err := app.DownvoteComment(comment.ID); err != nil {
		t.Fatalf("downvote: %v", err)
	}

	comments, err := app.GetCommentsByPost(postID)
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Score != -1 {
		t.Errorf("expected score -1 after downvote, got %d", comments[0].Score)
	}
}

func TestDownvoteCommentTogglesFromUpvote(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	comment, _ := app.AddLocalComment(identity.PublicKey, postID, "", "body")

	if err := app.UpvoteComment(comment.ID); err != nil {
		t.Fatalf("upvote: %v", err)
	}
	comments, _ := app.GetCommentsByPost(postID)
	if comments[0].Score != 1 {
		t.Fatalf("expected score 1 after upvote, got %d", comments[0].Score)
	}

	// Downvoting after an upvote should net to -1 (delta of 2: +1 -> -1).
	if err := app.DownvoteComment(comment.ID); err != nil {
		t.Fatalf("downvote: %v", err)
	}
	comments, _ = app.GetCommentsByPost(postID)
	if comments[0].Score != -1 {
		t.Errorf("expected score -1 after switching to downvote, got %d", comments[0].Score)
	}
}

func TestDownvoteCommentRequiresIdentity(t *testing.T) {
	app := NewApp()
	dbPath := t.TempDir() + "/test.db"
	app.SetDatabasePath(dbPath)
	if err := app.initDatabase(); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		if app.db != nil {
			_, _ = app.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
			_ = app.db.Close()
		}
	})

	if err := app.DownvoteComment("any-comment-id"); err == nil {
		t.Error("expected error when local identity is missing")
	}
}

// -----------------------------------------------------------------------------
// UpdateLocalComment
// -----------------------------------------------------------------------------

func TestUpdateLocalCommentChangesBody(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	original, err := app.AddLocalComment(identity.PublicKey, postID, "", "original body")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	updated, err := app.UpdateLocalComment(identity.PublicKey, original.ID, "edited body")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Body != "edited body" {
		t.Errorf("expected new body, got %q", updated.Body)
	}
	if updated.ID != original.ID {
		t.Errorf("update should keep the same id, got %q vs %q", updated.ID, original.ID)
	}
	if updated.PostID != original.PostID {
		t.Errorf("postID should be preserved")
	}
	if updated.Lamport <= original.Lamport {
		t.Errorf("lamport should advance after edit, got %d (was %d)", updated.Lamport, original.Lamport)
	}
}

func TestUpdateLocalCommentRejectsNonAuthor(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	comment, _ := app.AddLocalComment(identity.PublicKey, postID, "", "original")

	_, mnemonicPubkey := generateRemoteIdentity(t)
	_, err := app.UpdateLocalComment(mnemonicPubkey, comment.ID, "tampered body")
	if err == nil {
		t.Fatal("expected non-author update to be rejected")
	}
	if !strings.Contains(err.Error(), "only author") {
		t.Errorf("expected 'only author' error, got %v", err)
	}
}

func TestUpdateLocalCommentNotFound(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	_, err := app.UpdateLocalComment(identity.PublicKey, "does-not-exist", "anything")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' message, got %v", err)
	}
}

func TestUpdateLocalCommentRequiresPubkeyAndID(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if _, err := app.UpdateLocalComment("   ", "comment-1", "body"); err == nil {
		t.Error("expected error for empty pubkey")
	}
	if _, err := app.UpdateLocalComment(identity.PublicKey, "   ", "body"); err == nil {
		t.Error("expected error for empty comment id")
	}
}

// -----------------------------------------------------------------------------
// StoreCommentImageDataURL
// -----------------------------------------------------------------------------

func TestStoreCommentImageDataURLRejectsEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.StoreCommentImageDataURL(""); err == nil {
		t.Error("expected error for empty payload")
	}
	if _, err := app.StoreCommentImageDataURL("   "); err == nil {
		t.Error("expected error for whitespace payload")
	}
}

func TestStoreCommentImageDataURLRejectsNonDataURL(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.StoreCommentImageDataURL("https://example.com/x.png"); err == nil {
		t.Error("expected error for non-data URL")
	}
}

func TestStoreCommentImageDataURLRejectsNonBase64(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.StoreCommentImageDataURL("data:image/png,raw-bytes"); err == nil {
		t.Error("expected error when base64 marker missing")
	}
}

func TestStoreCommentImageDataURLRejectsBadEncoding(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.StoreCommentImageDataURL("data:image/png;base64,not-valid-base64==="); err == nil {
		t.Error("expected error for malformed base64")
	}
}

func TestStoreCommentImageDataURLAcceptsArbitraryBytes(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// prepareImageAssets falls back to passing source through verbatim
	// when image.Decode fails. That's the documented behavior we lock in.
	raw := []byte("synthetic-payload-for-storage")
	encoded := base64.StdEncoding.EncodeToString(raw)
	dataURL := "data:image/png;base64," + encoded

	att, err := app.StoreCommentImageDataURL(dataURL)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if att.Kind != "media_cid" {
		t.Errorf("expected kind=media_cid, got %q", att.Kind)
	}
	if att.Ref == "" {
		t.Error("expected non-empty CID ref")
	}
	if att.SizeBytes != int64(len(raw)) {
		t.Errorf("size bytes should reflect payload length, got %d (want %d)", att.SizeBytes, len(raw))
	}

	// Verify the blob was actually stored under the returned CID.
	media, err := app.GetMediaByCID(att.Ref)
	if err != nil {
		t.Fatalf("retrieve stored blob: %v", err)
	}
	if media.Mime == "" {
		t.Error("retrieved media should report a MIME type")
	}
}
