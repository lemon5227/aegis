package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func newC1TestApp(t *testing.T) *App {
	t.Helper()

	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "c1_test.db"))
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

func TestC1GovernancePolicyAffectsHistoricalVisibility(t *testing.T) {
	app := newC1TestApp(t)
	pubkey := "target-user"
	now := time.Now().Unix()

	message := ForumMessage{
		ID:          buildMessageID(pubkey, "history-post", now-3600),
		Pubkey:      pubkey,
		Title:       "history",
		Body:        "history body",
		Content:     "",
		Score:       0,
		Timestamp:   now - 3600,
		Zone:        "public",
		SubID:       "general",
		Visibility:  "normal",
		IsProtected: 0,
	}
	if _, err := app.insertMessage(message); err != nil {
		t.Fatalf("insertMessage failed: %v", err)
	}

	if _, err := app.SetGovernancePolicy(false); err != nil {
		t.Fatalf("SetGovernancePolicy(false) failed: %v", err)
	}
	if err := app.upsertModeration(pubkey, "SHADOW_BAN", "admin-a", now, "future-only"); err != nil {
		t.Fatalf("upsertModeration shadow ban (policy off) failed: %v", err)
	}

	feed, err := app.GetFeedBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted failed: %v", err)
	}
	if len(feed) != 1 {
		t.Fatalf("expected 1 visible historical post when policy off, got %d", len(feed))
	}

	if _, err = app.SetGovernancePolicy(true); err != nil {
		t.Fatalf("SetGovernancePolicy(true) failed: %v", err)
	}
	if err = app.upsertModeration(pubkey, "SHADOW_BAN", "admin-a", now+1, "hide-history"); err != nil {
		t.Fatalf("upsertModeration shadow ban (policy on) failed: %v", err)
	}

	feed, err = app.GetFeedBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted failed: %v", err)
	}
	if len(feed) != 0 {
		t.Fatalf("expected 0 visible historical posts when policy on, got %d", len(feed))
	}

	if _, err = app.SetGovernancePolicy(false); err != nil {
		t.Fatalf("SetGovernancePolicy(false) failed: %v", err)
	}
	feed, err = app.GetFeedBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted failed: %v", err)
	}
	if len(feed) != 1 {
		t.Fatalf("expected 1 visible historical post after policy off restore, got %d", len(feed))
	}

	if _, err = app.SetGovernancePolicy(true); err != nil {
		t.Fatalf("SetGovernancePolicy(true) failed: %v", err)
	}
	feed, err = app.GetFeedBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted failed: %v", err)
	}
	if len(feed) != 0 {
		t.Fatalf("expected 0 visible historical posts after policy on reapply, got %d", len(feed))
	}

	if err = app.upsertModeration(pubkey, "UNBAN", "admin-a", now+2, "restore"); err != nil {
		t.Fatalf("upsertModeration unban failed: %v", err)
	}
	feed, err = app.GetFeedBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted failed: %v", err)
	}
	if len(feed) != 1 {
		t.Fatalf("expected 1 visible post after unban, got %d", len(feed))
	}
}

func TestC1ModerationLogsRecorded(t *testing.T) {
	app := newC1TestApp(t)
	now := time.Now().Unix()

	if err := app.upsertModeration("target-a", "SHADOW_BAN", "admin-a", now, "r1"); err != nil {
		t.Fatalf("upsertModeration shadow ban failed: %v", err)
	}
	if err := app.upsertModeration("target-a", "UNBAN", "admin-a", now+1, "r2"); err != nil {
		t.Fatalf("upsertModeration unban failed: %v", err)
	}

	logs, err := app.GetModerationLogs(10)
	if err != nil {
		t.Fatalf("GetModerationLogs failed: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 logs, got %d", len(logs))
	}
	if logs[0].Result != "applied" {
		t.Fatalf("expected latest log result=applied, got %s", logs[0].Result)
	}
	if logs[0].Action != "UNBAN" {
		t.Fatalf("expected latest log action=UNBAN, got %s", logs[0].Action)
	}
	if logs[1].Action != "SHADOW_BAN" {
		t.Fatalf("expected second log action=SHADOW_BAN, got %s", logs[1].Action)
	}
}

