package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func newBenchApp(b *testing.B) (*App, Identity) {
	b.Helper()
	app := NewApp()
	app.SetDatabasePath(filepath.Join(b.TempDir(), "bench.db"))
	if err := app.initDatabase(); err != nil {
		b.Fatalf("init: %v", err)
	}
	identity := seedLocalIdentityForBench(b, app)
	b.Cleanup(func() {
		if app.db != nil {
			_, _ = app.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
			_ = app.db.Close()
		}
	})
	return app, identity
}

func seedLocalIdentityForBench(b *testing.B, app *App) Identity {
	b.Helper()
	identity, err := app.GenerateIdentity()
	if err != nil {
		b.Fatalf("generate identity: %v", err)
	}
	return identity
}

// seedFeedBench inserts n public posts into the default sub.
func seedFeedBench(b *testing.B, app *App, identity Identity, n int) {
	b.Helper()
	now := time.Now().Unix()
	for i := 0; i < n; i++ {
		if _, err := app.db.Exec(
			`INSERT INTO messages (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility, title, body)
			 VALUES (?, ?, '', ?, ?, ?, 0, 'public', ?, 'normal', 't', 'b')`,
			fmt.Sprintf("bench-post-%d", i),
			identity.PublicKey,
			int64(i%5),
			now-int64(i*60),
			int64(i+1),
			defaultSubID,
		); err != nil {
			b.Fatalf("seed %d: %v", i, err)
		}
	}
}

// -----------------------------------------------------------------------------
// Pure-function benchmarks (no DB)
// -----------------------------------------------------------------------------

func BenchmarkComputeHotScore(b *testing.B) {
	now := time.Now().Unix()
	for i := 0; i < b.N; i++ {
		_ = computeHotScore(int64(i%100), now-int64(i*60), now)
	}
}

func BenchmarkCompareLamportVersionEqualLamport(b *testing.B) {
	left := LamportVersion{Lamport: 1000, Author: "author-A", OpID: "op-zzz"}
	right := LamportVersion{Lamport: 1000, Author: "author-B", OpID: "op-aaa"}
	for i := 0; i < b.N; i++ {
		_ = compareLamportVersion(left, right)
	}
}

func BenchmarkCompareLamportVersionDifferentLamport(b *testing.B) {
	left := LamportVersion{Lamport: 1000, Author: "a", OpID: "x"}
	right := LamportVersion{Lamport: 999, Author: "b", OpID: "y"}
	for i := 0; i < b.N; i++ {
		_ = compareLamportVersion(left, right)
	}
}

func BenchmarkBuildContentCID(b *testing.B) {
	body := "the quick brown fox jumps over the lazy dog"
	for i := 0; i < b.N; i++ {
		_ = buildContentCID(body)
	}
}

func BenchmarkBuildBinaryCID(b *testing.B) {
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildBinaryCID(data)
	}
}

func BenchmarkBuildMessageID(b *testing.B) {
	pubkey := "deadbeef1234567890deadbeef1234567890deadbeef1234567890deadbeef12"
	content := "post body content"
	now := time.Now().Unix()
	for i := 0; i < b.N; i++ {
		_ = buildMessageID(pubkey, content, now)
	}
}

func BenchmarkDeriveTitle(b *testing.B) {
	body := "this is a body that may be longer than twenty runes for the truncation path"
	for i := 0; i < b.N; i++ {
		_ = deriveTitle(body)
	}
}

func BenchmarkNormalizeSubID(b *testing.B) {
	inputs := []string{"  My-Sub_123  ", "abc", "with spaces and Punct!@#", "  ", "ALLCAPS"}
	for i := 0; i < b.N; i++ {
		_ = normalizeSubID(inputs[i%len(inputs)])
	}
}

func BenchmarkResolveOperationID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = resolveOperationID("", "ent-1", "auth-1", int64(i), "CREATE")
	}
}

// -----------------------------------------------------------------------------
// DB read benchmarks
// -----------------------------------------------------------------------------

func BenchmarkGetFeedBySub_100Posts(b *testing.B) {
	app, identity := newBenchApp(b)
	seedFeedBench(b, app, identity, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := app.GetFeedBySub(defaultSubID); err != nil {
			b.Fatalf("get feed: %v", err)
		}
	}
}

func BenchmarkGetFeedBySub_1000Posts(b *testing.B) {
	app, identity := newBenchApp(b)
	seedFeedBench(b, app, identity, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := app.GetFeedBySub(defaultSubID); err != nil {
			b.Fatalf("get feed: %v", err)
		}
	}
}

func BenchmarkGetFeedBySubSortedHot_1000Posts(b *testing.B) {
	app, identity := newBenchApp(b)
	seedFeedBench(b, app, identity, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := app.GetFeedBySubSorted(defaultSubID, "hot"); err != nil {
			b.Fatalf("get feed sorted: %v", err)
		}
	}
}

