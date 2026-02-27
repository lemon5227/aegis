import { Post, SortMode, Profile } from '../types';
import { PostCard } from './PostCard';

interface FeedProps {
  posts: Array<Post & { reason?: string; isSubscribed?: boolean }>;
  sortMode: SortMode;
  profiles: Record<string, Profile>;
  onSortChange: (mode: SortMode) => void;
  onUpvote: (postId: string) => void;
  onPostClick: (post: Post) => void;
  onToggleFavorite?: (postId: string) => void;
}

export function Feed({ posts, sortMode, profiles, onSortChange, onUpvote, onPostClick, onToggleFavorite }: FeedProps) {
  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6 space-y-4">
      <div className="flex items-center justify-between mb-2">
        <div className="flex gap-2">
          <button
            onClick={() => onSortChange('hot')}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${
              sortMode === 'hot'
                ? 'bg-warm-accent text-white shadow-sm'
                : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-sidebar/50 dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-slate-200'
            }`}
          >
            Hot
          </button>
          <button
            onClick={() => onSortChange('new')}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors ${
              sortMode === 'new'
                ? 'bg-warm-accent text-white shadow-sm'
                : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-sidebar/50 dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-slate-200'
            }`}
          >
            New
          </button>
        </div>
      </div>
      
      {posts.length === 0 ? (
        <div className="text-center py-12 text-warm-text-secondary dark:text-slate-400">
          <span className="material-icons text-4xl mb-4">inbox</span>
          <p>No posts yet. Be the first to post!</p>
        </div>
      ) : (
        posts.map((post) => (
          <PostCard
            key={post.id}
            post={post}
            authorProfile={profiles[post.pubkey]}
            onUpvote={onUpvote}
            onClick={onPostClick}
            isRecommended={!!(post.reason && post.reason.includes('recommended') && !post.isSubscribed)}
          />
        ))
      )}
    </div>
  );
}
