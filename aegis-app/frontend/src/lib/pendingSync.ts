import { PendingSyncAction, PendingSyncActionKind } from '../types';
import { NetworkHealthSnapshot } from './networkHealth';

const PENDING_SYNC_STORAGE_KEY = 'aegis:pending-sync:v1';

function canUseStorage(): boolean {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

export function listPendingSyncActions(): PendingSyncAction[] {
  if (!canUseStorage()) {
    return [];
  }
  const rawValue = window.localStorage.getItem(PENDING_SYNC_STORAGE_KEY);
  if (!rawValue) {
    return [];
  }

  try {
    const parsed = JSON.parse(rawValue) as PendingSyncAction[];
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .filter((item) => item && typeof item.id === 'string' && typeof item.kind === 'string')
      .sort((left, right) => right.createdAt - left.createdAt);
  } catch {
    return [];
  }
}

function savePendingSyncActions(items: PendingSyncAction[]): void {
  if (!canUseStorage()) {
    return;
  }
  window.localStorage.setItem(PENDING_SYNC_STORAGE_KEY, JSON.stringify(items));
}

export function recordPendingSyncAction(kind: PendingSyncActionKind, entityId: string, summary: string): PendingSyncAction[] {
  const item: PendingSyncAction = {
    id: `${kind}:${entityId}:${Date.now()}`,
    kind,
    entityId,
    summary,
    createdAt: Date.now(),
  };
  const next = [item, ...listPendingSyncActions()].slice(0, 100);
  savePendingSyncActions(next);
  return next;
}

export function removePendingSyncAction(actionId: string): PendingSyncAction[] {
  const next = listPendingSyncActions().filter((item) => item.id !== actionId);
  savePendingSyncActions(next);
  return next;
}

export function reconcilePendingSyncActions(health: NetworkHealthSnapshot): PendingSyncAction[] {
  const current = listPendingSyncActions();
  if (current.length === 0) {
    return current;
  }

  const lastSyncMs = health.lastSyncAt > 0 ? health.lastSyncAt * 1000 : 0;
  if (health.level === 'offline' || health.peerCount === 0 || lastSyncMs === 0) {
    return current;
  }

  const next = current.filter((item) => item.createdAt > lastSyncMs);
  if (next.length !== current.length) {
    savePendingSyncActions(next);
  }
  return next;
}
