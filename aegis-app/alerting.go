package main

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ReleaseAlert struct {
	Key         string  `json:"key"`
	Metric      string  `json:"metric"`
	Level       string  `json:"level"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
	WindowSec   int64   `json:"windowSec"`
	TriggeredAt int64   `json:"triggeredAt"`
}

type releaseAlertRule struct {
	Key       string
	Metric    string
	Level     string
	WindowSec int64
	Breached  func(metrics ReleaseMetrics) (bool, float64, float64)
}

func (a *App) GetReleaseAlerts() []ReleaseAlert {
	a.releaseAlertMu.Lock()
	defer a.releaseAlertMu.Unlock()

	result := make([]ReleaseAlert, 0, len(a.releaseAlertActive))
	for _, alert := range a.releaseAlertActive {
		result = append(result, alert)
	}
	return result
}

func (a *App) TriggerReleaseAlertEvaluationNow() []ReleaseAlert {
	return a.evaluateReleaseAlertsAt(time.Now().Unix(), a.GetReleaseMetrics())
}

func (a *App) runReleaseAlertWorker(ctx context.Context) {
	interval := resolveReleaseAlertEvalInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	a.evaluateReleaseAlertsAt(time.Now().Unix(), a.GetReleaseMetrics())

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.evaluateReleaseAlertsAt(time.Now().Unix(), a.GetReleaseMetrics())
		}
	}
}

func (a *App) evaluateReleaseAlertsAt(now int64, metrics ReleaseMetrics) []ReleaseAlert {
	rules := releaseAlertRules()
	current := make(map[string]ReleaseAlert, len(rules))

	a.releaseAlertMu.Lock()
	for _, rule := range rules {
		breached, value, threshold := rule.Breached(metrics)
		if !breached {
			delete(a.releaseAlertState, rule.Key)
			continue
		}

		firstSeen, exists := a.releaseAlertState[rule.Key]
		if !exists || firstSeen <= 0 {
			firstSeen = now
			a.releaseAlertState[rule.Key] = firstSeen
		}
		if now-firstSeen < rule.WindowSec {
			continue
		}

		current[rule.Key] = ReleaseAlert{
			Key:         rule.Key,
			Metric:      rule.Metric,
			Level:       rule.Level,
			Value:       value,
			Threshold:   threshold,
			WindowSec:   rule.WindowSec,
			TriggeredAt: now,
		}
	}

	previous := a.releaseAlertActive
	a.releaseAlertActive = current
	a.releaseAlertMu.Unlock()

	if a.ctx != nil {
		for key, alert := range current {
			if _, existed := previous[key]; existed {
				continue
			}
			runtime.LogWarningf(
				a.ctx,
				"release_alert.raised key=%s metric=%s level=%s value=%.6f threshold=%.6f window_sec=%d",
				alert.Key, alert.Metric, alert.Level, alert.Value, alert.Threshold, alert.WindowSec,
			)
		}
		for key, alert := range previous {
			if _, stillActive := current[key]; stillActive {
				continue
			}
			runtime.LogInfof(
				a.ctx,
				"release_alert.recovered key=%s metric=%s level=%s",
				alert.Key, alert.Metric, alert.Level,
			)
		}
	}

	result := make([]ReleaseAlert, 0, len(current))
	for _, alert := range current {
		result = append(result, alert)
	}
	return result
}

func releaseAlertRules() []releaseAlertRule {
	return []releaseAlertRule{
		{
			Key:       "content_fetch_success_rate_warning",
			Metric:    "content_fetch_success_rate",
			Level:     "warning",
			WindowSec: 300,
			Breached: func(metrics ReleaseMetrics) (bool, float64, float64) {
				threshold := 0.95
				if metrics.ContentFetchAttempts == 0 {
					return false, metrics.ContentFetchSuccessRate, threshold
				}
				return metrics.ContentFetchSuccessRate < threshold, metrics.ContentFetchSuccessRate, threshold
			},
		},
		{
			Key:       "content_fetch_success_rate_critical",
			Metric:    "content_fetch_success_rate",
			Level:     "critical",
			WindowSec: 180,
			Breached: func(metrics ReleaseMetrics) (bool, float64, float64) {
				threshold := 0.85
				if metrics.ContentFetchAttempts == 0 {
					return false, metrics.ContentFetchSuccessRate, threshold
				}
				return metrics.ContentFetchSuccessRate < threshold, metrics.ContentFetchSuccessRate, threshold
			},
		},
		{
			Key:       "content_fetch_latency_p95_warning",
			Metric:    "content_fetch_latency_p95",
			Level:     "warning",
			WindowSec: 300,
			Breached: func(metrics ReleaseMetrics) (bool, float64, float64) {
				threshold := 3000.0
				if metrics.ContentFetchAttempts == 0 {
					return false, float64(metrics.ContentFetchLatencyP95), threshold
				}
				return float64(metrics.ContentFetchLatencyP95) > threshold, float64(metrics.ContentFetchLatencyP95), threshold
			},
		},
		{
			Key:       "content_fetch_latency_p95_critical",
			Metric:    "content_fetch_latency_p95",
			Level:     "critical",
			WindowSec: 180,
			Breached: func(metrics ReleaseMetrics) (bool, float64, float64) {
				threshold := 5000.0
				if metrics.ContentFetchAttempts == 0 {
					return false, float64(metrics.ContentFetchLatencyP95), threshold
				}
				return float64(metrics.ContentFetchLatencyP95) > threshold, float64(metrics.ContentFetchLatencyP95), threshold
			},
		},
		{
			Key:       "blob_cache_hit_rate_warning",
			Metric:    "blob_cache_hit_rate",
			Level:     "warning",
			WindowSec: 600,
			Breached: func(metrics ReleaseMetrics) (bool, float64, float64) {
				threshold := 0.60
				if metrics.BlobCacheHits+metrics.BlobCacheMisses == 0 {
					return false, metrics.BlobCacheHitRate, threshold
				}
				return metrics.BlobCacheHitRate < threshold, metrics.BlobCacheHitRate, threshold
			},
		},
		{
			Key:       "sync_lag_seconds_warning",
			Metric:    "sync_lag_seconds",
			Level:     "warning",
			WindowSec: 300,
			Breached: func(metrics ReleaseMetrics) (bool, float64, float64) {
				threshold := 180.0
				return float64(metrics.SyncLagSeconds) > threshold, float64(metrics.SyncLagSeconds), threshold
			},
		},
		{
			Key:       "sync_lag_seconds_critical",
			Metric:    "sync_lag_seconds",
			Level:     "critical",
			WindowSec: 180,
			Breached: func(metrics ReleaseMetrics) (bool, float64, float64) {
				threshold := 600.0
				return float64(metrics.SyncLagSeconds) > threshold, float64(metrics.SyncLagSeconds), threshold
			},
		},
	}
}

func resolveReleaseAlertEvalInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("AEGIS_RELEASE_ALERT_EVAL_INTERVAL_SEC"))
	if raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return 30 * time.Second
}
