package main

import (
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// GetPrivateFeed
// -----------------------------------------------------------------------------

func TestGetPrivateFeedEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	feed, err := app.GetPrivateFeed()
	if err != nil {
		t.Fatalf("get private feed: %v", err)
	}
	if len(feed) != 0 {
		t.Errorf("expected empty feed, got %d", len(feed))
	}
	if feed == nil {
		t.Error("result should be non-nil empty slice (frontend expects [])")
	}
}

func TestGetPrivateFeedOnlyReturnsPrivateZone(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	now := int64(1_700_000_000)
	if _, err := app.db.Exec(
		`INSERT INTO messages (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, title, body)
		 VALUES (?, ?, '', 0, ?, 1, 0, 'public', ?, 'normal', 'public-title', 'public-body'),
		        (?, ?, '', 0, ?, 2, 0, 'private', ?, 'normal', 'private-title', 'private-body')`,
		"public-1", identity.PublicKey, now, defaultSubID,
		"private-1", identity.PublicKey, now+1, defaultSubID,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	feed, err := app.GetPrivateFeed()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(feed) != 1 {
		t.Fatalf("expected 1 private message, got %d (%+v)", len(feed), feed)
	}
	if feed[0].ID != "private-1" {
		t.Errorf("expected private-1, got %q", feed[0].ID)
	}
	if feed[0].Zone != "private" {
		t.Errorf("zone should be private, got %q", feed[0].Zone)
	}
}

func TestGetPrivateFeedOrderedDesc(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	now := int64(1_700_000_000)
	for i, ts := range []int64{now, now + 100, now + 50} {
		if _, err := app.db.Exec(
			`INSERT INTO messages (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, title, body)
			 VALUES (?, ?, '', 0, ?, ?, 0, 'private', ?, 'normal', 'private', 'body')`,
			"private-"+string(rune('a'+i)), identity.PublicKey, ts, int64(i+1), defaultSubID,
		); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	feed, err := app.GetPrivateFeed()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(feed) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(feed))
	}
	// Most recent first.
	for i := 1; i < len(feed); i++ {
		if feed[i].Timestamp > feed[i-1].Timestamp {
			t.Errorf("private feed not ordered DESC: idx %d=%d > idx %d=%d", i, feed[i].Timestamp, i-1, feed[i-1].Timestamp)
		}
	}
}

// -----------------------------------------------------------------------------
// upsertContentBlob
// -----------------------------------------------------------------------------

func TestUpsertContentBlobValidation(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.upsertContentBlob("", "body", 4); err == nil {
		t.Error("expected error for empty cid")
	}
	if err := app.upsertContentBlob("cid", "", 4); err == nil {
		t.Error("expected error for empty body")
	}
	if err := app.upsertContentBlob("   ", "   ", 4); err == nil {
		t.Error("expected error for whitespace-only inputs")
	}
}

func TestUpsertContentBlobInsertAndOverwrite(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	body1 := "first version of the body"
	if err := app.upsertContentBlob("cid-1", body1, 0); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := app.GetPostBodyByCID("cid-1")
	if err != nil {
		t.Fatalf("get by cid: %v", err)
	}
	if got.Body != body1 {
		t.Errorf("body mismatch: got %q want %q", got.Body, body1)
	}
	// Default sizeBytes (zero passed) should fall back to len(body).
	if got.SizeBytes != int64(len(body1)) {
		t.Errorf("size: got %d, want %d", got.SizeBytes, len(body1))
	}

	// Overwrite with explicit size.
	body2 := "edited body content"
	if err := app.upsertContentBlob("cid-1", body2, int64(len(body2))); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, _ = app.GetPostBodyByCID("cid-1")
	if got.Body != body2 {
		t.Errorf("upsert did not overwrite: got %q", got.Body)
	}
}

// -----------------------------------------------------------------------------
// insertModerationLogIfAbsent
// -----------------------------------------------------------------------------

func TestInsertModerationLogIfAbsentValidation(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	cases := []struct {
		name string
		log  ModerationLog
	}{
		{"missing target", ModerationLog{Action: "SHADOW_BAN", SourceAdmin: "admin", Timestamp: 1}},
		{"missing action", ModerationLog{TargetPubkey: "t", SourceAdmin: "admin", Timestamp: 1}},
		{"missing admin", ModerationLog{TargetPubkey: "t", Action: "SHADOW_BAN", Timestamp: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inserted, err := app.insertModerationLogIfAbsent(tc.log)
			if err == nil {
				t.Error("expected validation error")
			}
			if inserted {
				t.Error("inserted=true on validation error")
			}
		})
	}
}

