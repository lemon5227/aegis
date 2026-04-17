import { Post } from '../types';
import { Profile } from '../types';
import { parsePostContent } from '../lib/postContent';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

interface PostCardProps {
  post: Post;
  authorProfile?: Profile;
  onUpvote: (postId: string) => void;
  onClick: (post: Post) => void;
  onAuthorClick?: (pubkey: string) => void;
  onShare?: (post: Post) => void;
  isRecommended?: boolean;
  isFavorited?: boolean;
  onToggleFavorite?: (postId: string) => void;
}

function formatTimeAgo(timestamp: number): string {
  const now = Date.now();
  const diff = now - timestamp * 1000;
  
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);
  
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 30) return `${days}d ago`;
  return `${Math.floor(days / 30)}mo ago`;
}

function getInitials(name: string): string {
  return name.slice(0, 2).toUpperCase();
}

export function PostCard({ post, authorProfile, onUpvote, onClick, onAuthorClick, onShare, isRecommended, isFavorited, onToggleFavorite }: PostCardProps) {
  const displayName = authorProfile?.displayName || post.pubkey.slice(0, 8);
  const avatarUrl = authorProfile?.avatarURL;
  const parsedContent = parsePostContent(post.bodyPreview, post.bodyPreview);
  const isLinkPost = parsedContent.mode === 'link';
  const isPartialContent = !parsedContent.preview.trim() && !!post.contentCid;

  return (
    <article 
      onClick={() => onClick(post)}
      className="bg-warm-card dark:bg-surface-dark rounded-xl p-5 shadow-soft hover:shadow-md border border-warm-border dark:border-border-dark hover:border-warm-accent/40 dark:hover:border-slate-600 transition-all cursor-pointer group"
    >
      <div className="flex gap-4">
        <div className="flex flex-col items-center gap-1 min-w-[2rem] pt-1">
          <button 
            onClick={(e) => {
              e.stopPropagation();
              onUpvote(post.id);
            }}
            className="text-warm-text-secondary hover:text-warm-accent hover:bg-warm-accent/10 rounded p-1 transition-colors"
          >
            <span className="material-icons-round text-xl">arrow_upward</span>
          </button>
          <span className="text-sm font-bold text-warm-text-primary dark:text-slate-300">
            {post.score || 0}
          </span>
        </div>
        
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            {avatarUrl ? (
              <img 
                className="w-5 h-5 rounded-full object-cover" 
                src={avatarUrl} 
                alt={displayName}
              />
            ) : (
              <div className="w-5 h-5 rounded-full bg-warm-accent flex items-center justify-center text-white text-[10px] font-bold">
                {getInitials(displayName)}
              </div>
            )}
            <button
              onClick={(e) => {
                e.stopPropagation();
                onAuthorClick?.(post.pubkey);
              }}
              className="text-xs font-medium text-warm-text-secondary dark:text-slate-400 hover:underline"
            >
              {displayName}
            </button>
            <span className="text-xs text-warm-text-secondary dark:text-slate-400">•</span>
            <span className="text-xs text-warm-text-secondary dark:text-slate-400">
              {formatTimeAgo(post.timestamp)}
            </span>
            <span className="bg-warm-sidebar dark:bg-surface-lighter text-warm-text-secondary dark:text-slate-400 text-[10px] px-2 py-0.5 rounded-full font-medium ml-2 border border-warm-border dark:border-slate-700">
              #{post.subId}
            </span>
            {isRecommended && (
              <span className="bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400 text-[10px] px-2 py-0.5 rounded-full font-medium border border-yellow-200 dark:border-yellow-800 flex items-center gap-1">
                <span className="material-icons-round text-[10px]">auto_awesome</span>
                Recommended
              </span>
            )}
            {post.isPinned && (
              <span className="bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 text-[10px] px-2 py-0.5 rounded-full font-medium border border-blue-200 dark:border-blue-800 flex items-center gap-1">
                <span className="material-icons-round text-[10px]">keep</span>
                Pinned
              </span>
            )}
            {post.isLocked && (
              <span className="bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 text-[10px] px-2 py-0.5 rounded-full font-medium border border-slate-200 dark:border-slate-700 flex items-center gap-1">
                <span className="material-icons-round text-[10px]">lock</span>
                Locked
              </span>
            )}
            {isPartialContent && (
              <span className="bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 text-[10px] px-2 py-0.5 rounded-full font-medium border border-slate-200 dark:border-slate-700">
                Syncing content
              </span>
            )}
          </div>
          
          <h2 className="text-lg font-bold text-warm-text-primary dark:text-white mb-2 group-hover:text-warm-accent transition-colors">
            {post.title}
          </h2>
          {isLinkPost && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                if (parsedContent.linkURL) {
                  BrowserOpenURL(parsedContent.linkURL);
                }
              }}
              className="mb-3 flex w-full items-center justify-between gap-3 rounded-xl border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark px-3 py-2 text-left hover:border-warm-accent/50"
            >
              <div className="min-w-0">
                <div className="text-[10px] font-semibold uppercase tracking-wide text-warm-text-secondary dark:text-slate-400">
                  External Link
                </div>
                <div className="truncate text-sm font-medium text-warm-text-primary dark:text-white">
                  {parsedContent.linkHostname || parsedContent.linkURL}
                </div>
              </div>
              <span className="material-icons-outlined text-base text-warm-text-secondary dark:text-slate-400">open_in_new</span>
            </button>
          )}
          {parsedContent.preview && (
            <p className="text-sm text-warm-text-secondary dark:text-slate-400 leading-relaxed mb-3 line-clamp-2">
              {parsedContent.preview}
            </p>
          )}
          
          <div className="flex items-center gap-4">
            <button 
              onClick={(e) => e.stopPropagation()}
              className="flex items-center gap-1.5 text-xs font-medium text-warm-text-secondary dark:text-slate-400 hover:bg-warm-sidebar dark:hover:bg-surface-lighter px-2 py-1 rounded transition-colors"
            >
              <span className="material-icons-outlined text-base">chat_bubble_outline</span>
              Comments
            </button>
            <button 
              onClick={(e) => {
                e.stopPropagation();
                onShare?.(post);
              }}
              className="flex items-center gap-1.5 text-xs font-medium text-warm-text-secondary dark:text-slate-400 hover:bg-warm-sidebar dark:hover:bg-surface-lighter px-2 py-1 rounded transition-colors"
            >
              <span className="material-icons-outlined text-base">share</span>
              Share
            </button>
            <button 
              onClick={(e) => {
                e.stopPropagation();
                if (onToggleFavorite) onToggleFavorite(post.id);
              }}
              className={`flex items-center gap-1.5 text-xs font-medium px-2 py-1 rounded transition-colors ${
                isFavorited
                  ? 'text-warm-accent bg-warm-accent/10'
                  : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-sidebar dark:hover:bg-surface-lighter'
              }`}
            >
              <span className="material-icons-outlined text-base">
                {isFavorited ? 'bookmark' : 'bookmark_border'}
              </span>
              {isFavorited ? 'Saved' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </article>
  );
}
