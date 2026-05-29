package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------
// makeSQLPlaceholders
// -----------------------------------------------------------------------------

func TestMakeSQLPlaceholders(t *testing.T) {
	cases := []struct {
		count int
		want  string
	}{
		{-1, ""},
		{0, ""},
		{1, "?"},
		{2, "?,?"},
		{4, "?,?,?,?"},
	}
	for _, tc := range cases {
		got := makeSQLPlaceholders(tc.count)
		if got != tc.want {
			t.Errorf("makeSQLPlaceholders(%d) = %q, want %q", tc.count, got, tc.want)
		}
	}
}

// -----------------------------------------------------------------------------
// fallbackOperationID + resolveOperationID + generateOperationID
// -----------------------------------------------------------------------------

func TestFallbackOperationIDFormat(t *testing.T) {
	got := fallbackOperationID(" ent-1 ", " author-A ", 42, " CREATE ")
	want := "ent-1:author-A:42:create"
	if got != want {
		t.Errorf("fallback id: got %q, want %q", got, want)
	}
}

func TestFallbackOperationIDClampsNegativeLamport(t *testing.T) {
	got := fallbackOperationID("e", "a", -7, "DELETE")
	if !strings.HasSuffix(got, ":0:delete") {
		t.Errorf("expected lamport clamped to 0, got %q", got)
	}
}

func TestResolveOperationIDPrefersExisting(t *testing.T) {
	got := resolveOperationID("  explicit-op  ", "ent", "auth", 1, "CREATE")
	if got != "explicit-op" {
		t.Errorf("expected explicit op id to be returned trimmed, got %q", got)
	}
}

func TestResolveOperationIDFallsBackWhenEmpty(t *testing.T) {
	got := resolveOperationID("   ", "ent", "auth", 5, "CREATE")
	want := "ent:auth:5:create"
	if got != want {
		t.Errorf("fallback should be used when opID empty: got %q want %q", got, want)
	}
}

func TestGenerateOperationIDIncludesNonce(t *testing.T) {
	a := generateOperationID("ent", "author", 10)
	b := generateOperationID("ent", "author", 10)

	if a == b {
		t.Errorf("expected randomized nonce to make ids unique: a=%s b=%s", a, b)
	}
	for _, id := range []string{a, b} {
		if !strings.HasPrefix(id, "ent:author:10:") {
			t.Errorf("op id missing prefix: %q", id)
		}
		// nonce is hex-encoded over defaultOpNonceBytes bytes -> 2*N hex chars.
		parts := strings.SplitN(id, ":", 4)
		if len(parts) != 4 || len(parts[3]) != defaultOpNonceBytes*2 {
			t.Errorf("op id nonce length incorrect: %q (parts=%v)", id, parts)
		}
	}
}

// -----------------------------------------------------------------------------
// CID + message id helpers
// -----------------------------------------------------------------------------

func TestBuildBinaryCIDDeterministicAndPrefixed(t *testing.T) {
	data := []byte("hello world")
	cid := buildBinaryCID(data)

	if !strings.HasPrefix(cid, "cidv1-bin-") {
		t.Errorf("missing cidv1-bin- prefix: %q", cid)
	}
	expected := "cidv1-bin-" + hex.EncodeToString(sha256Sum(data))
	if cid != expected {
		t.Errorf("cid mismatch:\n got  %s\n want %s", cid, expected)
	}
	if buildBinaryCID(data) != cid {
		t.Errorf("expected deterministic output for same input")
	}
	if buildBinaryCID([]byte("hello world!")) == cid {
		t.Errorf("different input must produce different cid")
	}
}

func TestBuildContentCIDIgnoresLeadingTrailingWhitespace(t *testing.T) {
	a := buildContentCID("  body  ")
	b := buildContentCID("body")
	if a != b {
		t.Errorf("content cid should be trim-stable: %q vs %q", a, b)
	}
	if !strings.HasPrefix(a, "cidv1-") || strings.HasPrefix(a, "cidv1-bin-") {
		t.Errorf("content cid should use cidv1- (non-bin) prefix, got %q", a)
	}
}