func TestInsertModerationLogIfAbsentInsertsAndDedupes(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	log := ModerationLog{
		TargetPubkey: "target-1",
		Action:       "SHADOW_BAN",
		SourceAdmin:  "admin-1",
		Timestamp:    1_700_000_000,
		Lamport:      5,
		Reason:       "spam",
		// Result intentionally empty -> defaults to "applied".
	}

	inserted, err := app.insertModerationLogIfAbsent(log)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if !inserted {
		t.Error("first insert should report true")
	}

	// Second identical call should NOT insert (returns false, no error).
	inserted, err = app.insertModerationLogIfAbsent(log)
	if err != nil {
		t.Fatalf("dedupe call: %v", err)
	}
	if inserted {
		t.Error("dedupe should report false for identical log")
	}

	// Verify only a single row exists.
	logs, err := app.GetModerationLogs(100)
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	count := 0
	for _, l := range logs {
		if l.TargetPubkey == "target-1" && l.Action == "SHADOW_BAN" && l.SourceAdmin == "admin-1" {
			count++
			if l.Result != "applied" {
				t.Errorf("expected default result='applied', got %q", l.Result)
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 log row for target-1, got %d", count)
	}
}

func TestInsertModerationLogIfAbsentNormalizesAction(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Lowercase action should be normalized to upper.
	log := ModerationLog{
		TargetPubkey: "target-norm",
		Action:       "  shadow_ban  ",
		SourceAdmin:  "admin-x",
		Timestamp:    1,
	}
	if _, err := app.insertModerationLogIfAbsent(log); err != nil {
		t.Fatalf("insert: %v", err)
	}
	logs, _ := app.GetModerationLogs(100)
	for _, l := range logs {
		if l.TargetPubkey == "target-norm" && l.Action != "SHADOW_BAN" {
			t.Errorf("action should be uppercased: got %q", l.Action)
		}
	}
}

// -----------------------------------------------------------------------------
// listPublicCommentDigestsByPostSince
// -----------------------------------------------------------------------------

func TestListPublicCommentDigestsByPostSinceEmptyPost(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	got, err := app.listPublicCommentDigestsByPostSince("", 0, 100)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty post id should return [], got %d", len(got))
	}
}

func TestListPublicCommentDigestsByPostSinceFiltersByPostAndTimestamp(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "host", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	// Add three comments at increasing timestamps.
	for i, body := range []string{"first", "second", "third"} {
		if _, err := app.AddLocalComment(identity.PublicKey, postID, "", body); err != nil {
			t.Fatalf("comment %d: %v", i, err)
		}
	}

	// Find the timestamp of the second comment to use as a `since` cutoff.
	all, err := app.GetCommentsByPost(postID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(all))
	}
	cutoff := all[1].Timestamp

	digests, err := app.listPublicCommentDigestsByPostSince(postID, cutoff, 100)
	if err != nil {
		t.Fatalf("digests: %v", err)
	}
	for _, d := range digests {
		if d.Timestamp < cutoff {
			t.Errorf("digest at %d is older than cutoff %d", d.Timestamp, cutoff)
		}
		if d.PostID != postID {
			t.Errorf("digest from wrong post: %q", d.PostID)
		}
	}

	// Whitespace post id should be trimmed.
	if _, err := app.listPublicCommentDigestsByPostSince("  "+postID+"  ", 0, 100); err != nil {
		t.Fatalf("trim case: %v", err)
	}
}

func TestListPublicCommentDigestsByPostSinceLimitClampsHigh(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "h", "b", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	if _, err := app.AddLocalComment(identity.PublicKey, postID, "", "c1"); err != nil {
		t.Fatalf("comment: %v", err)
	}

	// limit=99999 should be clamped to 200 (no overflow). Single comment fits comfortably.
	digests, err := app.listPublicCommentDigestsByPostSince(postID, 0, 99999)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(digests) != 1 {
		t.Errorf("expected 1 digest, got %d", len(digests))
	}
}

// -----------------------------------------------------------------------------
// listFavoriteOpsSince
// -----------------------------------------------------------------------------

func TestListFavoriteOpsSinceEmptyPubkey(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	got, err := app.listFavoriteOpsSince("   ", 0, 100)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty pubkey should return [], got %d", len(got))
	}
}

func TestListFavoriteOpsSinceOrdersAscByCreatedAt(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	pubkey := "favorite-owner"
	for i, row := range []struct {
		opID      string
		createdAt int64
	}{
		{"op-c", 300},
		{"op-a", 100},
		{"op-b", 200},
	} {
		if _, err := app.db.Exec(
			`INSERT INTO post_favorite_ops (op_id, pubkey, post_id, op, created_at, signature)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			row.opID, pubkey, "post-x", "ADD", row.createdAt, "sig",
		); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	got, err := app.listFavoriteOpsSince(pubkey, 0, 100)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 ops, got %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].CreatedAt < got[i-1].CreatedAt {
			t.Errorf("ops not ASC ordered: idx %d=%d, idx %d=%d", i, got[i].CreatedAt, i-1, got[i-1].CreatedAt)
		}
	}
}

func TestListFavoriteOpsSinceFiltersByPubkeyAndSince(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.db.Exec(
		`INSERT INTO post_favorite_ops (op_id, pubkey, post_id, op, created_at, signature)
		 VALUES ('op-1', 'pk-A', 'post-1', 'ADD', 100, 's'),
		        ('op-2', 'pk-A', 'post-2', 'ADD', 200, 's'),
		        ('op-3', 'pk-B', 'post-3', 'ADD', 300, 's')`,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := app.listFavoriteOpsSince("pk-A", 150, 100)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 row (pk-A AND created_at>=150), got %d (%+v)", len(got), got)
	}
	if got[0].OpID != "op-2" {
		t.Errorf("expected op-2, got %q", got[0].OpID)
	}

	if !strings.HasPrefix(got[0].Pubkey, "pk-") {
		t.Errorf("pubkey scan: got %q", got[0].Pubkey)
	}
}

func TestListFavoriteOpsSinceClampsNegativeSince(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.db.Exec(
		`INSERT INTO post_favorite_ops (op_id, pubkey, post_id, op, created_at, signature)
		 VALUES ('op-1', 'pk-A', 'post-1', 'ADD', 100, 's')`,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Negative since clamps to 0 (not an error).
	got, err := app.listFavoriteOpsSince("pk-A", -500, 100)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected negative since clamped, got %d rows", len(got))
	}
}
