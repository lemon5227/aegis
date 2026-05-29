package main

import (
	"testing"
)
// Mute Users Tests

func TestMuteAndUnmuteUser(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	targetPubkey := "deadbeef1234567890deadbeef1234567890deadbeef1234567890deadbeef12"

	// Initially not muted
	muted, err := app.IsMuted(targetPubkey)
	if err != nil {
		t.Fatalf("is muted: %v", err)
	}
	if muted {
		t.Error("user should not be muted initially")
	}

	// Mute the user
	if err := app.MuteUser(targetPubkey, "spammer"); err != nil {
		t.Fatalf("mute user: %v", err)
	}

	muted, err = app.IsMuted(targetPubkey)
	if err != nil {
		t.Fatalf("is muted after mute: %v", err)
	}
	if !muted {
		t.Error("user should be muted")
	}

	// Get muted users
	users, err := app.GetMutedUsers()
	if err != nil {
		t.Fatalf("get muted users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 muted user, got %d", len(users))
	}
	if users[0].Reason != "spammer" {
		t.Errorf("expected reason 'spammer', got %q", users[0].Reason)
	}

	// Unmute
	if err := app.UnmuteUser(targetPubkey); err != nil {
		t.Fatalf("unmute: %v", err)
	}

	muted, err = app.IsMuted(targetPubkey)
	if err != nil {
		t.Fatalf("is muted after unmute: %v", err)
	}
	if muted {
		t.Error("user should not be muted after unmute")
	}
}

func TestMuteUserIdempotent(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	pubkey := "abc123"

	if err := app.MuteUser(pubkey, "first"); err != nil {
		t.Fatalf("first mute: %v", err)
	}

	// Mute again - should update reason
	if err := app.MuteUser(pubkey, "second"); err != nil {
		t.Fatalf("second mute: %v", err)
	}

	users, _ := app.GetMutedUsers()
	if len(users) != 1 {
		t.Errorf("expected 1 muted user after re-mute, got %d", len(users))
	}
	if users[0].Reason != "second" {
		t.Errorf("expected updated reason 'second', got %q", users[0].Reason)
	}
}

func TestMuteUserValidation(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Empty pubkey should fail
	if err := app.MuteUser("", "reason"); err == nil {
		t.Error("expected error for empty pubkey")
	}

	if err := app.MuteUser("   ", "reason"); err == nil {
		t.Error("expected error for whitespace pubkey")
	}
}

func TestUnmuteUserIdempotent(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Unmute non-existent user should not error
	if err := app.UnmuteUser("nonexistent"); err != nil {
		t.Errorf("unmute non-existent should not error: %v", err)
	}
}

func TestGetMutedPubkeys(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.MuteUser("user1", ""); err != nil {
		t.Fatalf("mute: %v", err)
	}
	if err := app.MuteUser("user2", "reason"); err != nil {
		t.Fatalf("mute: %v", err)
	}

	pubkeys, err := app.GetMutedPubkeys()
	if err != nil {
		t.Fatalf("get muted pubkeys: %v", err)
	}
	if len(pubkeys) != 2 {
		t.Errorf("expected 2 pubkeys, got %d", len(pubkeys))
	}
	if _, ok := pubkeys["user1"]; !ok {
		t.Error("user1 not in pubkeys")
	}
	if _, ok := pubkeys["user2"]; !ok {
		t.Error("user2 not in pubkeys")
	}
}

func TestMuteReasonTruncation(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	longReason := ""
	for i := 0; i < 300; i++ {
		longReason += "x"
	}

	if err := app.MuteUser("user1", longReason); err != nil {
		t.Fatalf("mute: %v", err)
	}

	users, _ := app.GetMutedUsers()
	if len(users) != 1 {
		t.Fatal("expected 1 user")
	}
	if len([]rune(users[0].Reason)) > 200 {
		t.Errorf("reason should be truncated to 200, got %d", len([]rune(users[0].Reason)))
	}
}

// Post Read Tracking Tests

func TestMarkAndCheckPostRead(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Read Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	// Initially not read
	read, err := app.IsPostRead(postID)
	if err != nil {
		t.Fatalf("is post read: %v", err)
	}
	if read {
		t.Error("post should not be read initially")
	}

	// Mark as read
	if err := app.MarkPostRead(postID); err != nil {
		t.Fatalf("mark read: %v", err)
	}

	read, err = app.IsPostRead(postID)
	if err != nil {
		t.Fatalf("is post read after mark: %v", err)
	}
	if !read {
		t.Error("post should be read after marking")
	}
}

func TestMarkPostReadIdempotent(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	postID := "post-id-1"

	if err := app.MarkPostRead(postID); err != nil {
		t.Fatalf("first mark: %v", err)
	}
	if err := app.MarkPostRead(postID); err != nil {
		t.Fatalf("second mark: %v", err)
	}

	ids, err := app.GetReadPostIDs()
	if err != nil {
		t.Fatalf("get read: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 read post, got %d", len(ids))
	}
}

