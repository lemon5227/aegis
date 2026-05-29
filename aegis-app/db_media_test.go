package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// upsertMediaBlobRaw
// -----------------------------------------------------------------------------

func TestUpsertMediaBlobRawValidation(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.upsertMediaBlobRaw("", "image/png", []byte("x"), 1, 1, false); err == nil {
		t.Error("expected error for empty cid")
	}
	if err := app.upsertMediaBlobRaw("cid-1", "image/png", nil, 1, 1, false); err == nil {
		t.Error("expected error for empty data")
	}
	if err := app.upsertMediaBlobRaw("cid-1", "image/png", []byte{}, 1, 1, false); err == nil {
		t.Error("expected error for zero-length data")
	}
}

func TestUpsertMediaBlobRawInsertAndRead(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	payload := []byte("synthetic-image-bytes")
	cid := buildBinaryCID(payload)

	if err := app.upsertMediaBlobRaw(cid, "image/png", payload, 64, 32, false); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	media, raw, err := app.getMediaBlobRawLocal(cid)
	if err != nil {
		t.Fatalf("get raw: %v", err)
	}
	if media.ContentCID != cid {
		t.Errorf("cid mismatch: got %q", media.ContentCID)
	}
	if media.Mime != "image/png" {
		t.Errorf("mime mismatch: got %q", media.Mime)
	}
	if media.SizeBytes != int64(len(payload)) {
		t.Errorf("size mismatch: got %d", media.SizeBytes)
	}
	if media.Width != 64 || media.Height != 32 {
		t.Errorf("dimensions mismatch: got %dx%d", media.Width, media.Height)
	}
	if media.IsThumbnail {
		t.Error("expected IsThumbnail=false")
	}
	if string(raw) != string(payload) {
		t.Errorf("raw bytes mismatch")
	}
	if media.DataBase64 != base64.StdEncoding.EncodeToString(payload) {
		t.Errorf("DataBase64 should match base64 of raw")
	}
}

func TestUpsertMediaBlobRawOverwritesExisting(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	cid := "cid-overwrite"
	if err := app.upsertMediaBlobRaw(cid, "image/png", []byte("first"), 10, 10, false); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if err := app.upsertMediaBlobRaw(cid, "image/jpeg", []byte("second-payload"), 20, 30, true); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	media, _, err := app.getMediaBlobRawLocal(cid)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if media.Mime != "image/jpeg" {
		t.Errorf("mime should be updated, got %q", media.Mime)
	}
	if media.SizeBytes != int64(len("second-payload")) {
		t.Errorf("size should reflect new payload, got %d", media.SizeBytes)
	}
	if !media.IsThumbnail {
		t.Error("IsThumbnail should be true after upsert")
	}
}

func TestGetMediaBlobLocalDropsRawCopy(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	cid := "cid-wrapper"
	if err := app.upsertMediaBlobRaw(cid, "image/png", []byte("x"), 1, 1, false); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	media, err := app.getMediaBlobLocal(cid)
	if err != nil {
		t.Fatalf("get local: %v", err)
	}
	if media.ContentCID != cid {
		t.Errorf("cid mismatch: got %q", media.ContentCID)
	}
	// DataBase64 must still be populated even though raw is no longer surfaced.
	if media.DataBase64 == "" {
		t.Error("DataBase64 should be populated by the wrapper")
	}
}

// -----------------------------------------------------------------------------
// hasMediaBlobLocal
// -----------------------------------------------------------------------------

func TestHasMediaBlobLocal(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if got, err := app.hasMediaBlobLocal(""); err != nil || got {
		t.Errorf("empty cid should be false/nil, got %v err=%v", got, err)
	}
	if got, err := app.hasMediaBlobLocal("not-stored-yet"); err != nil || got {
		t.Errorf("missing cid should be false, got %v err=%v", got, err)
	}

	if err := app.upsertMediaBlobRaw("cid-present", "image/png", []byte("x"), 1, 1, false); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := app.hasMediaBlobLocal("  cid-present  ") // trim tolerance
	if err != nil {
		t.Fatalf("has: %v", err)
	}
	if !got {
		t.Error("expected true for present cid (with whitespace)")
	}
}

// -----------------------------------------------------------------------------
// GetMediaByCID
// -----------------------------------------------------------------------------

func TestGetMediaByCIDValidation(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.GetMediaByCID(""); err == nil {
		t.Error("expected error for empty cid")
	}
	if _, err := app.GetMediaByCID("   "); err == nil {
		t.Error("expected error for whitespace cid")
	}
}

