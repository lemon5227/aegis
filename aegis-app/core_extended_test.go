package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestAppWithIdentity(t *testing.T) (*App, Identity) {
	t.Helper()
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "test.db"))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("init database: %v", err)
	}
	identity := seedLocalIdentity(t, app)
	t.Cleanup(func() {
		if app.db != nil {
			// Checkpoint WAL and truncate before closing so the -wal and -shm
			// files are cleaned up and won't block t.TempDir() removal.
			_, _ = app.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
			_ = app.db.Close()
		}
	})
	return app, identity
}

// retryOnBusy retries an operation that may fail with SQLITE_BUSY.
// Returns the last error if all retries fail.
func retryOnBusy(t *testing.T, attempts int, fn func() error) error {
	t.Helper()
	var lastErr error
	for i := 0; i < attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		// Retry on lock errors
		if !strings.Contains(strings.ToLower(err.Error()), "database is locked") &&
			!strings.Contains(strings.ToLower(err.Error()), "busy") {
			return err
		}
		time.Sleep(time.Duration(50*(i+1)) * time.Millisecond)
	}
	return lastErr
}

func TestUpdateProfileAndGetDetails(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	profile, err := app.UpdateProfile("Alice", "https://example.com/avatar.png")
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if profile.DisplayName != "Alice" {
		t.Errorf("display name mismatch: got %q", profile.DisplayName)
	}

	details, err := app.UpdateProfileDetails("Alice Updated", "https://example.com/avatar2.png", "Hello world bio")
	if err != nil {
		t.Fatalf("update profile details: %v", err)
	}
	if details.Bio != "Hello world bio" {
		t.Errorf("bio mismatch: got %q", details.Bio)
	}
	if details.DisplayName != "Alice Updated" {
		t.Errorf("display name mismatch: got %q", details.DisplayName)
	}

	loaded, err := app.GetProfileDetails(identity.PublicKey)
	if err != nil {
		t.Fatalf("get profile details: %v", err)
	}
	if loaded.Bio != "Hello world bio" {
		t.Errorf("loaded bio mismatch: got %q", loaded.Bio)
	}
}

func TestUpdateProfileDetailsTruncation(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	longBio := ""
	for i := 0; i < 200; i++ {
		longBio += "x"
	}

	details, err := app.UpdateProfileDetails("Name", "", longBio)
	if err != nil {
		t.Fatalf("update profile details: %v", err)
	}
	if len([]rune(details.Bio)) > 160 {
		t.Errorf("bio should be truncated to 160 chars, got %d", len([]rune(details.Bio)))
	}
}

func TestUpvoteAndDownvotePost(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Vote Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, err := app.GetFeedBySub(defaultSubID)
	if err != nil || len(posts) == 0 {
		t.Fatalf("get feed: %v", err)
	}
	postID := posts[0].ID

	if err := app.UpvotePost(postID); err != nil {
		t.Fatalf("upvote: %v", err)
	}

	posts, _ = app.GetFeedBySub(defaultSubID)
	if posts[0].Score < 1 {
		t.Errorf("expected score >= 1 after upvote, got %d", posts[0].Score)
	}

	if err := app.DownvotePost(postID); err != nil {
		t.Fatalf("downvote: %v", err)
	}
}

