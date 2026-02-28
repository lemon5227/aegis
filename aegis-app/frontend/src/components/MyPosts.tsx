import { useState, useEffect } from 'react';
import { Post, Profile } from '../types';
import { PostCard } from './PostCard';
import { GetMyPosts } from '../../wailsjs/go/main/App';

const myPostsCache = new Map<string, Post[]>();

interface MyPostsProps {
  currentPubkey: string;
  refreshToken?: number;
  profiles: Record<string, Profile>;
  onUpvote: (postId: string) => void;
  onPostClick: (post: Post) => void;
}

export function MyPosts({ currentPubkey, refreshToken = 0, profiles, onUpvote, onPostClick }: MyPostsProps) {
  const cacheKey = currentPubkey.trim();
  const [myPosts, setMyPosts] = useState<Post[]>(() => myPostsCache.get(cacheKey) || []);
  const [loading, setLoading] = useState(() => !myPostsCache.has(cacheKey));

  useEffect(() => {
    const hasWailsRuntime = !!(window as any)?.go?.main?.App;
    if (!hasWailsRuntime) {
      setLoading(false);
      return;
    }

    let alive = true;
    const cached = myPostsCache.get(cacheKey);
    if (cached) {
      setMyPosts(cached);
      setLoading(false);
    } else {
      setLoading(true);
    }

    const loadMyPosts = async () => {
      try {
        const page = await GetMyPosts(100, '');
        const mapped: Post[] = (page.items || []).map((item: any) => ({
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
        if (!alive) return;
        myPostsCache.set(cacheKey, mapped);
        setMyPosts(mapped);
      } catch (e) {
        if (!alive) return;
        console.error('Failed to load my posts:', e);
        setMyPosts(cached || []);
      } finally {
        if (!alive) return;
        setLoading(false);
      }
    };

    void loadMyPosts();
    return () => {
      alive = false;
    };
  }, [cacheKey, refreshToken]);

  if (loading) {
    return (
      <div className="flex-1 overflow-y-auto p-4 md:p-6">
        <div className="max-w-2xl mx-auto">
          <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white mb-6">My Posts</h1>
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
        <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white mb-6">My Posts</h1>
        
        {myPosts.length === 0 ? (
          <div className="text-center py-12 text-warm-text-secondary dark:text-slate-400">
            <span className="material-icons text-4xl mb-4">article</span>
            <p>You haven't posted anything yet.</p>
            <p className="text-sm mt-2">Create your first post to see it here!</p>
          </div>
        ) : (
          <div className="space-y-4">
            {myPosts.map((post) => (
              <PostCard
                key={post.id}
                post={post}
                authorProfile={profiles[post.pubkey]}
                onUpvote={onUpvote}
                onClick={onPostClick}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
