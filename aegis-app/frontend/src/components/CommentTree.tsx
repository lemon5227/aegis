import { useState } from 'react';
import { Comment, Profile } from '../types';

interface CommentItemProps {
  comment: Comment;
  profiles: Record<string, Profile>;
  onReply: (parentId: string) => void;
  onUpvote: (commentId: string) => void;
  depth?: number;
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

export function CommentItem({ comment, profiles, onReply, onUpvote, depth = 0 }: CommentItemProps) {
  const profile = profiles[comment.pubkey];
  const displayName = profile?.displayName || comment.pubkey.slice(0, 8);
  const avatarUrl = profile?.avatarURL;

  return (
    <div className={`${depth > 0 ? 'ml-5 md:ml-10 mt-2 relative pl-6 border-l-2 border-warm-border dark:border-border-dark' : 'mb-6 relative'}`}>
      <div className="flex gap-3 relative group z-10">
        <div className="flex-shrink-0">
          {avatarUrl ? (
            <img 
              className="w-10 h-10 rounded-full object-cover" 
              src={avatarUrl} 
              alt={displayName}
            />
          ) : (
            <div className="w-10 h-10 rounded-full bg-warm-accent flex items-center justify-center text-white font-bold text-sm">
              {getInitials(displayName)}
            </div>
          )}
        </div>
        
        <div className="flex-1">
          <div className="bg-warm-surface dark:bg-surface-dark border border-warm-border/40 dark:border-border-dark/40 rounded-xl p-3 md:p-4 hover:border-warm-border dark:hover:border-border-dark transition-colors shadow-sm">
            <div className="flex items-center justify-between mb-1">
              <div className="flex items-center gap-2">
                <span className="font-bold text-sm text-warm-text-primary dark:text-white">
                  @{displayName}
                </span>
                <span className="text-xs text-warm-text-secondary dark:text-slate-400">
                  {formatTimeAgo(comment.timestamp)}
                </span>
              </div>
            </div>
            <p className="text-sm text-warm-text-secondary dark:text-slate-300 mb-3">
              {comment.body}
            </p>
            <div className="flex items-center gap-4">
              <button 
                onClick={() => onUpvote(comment.id)}
                className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors"
              >
                <span className="material-icons text-base">thumb_up_alt</span>
                <span className="text-xs font-medium">{comment.score || 0}</span>
              </button>
              <button 
                onClick={() => onReply(comment.id)}
                className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors"
              >
                <span className="material-icons text-base">chat_bubble_outline</span>
                <span className="text-xs font-medium">Reply</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

interface CommentTreeProps {
  comments: Comment[];
  profiles: Record<string, Profile>;
  onReply: (parentId: string) => void;
  onUpvote: (commentId: string) => void;
}

export function CommentTree({ comments, profiles, onReply, onUpvote }: CommentTreeProps) {
  const rootComments = comments.filter(c => !c.parentId || c.parentId === '');
  
  const renderComment = (comment: Comment, depth: number = 0) => {
    const children = comments.filter(c => c.parentId === comment.id);
    
    return (
      <div key={comment.id}>
        <CommentItem
          comment={comment}
          profiles={profiles}
          onReply={onReply}
          onUpvote={onUpvote}
          depth={depth}
        />
        {children.map(child => renderComment(child, depth + 1))}
      </div>
    );
  };

  if (comments.length === 0) {
    return (
      <div className="text-center py-8 text-warm-text-secondary dark:text-slate-400">
        <span className="material-icons text-4xl mb-2">chat_bubble_outline</span>
        <p>No comments yet. Be the first to comment!</p>
      </div>
    );
  }

  return (
    <div>
      {rootComments.map(comment => renderComment(comment))}
    </div>
  );
}
