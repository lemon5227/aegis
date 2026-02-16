import { useState, useEffect } from 'react';
import { Post, Profile } from '../types';
import { PostCard } from './PostCard';

interface FavoritesProps {
  allPosts: Post[];
  profiles: Record<string, Profile>;
  onUpvote: (postId: string) => void;
  onPostClick: (post: Post) => void;
}

export function Favorites({ allPosts, profiles, onUpvote, onPostClick }: FavoritesProps) {
  const [favoriteIds, setFavoriteIds] = useState<string[]>([]);
  const [favoritePosts, setFavoritePosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const stored = localStorage.getItem('favorites');
    if (stored) {
      try {
        const ids = JSON.parse(stored);
        setFavoriteIds(ids);
        // 从 allPosts 中筛选收藏的帖子
        const favorites = allPosts.filter(p => ids.includes(p.id));
        setFavoritePosts(favorites);
      } catch (e) {
        console.error('Failed to parse favorites:', e);
      }
    }
    setLoading(false);
  }, [allPosts]);

  const handleRemoveFavorite = (postId: string) => {
    const newIds = favoriteIds.filter(id => id !== postId);
    setFavoriteIds(newIds);
    localStorage.setItem('favorites', JSON.stringify(newIds));
    setFavoritePosts(prev => prev.filter(p => p.id !== postId));
  };

  if (loading) {
    return (
      <div className="flex-1 overflow-y-auto p-4 md:p-6">
        <div className="max-w-2xl mx-auto">
          <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white mb-6">Favorites</h1>
          <div className="text-center py-12 text-warm-text-secondary dark:text-slate-400">
            Loading...
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6">
      <div className="max-w-2xl mx-auto">
        <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white mb-6">Favorites</h1>
        
        {favoritePosts.length === 0 ? (
          <div className="text-center py-12 text-warm-text-secondary dark:text-slate-400">
            <span className="material-icons text-4xl mb-4">star_border</span>
            <p>No favorites yet.</p>
            <p className="text-sm mt-2">Save posts to see them here!</p>
          </div>
        ) : (
          <div className="space-y-4">
            {favoritePosts.map((post) => (
              <div key={post.id} className="relative group">
                <PostCard
                  post={post}
                  authorProfile={profiles[post.pubkey]}
                  onUpvote={onUpvote}
                  onClick={onPostClick}
                />
                <button
                  onClick={() => handleRemoveFavorite(post.id)}
                  className="absolute top-2 right-2 p-1.5 rounded-lg bg-warm-accent/80 text-white opacity-0 group-hover:opacity-100 transition-opacity"
                  title="Remove from favorites"
                >
                  <span className="material-icons text-sm">star</span>
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
