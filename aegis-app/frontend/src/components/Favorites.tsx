import { useState, useEffect } from 'react';
import { Post, Profile } from '../types';
import { PostCard } from './PostCard';
import { GetFavorites, RemoveFavorite } from '../../wailsjs/go/main/App';

const favoritesCache = new Map<string, Post[]>();

interface FavoritesProps {
  allPosts: Post[]; // Kept for interface compatibility but we fetch real favorites now
  refreshToken?: number;
  currentPubkey?: string;
  profiles: Record<string, Profile>;
  onUpvote: (postId: string) => void;
  onPostClick: (post: Post) => void;
  onToggleFavorite?: (postId: string) => void;
}

export function Favorites({ refreshToken = 0, currentPubkey = '', profiles, onUpvote, onPostClick, onToggleFavorite }: FavoritesProps) {
  const cacheKey = currentPubkey.trim() || 'anonymous';
  const [favoritePosts, setFavoritePosts] = useState<Post[]>(() => favoritesCache.get(cacheKey) || []);
  const [loading, setLoading] = useState(() => !favoritesCache.has(cacheKey));

  const loadFavorites = async () => {
    const cached = favoritesCache.get(cacheKey);
    if (cached) {
      setFavoritePosts(cached);
      setLoading(false);
    } else {
      setLoading(true);
    }
    try {
      const page = await GetFavorites(50, '');
      const mapped: Post[] = page.items.map((item: any) => ({
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
      }));

      favoritesCache.set(cacheKey, mapped);
      setFavoritePosts(mapped);
    } catch (e) {
      console.error('Failed to load favorites:', e);
      setFavoritePosts(cached || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadFavorites();
  }, [refreshToken, cacheKey]);

  const handleRemoveFavorite = async (postId: string) => {
    try {
      await RemoveFavorite(postId);
      setFavoritePosts((prev) => {
        const next = prev.filter((p) => p.id !== postId);
        favoritesCache.set(cacheKey, next);
        return next;
      });
      if (onToggleFavorite) onToggleFavorite(postId); // Notify parent to update global state
    } catch (e) {
      console.error('Failed to remove favorite:', e);
    }
  };

  if (loading) {
    return (
      <div className="flex-1 overflow-y-auto p-4 md:p-6">
        <div className="max-w-2xl mx-auto">
          <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white mb-6">Favorites</h1>
          <div className="text-center py-12 text-warm-text-secondary dark:text-slate-400">
            <span className="material-icons animate-spin text-2xl mb-2">refresh</span>
            <p>Loading favorites...</p>
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
                  isFavorited={true}
                  onToggleFavorite={() => handleRemoveFavorite(post.id)}
                />
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