func TestGetFeedStream(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	for i := 0; i < 3; i++ {
		if err := app.PublishPostStructuredToSub(identity.PublicKey, "Stream Post", "body", defaultSubID); err != nil {
			t.Fatalf("publish post %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	stream, err := app.GetFeedStream(10)
	if err != nil {
		t.Fatalf("get feed stream: %v", err)
	}
	if len(stream.Items) < 3 {
		t.Errorf("expected at least 3 items in stream, got %d", len(stream.Items))
	}
}

func TestGetFeedStreamWithStrategy(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Strategy Post", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	stream, err := app.GetFeedStreamWithStrategy(10, "hot-v1")
	if err != nil {
		t.Fatalf("get feed stream hot-v1: %v", err)
	}
	if len(stream.Items) < 1 {
		t.Errorf("expected at least 1 item, got %d", len(stream.Items))
	}

	// Invalid strategy should fallback
	stream, err = app.GetFeedStreamWithStrategy(10, "nonexistent")
	if err != nil {
		t.Fatalf("get feed stream fallback: %v", err)
	}
	_ = stream
}

func TestGetStorageUsage(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Storage Test", "some body content", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	usage, err := app.GetStorageUsage()
	if err != nil {
		t.Fatalf("get storage usage: %v", err)
	}
	// Should have some bytes used after publishing
	if usage.PublicUsedBytes < 0 {
		t.Errorf("public used bytes should not be negative: %d", usage.PublicUsedBytes)
	}
}

func TestGetPrivacySettings(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	settings, err := app.GetPrivacySettings()
	if err != nil {
		t.Fatalf("get privacy settings: %v", err)
	}
	// Default settings should be returned
	_ = settings
}

func TestSetPrivacySettings(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	_, err := app.SetPrivacySettings(false, false)
	if err != nil {
		t.Fatalf("set privacy settings: %v", err)
	}

	settings, err := app.GetPrivacySettings()
	if err != nil {
		t.Fatalf("get privacy settings: %v", err)
	}
	if settings.ShowOnlineStatus != false {
		t.Error("expected show online status to be false")
	}
}

func TestGetNotificationsWithCursor(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	page, err := app.GetNotifications(10, "")
	if err != nil {
		t.Fatalf("get notifications: %v", err)
	}
	// Should return valid page without error
	_ = page
}

func TestMarkAllNotificationsRead(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.MarkAllNotificationsRead(); err != nil {
		t.Fatalf("mark all read: %v", err)
	}

	count, err := app.GetUnreadNotificationCount()
	if err != nil {
		t.Fatalf("get unread count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 unread after mark all, got %d", count)
	}
}

func TestSubscribeAndUnsubscribeSub(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.CreateSub("sub-test", "Test Sub", "desc"); err != nil {
		t.Fatalf("create sub: %v", err)
	}

	if _, err := app.SubscribeSub("sub-test"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	subs, err := app.GetSubscribedSubs()
	if err != nil {
		t.Fatalf("get subscribed: %v", err)
	}
	found := false
	for _, s := range subs {
		if s.ID == "sub-test" {
			found = true
		}
	}
	if !found {
		t.Error("subscribed sub not found")
	}

	if err := app.UnsubscribeSub("sub-test"); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
}

func TestGetModerationLogs(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	logs, err := app.GetModerationLogs(10)
	if err != nil {
		t.Fatalf("get moderation logs: %v", err)
	}
	// Should return empty list without error
	_ = logs
}

func TestGetVersionHistory(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	history, err := app.GetVersionHistory(10)
	if err != nil {
		t.Fatalf("get version history: %v", err)
	}
	_ = history
}

func TestGetReleaseMetrics(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	metrics := app.GetReleaseMetrics()
	// Should return valid metrics struct
	if metrics.ContentFetchAttempts < 0 {
		t.Error("content fetch attempts should not be negative")
	}
}

func TestGetAntiEntropyStats(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	stats := app.GetAntiEntropyStats()
	// Should return valid stats struct
	_ = stats
}

func TestListEntityOps(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Entity Ops Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, _ := app.GetFeedBySub(defaultSubID)
	if len(posts) == 0 {
		t.Fatal("no posts")
	}

	// ListEntityOps requires dev mode; verify it returns appropriate error
	_, err := app.ListEntityOps("post", posts[0].ID, 10)
	if err == nil {
		// If dev mode is enabled, we should get results
		return
	}
	// Expected error in non-dev mode
	if err.Error() != "timeline is available in dev mode only" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostIndexByID(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Index Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, _ := app.GetFeedBySub(defaultSubID)
	if len(posts) == 0 {
		t.Fatal("no posts")
	}

	index, err := app.GetPostIndexByID(posts[0].ID)
	if err != nil {
		t.Fatalf("get post index by id: %v", err)
	}
	if index.Title != "Index Test" {
		t.Errorf("title mismatch: got %q", index.Title)
	}
}

func TestDeletePost(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Delete Me", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, _ := app.GetFeedBySub(defaultSubID)
	if len(posts) == 0 {
		t.Fatal("no posts")
	}
	postID := posts[0].ID

	if err := app.PublishDeletePost(identity.PublicKey, postID); err != nil {
		t.Fatalf("delete post: %v", err)
	}

	posts, _ = app.GetFeedBySub(defaultSubID)
	for _, p := range posts {
		if p.ID == postID && p.Visibility != "deleted" {
			t.Error("deleted post should not appear in feed or should be marked deleted")
		}
	}
}

func TestDeleteComment(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Comment Delete Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "To be deleted")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	if err := app.PublishDeleteComment(identity.PublicKey, comment.ID); err != nil {
		t.Fatalf("delete comment: %v", err)
	}
}

func TestP2PConfigSaveAndLoad(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	saved, err := app.SaveP2PConfig(40200, []string{"/ip4/1.2.3.4/tcp/40100/p2p/QmTest"}, true)
	if err != nil {
		t.Fatalf("save p2p config: %v", err)
	}
	if saved.ListenPort != 40200 {
		t.Errorf("listen port mismatch: got %d", saved.ListenPort)
	}

	loaded, err := app.GetP2PConfig()
	if err != nil {
		t.Fatalf("get p2p config: %v", err)
	}
	if loaded.ListenPort != 40200 {
		t.Errorf("loaded listen port mismatch: got %d", loaded.ListenPort)
	}
	if !loaded.AutoStart {
		t.Error("auto start should be true")
	}
}

func TestSetPostPinnedAndLocked(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("add admin: %v", err)
	}

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Pin Lock Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	if err := app.SetPostPinned(postID, true); err != nil {
		t.Fatalf("set pinned: %v", err)
	}
	if err := app.SetPostLocked(postID, true); err != nil {
		t.Fatalf("set locked: %v", err)
	}

	index, _ := app.GetPostIndexByID(postID)
	if !index.IsPinned {
		t.Error("post should be pinned")
	}
	if !index.IsLocked {
		t.Error("post should be locked")
	}
}

func TestGetPostBodyByID(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Body By ID", "full body text here", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, _ := app.GetFeedBySub(defaultSubID)
	if len(posts) == 0 {
		t.Fatal("no posts")
	}

	blob, err := app.GetPostBodyByID(posts[0].ID)
	if err != nil {
		t.Fatalf("get post body by id: %v", err)
	}
	if blob.Body != "full body text here" {
		t.Errorf("body mismatch: got %q", blob.Body)
	}
}

func TestRunTombstoneGC(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	result, err := app.RunTombstoneGC(30, 2, 100)
	if err != nil {
		t.Fatalf("run tombstone gc: %v", err)
	}
	// Should complete without error even with no tombstones
	_ = result
}

func TestGetReleaseAlerts(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	alerts := app.GetReleaseAlerts()
	// Should return empty slice without panic
	if alerts == nil {
		// nil is acceptable for empty alerts
	}
}

func TestResetLocalTestData(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Reset Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	if err := app.ResetLocalTestData(); err != nil {
		t.Fatalf("reset local test data: %v", err)
	}

	posts, _ := app.GetFeedBySub(defaultSubID)
	if len(posts) != 0 {
		t.Errorf("expected 0 posts after reset, got %d", len(posts))
	}
}

func TestGetFeedBySubSorted(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	for i := 0; i < 3; i++ {
		if err := app.PublishPostStructuredToSub(identity.PublicKey, "Sort Test", "body", defaultSubID); err != nil {
			t.Fatalf("publish: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	posts, err := app.GetFeedBySubSorted(defaultSubID, "new")
	if err != nil {
		t.Fatalf("get feed sorted new: %v", err)
	}
	if len(posts) < 3 {
		t.Errorf("expected 3+ posts, got %d", len(posts))
	}

	posts, err = app.GetFeedBySubSorted(defaultSubID, "hot")
	if err != nil {
		t.Fatalf("get feed sorted hot: %v", err)
	}
	if len(posts) < 3 {
		t.Errorf("expected 3+ posts for hot, got %d", len(posts))
	}
}

func TestAddCommentWithAttachments(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Attach Test", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	attachments := []CommentAttachment{
		{Kind: "external_url", Ref: "https://example.com/image.png", Mime: "image/png"},
	}

	comment, err := app.AddLocalCommentWithAttachments(identity.PublicKey, postID, "", "With attachment", attachments)
	if err != nil {
		t.Fatalf("add comment with attachments: %v", err)
	}
	if len(comment.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(comment.Attachments))
	}
}

func TestGetCommentsByPostEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	comments, err := app.GetCommentsByPost("nonexistent-post-id")
	if err != nil {
		t.Fatalf("get comments: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comments for nonexistent post, got %d", len(comments))
	}
}

func TestGetFeedEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	posts, err := app.GetFeed()
	if err != nil {
		t.Fatalf("get feed: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts in empty feed, got %d", len(posts))
	}
}

func TestGetMyPosts(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "My Post 1", "body", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	page, err := app.GetMyPosts(10, "")
	if err != nil {
		t.Fatalf("get my posts: %v", err)
	}
	if len(page.Items) < 1 {
		t.Errorf("expected at least 1 post, got %d", len(page.Items))
	}
}

func TestSearchPosts(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Searchable Unique Title", "searchable body", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	results, err := app.SearchPosts("Searchable Unique", "", 10)
	if err != nil {
		t.Fatalf("search posts: %v", err)
	}
	if len(results) < 1 {
		t.Errorf("expected at least 1 search result, got %d", len(results))
	}
}

func TestIsFavorited(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Fav Check", "body", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	fav, err := app.IsFavorited(postID)
	if err != nil {
		t.Fatalf("is favorited: %v", err)
	}
	if fav {
		t.Error("should not be favorited initially")
	}

	if err := app.AddFavorite(postID); err != nil {
		t.Fatalf("add favorite: %v", err)
	}

	fav, err = app.IsFavorited(postID)
	if err != nil {
		t.Fatalf("is favorited after add: %v", err)
	}
	if !fav {
		t.Error("should be favorited after add")
	}
}

func TestRemoveFavorite(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := retryOnBusy(t, 5, func() error {
		return app.PublishPostStructuredToSub(identity.PublicKey, "Remove Fav", "body", defaultSubID)
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	if err := retryOnBusy(t, 5, func() error {
		return app.AddFavorite(postID)
	}); err != nil {
		t.Fatalf("add favorite: %v", err)
	}

	if err := retryOnBusy(t, 5, func() error {
		return app.RemoveFavorite(postID)
	}); err != nil {
		t.Fatalf("remove favorite: %v", err)
	}
}

func TestUpvoteComment(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Comment Vote", "body", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "Vote me")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	if err := app.UpvoteComment(comment.ID); err != nil {
		t.Fatalf("upvote comment: %v", err)
	}
}

func TestGetPostsByAuthor(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Author Post", "body", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	posts, err := app.GetPostsByAuthor(identity.PublicKey, 10)
	if err != nil {
		t.Fatalf("get posts by author: %v", err)
	}
	if len(posts) < 1 {
		t.Errorf("expected at least 1 post by author, got %d", len(posts))
	}
}

func TestGetFavorites(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Fav Page", "body", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	if err := app.AddFavorite(postID); err != nil {
		t.Fatalf("add favorite: %v", err)
	}

	page, err := app.GetFavorites(10, "")
	if err != nil {
		t.Fatalf("get favorites: %v", err)
	}
	if len(page.Items) < 1 {
		t.Errorf("expected at least 1 favorite, got %d", len(page.Items))
	}
}
