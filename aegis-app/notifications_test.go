package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tyler-smith/go-bip39"
)

// -----------------------------------------------------------------------------
// Pure helpers
// -----------------------------------------------------------------------------

func TestBuildNotificationIDDeterministic(t *testing.T) {
	a := buildNotificationID("post_upvote", "voter1", "post-abc")
	b := buildNotificationID("post_upvote", "voter1", "post-abc")
	if a != b {
		t.Fatalf("expected deterministic notification id, got %q vs %q", a, b)
	}
	if len(a) != 16 {
		t.Fatalf("expected 16-char id, got %d", len(a))
	}

	c := buildNotificationID("post_upvote", "voter2", "post-abc")
	if a == c {
		t.Fatalf("different sources should produce different ids")
	}
}

// -----------------------------------------------------------------------------
// maybeNotifyPostVote / maybeNotifyCommentVote behavior
// -----------------------------------------------------------------------------

// generateRemoteIdentity creates a fresh BIP39 mnemonic and returns the
// derived hex pubkey along with the mnemonic. Useful to simulate a remote
// peer that signs messages without overwriting the local app's identity.
func generateRemoteIdentity(t *testing.T) (mnemonic, pubkeyHex string) {
	t.Helper()
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		t.Fatalf("entropy: %v", err)
	}
	m, err := bip39.NewMnemonic(entropy)
	if err != nil {
		t.Fatalf("mnemonic: %v", err)
	}
	app := NewApp() // no DB — only used for pure derivation
	pub, _, err := app.deriveKeypairFromMnemonic(m)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	return m, hex.EncodeToString(pub)
}

// generateVoterPubkeyHex returns just a fresh ed25519 pubkey hex (no mnemonic),
// for tests that don't need to sign.
func generateVoterPubkeyHex(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate voter key: %v", err)
	}
	return hex.EncodeToString(pub)
}

// signAsRemote builds a signature payload for `msg`, signs it with
// `mnemonic` (foreign to the local app), and attaches the signature.
func signAsRemote(t *testing.T, app *App, mnemonic string, msg IncomingMessage) IncomingMessage {
	t.Helper()
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}
	msg = normalizeIncomingMessageForSignature(msg)
	payload, err := buildIncomingMessageSignaturePayload(msg)
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}
	sig, err := app.SignMessage(mnemonic, payload)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}
	msg.Signature = sig
	return msg
}

