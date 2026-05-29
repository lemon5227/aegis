package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProcessIncomingMessagePost(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	msg := IncomingMessage{
		Type:          "POST",
		OpType:        postOpTypeCreate,
		OpID:          "test-op-1",
		SchemaVersion: lamportSchemaV2,
		AuthScope:     authScopeUser,
		ID:            "test-post-1",
		Pubkey:        identity.PublicKey,
		Title:         "Incoming Post",
		Body:          "Body content",
		ContentCID:    buildContentCID("Body content"),
		SubID:         defaultSubID,
		Timestamp:     time.Now().Unix(),
		Lamport:       1,
	}

	signed, err := app.signIncomingMessage(msg)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	payload, _ := json.Marshal(signed)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process incoming post: %v", err)
	}

	posts, _ := app.GetFeedBySub(defaultSubID)
	found := false
	for _, p := range posts {
		if p.Title == "Incoming Post" {
			found = true
		}
	}
	if !found {
		t.Error("processed post not found in feed")
	}
}

func TestProcessIncomingMessageComment(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	// First create a post
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Comment Target", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	msg := IncomingMessage{
		Type:          "COMMENT",
		OpType:        "CREATE",
		OpID:          "test-comment-op-1",
		SchemaVersion: lamportSchemaV2,
		AuthScope:     authScopeUser,
		ID:            "test-comment-1",
		PostID:        postID,
		Pubkey:        identity.PublicKey,
		Body:          "Incoming comment",
		Timestamp:     time.Now().Unix(),
		Lamport:       2,
	}

	signed, err := app.signIncomingMessage(msg)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	payload, _ := json.Marshal(signed)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process incoming comment: %v", err)
	}

	comments, _ := app.GetCommentsByPost(postID)
	if len(comments) == 0 {
		t.Error("processed comment not found")
	}
}

func TestProcessIncomingMessageProfileUpdate(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	msg := IncomingMessage{
		Type:        "PROFILE_UPDATE",
		Pubkey:      identity.PublicKey,
		DisplayName: "New Name",
		AvatarURL:   "https://example.com/new.png",
		Timestamp:   time.Now().Unix(),
	}

	signed, err := app.signIncomingMessage(msg)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	payload, _ := json.Marshal(signed)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process profile update: %v", err)
	}

	profile, err := app.GetProfile(identity.PublicKey)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.DisplayName != "New Name" {
		t.Errorf("display name mismatch: got %q", profile.DisplayName)
	}
}

func TestProcessIncomingMessageSubCreate(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	msg := IncomingMessage{
		Type:      "SUB_CREATE",
		SubID:     "new-sub",
		SubTitle:  "New Sub",
		SubDesc:   "A new sub-community",
		Timestamp: time.Now().Unix(),
	}

	payload, _ := json.Marshal(msg)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process sub create: %v", err)
	}

	subs, _ := app.GetSubs()
	found := false
	for _, s := range subs {
		if s.ID == "new-sub" {
			found = true
		}
	}
	if !found {
		t.Error("created sub not found")
	}
}

func TestProcessIncomingMessageVote(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Vote Target", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	msg := IncomingMessage{
		Type:        "POST_UPVOTE",
		PostID:      postID,
		VoterPubkey: identity.PublicKey,
		Pubkey:      identity.PublicKey,
		Timestamp:   time.Now().Unix(),
	}

	signed, err := app.signIncomingMessage(msg)
	if err != nil {
		t.Fatalf("sign vote: %v", err)
	}

	payload, _ := json.Marshal(signed)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process vote: %v", err)
	}
}

