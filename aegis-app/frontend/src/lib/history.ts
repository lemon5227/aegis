import { RecentlyViewedEntry } from '../types';

const HISTORY_STORAGE_KEY = 'aegis:recently-viewed:v1';
const MAX_HISTORY_ITEMS = 50;

function canUseStorage(): boolean {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

export function listRecentlyViewed(): RecentlyViewedEntry[] {
  if (!canUseStorage()) {
    return [];
  }
  const rawValue = window.localStorage.getItem(HISTORY_STORAGE_KEY);
  if (!rawValue) {
    return [];
  }

  try {
    const parsed = JSON.parse(rawValue) as RecentlyViewedEntry[];
    return Array.isArray(parsed)
      ? parsed.filter((item) => item && typeof item.postId === 'string' && typeof item.viewedAt === 'number')
      : [];
  } catch {
    return [];
  }
}

export function recordRecentlyViewed(postId: string): void {
  if (!canUseStorage()) {
    return;
  }
  const normalizedPostId = postId.trim();
  if (!normalizedPostId) {
    return;
  }

  const nextEntries = [
    { postId: normalizedPostId, viewedAt: Date.now() },
    ...listRecentlyViewed().filter((item) => item.postId !== normalizedPostId),
  ].slice(0, MAX_HISTORY_ITEMS);

  window.localStorage.setItem(HISTORY_STORAGE_KEY, JSON.stringify(nextEntries));
}

export function clearRecentlyViewed(postId?: string): void {
  if (!canUseStorage()) {
    return;
  }
  if (!postId) {
    window.localStorage.removeItem(HISTORY_STORAGE_KEY);
    return;
  }
  const normalizedPostId = postId.trim();
  const nextEntries = listRecentlyViewed().filter((item) => item.postId !== normalizedPostId);
  window.localStorage.setItem(HISTORY_STORAGE_KEY, JSON.stringify(nextEntries));
}
