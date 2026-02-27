import { useState, useRef, useEffect } from 'react';
import { Post, Comment, Profile } from '../types';
import { CommentTree } from './CommentTree';
import { StoreCommentImageDataURL } from '../../wailsjs/go/main/App';

interface PostDetailProps {
  post: Post & { isFavorited?: boolean };
  body: string;
  comments: Comment[];
  profiles: Record<string, Profile>;
  currentPubkey?: string;
  onBack: () => void;
  onUpvote: (postId: string) => void;
  onDownvote: (postId: string) => void;
  onReply: (parentId: string, body: string, localImageDataURLs?: string[], externalImageURLs?: string[]) => Promise<void>;
  onCommentUpvote: (commentId: string) => void;
  onCommentDownvote: (commentId: string) => void;
  onDeletePost: (postId: string) => Promise<void>;
  onDeleteComment: (commentId: string) => Promise<void>;
  onViewOperationTimeline: (entityType: 'post' | 'comment', entityId: string) => void;
  isDevMode?: boolean;
  onToggleFavorite?: (postId: string) => void;
}

function formatTimeAgo(timestamp: number): string {
  const now = Date.now();
  const diff = now - timestamp;

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

function linkifyAndMarkdown(text: string): React.ReactNode {
  if (!text) return null;
  const parts = text.split(/(!\[.*?\]\(.*?\)|```[\s\S]*?```|`[^`]+`)/g);
  return parts.map((part, i) => {
    if (part.startsWith('![') && part.includes('](') && part.endsWith(')')) {
      const altMatch = part.match(/^!\[(.*?)\]/);
      const urlMatch = part.match(/\((.*?)\)$/);
      if (altMatch && urlMatch) {
        return (
          <img
            key={i}
            src={urlMatch[1]}
            alt={altMatch[1]}
            className="max-w-full h-auto rounded-lg my-2 border border-warm-border dark:border-border-dark"
            loading="lazy"
          />
        );
      }
    }
    if (part.startsWith('```') && part.endsWith('```')) {
      const code = part.slice(3, -3);
      return (
        <pre key={i} className="bg-warm-sidebar dark:bg-surface-lighter p-3 rounded-lg overflow-x-auto my-2 border border-warm-border dark:border-border-dark text-xs font-mono">
          <code>{code}</code>
        </pre>
      );
    }
    if (part.startsWith('`') && part.endsWith('`')) {
      return (
        <code key={i} className="bg-warm-sidebar dark:bg-surface-lighter px-1.5 py-0.5 rounded text-xs font-mono border border-warm-border dark:border-border-dark">
          {part.slice(1, -1)}
        </code>
      );
    }
    return part;
  });
}

export function PostDetail({
  post,
  body,
  comments,
  profiles,
  currentPubkey,
  onBack,
  onUpvote,
  onDownvote,
  onReply,
  onCommentUpvote,
  onCommentDownvote,
  onDeletePost,
  onDeleteComment,
  onViewOperationTimeline,
  isDevMode,
  onToggleFavorite,
}: PostDetailProps) {
  const [replyContent, setReplyContent] = useState('');
  const [replyToId, setReplyToId] = useState<string | null>(null);
  const [replyBusy, setReplyBusy] = useState(false);
  const [replyMessage, setReplyMessage] = useState('');
  const replyInputRef = useRef<HTMLTextAreaElement>(null);
  const [pendingLocalImages, setPendingLocalImages] = useState<string[]>([]);
  const [pendingExternalImages, setPendingExternalImages] = useState<string[]>([]);
  const imageFileInputRef = useRef<HTMLInputElement>(null);
  const [imageInsertBusy, setImageInsertBusy] = useState(false);
  const [previewImageSrc, setPreviewImageSrc] = useState<string | null>(null);
  const [deletePostArmed, setDeletePostArmed] = useState(false);

  const authorProfile = profiles[post.pubkey];
  const displayName = authorProfile?.displayName || post.pubkey.slice(0, 8);
  const avatarUrl = authorProfile?.avatarURL;
  const canDeletePost = currentPubkey && currentPubkey === post.pubkey;

  const replyingToComment = replyToId ? comments.find((c) => c.id === replyToId) : null;

  useEffect(() => {
    if (replyToId && replyInputRef.current) {
      replyInputRef.current.focus();
      // Scroll to input
      replyInputRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }, [replyToId]);

  const handleSubmitReply = async () => {
    if ((!replyContent.trim() && pendingLocalImages.length === 0 && pendingExternalImages.length === 0) || replyBusy) return;
    setReplyBusy(true);
    setReplyMessage('');
    try {
      await onReply(
        replyToId || '',
        replyContent,
        pendingLocalImages.length > 0 ? pendingLocalImages : undefined,
        pendingExternalImages.length > 0 ? pendingExternalImages : undefined
      );
      setReplyContent('');
      setReplyToId(null);
      setPendingLocalImages([]);
      setPendingExternalImages([]);
      setReplyMessage('Reply posted!');
      setTimeout(() => setReplyMessage(''), 3000);
    } catch (e: any) {
      console.error('Reply failed:', e);
      setReplyMessage('Failed to post reply.');
    } finally {
      setReplyBusy(false);
    }
  };

  const handleSelectLocalImage = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 2 * 1024 * 1024) {
      setReplyMessage('Image too large (max 2MB)');
      setTimeout(() => setReplyMessage(''), 3000);
      return;
    }
    setImageInsertBusy(true);
    const reader = new FileReader();
    reader.onload = async (ev) => {
      const dataURL = ev.target?.result as string;
      if (dataURL) {
        try {
          await StoreCommentImageDataURL(dataURL); // Pre-validate/upload logic if needed, or just keep as data URL for now to send with post
          setPendingLocalImages((prev) => [...prev, dataURL]);
          setReplyMessage('Image attached');
          setTimeout(() => setReplyMessage(''), 2000);
        } catch (err: any) {
          setReplyMessage(`Failed to attach image: ${err.message}`);
        }
      }
      setImageInsertBusy(false);
      if (imageFileInputRef.current) imageFileInputRef.current.value = '';
    };
    reader.readAsDataURL(file);
  };

  const handleInsertImageURL = () => {
    const url = prompt('Enter image URL:');
    if (url) {
      if (url.match(/^https?:\/\/.+/)) {
        setPendingExternalImages((prev) => [...prev, url]);
      } else {
        alert('Invalid URL');
      }
    }
  };

  const handleInsertCodeBlock = () => {
    const start = replyInputRef.current?.selectionStart || 0;
    const end = replyInputRef.current?.selectionEnd || 0;
    const text = replyContent;
    const before = text.substring(0, start);
    const selected = text.substring(start, end);
    const after = text.substring(end);
    const insertion = selected ? `\`${selected}\`` : '```\ncode\n```';
    setReplyContent(before + insertion + after);
    setTimeout(() => {
      replyInputRef.current?.focus();
      const newCursorPos = before.length + insertion.length;
      replyInputRef.current?.setSelectionRange(newCursorPos, newCursorPos);
    }, 0);
  };

  const focusReplyInput = () => {
    if (replyInputRef.current) {
      replyInputRef.current.focus();
      replyInputRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  };

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6 bg-warm-bg dark:bg-background-dark">
      <div className="max-w-4xl mx-auto">
        <button
          onClick={onBack}
          className="mb-4 flex items-center text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary transition-colors"
        >
          <span className="material-icons text-lg mr-1">arrow_back</span>
          Back to Feed
        </button>

        <article className="bg-warm-card dark:bg-surface-dark rounded-xl p-6 shadow-soft border border-warm-border dark:border-border-dark mb-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              {avatarUrl ? (
                <img
                  className="w-10 h-10 rounded-full object-cover ring-2 ring-warm-bg dark:ring-border-dark"
                  src={avatarUrl}
                  alt={displayName}
                />
              ) : (
                <div className="w-10 h-10 rounded-full bg-warm-accent flex items-center justify-center text-white font-bold ring-2 ring-warm-bg dark:ring-border-dark">
                  {getInitials(displayName)}
                </div>
              )}
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-bold text-warm-text-primary dark:text-white hover:underline cursor-pointer">
                    {displayName}
                  </span>
                  <span className="bg-warm-sidebar dark:bg-surface-lighter text-warm-text-secondary dark:text-slate-400 text-[10px] px-2 py-0.5 rounded-full font-medium border border-warm-border dark:border-slate-700">
                    #{post.subId}
                  </span>
                </div>
                <div className="text-xs text-warm-text-secondary dark:text-slate-400 flex items-center gap-1">
                  <span>{formatTimeAgo(post.timestamp)}</span>
                  {isDevMode && (
                    <>
                      <span>•</span>
                      <button
                        onClick={() => onViewOperationTimeline('post', post.id)}
                        className="hover:text-warm-accent underline"
                        title="View operation timeline"
                      >
                        {post.id.slice(0, 8)}
                      </button>
                    </>
                  )}
                </div>
              </div>
            </div>
            {canDeletePost && (
              <button
                onClick={() => setDeletePostArmed(true)}
                className="text-warm-text-secondary hover:text-red-600 transition-colors p-2 rounded-full hover:bg-red-50 dark:hover:bg-red-900/20"
                title="Delete Post"
              >
                <span className="material-icons text-xl">delete_outline</span>
              </button>
            )}
          </div>

          <h1 className="text-2xl md:text-3xl font-bold text-warm-text-primary dark:text-white mb-4 leading-tight">
            {post.title}
          </h1>

          <div className="prose dark:prose-invert max-w-none mb-6 text-warm-text-secondary dark:text-slate-300 leading-relaxed text-base break-words">
            {post.imageCid && (
              <div className="mb-4 rounded-xl overflow-hidden border border-warm-border dark:border-border-dark bg-black/5">
                <img
                  src={`http://127.0.0.1:36660/blob/${post.imageCid}`}
                  alt="Post content"
                  className="w-full h-auto max-h-[600px] object-contain mx-auto"
                  loading="lazy"
                  onClick={() => setPreviewImageSrc(`http://127.0.0.1:36660/blob/${post.imageCid}`)}
                  style={{ cursor: 'zoom-in' }}
                />
              </div>
            )}
            {linkifyAndMarkdown(body)}
          </div>

          <div className="flex items-center justify-between border-t border-warm-border/50 dark:border-border-dark pt-4">
            <div className="flex items-center gap-4">
              <div className="flex items-center bg-warm-sidebar dark:bg-surface-lighter rounded-lg border border-warm-border dark:border-border-dark overflow-hidden">
                <button
                  onClick={() => onUpvote(post.id)}
                  className="px-3 py-1.5 hover:bg-warm-bg dark:hover:bg-border-dark text-warm-text-secondary hover:text-warm-accent transition-colors border-r border-warm-border/50 dark:border-border-dark"
                >
                  <span className="material-icons-round text-lg">arrow_upward</span>
                </button>
                <span className="px-3 text-sm font-bold text-warm-text-primary dark:text-white tabular-nums">
                  {post.score || 0}
                </span>
                <button
                  onClick={() => onDownvote(post.id)}
                  className="px-3 py-1.5 hover:bg-warm-bg dark:hover:bg-border-dark text-warm-text-secondary hover:text-warm-accent transition-colors border-l border-warm-border/50 dark:border-border-dark"
                >
                  <span className="material-icons-round text-lg">arrow_downward</span>
                </button>
              </div>

              <button
                onClick={() => focusReplyInput()}
                className="flex items-center gap-2 text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary px-3 py-1.5 rounded-lg hover:bg-warm-sidebar dark:hover:bg-surface-lighter transition-colors"
              >
                <span className="material-icons-outlined text-lg">chat_bubble_outline</span>
                {comments.length} Comments
              </button>

              <button className="flex items-center gap-2 text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary px-3 py-1.5 rounded-lg hover:bg-warm-sidebar dark:hover:bg-surface-lighter transition-colors">
                <span className="material-icons-outlined text-lg">share</span>
                Share
              </button>

              <button
                onClick={() => {
                  if (onToggleFavorite) onToggleFavorite(post.id);
                }}
                className={`flex items-center gap-2 text-sm font-medium px-3 py-1.5 rounded-lg transition-colors ${
                  post.isFavorited
                    ? 'text-warm-accent bg-warm-accent/10 hover:bg-warm-accent/20'
                    : 'text-warm-text-secondary hover:text-warm-text-primary hover:bg-warm-sidebar dark:hover:bg-surface-lighter'
                }`}
              >
                <span className="material-icons-outlined text-lg">
                  {post.isFavorited ? 'bookmark' : 'bookmark_border'}
                </span>
                {post.isFavorited ? 'Saved' : 'Save'}
              </button>
            </div>

            <div className="text-xs text-warm-text-secondary dark:text-slate-400 font-mono">
              ID: {post.id.slice(0, 8)}
            </div>
          </div>
        </article>

        <div className="mb-8">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-bold text-warm-text-primary dark:text-white">
              Comments <span className="text-warm-text-secondary dark:text-slate-400 text-sm font-normal">({comments.length})</span>
            </h3>
            <div className="flex items-center gap-2">
              <span className="text-xs font-medium text-warm-text-secondary dark:text-slate-400 uppercase tracking-wide">Sort by:</span>
              <div className="relative">
                <select
                  className="appearance-none bg-warm-card dark:bg-surface-dark border border-warm-border dark:border-border-dark text-sm font-bold text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent cursor-pointer py-1.5 pl-3 pr-8 rounded-lg outline-none transition-colors"
                >
                  <option value="best">Best</option>
                  <option value="newest">Newest</option>
                  <option value="controversial">Controversial</option>
                </select>
                <span className="material-icons absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none text-warm-text-secondary dark:text-slate-400 text-base">
                  expand_more
                </span>
              </div>
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
              {(pendingLocalImages.length > 0 || pendingExternalImages.length > 0) && (
                <div className="mt-2 flex flex-wrap gap-2">
                  {pendingLocalImages.map((src, idx) => (
                    <div key={`${src.slice(0, 24)}-${idx}`} className="relative group">
                      <img
                        src={src}
                        alt={`Pending attachment ${idx + 1}`}
                        className="h-16 w-16 rounded-md object-cover border border-warm-border dark:border-border-dark cursor-zoom-in"
                        onClick={() => setPreviewImageSrc(src)}
                      />
                      <button
                        onClick={() => setPendingLocalImages((prev) => prev.filter((_, i) => i !== idx))}
                        className="absolute -top-2 -right-2 rounded-full bg-red-500 text-white text-xs leading-none w-5 h-5 hidden group-hover:flex items-center justify-center"
                        title="Remove image"
                      >
                        x
                      </button>
                    </div>
                  ))}
                  {pendingExternalImages.map((src, idx) => (
                    <div key={`${src}-${idx}`} className="relative group">
                      <div
                        className="h-16 w-40 rounded-md border border-warm-border dark:border-border-dark bg-warm-surface dark:bg-surface-dark text-xs text-warm-text-secondary dark:text-slate-300 px-2 py-1 overflow-hidden break-all cursor-pointer"
                        onClick={() => setPreviewImageSrc(src)}
                        title={src}
                      >
                        {src}
                      </div>
                      <button
                        onClick={() => setPendingExternalImages((prev) => prev.filter((_, i) => i !== idx))}
                        className="absolute -top-2 -right-2 rounded-full bg-red-500 text-white text-xs leading-none w-5 h-5 hidden group-hover:flex items-center justify-center"
                        title="Remove image"
                      >
                        x
                      </button>
                    </div>
                  ))}
                </div>
              )}
              <div className="mt-2 flex items-center justify-end gap-2">
                <button
                  onClick={() => imageFileInputRef.current?.click()}
                  disabled={imageInsertBusy || replyBusy}
                  className="p-1.5 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors rounded"
                  title="Insert local image"
                >
                  <span className="material-icons text-lg">image</span>
                </button>
                <input
                  ref={imageFileInputRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={handleSelectLocalImage}
                />
                <button
                  onClick={handleInsertImageURL}
                  className="p-1.5 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors rounded"
                  title="Insert external image URL"
                >
                  <span className="material-icons text-lg">link</span>
                </button>
                <button
                  onClick={handleInsertCodeBlock}
                  className="p-1.5 text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors rounded"
                >
                  <span className="material-icons text-lg">code</span>
                </button>
                <button
                  onClick={handleSubmitReply}
                  disabled={(!replyContent.trim() && pendingLocalImages.length === 0 && pendingExternalImages.length === 0) || replyBusy}
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
            onDownvote={onCommentDownvote}
            currentPubkey={currentPubkey}
            onDelete={onDeleteComment}
            onImageClick={(src) => setPreviewImageSrc(src)}
          />
        </div>
      </div>

      {previewImageSrc && (
        <div className="fixed inset-0 z-[80] bg-black/80 flex items-center justify-center p-4" onClick={() => setPreviewImageSrc(null)}>
          <img
            src={previewImageSrc}
            alt="Preview"
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
      {canDeletePost && deletePostArmed && (
        <div className="fixed inset-0 z-[85] flex items-center justify-center bg-black/60 backdrop-blur-sm px-4 transition-all duration-300" onClick={() => setDeletePostArmed(false)}>
          <div
            className="w-full max-w-sm rounded-2xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark backdrop-blur-md p-6 shadow-2xl flex flex-col items-center text-center transform scale-100 transition-all"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4">
              <span className="material-icons text-warm-accent text-5xl">warning_amber</span>
            </div>
            <h4 className="text-lg font-bold text-warm-text-primary dark:text-white mb-2">Delete Post</h4>
            <p className="text-sm text-warm-text-secondary dark:text-slate-300 mb-6">
              This action cannot be undone. Are you sure you want to permanently delete this post?
            </p>
            <div className="flex items-center gap-3 w-full">
              <button
                onClick={() => setDeletePostArmed(false)}
                className="flex-1 py-2.5 rounded-xl font-bold border border-warm-border dark:border-border-dark text-warm-text-secondary dark:text-slate-300 hover:text-warm-text-primary dark:hover:text-white hover:bg-warm-bg dark:hover:bg-surface-lighter transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  try {
                    await onDeletePost(post.id);
                    setDeletePostArmed(false);
                  } catch (error: any) {
                    setReplyMessage(error?.message || 'Failed to delete post.');
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
