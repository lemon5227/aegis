package main

import (
	"path/filepath"
	"testing"
)

func TestR2DigestWindowReturnsOnlySinceTimestamp(t *testing.T) {
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "r2_digest_window.db"))

	if err := app.initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}
	defer func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	}()

	msg1, err := app.AddLocalPostStructuredToSub("pk1", "old-1", "body-1", "public", "general")
	if err != nil {
		t.Fatalf("AddLocalPostStructuredToSub old-1 failed: %v", err)
	}
	msg2, err := app.AddLocalPostStructuredToSub("pk2", "old-2", "body-2", "public", "general")
	if err != nil {
		t.Fatalf("AddLocalPostStructuredToSub old-2 failed: %v", err)
	}
	msg3, err := app.AddLocalPostStructuredToSub("pk3", "new-1", "body-3", "public", "general")
	if err != nil {
		t.Fatalf("AddLocalPostStructuredToSub new-1 failed: %v", err)
	}

	since := msg2.Timestamp
	digests, err := app.listPublicPostDigestsSince(since, 20)
	if err != nil {
		t.Fatalf("listPublicPostDigestsSince failed: %v", err)
	}
	if len(digests) == 0 {
		t.Fatalf("expected non-empty digests")
	}

	for _, digest := range digests {
		if digest.Timestamp < since {
			t.Fatalf("expected digest timestamp >= since, got %d < %d", digest.Timestamp, since)
		}
	}

	foundNewest := false
	for _, digest := range digests {
		if digest.ID == msg3.ID {
			foundNewest = true
			break
		}
	}
	if !foundNewest {
		t.Fatalf("expected latest message %s to be included", msg3.ID)
	}

	if msg1.ID == "" {
		t.Fatalf("sanity check failed: msg1 id should not be empty")
	}
}

func TestR2LatestPublicTimestamp(t *testing.T) {
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "r2_latest_ts.db"))

	if err := app.initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}
	defer func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	}()

	latest, err := app.getLatestPublicPostTimestamp()
	if err != nil {
		t.Fatalf("getLatestPublicPostTimestamp failed: %v", err)
	}
	if latest != 0 {
		t.Fatalf("expected empty latest timestamp to be 0, got %d", latest)
	}

	msg, err := app.AddLocalPostStructuredToSub("pk", "title", "body", "public", "general")
	if err != nil {
		t.Fatalf("AddLocalPostStructuredToSub failed: %v", err)
	}

	latest, err = app.getLatestPublicPostTimestamp()
	if err != nil {
		t.Fatalf("getLatestPublicPostTimestamp failed after insert: %v", err)
	}
	if latest < msg.Timestamp {
		t.Fatalf("expected latest >= message timestamp, got latest=%d msg=%d", latest, msg.Timestamp)
	}
}
