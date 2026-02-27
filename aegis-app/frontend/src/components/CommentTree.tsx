import { useEffect, useState } from 'react';
import { Comment, Profile } from '../types';
import { GetMediaByCID } from '../../wailsjs/go/main/App';

interface CommentItemProps {
  comment: Comment;
  profiles: Record<string, Profile>;
  onReply: (parentId: string) => void;
  onUpvote: (commentId: string) => void;
  onDownvote?: (commentId: string) => void;
  onImageClick: (src: string) => void;
  currentPubkey?: string;
  onDelete?: (commentId: string) => Promise<void> | void;
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

function renderRichText(content: string, onImageClick: (src: string) => void) {
  const text = content || '';
  const lines = text.split('\n');
  return lines.map((line, index) => {
    const imageMatch = line.trim().match(/^!\[[^\]]*\]\(([^)]+)\)$/);
    if (imageMatch && imageMatch[1]) {
      const src = imageMatch[1].trim();
      return (
        <img
          key={`img-${index}`}
          src={src}
          alt="comment image"
          className="max-h-64 w-auto rounded-lg border border-warm-border dark:border-border-dark cursor-zoom-in"
          onClick={() => onImageClick(src)}
        />
      );
    }

    return (
      <p key={`txt-${index}`} className="whitespace-pre-wrap break-words">
        {line}
      </p>
    );
  });
}

