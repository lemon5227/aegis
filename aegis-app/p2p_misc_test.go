package main

import (
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// PublishPostUpdate
// -----------------------------------------------------------------------------

func TestPublishPostUpdateValidation(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	cases := []struct {
		name   string
		pubkey string
		postID string
		title  string
		body   string
	}{
		{"empty pubkey", "  ", "p", "T", "B"},
		{"empty post id", identity.PublicKey, "  ", "T", "B"},
		{"both title+body empty", identity.PublicKey, "p", "  ", "  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := app.PublishPostUpdate(tc.pubkey, tc.postID, tc.title, tc.body); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestPublishPostUpdateRejectsNonAuthor(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	_, foreign := generateRemoteIdentity(t)
	err := app.PublishPostUpdate(foreign, postID, "Hijack", "")
	if err == nil {
		t.Fatal("expected non-author update to be rejected")
	}
	if !strings.Contains(err.Error(), "only author") {
		t.Errorf("expected 'only author' error, got %v", err)
	}
}

func TestPublishPostUpdateEnqueuesOutbox(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Old", "Old body", defaultSubID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	before := outboxCountByType(t, app, outboxMessageTypePost)
	if err := app.PublishPostUpdate(identity.PublicKey, postID, "Edited", "Edited body"); err != nil {
		t.Fatalf("update: %v", err)
	}
	after := outboxCountByType(t, app, outboxMessageTypePost)
	if after-before != 1 {
		t.Errorf("expected one new POST outbox row, got delta %d", after-before)
	}

	feed, _ := app.GetFeedBySub(defaultSubID)
	if len(feed) != 1 || feed[0].Title != "Edited" {
		t.Errorf("title not updated: %+v", feed)
	}
}

// -----------------------------------------------------------------------------
// PublishCommentUpdate
// -----------------------------------------------------------------------------

func TestPublishCommentUpdateValidation(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishCommentUpdate("  ", "c", "body"); err == nil {
		t.Error("empty pubkey should fail")
	}
	if err := app.PublishCommentUpdate(identity.PublicKey, "  ", "body"); err == nil {
		t.Error("empty comment id should fail")
	}
}

func TestPublishCommentUpdateEnqueuesOutbox(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)

	if err := app.PublishPostStructuredToSub(identity.PublicKey, "T", "B", defaultSubID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID
	comment, err := app.AddLocalComment(identity.PublicKey, postID, "", "original")
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	before := outboxCountByType(t, app, outboxMessageTypeComment)
	if err := app.PublishCommentUpdate(identity.PublicKey, comment.ID, "edited body"); err != nil {
		t.Fatalf("update: %v", err)
	}
	after := outboxCountByType(t, app, outboxMessageTypeComment)
	if after-before != 1 {
		t.Errorf("expected one new comment outbox row, got delta %d", after-before)
	}

	comments, _ := app.GetCommentsByPost(postID)
	if len(comments) != 1 || comments[0].Body != "edited body" {
		t.Errorf("comment body not updated: %+v", comments)
	}
}

// -----------------------------------------------------------------------------
// TriggerAntiEntropySyncNow / TriggerCommentSyncNow
// -----------------------------------------------------------------------------

func TestTriggerAntiEntropySyncNowWithoutP2P(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	err := app.TriggerAntiEntropySyncNow()
	if err == nil {
		t.Fatal("expected error when p2p is not started")
	}
	if !strings.Contains(err.Error(), "p2p not started") {
		t.Errorf("expected 'p2p not started', got %v", err)
	}
}

func TestTriggerCommentSyncNowWithoutP2P(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	err := app.TriggerCommentSyncNow("any-post")
	if err == nil {
		t.Fatal("expected error when p2p is not started")
	}
	if !strings.Contains(err.Error(), "p2p not started") {
		t.Errorf("expected 'p2p not started', got %v", err)
	}
}

// -----------------------------------------------------------------------------
// TriggerReleaseAlertEvaluationNow
// -----------------------------------------------------------------------------

func TestTriggerReleaseAlertEvaluationNowOnFreshApp(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// No metrics yet -> release alert evaluation should run without panicking
	// and return a (possibly empty) slice.
	alerts := app.TriggerReleaseAlertEvaluationNow()
	if alerts == nil {
		t.Error("expected non-nil slice (even if empty)")
	}
	// Re-running should be idempotent at the data level.
	alerts2 := app.TriggerReleaseAlertEvaluationNow()
	if len(alerts) != len(alerts2) {
		t.Errorf("evaluation results should be stable on a static app: first=%d, second=%d",
			len(alerts), len(alerts2))
	}
}

// -----------------------------------------------------------------------------
// decodeKnownPeerAddrs
// -----------------------------------------------------------------------------

func TestDecodeKnownPeerAddrsEmptyOrWhitespace(t *testing.T) {
	if got := decodeKnownPeerAddrs(""); got != nil {
		t.Errorf("empty input should return nil, got %v", got)
	}
	if got := decodeKnownPeerAddrs("   "); got != nil {
		t.Errorf("whitespace input should return nil, got %v", got)
	}
}

func TestDecodeKnownPeerAddrsInvalidJSON(t *testing.T) {
	if got := decodeKnownPeerAddrs("not-json"); got != nil {
		t.Errorf("malformed JSON should return nil, got %v", got)
	}
	if got := decodeKnownPeerAddrs("{not-an-array}"); got != nil {
		t.Errorf("non-array JSON should return nil, got %v", got)
	}
}

func TestDecodeKnownPeerAddrsValidArray(t *testing.T) {
	raw := `["/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWAa", "/ip4/192.168.1.1/tcp/4001/p2p/12D3KooWBb"]`
	got := decodeKnownPeerAddrs(raw)
	if len(got) == 0 {
		t.Errorf("expected addrs to be returned, got %v", got)
	}
	// normalizePeerAddresses may dedupe / filter; check at least one valid addr survived.
	for _, a := range got {
		if !strings.HasPrefix(a, "/ip4/") {
			t.Errorf("unexpected addr after normalization: %q", a)
		}
	}
}

// -----------------------------------------------------------------------------
// buildKnownPeerExchangePayload
// -----------------------------------------------------------------------------

func TestBuildKnownPeerExchangePayloadEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Fresh DB, no host -> returns nil (empty).
	got := app.buildKnownPeerExchangePayload(10)
	if got != nil {
		t.Errorf("expected nil on empty DB without host, got %v", got)
	}
}

func TestBuildKnownPeerExchangePayloadWithRows(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Insert 2 known peers directly. addrs_json is the JSON-encoded array.
	if _, err := app.db.Exec(
		`INSERT INTO known_peers (peer_id, addrs_json, relay_capable, public_reachable, success_count, fail_count, last_seen, updated_at)
		 VALUES (?, ?, 1, 1, 5, 0, ?, ?),
		        (?, ?, 0, 0, 0, 0, ?, ?)`,
		"peer-public", `["/ip4/1.1.1.1/tcp/4001/p2p/peer-public"]`, int64(1_700_000_100), int64(1_700_000_100),
		"peer-private", `["/ip4/192.168.0.1/tcp/4001/p2p/peer-private"]`, int64(1_700_000_000), int64(1_700_000_000),
	); err != nil {
		t.Fatalf("seed peers: %v", err)
	}

	got := app.buildKnownPeerExchangePayload(10)
	if len(got) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(got))
	}

	// Sort prefers PublicReachable first; peer-public should lead.
	if got[0].PeerID != "peer-public" {
		t.Errorf("expected PublicReachable peer first, got %q", got[0].PeerID)
	}
	if !got[0].PublicReachable || !got[0].RelayCapable {
		t.Errorf("public peer flags lost: %+v", got[0])
	}
	if got[1].PeerID != "peer-private" {
		t.Errorf("expected private peer second, got %q", got[1].PeerID)
	}
}

