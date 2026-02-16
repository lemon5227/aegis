import { useState, useEffect } from 'react';
import { Post, Profile } from '../types';
import { PostCard } from './PostCard';

interface MyPostsProps {
  currentPubkey: string;
  profiles: Record<string, Profile>;
  onUpvote: (postId: string) => void;
  onPostClick: (post: Post) => void;
}

export function MyPosts({ currentPubkey, profiles, onUpvote, onPostClick }: MyPostsProps) {
  const [myPosts, setMyPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // 从 localStorage 获取我发布的帖子
    const stored = localStorage.getItem('my_posts');
    if (stored) {
      try {
        const posts = JSON.parse(stored);
        setMyPosts(posts.filter((p: Post) => p.pubkey === currentPubkey));
      } catch (e) {
        console.error('Failed to parse my posts:', e);
      }
    }
    setLoading(false);
  }, [currentPubkey]);

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
