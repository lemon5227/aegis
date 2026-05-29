package main

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// resolveVoteBroadcastDebounce
// -----------------------------------------------------------------------------

func TestResolveVoteBroadcastDebounce(t *testing.T) {
	cases := []struct {
		env  string
		want time.Duration
	}{
		{"", 600 * time.Millisecond},                  // unset -> default
		{"   ", 600 * time.Millisecond},               // whitespace -> default
		{"not-a-number", 600 * time.Millisecond},      // invalid -> default
		{"50", 100 * time.Millisecond},                // below floor -> floored to 100
		{"100", 100 * time.Millisecond},               // exact floor
		{"500", 500 * time.Millisecond},               // valid mid-range
		{"3000", 3000 * time.Millisecond},             // exact ceiling
		{"99999", 3000 * time.Millisecond},            // above ceiling -> capped at 3000
		{"-100", 100 * time.Millisecond},              // negative -> floored to 100
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("AEGIS_VOTE_BROADCAST_DEBOUNCE_MS", tc.env)
			got := resolveVoteBroadcastDebounce()
			if got != tc.want {
				t.Errorf("debounce(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// scheduleVoteStateBroadcast (early-return guards)
//
// We don't assert on the goroutine's eventual outbox enqueue here; that path
// involves a debounce and is exercised by the integration tests that drive
// votes through ProcessIncomingMessage. We just lock down the input
// validation path so empty inputs never panic or schedule a goroutine that
// reads from a nil/empty key.
// -----------------------------------------------------------------------------

func TestScheduleVoteStateBroadcastEarlyReturnOnEmptyInputs(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// All of these should return immediately without scheduling a goroutine
	// or panicking on the (unkeyed) sequence map.
	app.scheduleVoteStateBroadcast("", "post-1", "")
	app.scheduleVoteStateBroadcast("   ", "post-1", "")
	app.scheduleVoteStateBroadcast("voter", "", "")
	app.scheduleVoteStateBroadcast("voter", "  ", "  ")

	app.voteBroadcastMu.Lock()
	if len(app.voteBroadcastSeq) != 0 {
		t.Errorf("guard rejection should not seed sequence map, got %v", app.voteBroadcastSeq)
	}
	app.voteBroadcastMu.Unlock()
}

func TestScheduleVoteStateBroadcastSeedsSequenceMap(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	app.scheduleVoteStateBroadcast("voter", "post-1", "")

	app.voteBroadcastMu.Lock()
	defer app.voteBroadcastMu.Unlock()
	if len(app.voteBroadcastSeq) == 0 {
		t.Error("expected sequence map to record the scheduled broadcast")
	}
	for k, seq := range app.voteBroadcastSeq {
		if seq < 1 {
			t.Errorf("sequence %q has bad value %d", k, seq)
		}
	}
}

// -----------------------------------------------------------------------------
// PublishPostStructured (default-sub wrapper)
// -----------------------------------------------------------------------------

func TestPublishPostStructuredRoutesToDefaultSub(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructured(identity.PublicKey, "Title", "Body"); err != nil {
		t.Fatalf("publish: %v", err)
	}

	feed, err := app.GetFeedBySub(defaultSubID)
	if err != nil {
		t.Fatalf("feed: %v", err)
	}
	if len(feed) != 1 {
		t.Fatalf("expected 1 post in default sub, got %d", len(feed))
	}
	if feed[0].SubID != defaultSubID {
		t.Errorf("expected sub id %q, got %q", defaultSubID, feed[0].SubID)
	}
}

func TestPublishPostStructuredRejectsEmptyPubkey(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.PublishPostStructured("   ", "T", "B"); err == nil {
		t.Error("expected error for empty pubkey")
	}
}

func TestPublishPostStructuredDerivesTitleFromBody(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	// Empty title should be derived from the body (first 20 runes).
	if err := app.PublishPostStructured(identity.PublicKey, "", "this body is the title source"); err != nil {
		t.Fatalf("publish: %v", err)
	}
	feed, _ := app.GetFeedBySub(defaultSubID)
	if len(feed) != 1 {
		t.Fatalf("expected 1 post, got %d", len(feed))
	}
	if feed[0].Title == "" {
		t.Error("title should have been derived from body")
	}
}

// -----------------------------------------------------------------------------
// AddLocalPostWithImageToSub
// -----------------------------------------------------------------------------

// makeTestPNGBase64 returns a base64-encoded PNG of the given size, useful for
// AddLocalPostWithImageToSub tests. Uses the helpers from db_image_test.go.
func makeTestPNGBase64(t *testing.T, w, h int) string {
	t.Helper()
	src := newOpaqueImage(w, h)
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatalf("encode test PNG: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestAddLocalPostWithImageToSubHappyPath(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	encoded := makeTestPNGBase64(t, 32, 16)

	post, err := app.AddLocalPostWithImageToSub(
		identity.PublicKey, "Title", "Body content", "public", defaultSubID,
		encoded, "image/png",
	)
	if err != nil {
		t.Fatalf("add post with image: %v", err)
	}
	if post.ImageCID == "" {
		t.Error("expected image CID to be populated")
	}
	if post.ImageWidth != 32 || post.ImageHeight != 16 {
		t.Errorf("dimensions mismatch: got %dx%d", post.ImageWidth, post.ImageHeight)
	}
	if post.ImageMIME == "" {
		t.Error("image mime should be populated")
	}
	if post.ImageSize <= 0 {
		t.Errorf("expected positive image size, got %d", post.ImageSize)
	}

	// Verify the blob is retrievable.
	if has, err := app.hasMediaBlobLocal(post.ImageCID); err != nil || !has {
		t.Errorf("image blob not retrievable: has=%v err=%v", has, err)
	}
}

func TestAddLocalPostWithImageToSubRejectsInvalidBase64(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	_, err := app.AddLocalPostWithImageToSub(
		identity.PublicKey, "T", "B", "public", defaultSubID,
		"%%not-base64%%", "image/png",
	)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestAddLocalPostWithImageToSubRejectsEmptyImage(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	_, err := app.AddLocalPostWithImageToSub(
		identity.PublicKey, "T", "B", "public", defaultSubID,
		"", "image/png",
	)
	if err == nil {
		t.Error("expected error for empty image payload")
	}
}

func TestAddLocalPostWithImageToSubLargeImageGetsThumb(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	// 1000x500 source - large enough to produce a separate thumbnail (max edge 320 for thumb).
	encoded := makeTestPNGBase64(t, 1000, 500)

	post, err := app.AddLocalPostWithImageToSub(
		identity.PublicKey, "T", "B", "public", defaultSubID,
		encoded, "image/png",
	)
	if err != nil {
		t.Fatalf("add post: %v", err)
	}
	if post.ImageCID == "" || post.ThumbCID == "" {
		t.Error("expected both image and thumb CIDs to be populated")
	}
	if post.ImageCID == post.ThumbCID {
		t.Error("for a large image, thumb CID should differ from image CID")
	}
	if has, _ := app.hasMediaBlobLocal(post.ThumbCID); !has {
		t.Error("thumbnail blob should be stored")
	}
}

// -----------------------------------------------------------------------------
// queryPostsBySubSet
// -----------------------------------------------------------------------------

func TestQueryPostsBySubSetEmptyList(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	got, err := app.queryPostsBySubSet(identity.PublicKey, nil, 10)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("nil sub list should return [], got %d", len(got))
	}
	if got == nil {
		t.Error("result should be non-nil empty slice")
	}

	got, _ = app.queryPostsBySubSet(identity.PublicKey, []string{}, 10)
	if len(got) != 0 {
		t.Errorf("empty sub list should return [], got %d", len(got))
	}
}

func TestQueryPostsBySubSetFiltersBySubAndVisibility(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	now := int64(1_700_000_000)
	otherAuthor, _ := generateRemoteIdentity(t)

	// 4 posts:
	//   sub-a, normal, by other         -> visible
	//   sub-a, deleted, by other        -> hidden (visibility filter)
	//   sub-b, normal, by viewer        -> visible
	//   sub-c, normal, by other         -> hidden (sub filter)
	rows := []struct {
		id, pubkey, sub, vis string
	}{
		{"p-a-other", otherAuthor, "sub-a", "normal"},
		{"p-a-deleted", otherAuthor, "sub-a", "deleted"},
		{"p-b-viewer", identity.PublicKey, "sub-b", "normal"},
		{"p-c-other", otherAuthor, "sub-c", "normal"},
	}
	for i, r := range rows {
		if _, err := app.db.Exec(
			`INSERT INTO messages (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, title, body)
			 VALUES (?, ?, '', 0, ?, ?, 0, 'public', ?, ?, 'title', 'body')`,
			r.id, r.pubkey, now+int64(i), int64(i+1), r.sub, r.vis,
		); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	got, err := app.queryPostsBySubSet(identity.PublicKey, []string{"sub-a", "sub-b"}, 10)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	gotIDs := map[string]bool{}
	for _, p := range got {
		gotIDs[p.ID] = true
	}
	if !gotIDs["p-a-other"] {
		t.Error("expected p-a-other (sub-a, normal) in result")
	}
	if !gotIDs["p-b-viewer"] {
		t.Error("expected p-b-viewer (sub-b, normal, by viewer) in result")
	}
	if gotIDs["p-a-deleted"] {
		t.Error("p-a-deleted (other author, deleted) should be filtered out")
	}
	if gotIDs["p-c-other"] {
		t.Error("p-c-other (sub-c not in list) should be filtered out")
	}
}

func TestQueryPostsBySubSetAuthorSeesOwnNonDeleted(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	now := int64(1_700_000_000)
	// Viewer's own posts: 'shadowed' is non-deleted -> visible to author.
	if _, err := app.db.Exec(
		`INSERT INTO messages (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, title, body)
		 VALUES (?, ?, '', 0, ?, 1, 0, 'public', ?, 'shadowed', 'title', 'body'),
		        (?, ?, '', 0, ?, 2, 0, 'public', ?, 'deleted',  'title', 'body')`,
		"own-shadow", identity.PublicKey, now, "sub-x",
		"own-deleted", identity.PublicKey, now+1, "sub-x",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := app.queryPostsBySubSet(identity.PublicKey, []string{"sub-x"}, 10)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	ids := map[string]bool{}
	for _, p := range got {
		ids[p.ID] = true
	}
	if !ids["own-shadow"] {
		t.Error("author should see own shadowed (non-deleted) post")
	}
	if ids["own-deleted"] {
		t.Error("author should NOT see own deleted post")
	}
}

func TestQueryPostsBySubSetDefaultLimitWhenNonPositive(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	// Insert 25 posts so we can verify limit defaults to 20.
	now := int64(1_700_000_000)
	for i := 0; i < 25; i++ {
		if _, err := app.db.Exec(
			`INSERT INTO messages (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, title, body)
			 VALUES (?, ?, '', 0, ?, ?, 0, 'public', ?, 'normal', 't', 'b')`,
			"limit-post-"+string(rune('a'+i%26))+string(rune('0'+i/10)), identity.PublicKey, now+int64(i), int64(i+1), defaultSubID,
		); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	got, err := app.queryPostsBySubSet(identity.PublicKey, []string{defaultSubID}, 0)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 20 {
		t.Errorf("expected default limit 20, got %d", len(got))
	}
}

func TestQueryPostsBySubSetNormalizesSubID(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	now := int64(1_700_000_000)
	if _, err := app.db.Exec(
		`INSERT INTO messages (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, title, body)
		 VALUES ('p-1', ?, '', 0, ?, 1, 0, 'public', 'my-sub', 'normal', 't', 'b')`,
		identity.PublicKey, now,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Caller passes "MY-SUB" with whitespace; normalizeSubID lowercases & trims.
	got, err := app.queryPostsBySubSet(identity.PublicKey, []string{"  MY-SUB  "}, 10)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected normalized lookup to match, got %d", len(got))
	}
}