func TestMaybeNotifyPostVoteCreatesUpvoteNotification(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "title", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	voter := generateVoterPubkeyHex(t)

	app.maybeNotifyPostVote(postID, voter, "up", time.Now().Unix())

	page, err := app.GetNotifications(20, "")
	if err != nil {
		t.Fatalf("get notifications: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(page.Items))
	}
	got := page.Items[0]
	if got.Type != NotifTypePostUpvote {
		t.Errorf("expected type %q, got %q", NotifTypePostUpvote, got.Type)
	}
	if got.SourcePubkey != voter {
		t.Errorf("expected source %q, got %q", voter, got.SourcePubkey)
	}
	if got.PostID != postID {
		t.Errorf("expected postID %q, got %q", postID, got.PostID)
	}
	if got.IsRead {
		t.Error("new notification should be unread")
	}
}

func TestMaybeNotifyPostVoteDownvoteUsesDownvoteType(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	app.maybeNotifyPostVote(postID, generateVoterPubkeyHex(t), "DOWN", time.Now().Unix())

	page, _ := app.GetNotifications(20, "")
	if len(page.Items) != 1 || page.Items[0].Type != NotifTypePostDownvote {
		t.Fatalf("expected single downvote notification, got %+v", page.Items)
	}
}

func TestMaybeNotifyPostVoteSkipsWhenLocalIsNotAuthor(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Insert a "remote" post (author != local pubkey). All NOT NULL columns must be set.
	remoteAuthor := generateVoterPubkeyHex(t)
	now := time.Now().Unix()
	if _, err := app.db.Exec(
		`INSERT INTO messages (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility)
		 VALUES (?, ?, ?, 0, ?, 1, ?, 'public', ?, 'normal')`,
		"remote-post-1", remoteAuthor, "remote body", now, int64(len("remote body")), defaultSubID,
	); err != nil {
		t.Fatalf("insert remote post: %v", err)
	}

	app.maybeNotifyPostVote("remote-post-1", generateVoterPubkeyHex(t), "up", now)

	page, _ := app.GetNotifications(20, "")
	if len(page.Items) != 0 {
		t.Fatalf("expected no notification when local is not the author, got %d", len(page.Items))
	}
}

func TestMaybeNotifyPostVoteSkipsForUnknownPost(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Should not panic or error; just a no-op.
	app.maybeNotifyPostVote("nonexistent-post", generateVoterPubkeyHex(t), "up", time.Now().Unix())
	app.maybeNotifyPostVote("", generateVoterPubkeyHex(t), "up", time.Now().Unix())

	page, _ := app.GetNotifications(20, "")
	if len(page.Items) != 0 {
		t.Fatalf("expected no notifications, got %d", len(page.Items))
	}
}

func TestMaybeNotifyPostVoteIdempotentForSameVoter(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	voter := generateVoterPubkeyHex(t)

	now := time.Now().Unix()
	app.maybeNotifyPostVote(postID, voter, "up", now)
	app.maybeNotifyPostVote(postID, voter, "up", now+1)
	app.maybeNotifyPostVote(postID, voter, "up", now+2)

	page, _ := app.GetNotifications(20, "")
	if len(page.Items) != 1 {
		t.Fatalf("dedupe should keep a single row per (type, source, target), got %d", len(page.Items))
	}
}

func TestMaybeNotifyCommentVoteCreatesNotification(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "comment body")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	voter := generateVoterPubkeyHex(t)
	app.maybeNotifyCommentVote(comment.ID, postID, voter, "up", time.Now().Unix())

	page, _ := app.GetNotifications(20, "")
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(page.Items))
	}
	got := page.Items[0]
	if got.Type != NotifTypeCommentUpvote {
		t.Errorf("expected %q, got %q", NotifTypeCommentUpvote, got.Type)
	}
	if got.TargetEntityID != comment.ID {
		t.Errorf("expected target %q, got %q", comment.ID, got.TargetEntityID)
	}
	if got.PostID != postID {
		t.Errorf("expected postID %q, got %q", postID, got.PostID)
	}
}

func TestMaybeNotifyCommentVoteSkipsWhenLocalIsNotAuthor(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	// Insert a comment whose author is *not* the local user.
	remoteAuthor := generateVoterPubkeyHex(t)
	now := time.Now().Unix()
	if _, err := app.db.Exec(
		`INSERT INTO comments (id, post_id, parent_id, pubkey, body, timestamp, lamport, current_op_id)
		 VALUES (?, ?, '', ?, 'remote', ?, 1, 'op-remote')`,
		"remote-comment-1", postID, remoteAuthor, now,
	); err != nil {
		t.Fatalf("insert remote comment: %v", err)
	}

	app.maybeNotifyCommentVote("remote-comment-1", postID, generateVoterPubkeyHex(t), "up", now)

	page, _ := app.GetNotifications(20, "")
	if len(page.Items) != 0 {
		t.Fatalf("expected 0 notifications, got %d", len(page.Items))
	}
}

// -----------------------------------------------------------------------------
// VOTE_SET path (previously uncovered) — verifies the refactored helper still
// classifies up vs down correctly through ProcessIncomingMessage. The voter is
// a remote identity, so signing happens with a foreign mnemonic.
// -----------------------------------------------------------------------------

func TestProcessIncomingMessagePostVoteSetTriggersNotification(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "title", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	voterMnemonic, voterPubkey := generateRemoteIdentity(t)

	msg := signAsRemote(t, app, voterMnemonic, IncomingMessage{
		Type:        "POST_VOTE_SET",
		OpID:        "vs-op-1",
		PostID:      postID,
		VoterPubkey: voterPubkey,
		Pubkey:      voterPubkey,
		VoteState:   "down",
		Timestamp:   time.Now().Unix(),
	})
	payload, _ := json.Marshal(msg)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process vote-set: %v", err)
	}

	page, _ := app.GetNotifications(20, "")
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(page.Items))
	}
	if page.Items[0].Type != NotifTypePostDownvote {
		t.Errorf("vote_state=down should map to %q, got %q", NotifTypePostDownvote, page.Items[0].Type)
	}
	if page.Items[0].SourcePubkey != voterPubkey {
		t.Errorf("expected source pubkey %q, got %q", voterPubkey, page.Items[0].SourcePubkey)
	}
}

