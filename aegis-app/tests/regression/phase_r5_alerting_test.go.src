package main

import "testing"

func TestR5ReleaseAlertRaisedAfterSustainWindow(t *testing.T) {
	app := NewApp()
	metrics := ReleaseMetrics{
		ContentFetchAttempts:    100,
		ContentFetchSuccess:     70,
		ContentFetchFailures:    30,
		ContentFetchSuccessRate: 0.70,
		ContentFetchLatencyP95:  6200,
		BlobCacheHits:           2,
		BlobCacheMisses:         8,
		BlobCacheHitRate:        0.20,
		SyncLagSeconds:          700,
	}

	_ = app.evaluateReleaseAlertsAt(1000, metrics)
	alerts := app.evaluateReleaseAlertsAt(1600, metrics)
	if len(alerts) == 0 {
		t.Fatalf("expected active alerts after sustain window")
	}

	byKey := make(map[string]struct{}, len(alerts))
	for _, alert := range alerts {
		byKey[alert.Key] = struct{}{}
	}
	expected := []string{
		"content_fetch_success_rate_warning",
		"content_fetch_success_rate_critical",
		"content_fetch_latency_p95_warning",
		"content_fetch_latency_p95_critical",
		"blob_cache_hit_rate_warning",
		"sync_lag_seconds_warning",
		"sync_lag_seconds_critical",
	}
	for _, key := range expected {
		if _, ok := byKey[key]; !ok {
			t.Fatalf("expected alert key %s to be active", key)
		}
	}
}

func TestR5ReleaseAlertRecoveryClearsActiveState(t *testing.T) {
	app := NewApp()
	bad := ReleaseMetrics{
		ContentFetchAttempts:    10,
		ContentFetchSuccess:     6,
		ContentFetchFailures:    4,
		ContentFetchSuccessRate: 0.60,
		ContentFetchLatencyP95:  7000,
		BlobCacheHits:           1,
		BlobCacheMisses:         9,
		BlobCacheHitRate:        0.10,
		SyncLagSeconds:          900,
	}
	ok := ReleaseMetrics{
		ContentFetchAttempts:    10,
		ContentFetchSuccess:     10,
		ContentFetchFailures:    0,
		ContentFetchSuccessRate: 1.0,
		ContentFetchLatencyP95:  200,
		BlobCacheHits:           9,
		BlobCacheMisses:         1,
		BlobCacheHitRate:        0.90,
		SyncLagSeconds:          10,
	}

	_ = app.evaluateReleaseAlertsAt(1000, bad)
	_ = app.evaluateReleaseAlertsAt(1700, bad)
	if len(app.GetReleaseAlerts()) == 0 {
		t.Fatalf("expected active alerts before recovery")
	}

	alerts := app.evaluateReleaseAlertsAt(1800, ok)
	if len(alerts) != 0 {
		t.Fatalf("expected alerts to clear after recovery, got %d", len(alerts))
	}
	if len(app.GetReleaseAlerts()) != 0 {
		t.Fatalf("expected stored active alerts to be cleared")
	}
}
