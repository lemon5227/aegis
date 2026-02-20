import { ChangeEvent, useEffect, useMemo, useRef, useState } from 'react';
import { Post, Profile, Comment } from '../types';
import { CommentTree } from './CommentTree';
import { GetPostMediaByID } from '../../wailsjs/go/main/App';

interface PostDetailProps {
  post: Post;
  body?: string;
  comments: Comment[];
  profiles: Record<string, Profile>;
  currentPubkey?: string;
  onBack: () => void;
  onUpvote: (postId: string) => void;
  onReply: (parentId: string, body: string, localImageDataURLs: string[], externalImageURLs: string[]) => Promise<void> | void;
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
  const [imageInsertBusy, setImageInsertBusy] = useState(false);
  const [replyMessage, setReplyMessage] = useState('');
  const [commentSort, setCommentSort] = useState<'best' | 'newest' | 'controversial'>('best');
  const [postImageSrc, setPostImageSrc] = useState('');
  const [pendingLocalImages, setPendingLocalImages] = useState<string[]>([]);
  const [pendingExternalImages, setPendingExternalImages] = useState<string[]>([]);
  const [previewImageSrc, setPreviewImageSrc] = useState<string | null>(null);
  const replyInputRef = useRef<HTMLTextAreaElement | null>(null);
  const imageFileInputRef = useRef<HTMLInputElement | null>(null);

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

  useEffect(() => {
    let active = true;
    const loadPostImage = async () => {
      if (!post.imageCid) {
        if (active) setPostImageSrc('');
        return;
      }
      try {
        const media = await GetPostMediaByID(post.id);
        if (!active) return;
        if (media?.dataBase64 && media?.mime) {
          setPostImageSrc(`data:${media.mime};base64,${media.dataBase64}`);
          return;
        }
        setPostImageSrc('');
      } catch {
        if (active) setPostImageSrc('');
      }
    };
    void loadPostImage();
    return () => {
      active = false;
    };
  }, [post.id, post.imageCid]);

  const handleSubmitReply = async () => {
    const textPart = replyContent.trim();
    if (!textPart && pendingLocalImages.length === 0 && pendingExternalImages.length === 0) return;

    setReplyBusy(true);
    setReplyMessage('');
    try {
      await onReply(replyToId || '', textPart, pendingLocalImages, pendingExternalImages);
      setReplyContent('');
      setPendingLocalImages([]);
      setPendingExternalImages([]);
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

  const addPendingLocalImage = (imageURL: string) => {
    const normalized = imageURL.trim();
    if (!normalized) return;
    setPendingLocalImages((prev) => [...prev, normalized]);
    focusReplyInput();
  };

  const addPendingExternalImage = (imageURL: string) => {
    const normalized = imageURL.trim();
    if (!normalized) return;
    setPendingExternalImages((prev) => [...prev, normalized]);
    focusReplyInput();
  };

  const readFileAsDataURL = (file: File): Promise<string> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '');
      reader.onerror = () => reject(new Error('Failed to read image file'));
      reader.readAsDataURL(file);
    });
  };

  const loadImageFromDataURL = (dataURL: string): Promise<HTMLImageElement> => {
    return new Promise((resolve, reject) => {
      const image = new Image();
      image.onload = () => resolve(image);
      image.onerror = () => reject(new Error('Failed to decode image'));
      image.src = dataURL;
    });
  };

  const canvasToBlob = (canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob> => {
    return new Promise((resolve, reject) => {
      canvas.toBlob((blob) => {
        if (!blob) {
          reject(new Error('Failed to convert image'));
          return;
        }
        resolve(blob);
      }, type, quality);
    });
  };

  const compressCommentImage = async (file: File): Promise<string> => {
    const MAX_SOURCE_BYTES = 10 * 1024 * 1024;
    const MAX_OUTPUT_BYTES = 180 * 1024;
    const MAX_DIMENSION = 960;

    if (file.size > MAX_SOURCE_BYTES) {
      throw new Error('Selected image is too large (>10MB). Try a smaller image.');
    }

    const sourceDataURL = await readFileAsDataURL(file);
    const image = await loadImageFromDataURL(sourceDataURL);

    const scale = Math.min(1, MAX_DIMENSION / Math.max(image.width, image.height));
    const targetWidth = Math.max(1, Math.round(image.width * scale));
    const targetHeight = Math.max(1, Math.round(image.height * scale));

    const canvas = document.createElement('canvas');
    canvas.width = targetWidth;
    canvas.height = targetHeight;
    const context = canvas.getContext('2d');
    if (!context) {
      throw new Error('Canvas unavailable');
    }
    context.drawImage(image, 0, 0, targetWidth, targetHeight);

    const qualityCandidates = [0.9, 0.82, 0.74, 0.66, 0.58, 0.5, 0.42, 0.34, 0.26];
    for (const quality of qualityCandidates) {
      const blob = await canvasToBlob(canvas, 'image/jpeg', quality);
      if (blob.size <= MAX_OUTPUT_BYTES) {
        return await readFileAsDataURL(new File([blob], 'comment.jpg', { type: 'image/jpeg' }));
      }
    }

    throw new Error('Image is still too large after compression. Try a smaller source image.');
  };

  const handleInsertImageURL = () => {
    const input = window.prompt('Paste image URL (https://...)');
    if (!input) return;
    try {
      const parsed = new URL(input.trim());
      if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
        setReplyMessage('Only http/https image URLs are supported.');
        return;
      }
        addPendingExternalImage(parsed.toString());
        setReplyMessage('Image URL attached to this comment.');
    } catch {
      setReplyMessage('Invalid image URL.');
    }
  };

  const handleSelectLocalImage = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      setReplyMessage('Please select an image file.');
      event.target.value = '';
      return;
    }

    try {
      setImageInsertBusy(true);
      setReplyMessage('Processing image...');
      const dataURL = await compressCommentImage(file);
      if (!dataURL) {
        setReplyMessage('Failed to read image file.');
      } else {
        addPendingLocalImage(dataURL);
        setReplyMessage('Image attached to this comment.');
      }
    } catch (error: any) {
      setReplyMessage(error?.message || 'Failed to insert local image.');
    } finally {
      setImageInsertBusy(false);
      event.target.value = '';
    }
  };

  const renderRichText = (raw: string) => {
    const text = raw || '';
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
            className="max-h-80 w-auto rounded-lg border border-warm-border dark:border-border-dark cursor-zoom-in"
            onClick={() => setPreviewImageSrc(src)}
          />
        );
      }
      return (
        <p key={`txt-${index}`} className="whitespace-pre-wrap break-words">
          {line}
        </p>
      );
    });
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
              {renderRichText(body || post.bodyPreview || 'No content')}
            </div>
            {postImageSrc && (
              <div className="mt-5">
                <img
                  src={postImageSrc}
                  alt="Post media"
                  className="max-h-[520px] w-auto rounded-xl border border-warm-border dark:border-border-dark cursor-zoom-in"
                  onClick={() => setPreviewImageSrc(postImageSrc)}
                />
              </div>
            )}
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
    </div>
  );
}
