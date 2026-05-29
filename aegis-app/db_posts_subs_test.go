package main

import (
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// AddLocalPostStructured (default-sub wrapper)
// -----------------------------------------------------------------------------

func TestAddLocalPostStructuredDefaultSub(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	post, err := app.AddLocalPostStructured(identity.PublicKey, "Title", "Body", "public")
	if err != nil {
		t.Fatalf("add post: %v", err)
	}
	if post.SubID != defaultSubID {
		t.Errorf("expected sub id %q, got %q", defaultSubID, post.SubID)
	}
	if post.Title != "Title" || post.Body != "Body" {
		t.Errorf("title/body mismatch: %+v", post)
	}

	// Verify it appears in the default sub's feed.
	feed, _ := app.GetFeedBySub(defaultSubID)
	found := false
	for _, p := range feed {
		if p.ID == post.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("post not visible in default-sub feed")
	}
}

// -----------------------------------------------------------------------------
// UpdateLocalPost
// -----------------------------------------------------------------------------

func TestUpdateLocalPostChangesTitleAndBody(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Original", "original body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	updated, err := app.UpdateLocalPost(identity.PublicKey, postID, "Edited title", "edited body")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "Edited title" || updated.Body != "edited body" {
		t.Errorf("update did not apply: %+v", updated)
	}
	if updated.ID != postID {
		t.Errorf("update should preserve id, got %q (was %q)", updated.ID, postID)
	}
}

func TestUpdateLocalPostTitleOnlyKeepsBody(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Old title", "kept body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	updated, err := app.UpdateLocalPost(identity.PublicKey, postID, "New title", "")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "New title" {
		t.Errorf("title should change, got %q", updated.Title)
	}
	if updated.Body != "kept body" {
		t.Errorf("body should be preserved when empty body passed, got %q", updated.Body)
	}
}

func TestUpdateLocalPostBodyOnlyKeepsTitle(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Kept title", "old body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	updated, err := app.UpdateLocalPost(identity.PublicKey, postID, "", "new body")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "Kept title" {
		t.Errorf("title should be preserved when empty title passed, got %q", updated.Title)
	}
	if updated.Body != "new body" {
		t.Errorf("body should change, got %q", updated.Body)
	}
}

func TestUpdateLocalPostRejectsBothEmpty(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	if _, err := app.UpdateLocalPost(identity.PublicKey, postID, "  ", "   "); err == nil {
		t.Error("expected error when both title and body are empty/whitespace")
	}
}

func TestUpdateLocalPostRejectsNonAuthor(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	_, foreign := generateRemoteIdentity(t)
	_, err := app.UpdateLocalPost(foreign, postID, "tampered", "")
	if err == nil {
		t.Fatal("expected non-author update to be rejected")
	}
	if !strings.Contains(err.Error(), "only author") {
		t.Errorf("expected 'only author' error, got %v", err)
	}
}

func TestUpdateLocalPostNotFound(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	_, err := app.UpdateLocalPost(identity.PublicKey, "missing-id", "x", "y")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' message, got %v", err)
	}
}

func TestUpdateLocalPostRequiresPubkeyAndID(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if _, err := app.UpdateLocalPost("   ", "any-id", "t", "b"); err == nil {
		t.Error("expected error for empty pubkey")
	}
	if _, err := app.UpdateLocalPost(identity.PublicKey, "  ", "t", "b"); err == nil {
		t.Error("expected error for empty post id")
	}
}

// -----------------------------------------------------------------------------
// SearchSubs
// -----------------------------------------------------------------------------

func TestSearchSubsEmptyKeywordReturnsEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.CreateSub("alpha", "Alpha", "first sub"); err != nil {
		t.Fatalf("create sub: %v", err)
	}

	got, err := app.SearchSubs("", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty keyword should return empty, got %d", len(got))
	}
}

func TestSearchSubsMatchesByIDTitleDescription(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.CreateSub("alpha", "Alpha title", "the first one"); err != nil {
		t.Fatalf("create alpha: %v", err)
	}
	if _, err := app.CreateSub("beta", "Different name", "alpha keyword in description"); err != nil {
		t.Fatalf("create beta: %v", err)
	}
	if _, err := app.CreateSub("gamma", "alpha-prefixed title", "unrelated"); err != nil {
		t.Fatalf("create gamma: %v", err)
	}
	if _, err := app.CreateSub("delta", "Delta", "no overlap"); err != nil {
		t.Fatalf("create delta: %v", err)
	}

	got, err := app.SearchSubs("alpha", 10)
	if err != nil {
		t.Fatalf("search alpha: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 matches (id/title/desc), got %d (%+v)", len(got), got)
	}

	// Exact id match must come first per the SearchSubs ORDER BY.
	if got[0].ID != "alpha" {
		t.Errorf("expected exact id match first, got %q", got[0].ID)
	}
}

