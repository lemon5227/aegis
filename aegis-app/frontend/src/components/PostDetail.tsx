import { useMemo, useRef, useState } from 'react';
import { Post, Profile, Comment } from '../types';
import { CommentTree } from './CommentTree';

interface PostDetailProps {
  post: Post;
  body?: string;
  comments: Comment[];
  profiles: Record<string, Profile>;
  currentPubkey?: string;
  onBack: () => void;
  onUpvote: (postId: string) => void;
  onReply: (parentId: string, body: string) => Promise<void> | void;
  onCommentUpvote: (commentId: string) => void;
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

export function PostDetail({ 
  post, 
  body, 
  comments,
  profiles,
  currentPubkey,
  onBack, 
  onUpvote,
  onReply,
  onCommentUpvote 
}: PostDetailProps) {
  const [replyContent, setReplyContent] = useState('');
  const [replyToId, setReplyToId] = useState<string | null>(null);
  const [replyBusy, setReplyBusy] = useState(false);
  const [replyMessage, setReplyMessage] = useState('');
  const [commentSort, setCommentSort] = useState<'best' | 'newest' | 'controversial'>('best');
  const replyInputRef = useRef<HTMLTextAreaElement | null>(null);

  const authorProfile = profiles[post.pubkey];
  const authorName = authorProfile?.displayName || post.pubkey.slice(0, 8);
  const authorAvatar = authorProfile?.avatarURL;

  const replyingToComment = useMemo(() => {
    if (!replyToId) return null;
    return comments.find((comment) => comment.id === replyToId) || null;
  }, [comments, replyToId]);

  const focusReplyInput = () => {
    window.setTimeout(() => {
      replyInputRef.current?.focus();
    }, 0);
  };

  const handleSubmitReply = async () => {
    if (!replyContent.trim()) return;
    setReplyBusy(true);
    setReplyMessage('');
    try {
      await onReply(replyToId || '', replyContent.trim());
      setReplyContent('');
      setReplyToId(null);
      setReplyMessage('Comment posted.');
    } catch (error) {
      console.error('Failed to post comment:', error);
      setReplyMessage('Failed to post comment. Please retry.');
    } finally {
      setReplyBusy(false);
    }
  };

  const handleInsertCodeBlock = () => {
    const addition = replyContent.trim() ? '\n\n```\n\n```' : '```\n\n```';
    setReplyContent((prev) => `${prev}${addition}`);
    focusReplyInput();
  };

  const handleInsertImageTemplate = () => {
    const addition = replyContent.trim() ? '\n\n![image](https://)' : '![image](https://)';
    setReplyContent((prev) => `${prev}${addition}`);
    focusReplyInput();
  };

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-8">
      <div className="max-w-4xl mx-auto pb-20">
        <article className="bg-warm-surface dark:bg-surface-dark rounded-2xl shadow-soft border border-warm-border/60 dark:border-border-dark/60 overflow-hidden mb-8">
          <div className="p-6 md:p-8 pb-4">
            <div className="flex items-center justify-between mb-6">
              <div className="flex items-center gap-3">
                {authorAvatar ? (
                  <img 
                    className="w-12 h-12 rounded-full ring-2 ring-warm-accent/20" 
                    src={authorAvatar} 
                    alt={authorName}
                  />
                ) : (
                  <div className="w-12 h-12 rounded-full bg-warm-accent flex items-center justify-center text-white font-bold ring-2 ring-warm-accent/20">
                    {getInitials(authorName)}
                  </div>
                )}
                <div>
                  <div className="flex items-center gap-2">
                    <h4 className="font-bold text-warm-text-primary dark:text-white">
                      @{authorName}
                    </h4>
                  </div>
                  <p className="text-xs text-warm-text-secondary dark:text-slate-400">
                    Posted {formatTimeAgo(post.timestamp)}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button className="p-2 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent hover:bg-warm-accent/10 rounded-full transition-colors">
                  <span className="material-icons">bookmark_border</span>
                </button>
                <button className="p-2 text-warm-text-secondary dark:text-slate-400 hover:text-warm-text-primary dark:hover:text-white hover:bg-warm-bg dark:hover:bg-surface-lighter rounded-full transition-colors">
                  <span className="material-icons">more_horiz</span>
                </button>
              </div>
            </div>
            
            <h1 className="text-2xl md:text-3xl font-bold text-warm-text-primary dark:text-white mb-6 leading-tight">
              {post.title}
            </h1>
            
            <div className="prose max-w-none text-warm-text-secondary dark:text-slate-300 leading-relaxed space-y-4">
              {body || post.bodyPreview || 'No content'}
            </div>
          </div>
          
          <div className="bg-warm-bg/40 dark:bg-background-dark/40 px-6 py-4 border-t border-warm-border/60 dark:border-border-dark/60 flex items-center justify-between">
            <div className="flex items-center gap-1 bg-warm-surface dark:bg-surface-dark border border-warm-border dark:border-border-dark rounded-lg p-1">
              <button 
                onClick={() => onUpvote(post.id)}
                className="p-1 px-2 rounded hover:bg-warm-bg dark:hover:bg-surface-lighter text-warm-accent transition-colors flex items-center gap-1"
              >
                <span className="material-icons text-lg">arrow_upward</span>
                <span className="text-sm font-bold">{post.score || 0}</span>
              </button>
              <div className="w-px h-4 bg-warm-border dark:bg-border-dark"></div>
              <button className="p-1 px-2 rounded hover:bg-warm-bg dark:hover:bg-surface-lighter text-warm-text-secondary dark:text-slate-400 hover:text-red-500 transition-colors">
                <span className="material-icons text-lg">arrow_downward</span>
              </button>
            </div>
            
            <div className="flex gap-3">
              <button
                onClick={() => {
                  setReplyToId(null);
                  setReplyMessage('');
                  focusReplyInput();
                }}
                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-warm-accent/10 hover:bg-warm-accent/20 text-warm-accent transition-colors"
              >
                <span className="material-icons text-lg">reply</span>
                <span className="text-sm font-medium">Reply</span>
              </button>
              <button className="flex items-center gap-2 px-4 py-2 rounded-lg hover:bg-warm-bg dark:hover:bg-surface-lighter text-warm-text-secondary dark:text-slate-400 transition-colors">
                <span className="material-icons text-lg">share</span>
                <span className="text-sm font-medium">Share</span>
              </button>
            </div>
          </div>
        </article>
        
        <div className="mt-8">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-xl font-bold text-warm-text-primary dark:text-white">
              Discussion <span className="text-warm-text-secondary dark:text-slate-400 text-base font-normal">({comments.length} comments)</span>
            </h3>
            <div className="flex items-center gap-2">
              <span className="text-xs text-warm-text-secondary dark:text-slate-400 font-medium">Sort by:</span>
              <select 
                value={commentSort}
                onChange={(e) => setCommentSort(e.target.value as any)}
                className="bg-transparent border-none text-sm font-bold text-warm-text-primary dark:text-white focus:ring-0 cursor-pointer p-0 pr-6"
              >
                <option value="best">Best</option>
                <option value="newest">Newest</option>
                <option value="controversial">Controversial</option>
              </select>
            </div>
          </div>
          
          <div className="flex gap-4 mb-10">
            <div className="w-10 h-10 rounded-full bg-warm-accent flex items-center justify-center text-white font-bold shrink-0">
              ?
            </div>
            <div className="flex-1 relative">
              {replyingToComment && (
                <div className="mb-2 flex items-center justify-between rounded-lg bg-warm-accent/10 px-3 py-2 text-xs text-warm-text-secondary dark:text-slate-300">
                  <span>
                    Replying to <strong>@{(profiles[replyingToComment.pubkey]?.displayName || replyingToComment.pubkey.slice(0, 8))}</strong>
                  </span>
                  <button
                    onClick={() => setReplyToId(null)}
                    className="text-warm-accent hover:underline"
                  >
                    Cancel
                  </button>
                </div>
              )}
              <textarea 
                ref={replyInputRef}
                value={replyContent}
                onChange={(e) => setReplyContent(e.target.value)}
                className="w-full bg-warm-surface dark:bg-surface-dark border border-warm-border dark:border-border-dark rounded-xl p-4 text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent placeholder-warm-text-secondary/40 dark:placeholder-slate-400/40 resize-none shadow-soft" 
                placeholder="What are your thoughts?" 
                rows={3}
              />
              <div className="mt-2 flex items-center justify-end gap-2">
                <button
                  onClick={handleInsertImageTemplate}
                  className="p-1.5 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors rounded"
                >
                  <span className="material-icons text-lg">image</span>
                </button>
                <button
                  onClick={handleInsertCodeBlock}
                  className="p-1.5 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors rounded"
                >
                  <span className="material-icons text-lg">code</span>
                </button>
                <button 
                  onClick={handleSubmitReply}
                  disabled={!replyContent.trim() || replyBusy}
                  className="bg-warm-accent hover:bg-warm-accent-hover text-white px-4 py-1.5 rounded-lg text-sm font-medium shadow-md shadow-warm-accent/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {replyBusy ? 'Posting...' : 'Comment'}
                </button>
              </div>
              <div className="mt-1 min-h-[18px] text-right text-xs text-warm-text-secondary dark:text-slate-400">
                {replyMessage}
              </div>
            </div>
          </div>
          
          <CommentTree
            comments={comments}
            profiles={profiles}
            onReply={(parentId) => {
              setReplyToId(parentId);
              setReplyMessage('');
              focusReplyInput();
            }}
            onUpvote={onCommentUpvote}
          />
        </div>
      </div>
    </div>
  );
}
