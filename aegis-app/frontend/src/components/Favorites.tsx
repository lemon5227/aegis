import { useState, useEffect } from 'react';
import { Post, Profile } from '../types';
import { PostCard } from './PostCard';
import { GetFavorites, RemoveFavorite } from '../../wailsjs/go/main/App';

interface FavoritesProps {
  allPosts: Post[]; // Kept for interface compatibility but we fetch real favorites now
  profiles: Record<string, Profile>;
  onUpvote: (postId: string) => void;
  onPostClick: (post: Post) => void;
  onToggleFavorite?: (postId: string) => void;
}

export function Favorites({ profiles, onUpvote, onPostClick, onToggleFavorite }: FavoritesProps) {
  const [favoritePosts, setFavoritePosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(true);

  const loadFavorites = async () => {
    setLoading(true);
    try {
      // Fetch favorites from backend
      // Using a reasonable limit for now, pagination can be added later
      const page = await GetFavorites(50, "");

      // Map PostIndex to Post
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

      setFavoritePosts(mapped);
    } catch (e) {
      console.error('Failed to load favorites:', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadFavorites();
  }, []);

  const handleRemoveFavorite = async (postId: string) => {
    try {
      await RemoveFavorite(postId);
      setFavoritePosts(prev => prev.filter(p => p.id !== postId));
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
