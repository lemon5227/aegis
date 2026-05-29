package main

import (
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// resolveFetchRetryAttempts (env-driven)
// -----------------------------------------------------------------------------

func TestResolveFetchRetryAttempts(t *testing.T) {
	cases := []struct {
		env  string
		want int
	}{
		{"", 1},          // unset → default
		{"   ", 1},       // whitespace-only → default
		{"not-a-number", 1},
		{"0", 1},         // <= 0 falls back to default
		{"-3", 1},
		{"1", 1},
		{"2", 2},
		{"3", 3},
		{"4", 3},         // capped at 3
		{"99", 3},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("AEGIS_FETCH_RETRY_ATTEMPTS", tc.env)
			got := resolveFetchRetryAttempts()
			if got != tc.want {
				t.Errorf("resolveFetchRetryAttempts() with env %q = %d, want %d", tc.env, got, tc.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// percentile95Latency (pure)
// -----------------------------------------------------------------------------

func TestPercentile95LatencyEmpty(t *testing.T) {
	if got := percentile95Latency(nil); got != 0 {
		t.Errorf("nil input must return 0, got %d", got)
	}
	if got := percentile95Latency([]int64{}); got != 0 {
		t.Errorf("empty slice must return 0, got %d", got)
	}
}

func TestPercentile95LatencySingleSample(t *testing.T) {
	if got := percentile95Latency([]int64{42}); got != 42 {
		t.Errorf("single sample must return itself, got %d", got)
	}
}

func TestPercentile95LatencyTwentySamples(t *testing.T) {
	// Sorted: 1..20. p95 index = (20*95-1)/100 = 1899/100 = 18 → element 19 (0-indexed 18).
	samples := []int64{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	got := percentile95Latency(samples)
	if got != 19 {
		t.Errorf("p95 of 1..20 should be 19, got %d", got)
	}
}

func TestPercentile95LatencyDoesNotMutateInput(t *testing.T) {
	in := []int64{5, 1, 4, 2, 3}
	original := append([]int64(nil), in...)

	_ = percentile95Latency(in)

	for i := range original {
		if in[i] != original[i] {
			t.Errorf("input was mutated at index %d: before=%d after=%d", i, original[i], in[i])
		}
	}
}

func TestPercentile95LatencyAllEqual(t *testing.T) {
	samples := []int64{50, 50, 50, 50, 50}
	if got := percentile95Latency(samples); got != 50 {
		t.Errorf("uniform samples should return that value, got %d", got)
	}
}

// -----------------------------------------------------------------------------
// noteBlobCacheHit / noteBlobCacheMiss → GetReleaseMetrics
// -----------------------------------------------------------------------------

func TestBlobCacheCountersAndHitRate(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Baseline: no lookups → hit rate is 0 (not NaN, not divide-by-zero).
	metrics := app.GetReleaseMetrics()
	if metrics.BlobCacheHits != 0 || metrics.BlobCacheMisses != 0 {
		t.Errorf("expected zero counters, got hits=%d misses=%d", metrics.BlobCacheHits, metrics.BlobCacheMisses)
	}
	if metrics.BlobCacheHitRate != 0 {
		t.Errorf("expected hit rate 0 with no lookups, got %v", metrics.BlobCacheHitRate)
	}

	// 3 hits, 1 miss → 0.75 hit rate.
	app.noteBlobCacheHit()
	app.noteBlobCacheHit()
	app.noteBlobCacheHit()
	app.noteBlobCacheMiss()

	metrics = app.GetReleaseMetrics()
	if metrics.BlobCacheHits != 3 {
		t.Errorf("hits: want 3, got %d", metrics.BlobCacheHits)
	}
	if metrics.BlobCacheMisses != 1 {
		t.Errorf("misses: want 1, got %d", metrics.BlobCacheMisses)
	}
	if metrics.BlobCacheHitRate != 0.75 {
		t.Errorf("hit rate: want 0.75, got %v", metrics.BlobCacheHitRate)
	}
}

func TestContentFetchCountersAndSuccessRate(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	app.noteContentFetchAttempt()
	app.noteContentFetchAttempt()
	app.noteContentFetchAttempt()
	app.noteContentFetchAttempt()
	app.noteContentFetchResult(true, 50*time.Millisecond)
	app.noteContentFetchResult(true, 100*time.Millisecond)
	app.noteContentFetchResult(true, 150*time.Millisecond)
	app.noteContentFetchResult(false, 200*time.Millisecond)

	metrics := app.GetReleaseMetrics()
	if metrics.ContentFetchAttempts != 4 {
		t.Errorf("attempts: want 4, got %d", metrics.ContentFetchAttempts)
	}
	if metrics.ContentFetchSuccess != 3 {
		t.Errorf("success: want 3, got %d", metrics.ContentFetchSuccess)
	}
	if metrics.ContentFetchFailures != 1 {
		t.Errorf("failures: want 1, got %d", metrics.ContentFetchFailures)
	}
	if metrics.ContentFetchSuccessRate != 0.75 {
		t.Errorf("success rate: want 0.75, got %v", metrics.ContentFetchSuccessRate)
	}
	// p95 of [50, 100, 150, 200] sorted: idx = (4*95-1)/100 = 379/100 = 3 → 200ms.
	if metrics.ContentFetchLatencyP95 != 200 {
		t.Errorf("p95 latency: want 200ms, got %dms", metrics.ContentFetchLatencyP95)
	}
}

func TestContentFetchResultClampsNegativeLatency(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// time.Duration arithmetic can produce negatives if the system clock jumps;
	// the implementation clamps to 0 to keep the latency stream consistent.
	app.noteContentFetchAttempt()
	app.noteContentFetchResult(true, -42*time.Millisecond)

	metrics := app.GetReleaseMetrics()
	if metrics.ContentFetchLatencyP95 != 0 {
		t.Errorf("negative latency must be clamped to 0, got %d", metrics.ContentFetchLatencyP95)
	}
}

func TestContentFetchLatencyRingBuffer(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Push more than maxFetchLatencySamples to ensure the ring buffer evicts old samples.
	overflow := maxFetchLatencySamples + 100
	for i := 0; i < overflow; i++ {
		app.noteContentFetchAttempt()
		app.noteContentFetchResult(true, time.Duration(i)*time.Millisecond)
	}

	app.observabilityMu.Lock()
	got := len(app.observabilityStats.ContentFetchLatency)
	app.observabilityMu.Unlock()

	if got != maxFetchLatencySamples {
		t.Errorf("latency buffer should be capped at %d, got %d", maxFetchLatencySamples, got)
	}

	// p95 should be drawn from the high tail (most recent values), not 0.
	metrics := app.GetReleaseMetrics()
	if metrics.ContentFetchLatencyP95 == 0 {
		t.Error("p95 should not be 0 when samples are present")
	}
}
