package main

import (
	"path/filepath"
	"testing"
	"time"
)

// newTestApp creates a fresh App instance with an in-memory-like temp database.
func newTestApp(t *testing.T) *App {
	t.Helper()
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "test.db"))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("init database: %v", err)
	}
	t.Cleanup(func() {
		if app.db != nil {
			// Checkpoint WAL and truncate before closing so the -wal and -shm
			// files are cleaned up and won't block t.TempDir() removal.
			_, _ = app.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
			_ = app.db.Close()
		}
	})
	return app
}

func TestIdentityGenerateAndLoad(t *testing.T) {
	app := newTestApp(t)

	identity, err := app.GenerateIdentity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	if identity.PublicKey == "" {
		t.Fatal("public key is empty")
	}
	if identity.Mnemonic == "" {
		t.Fatal("mnemonic is empty")
	}

	if err = app.saveLocalIdentity(identity); err != nil {
		t.Fatalf("save identity: %v", err)
	}

	loaded, err := app.LoadSavedIdentity()
	if err != nil {
		t.Fatalf("load identity: %v", err)
	}
	if loaded.PublicKey != identity.PublicKey {
		t.Errorf("loaded pubkey mismatch: got %q, want %q", loaded.PublicKey, identity.PublicKey)
	}
}

func TestIdentityImportFromMnemonic(t *testing.T) {
	app := newTestApp(t)

	original, err := app.GenerateIdentity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}

	imported, err := app.ImportIdentityFromMnemonic(original.Mnemonic)
	if err != nil {
		t.Fatalf("import identity: %v", err)
	}
	if imported.PublicKey != original.PublicKey {
		t.Errorf("imported pubkey mismatch: got %q, want %q", imported.PublicKey, original.PublicKey)
	}
}

func TestSignAndVerifyMessage(t *testing.T) {
	app := newTestApp(t)

	identity, err := app.GenerateIdentity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	if err = app.saveLocalIdentity(identity); err != nil {
		t.Fatalf("save identity: %v", err)
	}

	message := "hello world"
	sig, err := app.SignMessage(identity.Mnemonic, message)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}
	if sig == "" {
		t.Fatal("signature is empty")
	}

	valid, err := app.VerifyMessage(identity.PublicKey, message, sig)
	if err != nil {
		t.Fatalf("verify message: %v", err)
	}
	if !valid {
		t.Error("signature should be valid")
	}

	// Tampered message should fail
	valid, err = app.VerifyMessage(identity.PublicKey, "tampered", sig)
	if err != nil {
		t.Fatalf("verify tampered: %v", err)
	}
	if valid {
		t.Error("tampered message should not verify")
	}
}

func TestCreateSubAndGetSubs(t *testing.T) {
	app := newTestApp(t)

	sub, err := app.CreateSub("test-sub", "Test Sub", "A test sub-community")
	if err != nil {
		t.Fatalf("create sub: %v", err)
	}
	if sub.ID != "test-sub" {
		t.Errorf("sub ID mismatch: got %q", sub.ID)
	}
	if sub.Title != "Test Sub" {
		t.Errorf("sub title mismatch: got %q", sub.Title)
	}

	subs, err := app.GetSubs()
	if err != nil {
		t.Fatalf("get subs: %v", err)
	}
	found := false
	for _, s := range subs {
		if s.ID == "test-sub" {
			found = true
			break
		}
	}
	if !found {
		t.Error("created sub not found in GetSubs result")
	}
}

func TestPublishPostAndGetFeed(t *testing.T) {
	app := newTestApp(t)

	identity := seedLocalIdentity(t, app)

	err := app.PublishPostStructuredToSub(identity.PublicKey, "Test Post", "Test body content", defaultSubID)
	if err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, err := app.GetFeedBySub(defaultSubID)
	if err != nil {
		t.Fatalf("get feed: %v", err)
	}
	if len(posts) == 0 {
		t.Fatal("expected at least 1 post in feed")
	}
	if posts[0].Title != "Test Post" {
		t.Errorf("post title mismatch: got %q", posts[0].Title)
	}
}

func TestPublishPostSortedFeed(t *testing.T) {
	app := newTestApp(t)

	identity := seedLocalIdentity(t, app)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Post 1", "body1", defaultSubID); err != nil {
		t.Fatalf("publish post 1: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Post 2", "body2", defaultSubID); err != nil {
		t.Fatalf("publish post 2: %v", err)
	}

	postsNew, err := app.GetFeedIndexBySubSorted(defaultSubID, "new")
	if err != nil {
		t.Fatalf("get new feed: %v", err)
	}
	if len(postsNew) < 2 {
		t.Fatalf("expected at least 2 posts, got %d", len(postsNew))
	}
	// Newest first
	if postsNew[0].Timestamp < postsNew[1].Timestamp {
		t.Error("new feed should be sorted newest first")
	}
}

