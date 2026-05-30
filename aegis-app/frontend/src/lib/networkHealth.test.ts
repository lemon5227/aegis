import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { deriveNetworkHealth, formatRelativeNetworkTime } from './networkHealth';
import type { AntiEntropyStats, P2PStatus } from '../types';

const baseStatus = (overrides: Partial<P2PStatus> = {}): P2PStatus => ({
  started: true,
  peerId: 'peer-id',
  listenAddrs: [],
  announceAddrs: [],
  connectedPeers: [],
  topic: '',
  ...overrides,
});

const baseStats = (overrides: Partial<AntiEntropyStats> = {}): AntiEntropyStats => ({
  syncRequestsSent: 0,
  syncRequestsReceived: 0,
  syncResponsesReceived: 0,
  syncSummariesReceived: 0,
  indexInsertions: 0,
  blobFetchAttempts: 0,
  blobFetchSuccess: 0,
  blobFetchFailures: 0,
  lastSyncAt: 0,
  lastRemoteSummaryTs: 0,
  lastObservedSyncLagSec: 0,
  ...overrides,
});

describe('deriveNetworkHealth', () => {
  it('returns offline when status is null', () => {
    const got = deriveNetworkHealth(null, null);
    expect(got.level).toBe('offline');
    expect(got.peerCount).toBe(0);
  });

  it('returns offline when p2p has not started', () => {
    const got = deriveNetworkHealth(baseStatus({ started: false }), null);
    expect(got.level).toBe('offline');
  });

  it('returns degraded when started but no peers connected', () => {
    const got = deriveNetworkHealth(
      baseStatus({ started: true, connectedPeers: [] }),
      null,
    );
    expect(got.level).toBe('degraded');
    expect(got.label).toBe('Working Offline');
  });

  it('returns syncing when peers connected but lag exceeds 30s', () => {
    const got = deriveNetworkHealth(
      baseStatus({ connectedPeers: ['peer-a'] }),
      baseStats({ lastObservedSyncLagSec: 60, lastSyncAt: 1700000000 }),
    );
    expect(got.level).toBe('syncing');
  });

  it('returns healthy when peers connected and lag under threshold', () => {
    const got = deriveNetworkHealth(
      baseStatus({ connectedPeers: ['peer-a'] }),
      baseStats({ lastObservedSyncLagSec: 5, lastSyncAt: 1700000000 }),
    );
    expect(got.level).toBe('healthy');
    expect(got.label).toBe('Up To Date');
  });

  it('does not flag syncing when lag is high but lastSyncAt is zero', () => {
    // Edge case: lastSyncAt=0 means we have not synced yet, so the syncing
    // label would be misleading.
    const got = deriveNetworkHealth(
      baseStatus({ connectedPeers: ['peer-a'] }),
      baseStats({ lastObservedSyncLagSec: 60, lastSyncAt: 0 }),
    );
    expect(got.level).toBe('healthy');
  });

  it('reports peer count from connectedPeers length', () => {
    const got = deriveNetworkHealth(
      baseStatus({ connectedPeers: ['p1', 'p2', 'p3'] }),
      null,
    );
    expect(got.peerCount).toBe(3);
  });

  it('computes blob failure rate when stats are populated', () => {
    const got = deriveNetworkHealth(
      baseStatus({ connectedPeers: ['p'] }),
      baseStats({ blobFetchAttempts: 10, blobFetchFailures: 3, lastSyncAt: 1, lastObservedSyncLagSec: 5 }),
    );
    expect(got.blobFailureRate).toBeCloseTo(0.3, 5);
  });

  it('reports zero blob failure rate when no attempts', () => {
    const got = deriveNetworkHealth(
      baseStatus({ connectedPeers: ['p'] }),
      baseStats({ blobFetchAttempts: 0, blobFetchFailures: 0 }),
    );
    expect(got.blobFailureRate).toBe(0);
  });
});

describe('formatRelativeNetworkTime', () => {
  beforeEach(() => {
    // Pin Date.now() to a fixed instant so the relative formatting is stable.
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-30T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const NOW_SEC = Math.floor(new Date('2026-05-30T12:00:00Z').getTime() / 1000);

  it('returns "Never" for falsy timestamp', () => {
    expect(formatRelativeNetworkTime(0)).toBe('Never');
    expect(formatRelativeNetworkTime(NaN)).toBe('Never');
  });

  it('returns "Just now" within 10 seconds', () => {
    expect(formatRelativeNetworkTime(NOW_SEC - 5)).toBe('Just now');
    expect(formatRelativeNetworkTime(NOW_SEC)).toBe('Just now');
  });

  it('returns seconds for sub-minute deltas', () => {
    expect(formatRelativeNetworkTime(NOW_SEC - 30)).toBe('30s ago');
  });

  it('returns minutes for sub-hour deltas', () => {
    expect(formatRelativeNetworkTime(NOW_SEC - 5 * 60)).toBe('5m ago');
    expect(formatRelativeNetworkTime(NOW_SEC - 59 * 60)).toBe('59m ago');
  });

  it('returns hours for sub-day deltas', () => {
    expect(formatRelativeNetworkTime(NOW_SEC - 3 * 3600)).toBe('3h ago');
    expect(formatRelativeNetworkTime(NOW_SEC - 23 * 3600)).toBe('23h ago');
  });

  it('returns days for >= 1 day deltas', () => {
    expect(formatRelativeNetworkTime(NOW_SEC - 2 * 86400)).toBe('2d ago');
    expect(formatRelativeNetworkTime(NOW_SEC - 30 * 86400)).toBe('30d ago');
  });

  it('clamps negative deltas (timestamp in the future) to 0', () => {
    // Implementation uses Math.max(0, ...) on the diff; future timestamp
    // should display as 'Just now' rather than a negative count.
    expect(formatRelativeNetworkTime(NOW_SEC + 60)).toBe('Just now');
  });
});