func TestBuildMessageIDDifferentForDifferentTimestamps(t *testing.T) {
	a := buildMessageID("pubkey", "content", 1000)
	b := buildMessageID("pubkey", "content", 1001)
	if a == b {
		t.Errorf("timestamp should affect message id")
	}
	c := buildMessageID("pubkey-2", "content", 1000)
	if a == c {
		t.Errorf("pubkey should affect message id")
	}
	d := buildMessageID("pubkey", "content-2", 1000)
	if a == d {
		t.Errorf("content should affect message id")
	}
	if len(a) != 64 {
		t.Errorf("message id should be 64 hex chars (sha256), got %d", len(a))
	}
}

func sha256Sum(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

// -----------------------------------------------------------------------------
// deriveTitle / deriveBodyPreview
// -----------------------------------------------------------------------------

func TestDeriveTitle(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   \t\n  ", ""},
		{"short", "Hello", "Hello"},
		{"trims whitespace", "   trimmed   ", "trimmed"},
		{"truncates at 20 runes", "abcdefghijklmnopqrstuvwxyz", "abcdefghijklmnopqrst"},
		{"counts runes not bytes (CJK)", "你好世界你好世界你好世界你好世界你好世界另起一段", "你好世界你好世界你好世界你好世界你好世界"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveTitle(tc.body)
			if got != tc.want {
				t.Errorf("deriveTitle(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

func TestDeriveBodyPreviewRespectsMaxRunes(t *testing.T) {
	body := strings.Repeat("a", 500)
	got := deriveBodyPreview(body, 50)
	if len(got) != 50 {
		t.Errorf("expected 50 chars, got %d", len(got))
	}
}

func TestDeriveBodyPreviewDefaultsTo180(t *testing.T) {
	body := strings.Repeat("a", 500)
	got := deriveBodyPreview(body, 0)
	if len(got) != 180 {
		t.Errorf("expected default 180, got %d", len(got))
	}
	got2 := deriveBodyPreview(body, -5)
	if len(got2) != 180 {
		t.Errorf("expected default 180 for negative, got %d", len(got2))
	}
}

func TestDeriveBodyPreviewShortBodyKeptIntact(t *testing.T) {
	got := deriveBodyPreview("short body", 100)
	if got != "short body" {
		t.Errorf("expected short body kept, got %q", got)
	}
}

func TestDeriveBodyPreviewEmpty(t *testing.T) {
	if got := deriveBodyPreview("   ", 50); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// -----------------------------------------------------------------------------
// encodeMyPostsCursor / decodeMyPostsCursor
// -----------------------------------------------------------------------------

func TestMyPostsCursorRoundTrip(t *testing.T) {
	cases := []struct {
		ts     int64
		postID string
	}{
		{1, "p1"},
		{1234567890, "post-with-dashes-123"},
		{9999999999, "id|with|pipes"}, // pipes in body should still round-trip via SplitN
	}
	for _, tc := range cases {
		t.Run(tc.postID, func(t *testing.T) {
			cursor := encodeMyPostsCursor(tc.ts, tc.postID)
			ts, postID, err := decodeMyPostsCursor(cursor)
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

func TestDecodeMyPostsCursorEmptyReturnsZero(t *testing.T) {
	ts, postID, err := decodeMyPostsCursor("")
	if err != nil {
		t.Fatalf("expected nil error for empty cursor, got %v", err)
	}
	if ts != 0 || postID != "" {
		t.Errorf("expected zero values for empty cursor, got ts=%d postID=%q", ts, postID)
	}
}

func TestDecodeMyPostsCursorRejectsMalformed(t *testing.T) {
	cases := []string{
		"not-base64-!!!",
		"",          // handled separately above; included here as no-op
		"bm8tcGlwZQ==", // valid base64 but no | separator
	}
	// Skip empty in this loop since it's the not-an-error branch.
	for _, c := range cases {
		if c == "" {
			continue
		}
		if _, _, err := decodeMyPostsCursor(c); err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

func TestDecodeMyPostsCursorRejectsBadTimestamp(t *testing.T) {
	bad := encodeMyPostsCursor(0, "post-1") // ts=0 should be rejected
	if _, _, err := decodeMyPostsCursor(bad); err == nil {
		t.Error("expected error for ts=0")
	}
}

func TestDecodeMyPostsCursorRejectsEmptyPostID(t *testing.T) {
	bad := encodeMyPostsCursor(100, "   ")
	if _, _, err := decodeMyPostsCursor(bad); err == nil {
		t.Error("expected error for empty post id after trim")
	}
}

// -----------------------------------------------------------------------------
// normalizedImageMIME
// -----------------------------------------------------------------------------

func TestNormalizedImageMIME(t *testing.T) {
	cases := []struct {
		name   string
		hint   string
		format string
		want   string
	}{
		{"hint trumps format", "image/webp", "png", "image/webp"},
		{"hint normalized to lowercase", "  IMAGE/PNG  ", "jpeg", "image/png"},
		{"format jpeg", "", "jpeg", "image/jpeg"},
		{"format jpg alias", "", "JPG", "image/jpeg"},
		{"format png", "", "png", "image/png"},
		{"format gif", "", "gif", "image/gif"},
		{"unknown format defaults to jpeg", "", "tiff", "image/jpeg"},
		{"empty inputs default to jpeg", "", "", "image/jpeg"},
		{"non-image hint falls through to format", "text/plain", "png", "image/png"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizedImageMIME(tc.hint, tc.format)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// isDevModeEnabled / IsDevMode
// -----------------------------------------------------------------------------

func TestIsDevModeEnabled(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"random", false},
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"on", true},
		{"dev", true},
		{"  Dev  ", true},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("AEGIS_DEV_MODE", tc.value)
			if got := isDevModeEnabled(); got != tc.want {
				t.Errorf("isDevModeEnabled(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestAppIsDevModeMirrorsEnv(t *testing.T) {
	t.Setenv("AEGIS_DEV_MODE", "1")
	app := NewApp()
	if !app.IsDevMode() {
		t.Error("App.IsDevMode should report true when env var is set")
	}

	t.Setenv("AEGIS_DEV_MODE", "0")
	if app.IsDevMode() {
		t.Error("App.IsDevMode should report false when env var is disabled")
	}
}

// -----------------------------------------------------------------------------
// compareLamportVersion (drives multi-field tie-breaking)
// -----------------------------------------------------------------------------

func TestCompareLamportVersionByLamport(t *testing.T) {
	left := LamportVersion{Lamport: 5, Author: "z", OpID: "z"}
	right := LamportVersion{Lamport: 4, Author: "a", OpID: "a"}
	if compareLamportVersion(left, right) != 1 {
		t.Errorf("higher lamport must win regardless of author/op")
	}
	if compareLamportVersion(right, left) != -1 {
		t.Errorf("lower lamport must lose regardless of author/op")
	}
}

func TestCompareLamportVersionByAuthorTieBreak(t *testing.T) {
	left := LamportVersion{Lamport: 7, Author: "bbb", OpID: "z"}
	right := LamportVersion{Lamport: 7, Author: "aaa", OpID: "a"}
	if compareLamportVersion(left, right) != 1 {
		t.Errorf("on equal lamport, larger author wins")
	}
	if compareLamportVersion(right, left) != -1 {
		t.Errorf("on equal lamport, smaller author loses")
	}
}

func TestCompareLamportVersionByOpIDTieBreak(t *testing.T) {
	left := LamportVersion{Lamport: 9, Author: "same", OpID: "zzz"}
	right := LamportVersion{Lamport: 9, Author: "same", OpID: "aaa"}
	if compareLamportVersion(left, right) != 1 {
		t.Errorf("on equal lamport+author, larger op id wins")
	}
}

func TestCompareLamportVersionEqual(t *testing.T) {
	v := LamportVersion{Lamport: 1, Author: "a", OpID: "op"}
	if compareLamportVersion(v, v) != 0 {
		t.Error("equal versions should compare equal")
	}
}

func TestCompareLamportVersionTrimsWhitespace(t *testing.T) {
	left := LamportVersion{Lamport: 1, Author: "  a  ", OpID: "op"}
	right := LamportVersion{Lamport: 1, Author: "a", OpID: "op"}
	if compareLamportVersion(left, right) != 0 {
		t.Errorf("author whitespace should be ignored")
	}
}