func TestProcessIncomingMessageCommentVoteSetTriggersNotification(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "body")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}
	voterMnemonic, voterPubkey := generateRemoteIdentity(t)

	msg := signAsRemote(t, app, voterMnemonic, IncomingMessage{
		Type:        "COMMENT_VOTE_SET",
		OpID:        "cvs-op-1",
		PostID:      postID,
		CommentID:   comment.ID,
		VoterPubkey: voterPubkey,
		Pubkey:      voterPubkey,
		VoteState:   "up",
		Timestamp:   time.Now().Unix(),
	})
	payload, _ := json.Marshal(msg)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process cvs: %v", err)
	}

	page, _ := app.GetNotifications(20, "")
	if len(page.Items) != 1 || page.Items[0].Type != NotifTypeCommentUpvote {
		t.Fatalf("expected single comment_upvote notification, got %+v", page.Items)
	}
}

// -----------------------------------------------------------------------------
// GetNotifications cursor pagination (covers refactored scanNotifications)
// -----------------------------------------------------------------------------

func TestGetNotificationsPaginationCursorRoundtrip(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "title", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	// Generate 5 distinct voter keys so each vote produces a unique notification.
	now := time.Now().Unix()
	for i := 0; i < 5; i++ {
		app.maybeNotifyPostVote(postID, generateVoterPubkeyHex(t), "up", now+int64(i))
	}

	first, err := app.GetNotifications(2, "")
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(first.Items))
	}
	if strings.TrimSpace(first.NextCursor) == "" {
		t.Fatalf("expected non-empty cursor when more pages exist")
	}

	second, err := app.GetNotifications(2, first.NextCursor)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Items) != 2 {
		t.Fatalf("expected 2 items on second page, got %d", len(second.Items))
	}

	third, err := app.GetNotifications(2, second.NextCursor)
	if err != nil {
		t.Fatalf("third page: %v", err)
	}
	if len(third.Items) != 1 {
		t.Fatalf("expected 1 item on final page, got %d", len(third.Items))
	}
	if third.NextCursor != "" {
		t.Fatalf("expected empty cursor on final page, got %q", third.NextCursor)
	}

	// Deduplicate by id across pages — they should all be unique.
	seen := map[string]struct{}{}
	for _, n := range append(append(first.Items, second.Items...), third.Items...) {
		if _, dup := seen[n.ID]; dup {
			t.Fatalf("duplicate notification id %q across pages", n.ID)
		}
		seen[n.ID] = struct{}{}
	}
}

func TestGetNotificationsInvalidCursorReturnsError(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if _, err := app.GetNotifications(10, "%%%not-base64%%%"); err == nil {
		t.Fatal("expected error for malformed cursor")
	}
}

func TestGetNotificationsEmptyDBReturnsEmptyPage(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	page, err := app.GetNotifications(10, "")
	if err != nil {
		t.Fatalf("get notifications: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("expected empty list, got %d", len(page.Items))
	}
	if page.NextCursor != "" {
		t.Errorf("expected empty cursor, got %q", page.NextCursor)
	}
	if page.Items == nil {
		t.Error("Items should be non-nil even when empty (frontend expects [])")
	}
}


// -----------------------------------------------------------------------------
// Mark-read mutations + unread count
// -----------------------------------------------------------------------------

func seedNotificationsForLocalAuthor(t *testing.T, app *App, postID string, n int) []string {
	t.Helper()
	now := time.Now().Unix()
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		voter := generateVoterPubkeyHex(t)
		app.maybeNotifyPostVote(postID, voter, "up", now+int64(i))
		ids = append(ids, buildNotificationID(NotifTypePostUpvote, voter, postID))
	}
	return ids
}