func TestSearchSubsRespectsLimit(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	for _, id := range []string{"sub-a", "sub-b", "sub-c", "sub-d"} {
		if _, err := app.CreateSub(id, "Sub "+id, "test sub"); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	got, err := app.SearchSubs("sub", 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected limit=2 to cap results, got %d", len(got))
	}
}

func TestSearchSubsCaseInsensitive(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.CreateSub("MyClub", "MyClub", "MIXED case"); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := app.SearchSubs("MYCLUB", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected case-insensitive match, got %d", len(got))
	}
}

// -----------------------------------------------------------------------------
// isSubSubscribed (private but reachable through SubscribeSub/UnsubscribeSub)
// -----------------------------------------------------------------------------

func TestIsSubSubscribedReflectsSubscription(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.CreateSub("club", "Club", "desc"); err != nil {
		t.Fatalf("create sub: %v", err)
	}

	subscribed, err := app.isSubSubscribed("club")
	if err != nil {
		t.Fatalf("isSubSubscribed: %v", err)
	}
	if subscribed {
		t.Error("expected not subscribed before SubscribeSub")
	}

	if _, err := app.SubscribeSub("club"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	subscribed, _ = app.isSubSubscribed("club")
	if !subscribed {
		t.Error("expected subscribed after SubscribeSub")
	}

	if err := app.UnsubscribeSub("club"); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	subscribed, _ = app.isSubSubscribed("club")
	if subscribed {
		t.Error("expected not subscribed after UnsubscribeSub")
	}
}

func TestIsSubSubscribedNormalizesID(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.CreateSub("club", "Club", "desc"); err != nil {
		t.Fatalf("create sub: %v", err)
	}
	if _, err := app.SubscribeSub("CLUB"); err != nil {
		t.Fatalf("subscribe with caps: %v", err)
	}

	// Lookup with a different casing/whitespace should still resolve.
	subscribed, err := app.isSubSubscribed("  club  ")
	if err != nil {
		t.Fatalf("isSubSubscribed: %v", err)
	}
	if !subscribed {
		t.Error("expected normalization to handle casing/whitespace")
	}
}

// -----------------------------------------------------------------------------
// encodeFavoriteCursor / decodeFavoriteCursor
// -----------------------------------------------------------------------------

func TestFavoriteCursorRoundTrip(t *testing.T) {
	cases := []struct {
		ts     int64
		postID string
	}{
		{1, "post-1"},
		{1_700_000_000, "post-with-dashes-123"},
		{42, "id|with|pipes"},
	}
	for _, tc := range cases {
		t.Run(tc.postID, func(t *testing.T) {
			cursor := encodeFavoriteCursor(tc.ts, tc.postID)
			ts, postID, err := decodeFavoriteCursor(cursor)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if ts != tc.ts {
				t.Errorf("ts: got %d, want %d", ts, tc.ts)
			}
			if postID != tc.postID {
				t.Errorf("postID: got %q, want %q", postID, tc.postID)
			}
		})
	}
}

func TestDecodeFavoriteCursorEmptyReturnsZero(t *testing.T) {
	ts, postID, err := decodeFavoriteCursor("")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ts != 0 || postID != "" {
		t.Errorf("expected zero values, got ts=%d postID=%q", ts, postID)
	}
}

func TestDecodeFavoriteCursorRejectsMalformed(t *testing.T) {
	cases := []string{
		"not-base64-!!!",
		"bm8tcGlwZQ==", // valid base64 but no | separator
	}
	for _, c := range cases {
		if _, _, err := decodeFavoriteCursor(c); err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

func TestDecodeFavoriteCursorRejectsBadTimestamp(t *testing.T) {
	bad := encodeFavoriteCursor(0, "post-1")
	if _, _, err := decodeFavoriteCursor(bad); err == nil {
		t.Error("expected error for ts=0")
	}
	negative := encodeFavoriteCursor(-5, "post-1")
	if _, _, err := decodeFavoriteCursor(negative); err == nil {
		t.Error("expected error for negative ts")
	}
}

func TestDecodeFavoriteCursorRejectsEmptyPostID(t *testing.T) {
	bad := encodeFavoriteCursor(100, "  ")
	if _, _, err := decodeFavoriteCursor(bad); err == nil {
		t.Error("expected error for empty post id after trim")
	}
}