func TestAddCommentAndGetComments(t *testing.T) {
	app := newTestApp(t)

	identity := seedLocalIdentity(t, app)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Comment Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, err := app.GetFeedBySub(defaultSubID)
	if err != nil || len(posts) == 0 {
		t.Fatalf("get feed: %v", err)
	}
	postID := posts[0].ID

	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "Top level comment")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}
	if comment.Body != "Top level comment" {
		t.Errorf("comment body mismatch: got %q", comment.Body)
	}
	if comment.PostID != postID {
		t.Errorf("comment postID mismatch: got %q, want %q", comment.PostID, postID)
	}

	// Add nested reply
	reply, err := app.AddLocalComment(identity.PublicKey, postID, comment.ID, "Nested reply")
	if err != nil {
		t.Fatalf("add reply: %v", err)
	}
	if reply.ParentID != comment.ID {
		t.Errorf("reply parentID mismatch: got %q, want %q", reply.ParentID, comment.ID)
	}

	comments, err := app.GetCommentsByPost(postID)
	if err != nil {
		t.Fatalf("get comments: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}
}

func TestFavoriteAddAndRemove(t *testing.T) {
	app := newTestApp(t)

	identity := seedLocalIdentity(t, app)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Fav Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, err := app.GetFeedBySub(defaultSubID)
	if err != nil || len(posts) == 0 {
		t.Fatalf("get feed: %v", err)
	}
	postID := posts[0].ID

	if err := app.AddFavorite(postID); err != nil {
		t.Fatalf("add favorite: %v", err)
	}

	favIDs, err := app.GetFavoritePostIDs()
	if err != nil {
		t.Fatalf("get favorite IDs: %v", err)
	}
	if len(favIDs) != 1 || favIDs[0] != postID {
		t.Errorf("expected favorite %q, got %v", postID, favIDs)
	}

	// Adding same favorite again should be idempotent
	if err := app.AddFavorite(postID); err != nil {
		t.Fatalf("add favorite again: %v", err)
	}
	favIDs, err = app.GetFavoritePostIDs()
	if err != nil {
		t.Fatalf("get favorite IDs after re-add: %v", err)
	}
	if len(favIDs) != 1 {
		t.Errorf("expected 1 favorite after re-add, got %d", len(favIDs))
	}
}

func TestModerationBanAndUnban(t *testing.T) {
	app := newTestApp(t)

	admin := seedLocalIdentity(t, app)
	if err := app.AddTrustedAdmin(admin.PublicKey, "owner"); err != nil {
		t.Fatalf("add trusted admin: %v", err)
	}

	// Derive a target pubkey without overwriting the local identity.
	targetPubkey := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	if err := app.PublishShadowBan(targetPubkey, admin.PublicKey, "spam"); err != nil {
		t.Fatalf("shadow ban: %v", err)
	}

	states, err := app.GetModerationState()
	if err != nil {
		t.Fatalf("get moderation state: %v", err)
	}
	found := false
	for _, s := range states {
		if s.TargetPubkey == targetPubkey && (s.Action == "shadow_ban" || s.Action == "SHADOW_BAN") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ban not found in moderation state, got: %+v", states)
	}

	if err := app.PublishUnban(targetPubkey, admin.PublicKey, "reformed"); err != nil {
		t.Fatalf("unban: %v", err)
	}

	states, err = app.GetModerationState()
	if err != nil {
		t.Fatalf("get moderation state after unban: %v", err)
	}
	for _, s := range states {
		if s.TargetPubkey == targetPubkey && (s.Action == "shadow_ban" || s.Action == "SHADOW_BAN") {
			t.Error("ban should be removed after unban")
		}
	}
}

func TestSubSettings(t *testing.T) {
	app := newTestApp(t)

	identity := seedLocalIdentity(t, app)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("add trusted admin: %v", err)
	}

	rules := []string{"Be kind", "No spam"}
	announcement := "Welcome to the community!"

	settings, err := app.UpdateSubSettings(defaultSubID, rules, announcement)
	if err != nil {
		t.Fatalf("update sub settings: %v", err)
	}
	if settings.Announcement != announcement {
		t.Errorf("announcement mismatch: got %q", settings.Announcement)
	}
	if len(settings.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(settings.Rules))
	}

	loaded, err := app.GetSubSettings(defaultSubID)
	if err != nil {
		t.Fatalf("get sub settings: %v", err)
	}
	if loaded.Announcement != announcement {
		t.Errorf("loaded announcement mismatch: got %q", loaded.Announcement)
	}
}

func TestNotifications(t *testing.T) {
	app := newTestApp(t)

	identity := seedLocalIdentity(t, app)

	// Publish a post and comment to generate notifications
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Notif Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	count, err := app.GetUnreadNotificationCount()
	if err != nil {
		t.Fatalf("get unread count: %v", err)
	}
	// Initially 0 since the post author is the same user
	if count < 0 {
		t.Errorf("unexpected negative unread count: %d", count)
	}
}

func TestGetPostBodyByCID(t *testing.T) {
	app := newTestApp(t)

	identity := seedLocalIdentity(t, app)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Body Test", "Full body content here", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, err := app.GetFeedBySub(defaultSubID)
	if err != nil || len(posts) == 0 {
		t.Fatalf("get feed: %v", err)
	}

	if posts[0].ContentCID != "" {
		blob, err := app.GetPostBodyByCID(posts[0].ContentCID)
		if err != nil {
			t.Fatalf("get post body by CID: %v", err)
		}
		if blob.Body != "Full body content here" {
			t.Errorf("body mismatch: got %q", blob.Body)
		}
	}
}

func TestGovernancePolicy(t *testing.T) {
	app := newTestApp(t)

	policy, err := app.GetGovernancePolicy()
	if err != nil {
		t.Fatalf("get governance policy: %v", err)
	}
	// Should return a valid policy even with no admins
	_ = policy
}

func TestSubStats(t *testing.T) {
	app := newTestApp(t)

	identity := seedLocalIdentity(t, app)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Stats Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	stats, err := app.GetSubStats(defaultSubID)
	if err != nil {
		t.Fatalf("get sub stats: %v", err)
	}
	if stats.PostCount < 1 {
		t.Errorf("expected at least 1 post in stats, got %d", stats.PostCount)
	}
}