func TestGetMediaByCIDLocalCacheHit(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	beforeMetrics := app.GetReleaseMetrics()
	beforeHits := beforeMetrics.BlobCacheHits

	cid := "cid-cached"
	payload := []byte("local-blob")
	if err := app.upsertMediaBlobRaw(cid, "image/png", payload, 8, 8, false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	media, err := app.GetMediaByCID(cid)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if media.ContentCID != cid {
		t.Errorf("cid: got %q", media.ContentCID)
	}

	afterMetrics := app.GetReleaseMetrics()
	if afterMetrics.BlobCacheHits != beforeHits+1 {
		t.Errorf("hit counter should increment on local cache hit: before=%d after=%d", beforeHits, afterMetrics.BlobCacheHits)
	}
}

func TestGetMediaByCIDMissAndOfflineYieldsNotFound(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	beforeMetrics := app.GetReleaseMetrics()
	beforeMisses := beforeMetrics.BlobCacheMisses

	_, err := app.GetMediaByCID("not-in-db")
	if err == nil {
		t.Fatal("expected error when blob is missing and p2p is not started")
	}
	if !strings.Contains(err.Error(), "media not found") {
		t.Errorf("expected 'media not found', got %v", err)
	}

	afterMetrics := app.GetReleaseMetrics()
	if afterMetrics.BlobCacheMisses != beforeMisses+1 {
		t.Errorf("miss counter should increment: before=%d after=%d", beforeMisses, afterMetrics.BlobCacheMisses)
	}
}

// -----------------------------------------------------------------------------
// GetPostMediaByID
// -----------------------------------------------------------------------------

func TestGetPostMediaByIDValidation(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.GetPostMediaByID(""); err == nil {
		t.Error("expected error for empty post id")
	}
	if _, err := app.GetPostMediaByID("missing-post"); err == nil {
		t.Error("expected error for unknown post id")
	}
}

func TestGetPostMediaByIDPostHasNoImage(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	_, err := app.GetPostMediaByID(postID)
	if err == nil {
		t.Fatal("expected error when post has no image")
	}
	if !strings.Contains(err.Error(), "no image") {
		t.Errorf("expected 'no image' error, got %v", err)
	}
}

func TestGetPostMediaByIDReturnsBlob(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	// Seed a media blob and a post that references it.
	payload := []byte("media-bytes-for-post")
	cid := buildBinaryCID(payload)
	if err := app.upsertMediaBlobRaw(cid, "image/png", payload, 64, 64, false); err != nil {
		t.Fatalf("seed media: %v", err)
	}

	now := int64(1_700_000_000)
	postID := "post-with-image"
	if _, err := app.db.Exec(
		`INSERT INTO messages
		   (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, image_cid)
		 VALUES (?, ?, '', 0, ?, 1, 0, 'public', ?, 'normal', ?)`,
		postID, identity.PublicKey, now, defaultSubID, cid,
	); err != nil {
		t.Fatalf("seed post: %v", err)
	}

	media, err := app.GetPostMediaByID(postID)
	if err != nil {
		t.Fatalf("get post media: %v", err)
	}
	if media.ContentCID != cid {
		t.Errorf("expected cid %q, got %q", cid, media.ContentCID)
	}
}

// -----------------------------------------------------------------------------
// canServeMediaBlobToNetwork
// -----------------------------------------------------------------------------

func TestCanServeMediaBlobToNetworkEmptyCID(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	got, err := app.canServeMediaBlobToNetwork("")
	if err != nil {
		t.Fatalf("empty cid should not error, got %v", err)
	}
	if got {
		t.Error("empty cid should not be serveable")
	}
}

func TestCanServeMediaBlobToNetworkNoReferringMessage(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.upsertMediaBlobRaw("orphan-cid", "image/png", []byte("x"), 1, 1, false); err != nil {
		t.Fatalf("seed media: %v", err)
	}

	got, err := app.canServeMediaBlobToNetwork("orphan-cid")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if got {
		t.Error("blob with no public message reference must not be serveable")
	}
}

func TestCanServeMediaBlobToNetworkPublicNormalPost(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	cid := "served-cid"
	if err := app.upsertMediaBlobRaw(cid, "image/png", []byte("payload"), 16, 16, false); err != nil {
		t.Fatalf("seed media: %v", err)
	}

	now := int64(1_700_000_000)
	if _, err := app.db.Exec(
		`INSERT INTO messages
		   (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, image_cid)
		 VALUES (?, ?, '', 0, ?, 1, 0, 'public', ?, 'normal', ?)`,
		"public-post-1", identity.PublicKey, now, defaultSubID, cid,
	); err != nil {
		t.Fatalf("seed post: %v", err)
	}

	got, err := app.canServeMediaBlobToNetwork(cid)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !got {
		t.Error("expected public/normal post to make blob serveable")
	}
}

func TestCanServeMediaBlobToNetworkShadowedPostNotServed(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	cid := "shadowed-cid"
	if err := app.upsertMediaBlobRaw(cid, "image/png", []byte("payload"), 16, 16, false); err != nil {
		t.Fatalf("seed media: %v", err)
	}

	now := int64(1_700_000_000)
	if _, err := app.db.Exec(
		`INSERT INTO messages
		   (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, image_cid)
		 VALUES (?, ?, '', 0, ?, 1, 0, 'public', ?, 'shadowed', ?)`,
		"shadowed-post-1", identity.PublicKey, now, defaultSubID, cid,
	); err != nil {
		t.Fatalf("seed post: %v", err)
	}

	got, _ := app.canServeMediaBlobToNetwork(cid)
	if got {
		t.Error("shadowed post must not make its image serveable")
	}
}

func TestCanServeMediaBlobToNetworkPrivateZoneNotServed(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	cid := "private-cid"
	if err := app.upsertMediaBlobRaw(cid, "image/png", []byte("payload"), 16, 16, false); err != nil {
		t.Fatalf("seed media: %v", err)
	}

	now := int64(1_700_000_000)
	if _, err := app.db.Exec(
		`INSERT INTO messages
		   (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, image_cid)
		 VALUES (?, ?, '', 0, ?, 1, 0, 'private', ?, 'normal', ?)`,
		"private-post-1", identity.PublicKey, now, defaultSubID, cid,
	); err != nil {
		t.Fatalf("seed post: %v", err)
	}

	got, _ := app.canServeMediaBlobToNetwork(cid)
	if got {
		t.Error("private-zone post must not make its image serveable")
	}
}
