import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  listPendingSyncActions,
  recordPendingSyncAction,
  reconcilePendingSyncActions,
  removePendingSyncAction,
} from './pendingSync';
import type { NetworkHealthSnapshot } from './networkHealth';

const baseHealth = (overrides: Partial<NetworkHealthSnapshot> = {}): NetworkHealthSnapshot => ({
  level: 'healthy',
  label: 'Up To Date',
  summary: '',
  peerCount: 1,
  lagSeconds: 0,
  lastSyncAt: 0,
  lastRemoteSummaryTs: 0,
  blobFailureRate: 0,
  ...overrides,
});

describe('pendingSyncActions', () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-30T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
    window.localStorage.clear();
  });

  it('starts empty', () => {
    expect(listPendingSyncActions()).toEqual([]);
  });

  it('records an action', () => {
    const after = recordPendingSyncAction('post-create', 'post-1', 'created post-1');
    expect(after).toHaveLength(1);
    expect(after[0].kind).toBe('post-create');
    expect(after[0].entityId).toBe('post-1');
    expect(after[0].summary).toBe('created post-1');

    expect(listPendingSyncActions()).toHaveLength(1);
  });

  it('orders most recent first', () => {
    recordPendingSyncAction('post-create', 'p1', 'a');
    vi.advanceTimersByTime(100);
    recordPendingSyncAction('comment-create', 'c1', 'b');
    vi.advanceTimersByTime(100);
    recordPendingSyncAction('post-vote', 'p1', 'c');

    const got = listPendingSyncActions();
    expect(got.map((a) => a.kind)).toEqual(['post-vote', 'comment-create', 'post-create']);
  });

  it('caps at 100 entries', () => {
    for (let i = 0; i < 120; i++) {
      vi.advanceTimersByTime(1);
      recordPendingSyncAction('post-create', `p${i}`, '');
    }
    expect(listPendingSyncActions()).toHaveLength(100);
  });

  it('removePendingSyncAction drops a single entry by id', () => {
    const after = recordPendingSyncAction('post-create', 'p1', 'a');
    const id = after[0].id;
    recordPendingSyncAction('post-create', 'p2', 'b');

    const next = removePendingSyncAction(id);
    expect(next).toHaveLength(1);
    expect(next[0].entityId).toBe('p2');
  });

  it('survives malformed localStorage payloads', () => {
    window.localStorage.setItem('aegis:pending-sync:v1', 'not json');
    expect(listPendingSyncActions()).toEqual([]);
  });

  it('filters malformed entries inside a valid array', () => {
    window.localStorage.setItem(
      'aegis:pending-sync:v1',
      JSON.stringify([
        { id: 'a', kind: 'post-create', entityId: 'p1', summary: '', createdAt: 1 },
        { kind: 'post-create' }, // missing id
        null,
        'string',
        { id: 'b', kind: 'comment-create', entityId: 'c1', summary: '', createdAt: 2 },
      ]),
    );
    const got = listPendingSyncActions();
    expect(got.map((a) => a.id)).toEqual(['b', 'a']);
  });
});

describe('reconcilePendingSyncActions', () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-30T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
    window.localStorage.clear();
  });

  it('is a no-op when no actions are pending', () => {
    const got = reconcilePendingSyncActions(baseHealth({ level: 'healthy', lastSyncAt: 1700000000 }));
    expect(got).toEqual([]);
  });

  it('does nothing when offline', () => {
    recordPendingSyncAction('post-create', 'p1', 'a');
    const got = reconcilePendingSyncActions(baseHealth({ level: 'offline', lastSyncAt: 1700000000 }));
    expect(got).toHaveLength(1);
  });

  it('does nothing when peerCount=0', () => {
    recordPendingSyncAction('post-create', 'p1', 'a');
    const got = reconcilePendingSyncActions(baseHealth({ peerCount: 0, lastSyncAt: 1700000000 }));
    expect(got).toHaveLength(1);
  });

  it('does nothing when no successful sync has occurred', () => {
    recordPendingSyncAction('post-create', 'p1', 'a');
    const got = reconcilePendingSyncActions(baseHealth({ peerCount: 2, lastSyncAt: 0 }));
    expect(got).toHaveLength(1);
  });

  it('drops actions older than the last sync', () => {
    // Pin createdAt at 'now'.
    const oldAction = recordPendingSyncAction('post-create', 'p-old', 'a');
    expect(oldAction).toHaveLength(1);

    // Advance time and simulate a sync that finishes 'after' p-old was created.
    vi.advanceTimersByTime(60_000); // 60 seconds later
    const lastSyncAt = Math.floor(Date.now() / 1000) - 1;
    recordPendingSyncAction('post-create', 'p-new', 'b');

    const got = reconcilePendingSyncActions(
      baseHealth({ peerCount: 2, lastSyncAt }),
    );
    expect(got.map((a) => a.entityId)).toEqual(['p-new']);
  });

  it('keeps all actions when none are older than the last sync', () => {
    recordPendingSyncAction('post-create', 'p-fresh', '');
    // Move clock forward, but lastSyncAt is BEFORE the action was created.
    const lastSyncAt = Math.floor(Date.now() / 1000) - 60;

    const got = reconcilePendingSyncActions(
      baseHealth({ peerCount: 2, lastSyncAt }),
    );
    expect(got).toHaveLength(1);
  });
});