export function CommentItem({ comment, profiles, onReply, onUpvote, onDownvote, onImageClick, currentPubkey, onDelete, depth = 0 }: CommentItemProps) {
  const profile = profiles[comment.pubkey];
  const displayName = profile?.displayName || comment.pubkey.slice(0, 8);
  const avatarUrl = profile?.avatarURL;
  const canDelete = !!currentPubkey && currentPubkey === comment.pubkey;
  const [resolvedAttachmentImages, setResolvedAttachmentImages] = useState<string[]>([]);
  const [deleteArmed, setDeleteArmed] = useState(false);
  const attachmentKey = (comment.attachments || [])
    .map((item) => `${item.kind}:${item.ref}`)
    .join('|');

  useEffect(() => {
    let alive = true;

    const resolveMediaCIDWithRetry = async (cid: string): Promise<string | null> => {
      const attempts = 3;
      for (let attempt = 1; attempt <= attempts; attempt += 1) {
        try {
          const media = await GetMediaByCID(cid);
          if (media?.dataBase64 && media?.mime) {
            return `data:${media.mime};base64,${media.dataBase64}`;
          }
        } catch {
          // retry when remote media arrives slightly later
        }
        if (attempt < attempts) {
          await new Promise((resolve) => window.setTimeout(resolve, 300 * attempt));
        }
      }
      return null;
    };

    const run = async () => {
      const items = comment.attachments || [];
      if (items.length === 0) {
        if (alive) setResolvedAttachmentImages([]);
        return;
      }

      const output: string[] = [];
      for (const item of items) {
        if (item.kind === 'external_url' && item.ref) {
          output.push(item.ref);
          continue;
        }
        if (item.kind === 'media_cid' && item.ref) {
          const resolved = await resolveMediaCIDWithRetry(item.ref);
          if (resolved) {
            output.push(resolved);
          }
        }
      }
      if (alive) {
        setResolvedAttachmentImages(output);
      }
    };
    void run();
    return () => {
      alive = false;
    };
  }, [attachmentKey]);

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
            <div className="text-sm text-warm-text-secondary dark:text-slate-300 mb-3 space-y-2">
              {renderRichText(comment.body, onImageClick)}
            </div>
            {resolvedAttachmentImages.length > 0 && (
              <div className="mb-3 flex flex-wrap gap-2">
                {resolvedAttachmentImages.map((src, index) => (
                  <img
                    key={`${src.slice(0, 30)}-${index}`}
                    src={src}
                    alt="comment attachment"
                    className="max-h-64 w-auto rounded-lg border border-warm-border dark:border-border-dark cursor-zoom-in"
                    onClick={() => onImageClick(src)}
                  />
                ))}
              </div>
            )}
            <div className="flex items-center gap-4">
              <button 
                onClick={() => onUpvote(comment.id)}
                className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors"
              >
                <span className="material-icons text-base">thumb_up_alt</span>
                <span className="text-xs font-medium">{comment.score || 0}</span>
              </button>
              <button
                onClick={() => onDownvote?.(comment.id)}
                className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-red-500 transition-colors"
              >
                <span className="material-icons text-base">thumb_down_alt</span>
              </button>
              <button 
                onClick={() => onReply(comment.id)}
                className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors"
              >
                <span className="material-icons text-base">chat_bubble_outline</span>
                <span className="text-xs font-medium">Reply</span>
              </button>
              {canDelete && onDelete && (
                <button
                  onClick={() => {
                    setDeleteArmed(true);
                  }}
                  className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-red-500 transition-colors"
                >
                  <span className="material-icons text-base">delete</span>
                  <span className="text-xs font-medium">Delete</span>
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
      {canDelete && onDelete && deleteArmed && (
        <div className="fixed inset-0 z-[95] flex items-center justify-center bg-black/60 backdrop-blur-sm px-4 transition-all duration-300" onClick={() => setDeleteArmed(false)}>
          <div
            className="w-full max-w-sm rounded-2xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark backdrop-blur-md p-6 shadow-2xl flex flex-col items-center text-center transform scale-100 transition-all"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4">
              <span className="material-icons text-warm-accent text-5xl">warning_amber</span>
            </div>
            <h4 className="text-lg font-bold text-warm-text-primary dark:text-white mb-2">Delete Comment</h4>
            <p className="text-sm text-warm-text-secondary dark:text-slate-300 mb-6">
              This action cannot be undone. Are you sure you want to permanently delete this comment?
            </p>
            <div className="flex items-center gap-3 w-full">
              <button
                onClick={() => setDeleteArmed(false)}
                className="flex-1 py-2.5 rounded-xl font-bold border border-warm-border dark:border-border-dark text-warm-text-secondary dark:text-slate-300 hover:text-warm-text-primary dark:hover:text-white hover:bg-warm-bg dark:hover:bg-surface-lighter transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  try {
                    await onDelete(comment.id);
                    setDeleteArmed(false);
                  } catch (error) {
                    console.error('Failed to delete comment:', error);
                  }
                }}
                className="flex-1 py-2.5 rounded-xl font-bold bg-warm-accent hover:bg-warm-accent-hover text-white shadow-lg transition-all"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface CommentTreeProps {
  comments: Comment[];
  profiles: Record<string, Profile>;
  onReply: (parentId: string) => void;
  onUpvote: (commentId: string) => void;
  onDownvote?: (commentId: string) => void;
  currentPubkey?: string;
  onDelete?: (commentId: string) => Promise<void> | void;
  onImageClick?: (src: string) => void;
}

export function CommentTree({ comments, profiles, onReply, onUpvote, onDownvote, currentPubkey, onDelete, onImageClick }: CommentTreeProps) {
  const [previewImageSrc, setPreviewImageSrc] = useState<string | null>(null);

  const handleImageClick = (src: string) => {
    if (onImageClick) {
      onImageClick(src);
      return;
    }
    setPreviewImageSrc(src);
  };

  const commentIdSet = new Set(comments.map((c) => c.id));
  const rootComments = comments.filter((c) => !c.parentId || c.parentId === '' || !commentIdSet.has(c.parentId));

  const renderComment = (comment: Comment, depth: number = 0) => {
    const children = comments.filter(c => c.parentId === comment.id);

    return (
      <div key={comment.id}>
        <CommentItem
          comment={comment}
          profiles={profiles}
          onReply={onReply}
          onUpvote={onUpvote}
          onDownvote={onDownvote}
          onImageClick={handleImageClick}
          currentPubkey={currentPubkey}
          onDelete={onDelete}
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
      {previewImageSrc && (
        <div className="fixed inset-0 z-[90] bg-black/80 flex items-center justify-center p-4" onClick={() => setPreviewImageSrc(null)}>
          <img
            src={previewImageSrc}
            alt="Comment image preview"
            className="max-w-full max-h-full rounded-lg border border-white/20"
            onClick={(e) => e.stopPropagation()}
          />
          <button
            onClick={() => setPreviewImageSrc(null)}
            className="absolute top-4 right-4 text-white/90 hover:text-white"
            title="Close preview"
          >
            <span className="material-icons">close</span>
          </button>
        </div>
      )}
    </div>
  );
}
import { useEffect, useState } from 'react';
import { Comment, Profile } from '../types';
import { GetMediaByCID } from '../../wailsjs/go/main/App';

interface CommentItemProps {
  comment: Comment;
  profiles: Record<string, Profile>;
  onReply: (parentId: string) => void;
  onUpvote: (commentId: string) => void;
  onDownvote?: (commentId: string) => void;
  onImageClick: (src: string) => void;
  currentPubkey?: string;
  onDelete?: (commentId: string) => Promise<void> | void;
  onEdit?: (commentId: string, body: string) => Promise<void>;
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

function renderRichText(content: string, onImageClick: (src: string) => void) {
  const text = content || '';
  const lines = text.split('\n');
  return lines.map((line, index) => {
    const imageMatch = line.trim().match(/^!\[[^\]]*\]\(([^)]+)\)$/);
    if (imageMatch && imageMatch[1]) {
      const src = imageMatch[1].trim();
      return (
        <img
          key={`img-${index}`}
          src={src}
          alt="comment image"
          className="max-h-64 w-auto rounded-lg border border-warm-border dark:border-border-dark cursor-zoom-in"
          onClick={() => onImageClick(src)}
        />
      );
    }

    return (
      <p key={`txt-${index}`} className="whitespace-pre-wrap break-words">
        {line}
      </p>
    );
  });
}

export function CommentItem({ comment, profiles, onReply, onUpvote, onDownvote, onImageClick, currentPubkey, onDelete, onEdit, depth = 0 }: CommentItemProps) {
  const profile = profiles[comment.pubkey];
  const displayName = profile?.displayName || comment.pubkey.slice(0, 8);
  const avatarUrl = profile?.avatarURL;
  const canDelete = !!currentPubkey && currentPubkey === comment.pubkey;
  const canEdit = canDelete; // Same permission
  const [resolvedAttachmentImages, setResolvedAttachmentImages] = useState<string[]>([]);
  const [deleteArmed, setDeleteArmed] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editBody, setEditBody] = useState(comment.body);
  const [editBusy, setEditBusy] = useState(false);

  const attachmentKey = (comment.attachments || [])
    .map((item) => `${item.kind}:${item.ref}`)
    .join('|');

  useEffect(() => {
    setEditBody(comment.body);
  }, [comment.body]);

  useEffect(() => {
    let alive = true;

    const resolveMediaCIDWithRetry = async (cid: string): Promise<string | null> => {
      const attempts = 3;
      for (let attempt = 1; attempt <= attempts; attempt += 1) {
        try {
          const media = await GetMediaByCID(cid);
          if (media?.dataBase64 && media?.mime) {
            return `data:${media.mime};base64,${media.dataBase64}`;
          }
        } catch {
          // retry when remote media arrives slightly later
        }
        if (attempt < attempts) {
          await new Promise((resolve) => window.setTimeout(resolve, 300 * attempt));
        }
      }
      return null;
    };

    const run = async () => {
      const items = comment.attachments || [];
      if (items.length === 0) {
        if (alive) setResolvedAttachmentImages([]);
        return;
      }

      const output: string[] = [];
      for (const item of items) {
        if (item.kind === 'external_url' && item.ref) {
          output.push(item.ref);
          continue;
        }
        if (item.kind === 'media_cid' && item.ref) {
          const resolved = await resolveMediaCIDWithRetry(item.ref);
          if (resolved) {
            output.push(resolved);
          }
        }
      }
      if (alive) {
        setResolvedAttachmentImages(output);
      }
    };
    void run();
    return () => {
      alive = false;
    };
  }, [attachmentKey]);

  const handleSaveEdit = async () => {
    if (!onEdit || editBusy) return;
    setEditBusy(true);
    try {
      await onEdit(comment.id, editBody);
      setIsEditing(false);
    } catch (e: any) {
      console.error('Failed to edit comment:', e);
      alert('Failed to save comment changes: ' + e.message);
    } finally {
      setEditBusy(false);
    }
  };

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

            {isEditing ? (
              <div className="mb-3 space-y-2">
                <textarea
                  value={editBody}
                  onChange={(e) => setEditBody(e.target.value)}
                  className="w-full bg-warm-card dark:bg-surface-lighter border border-warm-border dark:border-border-dark rounded-lg p-2 text-sm text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent resize-y"
                  rows={3}
                />
                <div className="flex justify-end gap-2">
                  <button
                    onClick={() => {
                      setIsEditing(false);
                      setEditBody(comment.body);
                    }}
                    disabled={editBusy}
                    className="px-3 py-1 text-xs font-medium text-warm-text-secondary hover:text-warm-text-primary bg-warm-sidebar dark:bg-surface-lighter hover:bg-warm-border dark:hover:bg-border-dark rounded transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleSaveEdit}
                    disabled={editBusy}
                    className="px-3 py-1 text-xs font-medium text-white bg-warm-accent hover:bg-warm-accent-hover rounded transition-colors disabled:opacity-50"
                  >
                    {editBusy ? 'Saving...' : 'Save'}
                  </button>
                </div>
              </div>
            ) : (
              <div className="text-sm text-warm-text-secondary dark:text-slate-300 mb-3 space-y-2">
                {renderRichText(comment.body, onImageClick)}
              </div>
            )}

            {resolvedAttachmentImages.length > 0 && (
              <div className="mb-3 flex flex-wrap gap-2">
                {resolvedAttachmentImages.map((src, index) => (
                  <img
                    key={`${src.slice(0, 30)}-${index}`}
                    src={src}
                    alt="comment attachment"
                    className="max-h-64 w-auto rounded-lg border border-warm-border dark:border-border-dark cursor-zoom-in"
                    onClick={() => onImageClick(src)}
                  />
                ))}
              </div>
            )}
            <div className="flex items-center gap-4">
              <button
                onClick={() => onUpvote(comment.id)}
                className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors"
              >
                <span className="material-icons text-base">thumb_up_alt</span>
                <span className="text-xs font-medium">{comment.score || 0}</span>
              </button>
              <button
                onClick={() => onDownvote?.(comment.id)}
                className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-red-500 transition-colors"
              >
                <span className="material-icons text-base">thumb_down_alt</span>
              </button>
              <button
                onClick={() => onReply(comment.id)}
                className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors"
              >
                <span className="material-icons text-base">chat_bubble_outline</span>
                <span className="text-xs font-medium">Reply</span>
              </button>
              {canEdit && onEdit && !isEditing && (
                 <button
                  onClick={() => setIsEditing(true)}
                  className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-warm-primary transition-colors"
                >
                  <span className="material-icons text-base">edit</span>
                  <span className="text-xs font-medium">Edit</span>
                </button>
              )}
              {canDelete && onDelete && (
                <button
                  onClick={() => {
                    setDeleteArmed(true);
                  }}
                  className="flex items-center gap-1 text-warm-text-secondary dark:text-slate-400 hover:text-red-500 transition-colors"
                >
                  <span className="material-icons text-base">delete</span>
                  <span className="text-xs font-medium">Delete</span>
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
      {canDelete && onDelete && deleteArmed && (
        <div className="fixed inset-0 z-[95] flex items-center justify-center bg-black/60 backdrop-blur-sm px-4 transition-all duration-300" onClick={() => setDeleteArmed(false)}>
          <div
            className="w-full max-w-sm rounded-2xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark backdrop-blur-md p-6 shadow-2xl flex flex-col items-center text-center transform scale-100 transition-all"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4">
              <span className="material-icons text-warm-accent text-5xl">warning_amber</span>
            </div>
            <h4 className="text-lg font-bold text-warm-text-primary dark:text-white mb-2">Delete Comment</h4>
            <p className="text-sm text-warm-text-secondary dark:text-slate-300 mb-6">
              This action cannot be undone. Are you sure you want to permanently delete this comment?
            </p>
            <div className="flex items-center gap-3 w-full">
              <button
                onClick={() => setDeleteArmed(false)}
                className="flex-1 py-2.5 rounded-xl font-bold border border-warm-border dark:border-border-dark text-warm-text-secondary dark:text-slate-300 hover:text-warm-text-primary dark:hover:text-white hover:bg-warm-bg dark:hover:bg-surface-lighter transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  try {
                    await onDelete(comment.id);
                    setDeleteArmed(false);
                  } catch (error) {
                    console.error('Failed to delete comment:', error);
                  }
                }}
                className="flex-1 py-2.5 rounded-xl font-bold bg-warm-accent hover:bg-warm-accent-hover text-white shadow-lg transition-all"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface CommentTreeProps {
  comments: Comment[];
  profiles: Record<string, Profile>;
  onReply: (parentId: string) => void;
  onUpvote: (commentId: string) => void;
  onDownvote?: (commentId: string) => void;
  currentPubkey?: string;
  onDelete?: (commentId: string) => Promise<void> | void;
  onEdit?: (commentId: string, body: string) => Promise<void>;
  onImageClick?: (src: string) => void;
}

export function CommentTree({ comments, profiles, onReply, onUpvote, onDownvote, currentPubkey, onDelete, onEdit, onImageClick }: CommentTreeProps) {
  const [previewImageSrc, setPreviewImageSrc] = useState<string | null>(null);

  const handleImageClick = (src: string) => {
    if (onImageClick) {
      onImageClick(src);
      return;
    }
    setPreviewImageSrc(src);
  };

  const commentIdSet = new Set(comments.map((c) => c.id));
  const rootComments = comments.filter((c) => !c.parentId || c.parentId === '' || !commentIdSet.has(c.parentId));

  const renderComment = (comment: Comment, depth: number = 0) => {
    const children = comments.filter(c => c.parentId === comment.id);

    return (
      <div key={comment.id}>
        <CommentItem
          comment={comment}
          profiles={profiles}
          onReply={onReply}
          onUpvote={onUpvote}
          onDownvote={onDownvote}
          onImageClick={handleImageClick}
          currentPubkey={currentPubkey}
          onDelete={onDelete}
          onEdit={onEdit}
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
      {previewImageSrc && (
        <div className="fixed inset-0 z-[90] bg-black/80 flex items-center justify-center p-4" onClick={() => setPreviewImageSrc(null)}>
          <img
            src={previewImageSrc}
            alt="Comment image preview"
            className="max-w-full max-h-full rounded-lg border border-white/20"
            onClick={(e) => e.stopPropagation()}
          />
          <button
            onClick={() => setPreviewImageSrc(null)}
            className="absolute top-4 right-4 text-white/90 hover:text-white"
            title="Close preview"
          >
            <span className="material-icons">close</span>
          </button>
        </div>
      )}
    </div>
  );
}
