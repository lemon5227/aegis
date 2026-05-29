package main

import (
	"testing"
)

func TestParsePeerAddressesCSV(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 0},
		{"single", "/ip4/1.2.3.4/tcp/40100/p2p/QmTest", 1},
		{"comma_separated", "/ip4/1.2.3.4/tcp/40100/p2p/QmA,/ip4/5.6.7.8/tcp/40100/p2p/QmB", 2},
		{"newline_separated", "/ip4/1.2.3.4/tcp/40100/p2p/QmA\n/ip4/5.6.7.8/tcp/40100/p2p/QmB", 2},
		{"semicolon_separated", "/ip4/1.2.3.4/tcp/40100/p2p/QmA;/ip4/5.6.7.8/tcp/40100/p2p/QmB", 2},
		{"with_spaces", "  /ip4/1.2.3.4/tcp/40100/p2p/QmA  ", 1},
		{"empty_parts", ",,,", 0},
		{"duplicates", "/ip4/1.2.3.4/tcp/40100/p2p/QmA,/ip4/1.2.3.4/tcp/40100/p2p/QmA", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePeerAddressesCSV(tt.input)
			if len(result) != tt.expected {
				t.Errorf("expected %d addresses, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

func TestNormalizePeerAddresses(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected int
	}{
		{"nil", nil, 0},
		{"empty", []string{}, 0},
		{"all_empty", []string{"", "", ""}, 0},
		{"with_whitespace", []string{"  /addr1  ", "/addr2"}, 2},
		{"deduplicate", []string{"/addr1", "/addr1", "/addr2"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePeerAddresses(tt.input)
			if len(result) != tt.expected {
				t.Errorf("expected %d, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

func TestMergePeerAddressLists(t *testing.T) {
	merged := mergePeerAddressLists(
		[]string{"/addr1", "/addr2"},
		[]string{"/addr2", "/addr3"},
		[]string{"/addr3", "/addr4"},
	)
	if len(merged) != 4 {
		t.Errorf("expected 4 unique addresses, got %d: %v", len(merged), merged)
	}
}

func TestNormalizeP2PListenPort(t *testing.T) {
	tests := []struct {
		name        string
		input       int
		expected    int
		expectError bool
	}{
		{"zero_default", 0, 40100, false},
		{"negative_default", -1, 40100, false},
		{"valid", 40200, 40200, false},
		{"too_high", 65536, 0, true},
		{"max_valid", 65535, 65535, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeP2PListenPort(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestIsPublicIPv4(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"public_google", "8.8.8.8", true},
		{"public_cloudflare", "1.1.1.1", true},
		{"loopback", "127.0.0.1", false},
		{"private_10", "10.0.0.1", false},
		{"private_172_16", "172.16.0.1", false},
		{"private_172_31", "172.31.255.255", false},
		{"public_172_15", "172.15.0.1", true},
		{"public_172_32", "172.32.0.1", true},
		{"private_192_168", "192.168.1.1", false},
		{"linklocal_169_254", "169.254.1.1", false},
		{"invalid", "not.an.ip", false},
		{"empty", "", false},
		{"ipv6", "::1", false},
		{"with_whitespace", "  8.8.8.8  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPublicIPv4(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v for %q, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

func TestNormalizeAuthScope(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "user"},
		{"user", "user"},
		{"USER", "user"},
		{"admin", "admin"},
		{"ADMIN", "admin"},
		{"  user  ", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeAuthScope(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q for %q, got %q", tt.expected, tt.input, result)
			}
		})
	}
}

func TestEncodeDecodeSubRulesJSON(t *testing.T) {
	rules := []string{"Rule 1", "Rule 2", "Rule 3"}

	encoded, err := encodeSubRulesJSON(rules)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded := decodeSubRulesJSON(encoded)

	if len(decoded) != len(rules) {
		t.Errorf("expected %d rules, got %d", len(rules), len(decoded))
	}
	for i, r := range rules {
		if decoded[i] != r {
			t.Errorf("rule %d mismatch: got %q, want %q", i, decoded[i], r)
		}
	}
}

func TestNormalizeSubRules(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected int
	}{
		{"empty", []string{}, 0},
		{"normal", []string{"Rule 1", "Rule 2"}, 2},
		{"with_empty", []string{"Rule 1", "", "  ", "Rule 2"}, 2},
		{"with_whitespace", []string{"  Rule 1  ", "Rule 2"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSubRules(tt.input)
			if len(result) != tt.expected {
				t.Errorf("expected %d rules, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

func TestNormalizeFeedSortMode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hot", "hot"},
		{"new", "new"},
		{"top-day", "top-day"},
		{"HOT", "hot"},
		{"", "hot"},
		{"unknown", "hot"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeFeedSortMode(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q for %q, got %q", tt.expected, tt.input, result)
			}
		})
	}
}

func TestComputeHotScore(t *testing.T) {
	now := int64(1700000000)

	// Higher upvotes should produce higher score
	score1 := computeHotScore(10, now-3600, now)
	score2 := computeHotScore(100, now-3600, now)
	if score2 <= score1 {
		t.Errorf("higher upvotes should produce higher score: %v <= %v", score2, score1)
	}

	// Newer post should have higher score than older with same upvotes
	scoreNew := computeHotScore(10, now-60, now)
	scoreOld := computeHotScore(10, now-86400, now)
	if scoreNew <= scoreOld {
		t.Errorf("newer post should have higher score: %v <= %v", scoreNew, scoreOld)
	}
}

func TestVoteDelta(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected int64
	}{
		{"none_to_up", "none", "up", 1},
		{"none_to_down", "none", "down", -1},
		{"up_to_none", "up", "none", -1},
		{"down_to_none", "down", "none", 1},
		{"up_to_down", "up", "down", -2},
		{"down_to_up", "down", "up", 2},
		{"same", "up", "up", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := voteDelta(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestRecommendationStrategyRank(t *testing.T) {
	strategy := &HotV1Strategy{}
	if strategy.Name() != "hot-v1" {
		t.Errorf("expected hot-v1, got %q", strategy.Name())
	}

	now := int64(1700000000)
	candidates := []ForumMessage{
		{ID: "p1", Score: 10, Timestamp: now - 3600},
		{ID: "p2", Score: 100, Timestamp: now - 3600},
		{ID: "p3", Score: 50, Timestamp: now - 60},
	}

	items, err := strategy.Rank(candidates, "viewer-pubkey", now)
	if err != nil {
		t.Fatalf("rank: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestNewStrategyRank(t *testing.T) {
	strategy := &NewStrategy{}
	if strategy.Name() != "new" {
		t.Errorf("expected new, got %q", strategy.Name())
	}

	now := int64(1700000000)
	candidates := []ForumMessage{
		{ID: "p1", Timestamp: now - 3600},
		{ID: "p2", Timestamp: now - 60},
		{ID: "p3", Timestamp: now - 1800},
	}

	items, err := strategy.Rank(candidates, "viewer", now)
	if err != nil {
		t.Fatalf("rank: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
	// Newest first
	if items[0].Post.ID != "p2" {
		t.Errorf("expected newest first (p2), got %q", items[0].Post.ID)
	}
}

func TestGetStrategyDefault(t *testing.T) {
	// Empty name should default to hot-v1
	strategy, err := GetStrategy("")
	if err != nil {
		t.Fatalf("get default strategy: %v", err)
	}
	if strategy.Name() != "hot-v1" {
		t.Errorf("expected hot-v1 default, got %q", strategy.Name())
	}

	// Whitespace should also default
	strategy, err = GetStrategy("   ")
	if err != nil {
		t.Fatalf("get strategy with whitespace: %v", err)
	}
	if strategy.Name() != "hot-v1" {
		t.Errorf("expected hot-v1, got %q", strategy.Name())
	}

	// Case insensitive
	strategy, err = GetStrategy("HOT-V1")
	if err != nil {
		t.Fatalf("get HOT-V1: %v", err)
	}
	if strategy.Name() != "hot-v1" {
		t.Errorf("expected hot-v1, got %q", strategy.Name())
	}
}

func TestGetStrategyUnknownFallback(t *testing.T) {
	// Unknown strategy should fallback to hot-v1
	strategy, err := GetStrategy("nonexistent-strategy")
	if err != nil {
		t.Fatalf("get unknown strategy: %v", err)
	}
	if strategy.Name() != "hot-v1" {
		t.Errorf("expected hot-v1 fallback, got %q", strategy.Name())
	}
}

func TestNormalizeFeedStreamLimit(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 50},
		{-1, 50},
		{10, 10},
		{200, 200},
		{1000, 200}, // capped at 200
	}

	for _, tt := range tests {
		result := normalizeFeedStreamLimit(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeFeedStreamLimit(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeFeedStreamAlgorithm(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hot-v1", "hot-v1"},
		{"new", "new"},
		{"HOT-V1", "hot-v1"},
		{"", "hot-v1"},
		{"  hot-v1  ", "hot-v1"},
	}

	for _, tt := range tests {
		result := normalizeFeedStreamAlgorithm(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeFeedStreamAlgorithm(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestTopWindowStartUnix(t *testing.T) {
	now := int64(1_700_000_000)

	cases := []struct {
		name     string
		sortMode string
		want     int64
	}{
		{"top-day", "top-day", now - 24*60*60},
		{"top-week", "top-week", now - 7*24*60*60},
		{"top-month", "top-month", now - 30*24*60*60},
		{"empty falls through to default", "", 0},
		{"new defaults to 0", "new", 0},
		{"hot defaults to 0", "hot", 0},
		{"unknown defaults to 0", "definitely-not-a-mode", 0},
		{"whitespace trimmed", "   top-week   ", now - 7*24*60*60},
		{"case-insensitive via normalizeFeedSortMode", "TOP-DAY", now - 24*60*60},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := topWindowStartUnix(tc.sortMode, now)
			if got != tc.want {
				t.Errorf("topWindowStartUnix(%q, %d) = %d, want %d", tc.sortMode, now, got, tc.want)
			}
		})
	}
}