func TestC1ShadowBannedContentIsLocalOnly(t *testing.T) {
	app := newC1TestApp(t)
	now := time.Now().Unix()
	bannedPubkey := "target-a"

	if err := app.upsertModeration(bannedPubkey, "SHADOW_BAN", "admin-a", now, "ban"); err != nil {
		t.Fatalf("upsertModeration shadow ban failed: %v", err)
	}

	post, err := app.AddLocalPostStructuredToSub(bannedPubkey, "t", "b", "public", "general")
	if err != nil {
		t.Fatalf("expected AddLocalPostStructuredToSub to allow shadow-banned user local post: %v", err)
	}

	if _, err := app.AddLocalComment(bannedPubkey, post.ID, "", "comment"); err != nil {
		t.Fatalf("expected AddLocalComment to allow shadow-banned user local comment: %v", err)
	}

	if err := app.PublishPostStructuredToSub(bannedPubkey, "t2", "b2", "general"); err != nil {
		t.Fatalf("expected PublishPostStructuredToSub to keep local-only post for shadow-banned user: %v", err)
	}

	if err := app.PublishComment(bannedPubkey, post.ID, "", "comment2"); err != nil {
		t.Fatalf("expected PublishComment to keep local-only comment for shadow-banned user: %v", err)
	}

	feed, err := app.GetFeedBySubSorted("general", "new")
	if err != nil {
		t.Fatalf("GetFeedBySubSorted failed: %v", err)
	}
	if len(feed) == 0 {
		t.Fatalf("expected local shadow-banned user to still see own posts")
	}
}

func TestC1IncomingBannedCommentIgnored(t *testing.T) {
	app := newC1TestApp(t)
	now := time.Now().Unix()
	bannedPubkey := "target-b"

	if err := app.upsertModeration(bannedPubkey, "SHADOW_BAN", "admin-a", now, "ban"); err != nil {
		t.Fatalf("upsertModeration shadow ban failed: %v", err)
	}

	payload := []byte(fmt.Sprintf(`{"type":"COMMENT","id":"c-ban-1","pubkey":"%s","post_id":"post-1","parent_id":"","body":"should-not-appear","timestamp":%d}`,
		bannedPubkey,
		now+1,
	))

	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("ProcessIncomingMessage failed: %v", err)
	}

	comments, err := app.GetCommentsByPost("post-1")
	if err != nil {
		t.Fatalf("GetCommentsByPost failed: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("expected banned comment to be ignored, got %d comments", len(comments))
	}
}

func TestC1ModerationIgnoresOlderOutOfOrderUpdates(t *testing.T) {
	app := newC1TestApp(t)
	now := time.Now().Unix()
	target := "target-out-of-order"

	if err := app.upsertModeration(target, "UNBAN", "admin-a", now+10, "newer-unban"); err != nil {
		t.Fatalf("upsertModeration newer UNBAN failed: %v", err)
	}
	if err := app.upsertModeration(target, "SHADOW_BAN", "admin-a", now, "older-shadow-ban"); err != nil {
		t.Fatalf("upsertModeration older SHADOW_BAN failed: %v", err)
	}

	shadowed, err := app.isShadowBanned(target)
	if err != nil {
		t.Fatalf("isShadowBanned failed: %v", err)
	}
	if shadowed {
		t.Fatalf("expected target to remain unbanned after older out-of-order update")
	}

	logs, err := app.GetModerationLogs(20)
	if err != nil {
		t.Fatalf("GetModerationLogs failed: %v", err)
	}

	foundIgnoredOlder := false
	for _, entry := range logs {
		if entry.TargetPubkey == target && entry.Result == "ignored_older" {
			foundIgnoredOlder = true
			break
		}
	}

	if !foundIgnoredOlder {
		t.Fatalf("expected ignored_older moderation log entry for out-of-order update")
	}
}

func TestC1PublishGovernanceWithoutP2PRejectedAndNoLog(t *testing.T) {
	app := newC1TestApp(t)

	err := app.PublishUnban("target-offline", "admin-a", "offline-attempt")
	if err == nil {
		t.Fatalf("expected PublishUnban to fail when p2p is not started")
	}

	states, err := app.GetModerationState()
	if err != nil {
		t.Fatalf("GetModerationState failed: %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("expected no moderation state written on rejected offline governance, got %d", len(states))
	}

	logs, err := app.GetModerationLogs(20)
	if err != nil {
		t.Fatalf("GetModerationLogs failed: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("expected no moderation logs written on rejected offline governance, got %d", len(logs))
	}
}
