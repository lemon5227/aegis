package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestR5ReleaseMetricsDerivedValues(t *testing.T) {
	app := NewApp()

	app.noteBlobCacheHit()
	app.noteBlobCacheHit()
	app.noteBlobCacheMiss()

	app.noteContentFetchAttempt()
	app.noteContentFetchResult(true, 30*time.Millisecond)
	app.noteContentFetchAttempt()
	app.noteContentFetchResult(false, 120*time.Millisecond)
	app.noteContentFetchAttempt()
	app.noteContentFetchResult(true, 260*time.Millisecond)

	app.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
		stats.LastObservedSyncLagSec = 42
	})

	metrics := app.GetReleaseMetrics()
	if metrics.ContentFetchAttempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", metrics.ContentFetchAttempts)
	}
	if metrics.ContentFetchSuccess != 2 {
		t.Fatalf("expected 2 success, got %d", metrics.ContentFetchSuccess)
	}
	if metrics.ContentFetchFailures != 1 {
		t.Fatalf("expected 1 failure, got %d", metrics.ContentFetchFailures)
	}
	if metrics.ContentFetchSuccessRate < 0.66 || metrics.ContentFetchSuccessRate > 0.67 {
		t.Fatalf("unexpected success rate: %f", metrics.ContentFetchSuccessRate)
	}
	if metrics.BlobCacheHitRate < 0.66 || metrics.BlobCacheHitRate > 0.67 {
		t.Fatalf("unexpected hit rate: %f", metrics.BlobCacheHitRate)
	}
	if metrics.ContentFetchLatencyP95 != 260 {
		t.Fatalf("expected p95=260ms, got %d", metrics.ContentFetchLatencyP95)
	}
	if metrics.SyncLagSeconds != 42 {
		t.Fatalf("expected sync lag 42, got %d", metrics.SyncLagSeconds)
	}
}

func TestR5BlobCacheMetricsRecordedFromReadPath(t *testing.T) {
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "r5_cache_metrics.db"))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}
	defer func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	}()

	cid := buildContentCID("r5-local-body")
	if err := app.upsertContentBlob(cid, "r5-local-body", int64(len("r5-local-body"))); err != nil {
		t.Fatalf("upsertContentBlob failed: %v", err)
	}

	if _, err := app.GetPostBodyByCID(cid); err != nil {
		t.Fatalf("expected local hit, got error: %v", err)
	}
	if _, err := app.GetPostBodyByCID("cidv1-r5-missing"); err == nil {
		t.Fatalf("expected miss error")
	}

	metrics := app.GetReleaseMetrics()
	if metrics.BlobCacheHits < 1 {
		t.Fatalf("expected at least 1 cache hit, got %d", metrics.BlobCacheHits)
	}
	if metrics.BlobCacheMisses < 1 {
		t.Fatalf("expected at least 1 cache miss, got %d", metrics.BlobCacheMisses)
	}
}
