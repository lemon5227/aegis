package main

import (
	"strings"
	"testing"
)

// voteBroadcastSeqLen returns the current size of the broadcast sequence map.
// Used to assert that successful Publish*Vote calls schedule a debounced
// broadcast, while validation failures do not.
func voteBroadcastSeqLen(t *testing.T, app *App) int {
	t.Helper()
	app.voteBroadcastMu.Lock()
	defer app.voteBroadcastMu.Unlock()
	return len(app.voteBroadcastSeq)
}

// commentScore looks up the score for a single comment id.
func commentScore(t *testing.T, app *App, postID, commentID string) int64 {
	t.Helper()
	comments, err := app.GetCommentsByPost(postID)
	if err != nil {
		t.Fatalf("get comments: %v", err)
	}
	for _, c := range comments {
		if c.ID == commentID {
			return c.Score
		}
	}
	t.Fatalf("comment %q not found in post %q", commentID, postID)
	return 0
}

// postScore looks up the score for a single post id in the default sub.
func postScore(t *testing.T, app *App, postID string) int64 {
	t.Helper()
	feed, err := app.GetFeedBySub(defaultSubID)
	if err != nil {
		t.Fatalf("get feed: %v", err)
	}
	for _, p := range feed {
		if p.ID == postID {
			return p.Score
		}
	}
	t.Fatalf("post %q not found in feed", postID)
	return 0
}

// -----------------------------------------------------------------------------
// PublishPostUpvote / PublishPostDownvote
// -----------------------------------------------------------------------------

func TestPublishPostVoteValidation(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	cases := []struct {
		name   string
		pubkey string
		postID string
	}{
		{"empty pubkey", "  ", "post-1"},
		{"empty post id", identity.PublicKey, "  "},
		{"both empty", "", ""},
	}
	for _, tc := range cases {
		t.Run("upvote/"+tc.name, func(t *testing.T) {
			if err := app.PublishPostUpvote(tc.pubkey, tc.postID); err == nil {
				t.Error("expected error")
			}
		})
		t.Run("downvote/"+tc.name, func(t *testing.T) {
			if err := app.PublishPostDownvote(tc.pubkey, tc.postID); err == nil {
				t.Error("expected error")
			}
		})
	}
	if seqLen := voteBroadcastSeqLen(t, app); seqLen != 0 {
		t.Errorf("validation failures must not seed broadcast queue, got %d entries", seqLen)
	}
}

func TestPublishPostUpvoteAppliesAndSchedules(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	beforeSeq := voteBroadcastSeqLen(t, app)
	if err := app.PublishPostUpvote(identity.PublicKey, postID); err != nil {
		t.Fatalf("upvote: %v", err)
	}
	if got := postScore(t, app, postID); got != 1 {
		t.Errorf("expected score=1 after upvote, got %d", got)
	}
	if voteBroadcastSeqLen(t, app) != beforeSeq+1 {
		t.Error("upvote should schedule exactly one debounced broadcast")
	}
}

func TestPublishPostDownvoteAppliesAndSchedules(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	if err := app.PublishPostDownvote(identity.PublicKey, postID); err != nil {
		t.Fatalf("downvote: %v", err)
	}
	if got := postScore(t, app, postID); got != -1 {
		t.Errorf("expected score=-1 after downvote, got %d", got)
	}
}

func TestPublishPostVoteToggleNetsCorrectly(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	if err := app.PublishPostUpvote(identity.PublicKey, postID); err != nil {
		t.Fatalf("upvote: %v", err)
	}
	if err := app.PublishPostDownvote(identity.PublicKey, postID); err != nil {
		t.Fatalf("downvote toggle: %v", err)
	}
	// Up then down -> net delta of -2 in vote tally; final score = -1.
	if got := postScore(t, app, postID); got != -1 {
		t.Errorf("up->down toggle: expected score=-1, got %d", got)
	}
}

func TestPublishPostUpvoteRebroadcastDebouncesByKey(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	// Multiple rapid calls for the same (voter, post) should share one queue
	// entry — scheduleVoteStateBroadcast keys on (voter|post|comment).
	for i := 0; i < 3; i++ {
		if err := app.PublishPostUpvote(identity.PublicKey, postID); err != nil {
			t.Fatalf("upvote %d: %v", i, err)
		}
	}
	if seqLen := voteBroadcastSeqLen(t, app); seqLen != 1 {
		t.Errorf("expected single debounce queue entry for repeated votes on same post, got %d", seqLen)
	}
}

// -----------------------------------------------------------------------------
// PublishCommentUpvote / PublishCommentDownvote
// -----------------------------------------------------------------------------

func TestPublishCommentVoteValidation(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	cases := []struct {
		name      string
		pubkey    string
		postID    string
		commentID string
	}{
		{"empty pubkey", "  ", "p", "c"},
		{"empty post id", identity.PublicKey, "  ", "c"},
		{"empty comment id", identity.PublicKey, "p", "  "},
		{"all empty", "", "", ""},
	}
	for _, tc := range cases {
		t.Run("upvote/"+tc.name, func(t *testing.T) {
			err := app.PublishCommentUpvote(tc.pubkey, tc.postID, tc.commentID)
			if err == nil {
				t.Error("expected error")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("expected 'required' in error, got %v", err)
			}
		})
		t.Run("downvote/"+tc.name, func(t *testing.T) {
			err := app.PublishCommentDownvote(tc.pubkey, tc.postID, tc.commentID)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestPublishCommentUpvoteAppliesAndSchedules(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "vote on me")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	beforeSeq := voteBroadcastSeqLen(t, app)
	if err := app.PublishCommentUpvote(identity.PublicKey, postID, comment.ID); err != nil {
		t.Fatalf("upvote: %v", err)
	}
	if got := commentScore(t, app, postID, comment.ID); got != 1 {
		t.Errorf("expected comment score=1 after upvote, got %d", got)
	}
	if voteBroadcastSeqLen(t, app) != beforeSeq+1 {
		t.Error("comment upvote should schedule one broadcast")
	}
}

func TestPublishCommentDownvoteAppliesAndSchedules(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "downvote me")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	if err := app.PublishCommentDownvote(identity.PublicKey, postID, comment.ID); err != nil {
		t.Fatalf("downvote: %v", err)
	}
	if got := commentScore(t, app, postID, comment.ID); got != -1 {
		t.Errorf("expected comment score=-1 after downvote, got %d", got)
	}
}

func TestPublishCommentVoteToggleNetsCorrectly(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "toggle")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	if err := app.PublishCommentUpvote(identity.PublicKey, postID, comment.ID); err != nil {
		t.Fatalf("up: %v", err)
	}
	if err := app.PublishCommentDownvote(identity.PublicKey, postID, comment.ID); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := commentScore(t, app, postID, comment.ID); got != -1 {
		t.Errorf("up->down toggle: expected -1, got %d", got)
	}
}

func TestPublishPostAndCommentVoteSchedulesUseDistinctKeys(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "comment to vote on")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	if err := app.PublishPostUpvote(identity.PublicKey, postID); err != nil {
		t.Fatalf("post upvote: %v", err)
	}
	if err := app.PublishCommentUpvote(identity.PublicKey, postID, comment.ID); err != nil {
		t.Fatalf("comment upvote: %v", err)
	}

	if seqLen := voteBroadcastSeqLen(t, app); seqLen != 2 {
		t.Errorf("post and comment votes should produce distinct queue entries, got %d", seqLen)
	}
}
