import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { clearRecentlyViewed, listRecentlyViewed, recordRecentlyViewed } from './history';

describe('recentlyViewed', () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-30T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
    window.localStorage.clear();
  });

  it('returns empty list on a fresh storage', () => {
    expect(listRecentlyViewed()).toEqual([]);
  });

  it('records a viewed post', () => {
    recordRecentlyViewed('post-1');
    const got = listRecentlyViewed();
    expect(got).toHaveLength(1);
    expect(got[0].postId).toBe('post-1');
  });

  it('moves repeat views to the front (most recent first)', () => {
    recordRecentlyViewed('post-1');
    vi.advanceTimersByTime(1000);
    recordRecentlyViewed('post-2');
    vi.advanceTimersByTime(1000);
    recordRecentlyViewed('post-1'); // repeat -> moves to front

    const got = listRecentlyViewed();
    expect(got.map((e) => e.postId)).toEqual(['post-1', 'post-2']);
    expect(got[0].viewedAt).toBeGreaterThan(got[1].viewedAt);
  });

  it('caps the list at 50 entries', () => {
    for (let i = 0; i < 60; i++) {
      vi.advanceTimersByTime(1);
      recordRecentlyViewed(`post-${i}`);
    }
    expect(listRecentlyViewed()).toHaveLength(50);
  });

  it('ignores empty / whitespace post ids', () => {
    recordRecentlyViewed('');
    recordRecentlyViewed('   ');
    expect(listRecentlyViewed()).toHaveLength(0);
  });

  it('trims whitespace from recorded post ids', () => {
    recordRecentlyViewed('  post-trim  ');
    const got = listRecentlyViewed();
    expect(got[0].postId).toBe('post-trim');
  });

  it('clearRecentlyViewed() with no arg wipes everything', () => {
    recordRecentlyViewed('post-1');
    recordRecentlyViewed('post-2');
    clearRecentlyViewed();
    expect(listRecentlyViewed()).toEqual([]);
  });

  it('clearRecentlyViewed(postId) removes only that entry', () => {
    recordRecentlyViewed('keep-me');
    recordRecentlyViewed('drop-me');
    clearRecentlyViewed('drop-me');

    const got = listRecentlyViewed();
    expect(got).toHaveLength(1);
    expect(got[0].postId).toBe('keep-me');
  });

  it('survives malformed localStorage payloads', () => {
    window.localStorage.setItem('aegis:recently-viewed:v1', 'not json');
    expect(listRecentlyViewed()).toEqual([]);
  });

  it('filters out malformed entries inside a valid array', () => {
    window.localStorage.setItem(
      'aegis:recently-viewed:v1',
      JSON.stringify([
        { postId: 'good', viewedAt: 1700000000 },
        { postId: 123, viewedAt: 1700000001 }, // bad: postId not string
        { viewedAt: 1700000002 },                // bad: missing postId
        null,                                    // bad: not an object
        { postId: 'also-good', viewedAt: 1700000003 },
      ]),
    );
    const got = listRecentlyViewed();
    expect(got.map((e) => e.postId)).toEqual(['good', 'also-good']);
  });
});
