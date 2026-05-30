package main

import (
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Peer policy helpers
// =============================================================================

func TestParsePeerIDSet(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", []string{}},
		{"whitespace only", "   ", []string{}},
		{"single", "12D3KooWAa", []string{"12D3KooWAa"}},
		{"multiple csv", "peer-a,peer-b,peer-c", []string{"peer-a", "peer-b", "peer-c"}},
		{"trims whitespace", " peer-a , peer-b ,peer-c ", []string{"peer-a", "peer-b", "peer-c"}},
		{"skips blank entries", "peer-a,,,peer-b,", []string{"peer-a", "peer-b"}},
		{"dedupes via map", "peer-a,peer-a,peer-a", []string{"peer-a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePeerIDSet(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("len: got %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for _, w := range tc.want {
				if _, ok := got[w]; !ok {
					t.Errorf("expected %q in result %v", w, got)
				}
			}
		})
	}
}

func TestRefreshPeerPoliciesFromEnv(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	t.Setenv("AEGIS_P2P_BLACKLIST_PEERS", "evil-1,evil-2")
	t.Setenv("AEGIS_P2P_GREYLIST_PEERS", "warn-1")

	app.refreshPeerPoliciesFromEnv()

	app.peerPolicyMu.Lock()
	defer app.peerPolicyMu.Unlock()

	if _, ok := app.peerBlacklist["evil-1"]; !ok {
		t.Error("expected evil-1 in blacklist")
	}
	if _, ok := app.peerBlacklist["evil-2"]; !ok {
		t.Error("expected evil-2 in blacklist")
	}
	if _, ok := app.peerGreylist["warn-1"]; !ok {
		t.Error("expected warn-1 in greylist")
	}
}

func TestRefreshPeerPoliciesFromEnvReplacesBlacklist(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// First refresh seeds the blacklist.
	t.Setenv("AEGIS_P2P_BLACKLIST_PEERS", "old-evil")
	app.refreshPeerPoliciesFromEnv()
	app.peerPolicyMu.Lock()
	if _, ok := app.peerBlacklist["old-evil"]; !ok {
		app.peerPolicyMu.Unlock()
		t.Fatal("seeding blacklist failed")
	}
	app.peerPolicyMu.Unlock()

	// Second refresh with a new value must REPLACE the blacklist (not merge).
	t.Setenv("AEGIS_P2P_BLACKLIST_PEERS", "new-evil")
	app.refreshPeerPoliciesFromEnv()
	app.peerPolicyMu.Lock()
	defer app.peerPolicyMu.Unlock()
	if _, ok := app.peerBlacklist["old-evil"]; ok {
		t.Error("old-evil should have been replaced")
	}
	if _, ok := app.peerBlacklist["new-evil"]; !ok {
		t.Error("new-evil missing after refresh")
	}
}

func TestIsPeerBlockedEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	blocked, reason := app.isPeerBlocked("")
	if blocked || reason != "" {
		t.Errorf("empty peer id should be unblocked, got blocked=%v reason=%q", blocked, reason)
	}
	blocked, reason = app.isPeerBlocked("   ")
	if blocked || reason != "" {
		t.Errorf("whitespace peer id should be unblocked, got blocked=%v reason=%q", blocked, reason)
	}
}

func TestIsPeerBlockedBlacklist(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	app.peerPolicyMu.Lock()
	app.peerBlacklist["evil"] = struct{}{}
	app.peerPolicyMu.Unlock()

	blocked, reason := app.isPeerBlocked("evil")
	if !blocked {
		t.Fatal("blacklisted peer should report blocked")
	}
	if reason != "blacklist" {
		t.Errorf("expected reason 'blacklist', got %q", reason)
	}

	// Unrelated peer should not be blocked.
	blocked, _ = app.isPeerBlocked("friend")
	if blocked {
		t.Error("unrelated peer should not be blocked")
	}
}

func TestIsPeerBlockedGreylistActive(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	app.peerPolicyMu.Lock()
	app.peerGreylist["pending"] = time.Now().Unix() + 3600
	app.peerPolicyMu.Unlock()

	blocked, reason := app.isPeerBlocked("pending")
	if !blocked {
		t.Fatal("greylisted peer with future expiry should report blocked")
	}
	if reason != "greylist" {
		t.Errorf("expected reason 'greylist', got %q", reason)
	}
}

func TestIsPeerBlockedGreylistExpiresAndCleansUp(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	app.peerPolicyMu.Lock()
	app.peerGreylist["expired"] = 1 // clearly in the past
	app.peerPolicyMu.Unlock()

	blocked, reason := app.isPeerBlocked("expired")
	if blocked {
		t.Error("expired greylist entry should not block")
	}
	if reason != "" {
		t.Errorf("expected empty reason, got %q", reason)
	}

	// The expired entry should be evicted by the call.
	app.peerPolicyMu.Lock()
	if _, ok := app.peerGreylist["expired"]; ok {
		t.Error("expired entry should have been removed")
	}
	app.peerPolicyMu.Unlock()
}

func TestMarkPeerGreylistedAddsTTL(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	beforeNow := time.Now().Unix()
	app.markPeerGreylisted("naughty", "fail-rate-too-high")

	app.peerPolicyMu.Lock()
	defer app.peerPolicyMu.Unlock()
	until, ok := app.peerGreylist["naughty"]
	if !ok {
		t.Fatal("expected peer to be in greylist")
	}
	if until <= beforeNow {
		t.Errorf("expected TTL into the future, got until=%d (before=%d)", until, beforeNow)
	}
}

func TestMarkPeerGreylistedIgnoresEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	app.markPeerGreylisted("", "anything")
	app.markPeerGreylisted("   ", "anything")

	app.peerPolicyMu.Lock()
	defer app.peerPolicyMu.Unlock()
	if len(app.peerGreylist) != 0 {
		t.Errorf("empty peer id should be a no-op, got %v", app.peerGreylist)
	}
}

// =============================================================================
// Admin publish wrappers
// =============================================================================

func TestPublishShadowBanRequiresTrustedAdmin(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	err := app.PublishShadowBan("target", "untrusted-admin", "spam")
	if err == nil {
		t.Fatal("expected error for untrusted admin")
	}
	if !strings.Contains(err.Error(), "not trusted") {
		t.Errorf("expected 'not trusted' error, got %v", err)
	}
}

func TestPublishShadowBanValidation(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	if err := app.PublishShadowBan("  ", identity.PublicKey, "r"); err == nil {
		t.Error("empty target should fail")
	}
	if err := app.PublishShadowBan("t", "  ", "r"); err == nil {
		t.Error("empty admin should fail")
	}
}

func TestPublishShadowBanHappyPath(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	before := outboxCountByType(t, app, outboxMessageTypeGovernance)
	if err := app.PublishShadowBan("evil-author", identity.PublicKey, "spam"); err != nil {
		t.Fatalf("publish ban: %v", err)
	}
	after := outboxCountByType(t, app, outboxMessageTypeGovernance)
	if after-before != 1 {
		t.Errorf("expected 1 new GOVERNANCE outbox row, got delta %d", after-before)
	}

	// Moderation state should reflect the ban locally.
	state, _ := app.GetModerationState()
	found := false
	for _, m := range state {
		if m.TargetPubkey == "evil-author" && m.Action == "SHADOW_BAN" {
			found = true
		}
	}
	if !found {
		t.Error("expected SHADOW_BAN moderation row for evil-author")
	}
}

func TestPublishUnbanHappyPath(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	if err := app.PublishShadowBan("former-spammer", identity.PublicKey, "spam"); err != nil {
		t.Fatalf("seed ban: %v", err)
	}

	before := outboxCountByType(t, app, outboxMessageTypeGovernance)
	if err := app.PublishUnban("former-spammer", identity.PublicKey, "reformed"); err != nil {
		t.Fatalf("unban: %v", err)
	}
	after := outboxCountByType(t, app, outboxMessageTypeGovernance)
	if after-before != 1 {
		t.Errorf("expected 1 new GOVERNANCE row, got delta %d", after-before)
	}

	state, _ := app.GetModerationState()
	for _, m := range state {
		if m.TargetPubkey == "former-spammer" && m.Action == "SHADOW_BAN" {
			t.Errorf("SHADOW_BAN should have been replaced by UNBAN, got %+v", m)
		}
	}
}

func TestPublishSetPostPinnedHappyPath(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	before := outboxCountByType(t, app, outboxMessageTypePostPin)
	if err := app.PublishSetPostPinned(postID, true); err != nil {
		t.Fatalf("pin: %v", err)
	}
	after := outboxCountByType(t, app, outboxMessageTypePostPin)
	if after-before != 1 {
		t.Errorf("expected 1 new POST_PIN_SET row, got delta %d", after-before)
	}

	feed, _ := app.GetFeedBySub(defaultSubID)
	if len(feed) != 1 || !feed[0].IsPinned {
		t.Errorf("post should be pinned in local feed, got %+v", feed)
	}
}

func TestPublishSetPostPinnedRequiresAdmin(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)
	// Local identity is not a trusted admin yet.
	if err := app.PublishSetPostPinned("any-post", true); err == nil {
		t.Error("expected error when local identity is not trusted")
	}
}

func TestPublishSetPostPinnedValidation(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := app.PublishSetPostPinned("  ", true); err == nil {
		t.Error("empty post id should fail")
	}
}

func TestPublishSetPostLockedHappyPath(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	before := outboxCountByType(t, app, outboxMessageTypePostLock)
	if err := app.PublishSetPostLocked(postID, true); err != nil {
		t.Fatalf("lock: %v", err)
	}
	after := outboxCountByType(t, app, outboxMessageTypePostLock)
	if after-before != 1 {
		t.Errorf("expected 1 new POST_LOCK_SET row, got delta %d", after-before)
	}

	feed, _ := app.GetFeedBySub(defaultSubID)
	if len(feed) != 1 || !feed[0].IsLocked {
		t.Errorf("post should be locked in local feed, got %+v", feed)
	}
}

func TestPublishSetPostLockedRequiresAdmin(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)
	if err := app.PublishSetPostLocked("any-post", true); err == nil {
		t.Error("expected error when local identity is not trusted")
	}
}

func TestPublishSubSettingsUpdateHappyPath(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	before := outboxCountByType(t, app, outboxMessageTypeSubSettings)
	if err := app.PublishSubSettingsUpdate(defaultSubID, []string{"rule-1", "rule-2"}, "be nice"); err != nil {
		t.Fatalf("update: %v", err)
	}
	after := outboxCountByType(t, app, outboxMessageTypeSubSettings)
	if after-before != 1 {
		t.Errorf("expected 1 new SUB_SETTINGS_UPDATE row, got delta %d", after-before)
	}

	settings, err := app.GetSubSettings(defaultSubID)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if len(settings.Rules) != 2 {
		t.Errorf("expected 2 rules, got %v", settings.Rules)
	}
	if settings.Announcement != "be nice" {
		t.Errorf("expected announcement, got %q", settings.Announcement)
	}
}

func TestPublishSubSettingsUpdateRequiresAdmin(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)
	if err := app.PublishSubSettingsUpdate(defaultSubID, nil, ""); err == nil {
		t.Error("expected error when local identity is not trusted")
	}
}
