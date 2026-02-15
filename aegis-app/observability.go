package main

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxFetchLatencySamples = 512

type ObservabilityStats struct {
	ContentFetchAttempts int64
	ContentFetchSuccess  int64
	ContentFetchFailures int64
	BlobCacheHits        int64
	BlobCacheMisses      int64
	ContentFetchLatency  []int64
}

type ReleaseMetrics struct {
	ContentFetchSuccessRate float64 `json:"content_fetch_success_rate"`
	ContentFetchLatencyP95  int64   `json:"content_fetch_latency_p95"`
	BlobCacheHitRate        float64 `json:"blob_cache_hit_rate"`
	SyncLagSeconds          int64   `json:"sync_lag_seconds"`

	ContentFetchAttempts int64 `json:"content_fetch_attempts"`
	ContentFetchSuccess  int64 `json:"content_fetch_success"`
	ContentFetchFailures int64 `json:"content_fetch_failures"`
	BlobCacheHits        int64 `json:"blob_cache_hits"`
	BlobCacheMisses      int64 `json:"blob_cache_misses"`
}

func (a *App) noteBlobCacheHit() {
	a.observabilityMu.Lock()
	defer a.observabilityMu.Unlock()
	a.observabilityStats.BlobCacheHits++
}

func (a *App) noteBlobCacheMiss() {
	a.observabilityMu.Lock()
	defer a.observabilityMu.Unlock()
	a.observabilityStats.BlobCacheMisses++
}

func (a *App) noteContentFetchAttempt() {
	a.observabilityMu.Lock()
	defer a.observabilityMu.Unlock()
	a.observabilityStats.ContentFetchAttempts++
}

func (a *App) noteContentFetchResult(success bool, latency time.Duration) {
	a.observabilityMu.Lock()
	defer a.observabilityMu.Unlock()

	if success {
		a.observabilityStats.ContentFetchSuccess++
	} else {
		a.observabilityStats.ContentFetchFailures++
	}

	latencyMs := latency.Milliseconds()
	if latencyMs < 0 {
		latencyMs = 0
	}
	a.observabilityStats.ContentFetchLatency = append(a.observabilityStats.ContentFetchLatency, latencyMs)
	if len(a.observabilityStats.ContentFetchLatency) > maxFetchLatencySamples {
		overflow := len(a.observabilityStats.ContentFetchLatency) - maxFetchLatencySamples
		a.observabilityStats.ContentFetchLatency = a.observabilityStats.ContentFetchLatency[overflow:]
	}
}

func (a *App) GetReleaseMetrics() ReleaseMetrics {
	a.observabilityMu.Lock()
	snapshot := a.observabilityStats
	latency := append([]int64(nil), snapshot.ContentFetchLatency...)
	a.observabilityMu.Unlock()

	a.antiEntropyMu.Lock()
	syncLag := a.antiEntropyStats.LastObservedSyncLagSec
	a.antiEntropyMu.Unlock()

	metrics := ReleaseMetrics{
		ContentFetchAttempts: snapshot.ContentFetchAttempts,
		ContentFetchSuccess:  snapshot.ContentFetchSuccess,
		ContentFetchFailures: snapshot.ContentFetchFailures,
		BlobCacheHits:        snapshot.BlobCacheHits,
		BlobCacheMisses:      snapshot.BlobCacheMisses,
		SyncLagSeconds:       syncLag,
	}

	if snapshot.ContentFetchAttempts > 0 {
		metrics.ContentFetchSuccessRate = float64(snapshot.ContentFetchSuccess) / float64(snapshot.ContentFetchAttempts)
	}
	totalBlobLookup := snapshot.BlobCacheHits + snapshot.BlobCacheMisses
	if totalBlobLookup > 0 {
		metrics.BlobCacheHitRate = float64(snapshot.BlobCacheHits) / float64(totalBlobLookup)
	}
	metrics.ContentFetchLatencyP95 = percentile95Latency(latency)

	return metrics
}

func percentile95Latency(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}

	sorted := append([]int64(nil), samples...)
	sort.Slice(sorted, func(i int, j int) bool {
		return sorted[i] < sorted[j]
	})

	idx := (len(sorted)*95 - 1) / 100
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func resolveFetchRetryAttempts() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_FETCH_RETRY_ATTEMPTS"))
	if raw == "" {
		return 1
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 1
	}
	if value > 3 {
		return 3
	}
	return value
}
