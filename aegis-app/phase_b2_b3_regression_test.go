package main

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestAppWithDB(t *testing.T) *App {
	t.Helper()

	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "test.db"))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}

	t.Cleanup(func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	})

	return app
}

func TestB2PostUpvoteDedup(t *testing.T) {
	app := newTestAppWithDB(t)

	post, err := app.AddLocalPostStructuredToSub("author-a", "post-title", "post-body", "public", "general")
	if err != nil {
		t.Fatalf("AddLocalPostStructuredToSub failed: %v", err)
	}

	if err = app.applyPostUpvote("voter-a", post.ID); err != nil {
		t.Fatalf("first applyPostUpvote failed: %v", err)
	}
	if err = app.applyPostUpvote("voter-a", post.ID); err != nil {
		t.Fatalf("duplicate applyPostUpvote failed: %v", err)
	}
	if err = app.applyPostUpvote("voter-b", post.ID); err != nil {
		t.Fatalf("second voter applyPostUpvote failed: %v", err)
	}

	rows, err := app.GetFeedBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 post, got %d", len(rows))
	}
	if rows[0].Score != 2 {
		t.Fatalf("expected post score=2, got %d", rows[0].Score)
	}

	var voteCount int
	if err = app.db.QueryRow(`SELECT COUNT(1) FROM post_votes WHERE post_id = ?;`, post.ID).Scan(&voteCount); err != nil {
		t.Fatalf("query post_votes failed: %v", err)
	}
	if voteCount != 2 {
		t.Fatalf("expected 2 unique post votes, got %d", voteCount)
	}
}

func TestB2CommentUpvoteDedup(t *testing.T) {
	app := newTestAppWithDB(t)

	post, err := app.AddLocalPostStructuredToSub("author-a", "post-title", "post-body", "public", "general")
	if err != nil {
		t.Fatalf("AddLocalPostStructuredToSub failed: %v", err)
	}

	comment, err := app.AddLocalComment("author-b", post.ID, "", "hello")
	if err != nil {
		t.Fatalf("AddLocalComment failed: %v", err)
	}

	if err = app.applyCommentUpvote("voter-a", comment.ID, post.ID); err != nil {
		t.Fatalf("first applyCommentUpvote failed: %v", err)
	}
	if err = app.applyCommentUpvote("voter-a", comment.ID, post.ID); err != nil {
		t.Fatalf("duplicate applyCommentUpvote failed: %v", err)
	}
	if err = app.applyCommentUpvote("voter-b", comment.ID, post.ID); err != nil {
		t.Fatalf("second voter applyCommentUpvote failed: %v", err)
	}

	comments, err := app.GetCommentsByPost(post.ID)
	if err != nil {
		t.Fatalf("GetCommentsByPost failed: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Score != 2 {
		t.Fatalf("expected comment score=2, got %d", comments[0].Score)
	}

	var voteCount int
	if err = app.db.QueryRow(`SELECT COUNT(1) FROM comment_votes WHERE comment_id = ?;`, comment.ID).Scan(&voteCount); err != nil {
		t.Fatalf("query comment_votes failed: %v", err)
	}
	if voteCount != 2 {
		t.Fatalf("expected 2 unique comment votes, got %d", voteCount)
	}
}

func TestB3HotVsNewOrdering(t *testing.T) {
	app := newTestAppWithDB(t)
	now := time.Now().Unix()

	oldHigh := ForumMessage{
		ID:          buildMessageID("author-old", "old-high", now-96*3600),
		Pubkey:      "author-old",
		Title:       "old high",
		Body:        "old high body",
		Content:     "",
		Score:       160,
		Timestamp:   now - 96*3600,
		Zone:        "public",
		SubID:       "general",
		Visibility:  "normal",
		IsProtected: 0,
	}
	midBalanced := ForumMessage{
		ID:          buildMessageID("author-mid", "mid-balanced", now-12*3600),
		Pubkey:      "author-mid",
		Title:       "mid balanced",
		Body:        "mid balanced body",
		Content:     "",
		Score:       30,
		Timestamp:   now - 12*3600,
		Zone:        "public",
		SubID:       "general",
		Visibility:  "normal",
		IsProtected: 0,
	}
	newLow := ForumMessage{
		ID:          buildMessageID("author-new", "new-low", now-1*3600),
		Pubkey:      "author-new",
		Title:       "new low",
		Body:        "new low body",
		Content:     "",
		Score:       2,
		Timestamp:   now - 1*3600,
		Zone:        "public",
		SubID:       "general",
		Visibility:  "normal",
		IsProtected: 0,
	}

	if _, err := app.insertMessage(oldHigh); err != nil {
		t.Fatalf("insert oldHigh failed: %v", err)
	}
	if _, err := app.insertMessage(midBalanced); err != nil {
		t.Fatalf("insert midBalanced failed: %v", err)
	}
	if _, err := app.insertMessage(newLow); err != nil {
		t.Fatalf("insert newLow failed: %v", err)
	}

	hotRows, err := app.GetFeedBySubSorted("general", "hot")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted(hot) failed: %v", err)
	}
	newRows, err := app.GetFeedBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted(new) failed: %v", err)
	}
	if len(hotRows) < 3 || len(newRows) < 3 {
		t.Fatalf("expected at least 3 rows, got hot=%d new=%d", len(hotRows), len(newRows))
	}

	if hotRows[0].ID != midBalanced.ID {
		t.Fatalf("expected hot top=%s, got %s", midBalanced.ID, hotRows[0].ID)
	}
	if newRows[0].ID != newLow.ID {
		t.Fatalf("expected new top=%s, got %s", newLow.ID, newRows[0].ID)
	}
	if hotRows[0].ID == newRows[0].ID {
		t.Fatalf("expected hot/new top to differ, both are %s", hotRows[0].ID)
	}
}