func TestProcessIncomingMessageInvalidJSON(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	err := app.ProcessIncomingMessage([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestProcessIncomingMessageUnknownType(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	msg := IncomingMessage{
		Type:      "UNKNOWN_TYPE",
		Timestamp: time.Now().Unix(),
	}
	payload, _ := json.Marshal(msg)
	// Unknown types should be silently ignored or return nil
	_ = app.ProcessIncomingMessage(payload)
}

func TestProcessIncomingMessagePostDelete(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	// Create a post first
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "To Delete", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	msg := IncomingMessage{
		Type:          "POST_DELETE",
		OpType:        "DELETE",
		OpID:          "delete-op-1",
		SchemaVersion: lamportSchemaV2,
		AuthScope:     authScopeUser,
		PostID:        postID,
		Pubkey:        identity.PublicKey,
		Timestamp:     time.Now().Unix(),
		Lamport:       10,
	}

	signed, err := app.signIncomingMessage(msg)
	if err != nil {
		t.Fatalf("sign delete: %v", err)
	}

	payload, _ := json.Marshal(signed)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process post delete: %v", err)
	}
}

func TestProcessIncomingMessageCommentDelete(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Comment Del Target", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "To be deleted")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	msg := IncomingMessage{
		Type:          "COMMENT_DELETE",
		OpType:        "DELETE",
		OpID:          "comment-delete-op-1",
		SchemaVersion: lamportSchemaV2,
		AuthScope:     authScopeUser,
		CommentID:     comment.ID,
		Pubkey:        identity.PublicKey,
		Timestamp:     time.Now().Unix(),
		Lamport:       10,
	}

	signed, err := app.signIncomingMessage(msg)
	if err != nil {
		t.Fatalf("sign comment delete: %v", err)
	}

	payload, _ := json.Marshal(signed)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process comment delete: %v", err)
	}
}

func TestProcessIncomingMessageGovernance(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("add admin: %v", err)
	}

	targetPubkey := "deadbeef1234567890deadbeef1234567890deadbeef1234567890deadbeef12"

	msg := IncomingMessage{
		Type:         "SHADOW_BAN",
		OpID:         "ban-op-1",
		TargetPubkey: targetPubkey,
		AdminPubkey:  identity.PublicKey,
		Reason:       "spam",
		Timestamp:    time.Now().Unix(),
		Lamport:      5,
	}

	signed, err := app.signIncomingMessage(msg)
	if err != nil {
		t.Fatalf("sign ban: %v", err)
	}

	payload, _ := json.Marshal(signed)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process shadow ban: %v", err)
	}

	// Now unban
	msg2 := IncomingMessage{
		Type:         "UNBAN",
		OpID:         "unban-op-1",
		TargetPubkey: targetPubkey,
		AdminPubkey:  identity.PublicKey,
		Reason:       "reformed",
		Timestamp:    time.Now().Unix() + 1,
		Lamport:      6,
	}

	signed2, err := app.signIncomingMessage(msg2)
	if err != nil {
		t.Fatalf("sign unban: %v", err)
	}

	payload2, _ := json.Marshal(signed2)
	if err := app.ProcessIncomingMessage(payload2); err != nil {
		t.Fatalf("process unban: %v", err)
	}
}

func TestProcessIncomingMessageCommentVote(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "CV Target", "body", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "Vote on me")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	msg := IncomingMessage{
		Type:        "COMMENT_UPVOTE",
		CommentID:   comment.ID,
		PostID:      postID,
		VoterPubkey: identity.PublicKey,
		Pubkey:      identity.PublicKey,
		Timestamp:   time.Now().Unix(),
	}

	signed, err := app.signIncomingMessage(msg)
	if err != nil {
		t.Fatalf("sign comment vote: %v", err)
	}

	payload, _ := json.Marshal(signed)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("process comment upvote: %v", err)
	}
}

func TestProcessIncomingMessageDownvote(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := retryOnBusy(t, 5, func() error {
		return app.PublishPostStructuredToSub(identity.PublicKey, "DV Target", "body", defaultSubID)
	}); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	msg := IncomingMessage{
		Type:        "POST_DOWNVOTE",
		PostID:      postID,
		VoterPubkey: identity.PublicKey,
		Pubkey:      identity.PublicKey,
		Timestamp:   time.Now().Unix(),
	}

	signed, err := app.signIncomingMessage(msg)
	if err != nil {
		t.Fatalf("sign downvote: %v", err)
	}

	payload, _ := json.Marshal(signed)
	if err := retryOnBusy(t, 5, func() error {
		return app.ProcessIncomingMessage(payload)
	}); err != nil {
		t.Fatalf("process post downvote: %v", err)
	}
}