func BenchmarkGetFeedBySubSortedTopWeek_1000Posts(b *testing.B) {
	app, identity := newBenchApp(b)
	seedFeedBench(b, app, identity, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := app.GetFeedBySubSorted(defaultSubID, "top-week"); err != nil {
			b.Fatalf("get feed: %v", err)
		}
	}
}

// -----------------------------------------------------------------------------
// Moderation hot path
// -----------------------------------------------------------------------------

func BenchmarkShouldAcceptPublicContent_NoModeration(b *testing.B) {
	app, identity := newBenchApp(b)
	now := time.Now().Unix()
	authorPubkey := identity.PublicKey + "-other"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := app.shouldAcceptPublicContent(authorPubkey, int64(i), now, "content-id", ""); err != nil {
			b.Fatalf("check: %v", err)
		}
	}
}

func BenchmarkShouldAcceptPublicContent_AuthorIsViewer(b *testing.B) {
	app, identity := newBenchApp(b)
	now := time.Now().Unix()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Self-content path: author == viewer -> immediate accept (no DB query).
		if _, err := app.shouldAcceptPublicContent(identity.PublicKey, int64(i), now, "content-id", identity.PublicKey); err != nil {
			b.Fatalf("check: %v", err)
		}
	}
}

// BenchmarkShouldAcceptPublicContent_WithShadowBan exercises the path that
// hits GetGovernancePolicy, which is the path the policy cache exists to
// optimize. With a hot cache, this should cost a single DB query
// (getModerationSnapshot) plus a cache hit on the policy read.
func BenchmarkShouldAcceptPublicContent_WithShadowBan(b *testing.B) {
	app, identity := newBenchApp(b)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		b.Fatalf("seed admin: %v", err)
	}
	bannedAuthor := "banned-author-pubkey-deadbeef"
	if err := app.ApplyShadowBan(bannedAuthor, identity.PublicKey, "spam"); err != nil {
		b.Fatalf("ban: %v", err)
	}
	if _, err := app.SetGovernancePolicy(false); err != nil {
		b.Fatalf("set policy: %v", err)
	}
	// Warm the cache so the steady-state cost reflects cache hits, not the
	// initial population query.
	if _, err := app.GetGovernancePolicy(); err != nil {
		b.Fatalf("warm cache: %v", err)
	}

	now := time.Now().Unix()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := app.shouldAcceptPublicContent(bannedAuthor, int64(i), now, "content-id", "viewer-pubkey"); err != nil {
			b.Fatalf("check: %v", err)
		}
	}
}

// -----------------------------------------------------------------------------
// ProcessIncomingMessage (write path)
// -----------------------------------------------------------------------------

func BenchmarkProcessIncomingMessage_Post(b *testing.B) {
	app, identity := newBenchApp(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := IncomingMessage{
			Type:          "POST",
			OpType:        postOpTypeCreate,
			OpID:          fmt.Sprintf("op-%d", i),
			SchemaVersion: lamportSchemaV2,
			AuthScope:     authScopeUser,
			ID:            fmt.Sprintf("post-%d", i),
			Pubkey:        identity.PublicKey,
			Title:         "Title",
			Body:          "Body",
			ContentCID:    buildContentCID("Body"),
			SubID:         defaultSubID,
			Timestamp:     time.Now().Unix(),
			Lamport:       int64(i + 1),
		}
		signed, err := app.signIncomingMessage(msg)
		if err != nil {
			b.Fatalf("sign: %v", err)
		}
		payload, _ := json.Marshal(signed)
		if err := app.ProcessIncomingMessage(payload); err != nil {
			b.Fatalf("process: %v", err)
		}
	}
}

func BenchmarkProcessIncomingMessage_Vote(b *testing.B) {
	app, identity := newBenchApp(b)

	// Seed 1 post that we'll vote on.
	if err := app.PublishPostStructuredToSub(identity.PublicKey, "Target", "B", defaultSubID); err != nil {
		b.Fatalf("seed post: %v", err)
	}
	posts, _ := app.GetFeedBySub(defaultSubID)
	postID := posts[0].ID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := IncomingMessage{
			Type:        "POST_UPVOTE",
			OpID:        fmt.Sprintf("vote-%d", i),
			PostID:      postID,
			Pubkey:      identity.PublicKey,
			VoterPubkey: identity.PublicKey,
			Timestamp:   time.Now().Unix(),
		}
		signed, err := app.signIncomingMessage(msg)
		if err != nil {
			b.Fatalf("sign: %v", err)
		}
		payload, _ := json.Marshal(signed)
		if err := app.ProcessIncomingMessage(payload); err != nil {
			b.Fatalf("process: %v", err)
		}
	}
}

// -----------------------------------------------------------------------------
// Notification helpers (R1 refactor target)
// -----------------------------------------------------------------------------

func BenchmarkBuildNotificationID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = buildNotificationID("post_upvote", "voter-pubkey", fmt.Sprintf("post-%d", i))
	}
}

func BenchmarkScanNotifications_Empty(b *testing.B) {
	app, _ := newBenchApp(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := app.GetNotifications(20, ""); err != nil {
			b.Fatalf("get: %v", err)
		}
	}
}
