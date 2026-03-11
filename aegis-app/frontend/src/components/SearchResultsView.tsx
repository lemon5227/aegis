import { useState } from 'react';
import { Post, Profile, Sub } from '../types';
import { PostCard } from './PostCard';

type SearchSortMode = 'relevance' | 'new' | 'top';
type SearchTimeRange = 'all' | 'day' | 'week' | 'month' | 'year';

interface SearchResultsViewProps {
  query: string;
  subs: Sub[];
  posts: Array<Post & { isFavorited?: boolean }>;
  profiles: Record<string, Profile>;
  scopeSubId?: string;
  onSubClick: (subId: string) => void;
  onPostClick: (post: Post) => void;
  onAuthorClick?: (pubkey: string) => void;
  onShare?: (post: Post) => void;
  onUpvote: (postId: string) => void;
  onToggleFavorite?: (postId: string) => void;
}

function withinTimeRange(timestamp: number, range: SearchTimeRange): boolean {
  if (range === 'all') {
    return true;
  }

  const now = Date.now();
  const createdAt = timestamp * 1000;
  const hour = 60 * 60 * 1000;
  const thresholds: Record<Exclude<SearchTimeRange, 'all'>, number> = {
    day: 24 * hour,
    week: 7 * 24 * hour,
    month: 30 * 24 * hour,
    year: 365 * 24 * hour,
  };

  return now-createdAt <= thresholds[range];
}

export function SearchResultsView({
  query,
  subs,
  posts,
  profiles,
  scopeSubId,
  onSubClick,
  onPostClick,
  onAuthorClick,
  onShare,
  onUpvote,
  onToggleFavorite,
}: SearchResultsViewProps) {
  const [authorQuery, setAuthorQuery] = useState('');
  const [timeRange, setTimeRange] = useState<SearchTimeRange>('all');
  const [sortMode, setSortMode] = useState<SearchSortMode>('relevance');

  const normalizedAuthorQuery = authorQuery.trim().toLowerCase();
  const filteredPosts = posts
    .filter((post) => {
      if (!withinTimeRange(post.timestamp, timeRange)) {
        return false;
      }
      if (!normalizedAuthorQuery) {
        return true;
      }

      const profile = profiles[post.pubkey];
      const displayName = (profile?.displayName || '').toLowerCase();
      const pubkey = post.pubkey.toLowerCase();
      return displayName.includes(normalizedAuthorQuery) || pubkey.includes(normalizedAuthorQuery);
    })
    .sort((left, right) => {
      if (sortMode === 'new') {
        return right.timestamp - left.timestamp;
      }
      if (sortMode === 'top') {
        if (right.score === left.score) {
          return right.timestamp - left.timestamp;
        }
        return right.score - left.score;
      }
      return 0;
    });

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6">
      <div className="max-w-5xl mx-auto space-y-6">
        <div className="bg-warm-card dark:bg-surface-dark rounded-2xl border border-warm-border dark:border-border-dark p-5">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.18em] text-warm-text-secondary dark:text-slate-400">Search</p>
              <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white mt-1">
                Results for "{query}"
              </h1>
              <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-2">
                {scopeSubId ? `Scoped to ${scopeSubId}` : 'Searching across all subs'} · {filteredPosts.length} posts · {subs.length} subs
              </p>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 min-w-0 lg:min-w-[520px]">
              <input
                type="text"
                value={authorQuery}
                onChange={(e) => setAuthorQuery(e.target.value)}
                placeholder="Filter by author"
                className="w-full px-3 py-2 rounded-xl border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-sm text-warm-text-primary dark:text-white outline-none focus:ring-2 focus:ring-warm-accent"
              />
              <select
                value={timeRange}
                onChange={(e) => setTimeRange(e.target.value as SearchTimeRange)}
                className="w-full px-3 py-2 rounded-xl border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-sm text-warm-text-primary dark:text-white outline-none focus:ring-2 focus:ring-warm-accent"
              >
                <option value="all">All time</option>
                <option value="day">Past day</option>
                <option value="week">Past week</option>
                <option value="month">Past month</option>
                <option value="year">Past year</option>
              </select>
              <select
                value={sortMode}
                onChange={(e) => setSortMode(e.target.value as SearchSortMode)}
                className="w-full px-3 py-2 rounded-xl border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-sm text-warm-text-primary dark:text-white outline-none focus:ring-2 focus:ring-warm-accent"
              >
                <option value="relevance">Relevance</option>
                <option value="new">Newest</option>
                <option value="top">Top scored</option>
              </select>
            </div>
          </div>
        </div>

        {!scopeSubId && subs.length > 0 && (
          <section className="space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">
                Matching Subs
              </h2>
              <span className="text-xs text-warm-text-secondary dark:text-slate-400">{subs.length} found</span>
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              {subs.map((sub) => (
                <button
                  key={sub.id}
                  onClick={() => onSubClick(sub.id)}
                  className="text-left bg-warm-card dark:bg-surface-dark rounded-2xl border border-warm-border dark:border-border-dark p-4 hover:border-warm-accent/40 transition-colors"
                >
                  <div className="flex items-start gap-3">
                    <div className="w-11 h-11 rounded-xl bg-gradient-to-br from-warm-accent to-orange-400 text-white flex items-center justify-center shrink-0">
                      <span className="material-icons-outlined">forum</span>
                    </div>
                    <div className="min-w-0">
                      <div className="text-base font-semibold text-warm-text-primary dark:text-white">{sub.id}</div>
                      <div className="text-sm text-warm-text-secondary dark:text-slate-400 truncate">{sub.title || 'Untitled sub'}</div>
                      {sub.description && (
                        <div className="text-sm text-warm-text-secondary dark:text-slate-400 mt-2 line-clamp-2">{sub.description}</div>
                      )}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          </section>
        )}

        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">
              Matching Posts
            </h2>
            <span className="text-xs text-warm-text-secondary dark:text-slate-400">{filteredPosts.length} shown</span>
          </div>

          {filteredPosts.length === 0 ? (
            <div className="bg-warm-card dark:bg-surface-dark rounded-2xl border border-warm-border dark:border-border-dark py-14 text-center text-warm-text-secondary dark:text-slate-400">
              <span className="material-icons-outlined text-4xl mb-3">manage_search</span>
              <p>No posts match the current filters.</p>
            </div>
          ) : (
            filteredPosts.map((post) => (
              <PostCard
                key={post.id}
                post={post}
                authorProfile={profiles[post.pubkey]}
                onUpvote={onUpvote}
                onClick={onPostClick}
                onAuthorClick={onAuthorClick}
                onShare={onShare}
                isFavorited={post.isFavorited}
                onToggleFavorite={onToggleFavorite}
              />
            ))
          )}
        </section>
      </div>
    </div>
  );
}