func TestMarkPostsReadBatch(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	postIDs := []string{"p1", "p2", "p3", "p4", "p5"}
	if err := app.MarkPostsRead(postIDs); err != nil {
		t.Fatalf("mark batch: %v", err)
	}

	ids, err := app.GetReadPostIDs()
	if err != nil {
		t.Fatalf("get read: %v", err)
	}
	if len(ids) != 5 {
		t.Errorf("expected 5 read posts, got %d", len(ids))
	}
	for _, p := range postIDs {
		if _, ok := ids[p]; !ok {
			t.Errorf("post %q not marked as read", p)
		}
	}
}

func TestMarkPostsReadEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Empty list should not error
	if err := app.MarkPostsRead([]string{}); err != nil {
		t.Errorf("empty batch should not error: %v", err)
	}
	if err := app.MarkPostsRead(nil); err != nil {
		t.Errorf("nil batch should not error: %v", err)
	}
}

func TestClearReadHistory(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.MarkPostsRead([]string{"p1", "p2", "p3"}); err != nil {
		t.Fatalf("mark batch: %v", err)
	}

	if err := app.ClearReadHistory(); err != nil {
		t.Fatalf("clear: %v", err)
	}

	ids, _ := app.GetReadPostIDs()
	if len(ids) != 0 {
		t.Errorf("expected 0 read posts after clear, got %d", len(ids))
	}
}

func TestGetUnreadPostCount(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	// Publish 3 posts with unique titles
	for i := 0; i < 3; i++ {
		title := "Unread Test " + string(rune('A'+i))
		idx := i
		if err := retryOnBusy(t, 5, func() error {
			return app.PublishPostStructuredToSub(identity.PublicKey, title, "body", defaultSubID)
		}); err != nil {
			t.Fatalf("publish %d: %v", idx, err)
		}
	}

	count, err := app.GetUnreadPostCount(defaultSubID)
	if err != nil {
		t.Fatalf("get unread count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 unread, got %d", count)
	}

	// Get all posts directly from messages table to ensure we have a valid ID
	posts, err := app.GetFeedBySubSorted(defaultSubID, "new")
	if err != nil {
		t.Fatalf("get feed sorted: %v", err)
	}
	if len(posts) < 1 {
		t.Fatal("no posts found")
	}

	// Mark first one as read
	if err := retryOnBusy(t, 5, func() error {
		return app.MarkPostRead(posts[0].ID)
	}); err != nil {
		t.Fatalf("mark read: %v", err)
	}

	// Verify it's actually marked
	read, _ := app.IsPostRead(posts[0].ID)
	if !read {
		t.Fatalf("post %s should be read", posts[0].ID)
	}

	count, err = app.GetUnreadPostCount(defaultSubID)
	if err != nil {
		t.Fatalf("get unread count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 unread after one read, got %d (post=%s)", count, posts[0].ID)
	}
}

// User Preferences Tests

func TestSetAndGetUserPreference(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.SetUserPreference("theme", "dark"); err != nil {
		t.Fatalf("set: %v", err)
	}

	value, err := app.GetUserPreference("theme")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if value != "dark" {
		t.Errorf("expected 'dark', got %q", value)
	}
}

func TestUserPreferenceUpdate(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.SetUserPreference("theme", "dark"); err != nil {
		t.Fatalf("set: %v", err)
	}

	if err := app.SetUserPreference("theme", "light"); err != nil {
		t.Fatalf("update: %v", err)
	}

	value, _ := app.GetUserPreference("theme")
	if value != "light" {
		t.Errorf("expected 'light' after update, got %q", value)
	}
}

func TestGetMissingPreference(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	value, err := app.GetUserPreference("nonexistent")
	if err != nil {
		t.Fatalf("get missing: %v", err)
	}
	if value != "" {
		t.Errorf("expected empty string for missing key, got %q", value)
	}
}

func TestDeleteUserPreference(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.SetUserPreference("temp", "value"); err != nil {
		t.Fatalf("set: %v", err)
	}

	if err := app.DeleteUserPreference("temp"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	value, _ := app.GetUserPreference("temp")
	if value != "" {
		t.Errorf("expected empty after delete, got %q", value)
	}
}

func TestUserPreferenceValidation(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Empty key should fail
	if err := app.SetUserPreference("", "value"); err == nil {
		t.Error("expected error for empty key")
	}

	// Key too long should fail
	longKey := ""
	for i := 0; i < 100; i++ {
		longKey += "x"
	}
	if err := app.SetUserPreference(longKey, "value"); err == nil {
		t.Error("expected error for key > 64 chars")
	}

	// Value too large should fail
	largeValue := make([]byte, 5000)
	for i := range largeValue {
		largeValue[i] = 'x'
	}
	if err := app.SetUserPreference("key", string(largeValue)); err == nil {
		t.Error("expected error for value > 4096 bytes")
	}
}

func TestGetAllUserPreferences(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	prefs := map[string]string{
		"theme":     "dark",
		"language":  "en",
		"font_size": "medium",
	}

	for k, v := range prefs {
		if err := app.SetUserPreference(k, v); err != nil {
			t.Fatalf("set %s: %v", k, err)
		}
	}

	all, err := app.GetAllUserPreferences()
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 prefs, got %d", len(all))
	}

	// Should be sorted by key
	expectedKeys := []string{"font_size", "language", "theme"}
	for i, p := range all {
		if p.Key != expectedKeys[i] {
			t.Errorf("position %d: expected %q, got %q", i, expectedKeys[i], p.Key)
		}
	}
}
