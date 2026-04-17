import { useEffect, useState } from 'react';
import { Post, Profile } from '../types';
import { listRecentlyViewed, clearRecentlyViewed } from '../lib/history';
import { GetPostIndexByID } from '../../wailsjs/go/main/App';
import { PostCard } from './PostCard';

interface HistoryViewProps {
  refreshToken?: number;
  profiles: Record<string, Profile>;
  onPostClick: (post: Post) => void;
  onUpvote: (postId: string) => void;
  onAuthorClick?: (pubkey: string) => void;
  onShare?: (post: Post) => void;
}

function mapPostIndexToPost(item: any): Post {
  return {
    id: item.id,
    pubkey: item.pubkey,
    title: item.title,
    bodyPreview: item.bodyPreview || '',
    contentCid: item.contentCid || '',
    imageCid: item.imageCid || '',
    thumbCid: item.thumbCid || '',
    imageMime: item.imageMime || '',
    imageSize: item.imageSize || 0,
    imageWidth: item.imageWidth || 0,
    imageHeight: item.imageHeight || 0,
    score: item.score || 0,
    timestamp: item.timestamp || 0,
    zone: (item.zone || 'public') as 'private' | 'public',
    subId: item.subId || 'general',
    visibility: item.visibility || 'normal',
    isPinned: !!item.isPinned,
    pinnedAt: item.pinnedAt || 0,
    isLocked: !!item.isLocked,
    lockedAt: item.lockedAt || 0,
  };
}

export function HistoryView({ refreshToken = 0, profiles, onPostClick, onUpvote, onAuthorClick, onShare }: HistoryViewProps) {
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      setLoading(true);
      const entries = listRecentlyViewed();
      const resolved = await Promise.all(
        entries.map(async (entry) => {
          try {
            const index = await GetPostIndexByID(entry.postId);
            return { post: mapPostIndexToPost(index), viewedAt: entry.viewedAt };
          } catch {
            return null;
          }
        })
      );

      if (!alive) {
        return;
      }

      setPosts(
        resolved
          .filter((item): item is { post: Post; viewedAt: number } => item !== null)
          .sort((left, right) => right.viewedAt - left.viewedAt)
          .map((item) => item.post)
      );
      setLoading(false);
    };

    void load();
    return () => {
      alive = false;
    };
  }, [refreshToken]);

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6">
      <div className="max-w-4xl mx-auto space-y-6">
        <section className="rounded-3xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark p-6">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-xs uppercase tracking-[0.18em] text-warm-text-secondary dark:text-slate-400">History</p>
              <h1 className="mt-2 text-3xl font-bold text-warm-text-primary dark:text-white">Recently Viewed</h1>
              <p className="mt-3 text-sm text-warm-text-secondary dark:text-slate-300">
                Jump back into posts you opened recently. This history is stored locally on this device.
              </p>
            </div>
            {posts.length > 0 && (
              <button
                onClick={() => {
                  clearRecentlyViewed();
                  setPosts([]);
                }}
                className="rounded-lg border border-warm-border dark:border-border-dark px-4 py-2 text-sm font-medium text-warm-text-secondary dark:text-slate-300 hover:bg-warm-bg dark:hover:bg-background-dark"
              >
                Clear History
              </button>
            )}
          </div>
        </section>

        {loading ? (
          <div className="rounded-3xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark py-16 text-center text-warm-text-secondary dark:text-slate-400">
            Loading history...
          </div>
        ) : posts.length === 0 ? (
          <div className="rounded-3xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark py-16 text-center text-warm-text-secondary dark:text-slate-400">
            <span className="material-icons-outlined text-4xl mb-3">history</span>
            <p>No recently viewed posts yet.</p>
          </div>
        ) : (
          <div className="space-y-4">
            {posts.map((post) => (
              <PostCard
                key={post.id}
                post={post}
                authorProfile={profiles[post.pubkey]}
                onUpvote={onUpvote}
                onClick={onPostClick}
                onAuthorClick={onAuthorClick}
                onShare={onShare}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
