package main

import "testing"

func TestC2PostStoresContentCIDAndBlob(t *testing.T) {
	app := newTestAppWithDB(t)

	post, err := app.AddLocalPostStructuredToSub("author-c2", "c2 title", "c2 body payload", "public", "general")
	if err != nil {
		t.Fatalf("AddLocalPostStructuredToSub failed: %v", err)
	}
	if post.ContentCID == "" {
		t.Fatalf("expected post content cid to be set")
	}

	var cid string
	if err = app.db.QueryRow(`SELECT content_cid FROM messages WHERE id = ?;`, post.ID).Scan(&cid); err != nil {
		t.Fatalf("query messages content_cid failed: %v", err)
	}
	if cid == "" {
		t.Fatalf("expected content_cid stored in messages")
	}

	bodyBlob, err := app.GetPostBodyByCID(cid)
	if err != nil {
		t.Fatalf("GetPostBodyByCID failed: %v", err)
	}
	if bodyBlob.Body != "c2 body payload" {
		t.Fatalf("expected blob body=c2 body payload, got %q", bodyBlob.Body)
	}
}

func TestC2IndexFeedAndBodyFetchByPostID(t *testing.T) {
	app := newTestAppWithDB(t)

	post, err := app.AddLocalPostStructuredToSub("author-c2", "index title", "index body for preview", "public", "general")
	if err != nil {
		t.Fatalf("AddLocalPostStructuredToSub failed: %v", err)
	}

	indexes, err := app.GetFeedIndexBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedIndexBySubSorted failed: %v", err)
	}
	if len(indexes) == 0 {
		t.Fatalf("expected at least one index row")
	}
	if indexes[0].ContentCID == "" {
		t.Fatalf("expected index row content cid to be populated")
	}
	if indexes[0].BodyPreview == "" {
		t.Fatalf("expected index row body preview to be populated")
	}

	blob, err := app.GetPostBodyByID(post.ID)
	if err != nil {
		t.Fatalf("GetPostBodyByID failed: %v", err)
	}
	if blob.Body != "index body for preview" {
		t.Fatalf("expected GetPostBodyByID to return full body, got %q", blob.Body)
	}
}

func TestC2BlobLRUEvictKeepsMessageIndexes(t *testing.T) {
	app := newTestAppWithDB(t)

	firstBody := "12345678901234567890"
	secondBody := "abcdefghijabcdefghij"

	first, err := app.insertMessage(ForumMessage{
		ID:          buildMessageID("author-a", firstBody, 1),
		Pubkey:      "author-a",
		Title:       "first",
		Body:        firstBody,
		ContentCID:  buildContentCID(firstBody),
		Timestamp:   1,
		Zone:        "public",
		SubID:       "general",
		Visibility:  "normal",
		IsProtected: 0,
	})
	if err != nil {
		t.Fatalf("insert first message failed: %v", err)
	}

	if _, err = app.insertMessage(ForumMessage{
		ID:          buildMessageID("author-b", secondBody, 2),
		Pubkey:      "author-b",
		Title:       "second",
		Body:        secondBody,
		ContentCID:  buildContentCID(secondBody),
		Timestamp:   2,
		Zone:        "public",
		SubID:       "general",
		Visibility:  "normal",
		IsProtected: 0,
	}); err != nil {
		t.Fatalf("insert second message failed: %v", err)
	}

	tx, err := app.db.Begin()
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}

	if err = ensureBlobQuotaWithLRU(tx, "public", 20, 20, buildContentCID(secondBody)); err != nil {
		_ = tx.Rollback()
		t.Fatalf("ensureBlobQuotaWithLRU failed: %v", err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatalf("commit tx failed: %v", err)
	}

	feed, err := app.GetFeedBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted failed: %v", err)
	}
	if len(feed) != 2 {
		t.Fatalf("expected 2 index rows after blob eviction, got %d", len(feed))
	}

	if _, err = app.GetPostBodyByCID(first.ContentCID); err == nil {
		t.Fatalf("expected first post body blob to be evicted")
	}
}