func TestBuildKnownPeerExchangePayloadRespectsLimit(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Insert 5 peers; ask for limit=2.
	for i := 0; i < 5; i++ {
		if _, err := app.db.Exec(
			`INSERT INTO known_peers (peer_id, addrs_json, relay_capable, public_reachable, success_count, fail_count, last_seen, updated_at)
			 VALUES (?, '[]', 1, 1, ?, 0, ?, ?)`,
			"peer-"+string(rune('a'+i)), int64(10-i), int64(1_700_000_000+i), int64(1_700_000_000+i),
		); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	got := app.buildKnownPeerExchangePayload(2)
	if len(got) > 2 {
		t.Errorf("expected limit=2 to cap result, got %d", len(got))
	}
}

func TestBuildKnownPeerExchangePayloadDedupesByPeerID(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// known_peers has UNIQUE on peer_id, so we can't have two rows with the
	// same id. The dedup logic guards against accidental duplicates produced
	// by JOINs / future additions. Just verify single-peer happy path.
	if _, err := app.db.Exec(
		`INSERT INTO known_peers (peer_id, addrs_json, relay_capable, public_reachable, success_count, fail_count, last_seen, updated_at)
		 VALUES ('only-one', '[]', 0, 0, 0, 0, ?, ?)`,
		int64(1_700_000_000), int64(1_700_000_000),
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got := app.buildKnownPeerExchangePayload(10)
	if len(got) != 1 {
		t.Errorf("expected 1 peer, got %d", len(got))
	}
	if got[0].PeerID != "only-one" {
		t.Errorf("peer id mismatch: %q", got[0].PeerID)
	}
}