func TestGetUnreadNotificationCountTracksReads(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	if count, err := app.GetUnreadNotificationCount(); err != nil || count != 0 {
		t.Fatalf("expected 0 unread on empty db, got count=%d err=%v", count, err)
	}

	ids := seedNotificationsForLocalAuthor(t, app, postID, 3)

	count, err := app.GetUnreadNotificationCount()
	if err != nil {
		t.Fatalf("get unread: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 unread, got %d", count)
	}

	if err := app.MarkNotificationRead(ids[0]); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	count, _ = app.GetUnreadNotificationCount()
	if count != 2 {
		t.Fatalf("expected 2 unread after marking 1, got %d", count)
	}

	// Idempotent — second call should be a no-op.
	if err := app.MarkNotificationRead(ids[0]); err != nil {
		t.Fatalf("mark read (second time): %v", err)
	}
	count, _ = app.GetUnreadNotificationCount()
	if count != 2 {
		t.Fatalf("idempotent mark-read changed count: got %d, want 2", count)
	}

	// Marking a non-existent id should not error.
	if err := app.MarkNotificationRead("does-not-exist"); err != nil {
		t.Errorf("mark-read of unknown id should be silent, got %v", err)
	}
}

func TestMarkAllNotificationsReadFlipsAll(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "t", "b", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	seedNotificationsForLocalAuthor(t, app, postID, 5)

	if err := app.MarkAllNotificationsRead(); err != nil {
		t.Fatalf("mark all read: %v", err)
	}

	count, _ := app.GetUnreadNotificationCount()
	if count != 0 {
		t.Fatalf("expected 0 unread after MarkAll, got %d", count)
	}

	page, _ := app.GetNotifications(20, "")
	for _, n := range page.Items {
		if !n.IsRead {
			t.Errorf("notification %s should be read after MarkAll", n.ID)
		}
	}

	// Calling MarkAll again on an empty unread set should be a no-op.
	if err := app.MarkAllNotificationsRead(); err != nil {
		t.Errorf("idempotent MarkAll should not error, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// Vote-validation helpers (Round 2 extraction) — pure-function unit tests.
// -----------------------------------------------------------------------------

func TestResolvePostVoteFieldsHappyPath(t *testing.T) {
	voter, postID, err := resolvePostVoteFields(IncomingMessage{
		VoterPubkey: " voter-key ",
		PostID:      " post-1 ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if voter != "voter-key" {
		t.Errorf("voter not trimmed: %q", voter)
	}
	if postID != "post-1" {
		t.Errorf("postID not trimmed: %q", postID)
	}
}

func TestResolvePostVoteFieldsFallsBackToPubkey(t *testing.T) {
	voter, _, err := resolvePostVoteFields(IncomingMessage{
		Pubkey: "fallback-key",
		PostID: "p",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if voter != "fallback-key" {
		t.Errorf("expected fallback to Pubkey, got %q", voter)
	}
}

func TestResolvePostVoteFieldsRejectsMissing(t *testing.T) {
	cases := []struct {
		name string
		msg  IncomingMessage
	}{
		{"no voter or post", IncomingMessage{}},
		{"voter only", IncomingMessage{VoterPubkey: "v"}},
		{"post only", IncomingMessage{PostID: "p"}},
		{"whitespace only", IncomingMessage{VoterPubkey: "   ", PostID: "  "}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := resolvePostVoteFields(tc.msg); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestResolveCommentVoteFieldsHappyPath(t *testing.T) {
	voter, commentID, postID, err := resolveCommentVoteFields(IncomingMessage{
		VoterPubkey: "v",
		CommentID:   "c",
		PostID:      "p",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if voter != "v" || commentID != "c" || postID != "p" {
		t.Errorf("unexpected: voter=%q commentID=%q postID=%q", voter, commentID, postID)
	}
}

func TestResolveCommentVoteFieldsRejectsMissing(t *testing.T) {
	cases := []struct {
		name string
		msg  IncomingMessage
	}{
		{"all empty", IncomingMessage{}},
		{"missing comment id", IncomingMessage{VoterPubkey: "v", PostID: "p"}},
		{"missing post id", IncomingMessage{VoterPubkey: "v", CommentID: "c"}},
		{"missing voter", IncomingMessage{CommentID: "c", PostID: "p"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := resolveCommentVoteFields(tc.msg); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}
