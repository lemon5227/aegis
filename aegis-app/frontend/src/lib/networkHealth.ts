import { AntiEntropyStats, P2PStatus } from '../types';

export type NetworkHealthLevel = 'healthy' | 'syncing' | 'degraded' | 'offline';

export interface NetworkHealthSnapshot {
  level: NetworkHealthLevel;
  label: string;
  summary: string;
  peerCount: number;
  lagSeconds: number;
  lastSyncAt: number;
  lastRemoteSummaryTs: number;
  blobFailureRate: number;
}

function calculateBlobFailureRate(stats: AntiEntropyStats | null): number {
  if (!stats || stats.blobFetchAttempts <= 0) {
    return 0;
  }
  return stats.blobFetchFailures / stats.blobFetchAttempts;
}

export function deriveNetworkHealth(status: P2PStatus | null, stats: AntiEntropyStats | null): NetworkHealthSnapshot {
  const peerCount = status?.connectedPeers?.length || 0;
  const lagSeconds = stats?.lastObservedSyncLagSec || 0;
  const lastSyncAt = stats?.lastSyncAt || 0;
  const lastRemoteSummaryTs = stats?.lastRemoteSummaryTs || 0;
  const blobFailureRate = calculateBlobFailureRate(stats);

  if (!status?.started) {
    return {
      level: 'offline',
      label: 'Offline',
      summary: 'You can keep using the app. New activity will be sent when connection is available again.',
      peerCount,
      lagSeconds,
      lastSyncAt,
      lastRemoteSummaryTs,
      blobFailureRate,
    };
  }

  if (peerCount === 0) {
    return {
      level: 'degraded',
      label: 'Working Offline',
      summary: 'You can keep using the app. We will send your changes when the app reconnects.',
      peerCount,
      lagSeconds,
      lastSyncAt,
      lastRemoteSummaryTs,
      blobFailureRate,
    };
  }

  // Sync lag and blob failures are handled silently in the background.
  // No need to surface transient sync states to the user.

  return {
    level: 'healthy',
    label: 'Up To Date',
    summary: 'Everything looks current.',
    peerCount,
    lagSeconds,
    lastSyncAt,
    lastRemoteSummaryTs,
    blobFailureRate,
  };
}

export function formatRelativeNetworkTime(timestamp: number): string {
  if (!timestamp) {
    return 'Never';
  }

  const diffSeconds = Math.max(0, Math.floor(Date.now() / 1000 - timestamp));
  if (diffSeconds < 10) {
    return 'Just now';
  }
  if (diffSeconds < 60) {
    return `${diffSeconds}s ago`;
  }
  if (diffSeconds < 3600) {
    return `${Math.floor(diffSeconds / 60)}m ago`;
  }
  if (diffSeconds < 86400) {
    return `${Math.floor(diffSeconds / 3600)}h ago`;
  }
  return `${Math.floor(diffSeconds / 86400)}d ago`;
}
