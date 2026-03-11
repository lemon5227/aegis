import { useEffect, useMemo, useRef, useState } from 'react';
import { buildPostBody, isValidExternalURL } from '../lib/postContent';
import { CreatePostInput, PostComposerMode } from '../types';

const CREATE_POST_DRAFT_VERSION = 2;
const MAX_TITLE_LENGTH = 180;
const MAX_BODY_LENGTH = 12000;
const AUTOSAVE_DELAY_MS = 300;

type CreatePostDraft = {
  version: number;
  mode: PostComposerMode;
  title: string;
  body: string;
  linkURL: string;
  externalImageURL: string;
  updatedAt: number;
};

type DraftState = {
  mode: PostComposerMode;
  title: string;
  body: string;
  linkURL: string;
  externalImageURL: string;
};

type ImageState = {
  base64: string;
  mime: string;
  preview: string;
  message: string;
  busy: boolean;
};

const EMPTY_DRAFT: DraftState = {
  mode: 'text',
  title: '',
  body: '',
  linkURL: '',
  externalImageURL: '',
};

const EMPTY_IMAGE_STATE: ImageState = {
  base64: '',
  mime: '',
  preview: '',
  message: '',
  busy: false,
};

function getDraftStorageKey(subId: string, publicKey?: string): string {
  const normalizedSubId = subId.trim() || 'general';
  const normalizedPublicKey = (publicKey || 'anonymous').trim() || 'anonymous';
  return `aegis:create-post-draft:v${CREATE_POST_DRAFT_VERSION}:${normalizedPublicKey}:${normalizedSubId}`;
}

function isBrowserStorageAvailable(): boolean {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

function parseStoredDraft(rawValue: string | null): DraftState | null {
  if (!rawValue) return null;
  try {
    const parsed = JSON.parse(rawValue) as Partial<CreatePostDraft>;
    if (parsed.version !== CREATE_POST_DRAFT_VERSION) {
      return null;
    }
    return {
      mode: parsed.mode === 'link' ? 'link' : 'text',
      title: typeof parsed.title === 'string' ? parsed.title : '',
      body: typeof parsed.body === 'string' ? parsed.body : '',
      linkURL: typeof parsed.linkURL === 'string' ? parsed.linkURL : '',
      externalImageURL: typeof parsed.externalImageURL === 'string' ? parsed.externalImageURL : '',
    };
  } catch {
    return null;
  }
}

function loadDraft(key: string): DraftState | null {
  if (!isBrowserStorageAvailable()) return null;
  return parseStoredDraft(window.localStorage.getItem(key));
}

function persistDraft(key: string, draft: DraftState): void {
  if (!isBrowserStorageAvailable()) return;
  const payload: CreatePostDraft = {
    version: CREATE_POST_DRAFT_VERSION,
    mode: draft.mode,
    title: draft.title,
    body: draft.body,
    linkURL: draft.linkURL,
    externalImageURL: draft.externalImageURL,
    updatedAt: Date.now(),
  };
  window.localStorage.setItem(key, JSON.stringify(payload));
}

function removeDraft(key: string): void {
  if (!isBrowserStorageAvailable()) return;
  window.localStorage.removeItem(key);
}

function normalizeDraftInput(value: string): string {
  return value.replace(/\r\n/g, '\n');
}

function isDraftEmpty(draft: DraftState): boolean {
  return !draft.title.trim() && !draft.body.trim() && !draft.linkURL.trim() && !draft.externalImageURL.trim();
}

function validateExternalImageURL(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return '';
  if (!isValidExternalURL(trimmed)) {
    return 'External image URL is invalid.';
  }
  return '';
}

function validateLinkURL(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return 'Link URL is required for link posts.';
  }
  if (!isValidExternalURL(trimmed)) {
    return 'Link URL is invalid.';
  }
  return '';
}

function getSubmitValidationMessage(draft: DraftState): string {
  if (!draft.title.trim()) {
    return 'Title is required.';
  }
  if (draft.title.trim().length > MAX_TITLE_LENGTH) {
    return `Title must be ${MAX_TITLE_LENGTH} characters or fewer.`;
  }
  if (draft.body.length > MAX_BODY_LENGTH) {
    return `Content must be ${MAX_BODY_LENGTH} characters or fewer.`;
  }
  if (draft.mode === 'link') {
    const linkValidationMessage = validateLinkURL(draft.linkURL);
    if (linkValidationMessage) {
      return linkValidationMessage;
    }
  }
  return validateExternalImageURL(draft.externalImageURL);
}

function readFileAsDataURL(input: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '');
    reader.onerror = () => reject(new Error('Failed to read image.'));
    reader.readAsDataURL(input);
  });
}

function loadImageFromDataURL(dataURL: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error('Failed to decode image.'));
    image.src = dataURL;
  });
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) {
        reject(new Error('Failed to convert image.'));
        return;
      }
      resolve(blob);
    }, type, quality);
  });
}

async function compressPostImage(input: File): Promise<{ mime: string; base64: string; preview: string }> {
  const MAX_SOURCE_BYTES = 8 * 1024 * 1024;
  const MAX_DIMENSION = 1600;
  const MAX_OUTPUT_BYTES = 420 * 1024;

  if (input.size > MAX_SOURCE_BYTES) {
    throw new Error('Image too large (>8MB). Please use a smaller file.');
  }

  const sourceDataURL = await readFileAsDataURL(input);
  const image = await loadImageFromDataURL(sourceDataURL);
  const scale = Math.min(1, MAX_DIMENSION / Math.max(image.width, image.height));
  const targetWidth = Math.max(1, Math.round(image.width * scale));
  const targetHeight = Math.max(1, Math.round(image.height * scale));

  const canvas = document.createElement('canvas');
  canvas.width = targetWidth;
  canvas.height = targetHeight;
  const context = canvas.getContext('2d');
  if (!context) {
    throw new Error('Canvas unavailable.');
  }
  context.drawImage(image, 0, 0, targetWidth, targetHeight);

  const qualityCandidates = [0.9, 0.82, 0.74, 0.66, 0.58, 0.5, 0.42];
  for (const quality of qualityCandidates) {
    const blob = await canvasToBlob(canvas, 'image/jpeg', quality);
    if (blob.size > MAX_OUTPUT_BYTES) {
      continue;
    }

    const dataURL = await readFileAsDataURL(new File([blob], 'post.jpg', { type: 'image/jpeg' }));
    const marker = ';base64,';
    const markerIndex = dataURL.indexOf(marker);
    if (markerIndex <= 0) {
      throw new Error('Image encode failed.');
    }
    return {
      mime: 'image/jpeg',
      base64: dataURL.slice(markerIndex + marker.length),
      preview: dataURL,
    };
  }

  throw new Error('Could not compress enough. Try a smaller image or use external URL.');
}

interface CreatePostModalProps {
  isOpen: boolean;
  onClose: () => void;
  subId: string;
  authorPublicKey?: string;
  onCreate: (input: CreatePostInput) => Promise<void> | void;
}

export function CreatePostModal({ isOpen, onClose, subId, authorPublicKey, onCreate }: CreatePostModalProps) {
  const [draft, setDraft] = useState<DraftState>(EMPTY_DRAFT);
  const [imageState, setImageState] = useState<ImageState>(EMPTY_IMAGE_STATE);
  const [submitBusy, setSubmitBusy] = useState(false);
  const [submitMessage, setSubmitMessage] = useState('');
  const [draftRestored, setDraftRestored] = useState(false);
  const [draftSavedAt, setDraftSavedAt] = useState<number | null>(null);
  const [activePanel, setActivePanel] = useState<'compose' | 'preview'>('compose');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const autosaveTimerRef = useRef<number | null>(null);
  const draftKey = useMemo(() => getDraftStorageKey(subId, authorPublicKey), [authorPublicKey, subId]);
  const validationMessage = getSubmitValidationMessage(draft);
  const linkValidationMessage = draft.mode === 'link' ? validateLinkURL(draft.linkURL) : '';
  const externalImageValidationMessage = validateExternalImageURL(draft.externalImageURL);
  const previewBody = buildPostBody({
    mode: draft.mode,
    body: draft.body,
    linkURL: draft.linkURL,
  });

  useEffect(() => {
    return () => {
      if (autosaveTimerRef.current !== null) {
        window.clearTimeout(autosaveTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!isOpen) return;
    const storedDraft = loadDraft(draftKey);
    if (storedDraft) {
      setDraft(storedDraft);
      setDraftRestored(!isDraftEmpty(storedDraft));
      setSubmitMessage('');
      setImageState(EMPTY_IMAGE_STATE);
      setActivePanel('compose');
      return;
    }
    setDraft(EMPTY_DRAFT);
    setDraftRestored(false);
    setDraftSavedAt(null);
    setSubmitMessage('');
    setImageState(EMPTY_IMAGE_STATE);
    setActivePanel('compose');
  }, [draftKey, isOpen]);

  useEffect(() => {
    if (!isOpen) return;
    if (autosaveTimerRef.current !== null) {
      window.clearTimeout(autosaveTimerRef.current);
    }

    autosaveTimerRef.current = window.setTimeout(() => {
      if (isDraftEmpty(draft)) {
        removeDraft(draftKey);
        setDraftSavedAt(null);
        return;
      }
      persistDraft(draftKey, draft);
      setDraftSavedAt(Date.now());
    }, AUTOSAVE_DELAY_MS);

    return () => {
      if (autosaveTimerRef.current !== null) {
        window.clearTimeout(autosaveTimerRef.current);
      }
    };
  }, [draft, draftKey, isOpen]);

  if (!isOpen) return null;

  const applyDraftPatch = (patch: Partial<DraftState>) => {
    setDraft((current) => ({ ...current, ...patch }));
    setSubmitMessage('');
  };

  const resetImageState = () => {
    setImageState(EMPTY_IMAGE_STATE);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const discardDraft = () => {
    setDraft(EMPTY_DRAFT);
    setDraftRestored(false);
    setDraftSavedAt(null);
    setSubmitMessage('');
    resetImageState();
    removeDraft(draftKey);
  };

  const setComposerMode = (mode: PostComposerMode) => {
    setDraft((current) => {
      if (current.mode === mode) {
        return current;
      }
      return {
        ...current,
        mode,
        linkURL: mode === 'link' ? current.linkURL : '',
      };
    });
    setSubmitMessage('');
  };

  const handleSubmit = async () => {
    if (validationMessage) {
      setSubmitMessage(validationMessage);
      return;
    }
    setSubmitBusy(true);
    setSubmitMessage('');
    try {
      await onCreate({
        title: draft.title.trim(),
        body: buildPostBody({
          mode: draft.mode,
          body: draft.body,
          linkURL: draft.linkURL,
        }),
        imageBase64: imageState.base64.trim(),
        imageMime: imageState.mime.trim(),
        externalImageURL: draft.externalImageURL.trim(),
        mode: draft.mode,
        linkURL: draft.linkURL.trim(),
      });
      discardDraft();
      onClose();
    } catch (error: any) {
      setSubmitMessage(error?.message || 'Failed to publish post.');
    } finally {
      setSubmitBusy(false);
    }
  };

  const handleImageSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      setImageState((current) => ({ ...current, message: 'Please choose an image file.' }));
      event.target.value = '';
      return;
    }

    try {
      setImageState((current) => ({
        ...current,
        busy: true,
        message: 'Compressing image...',
      }));
      const result = await compressPostImage(file);
      setImageState({
        base64: result.base64,
        mime: result.mime,
        preview: result.preview,
        busy: false,
        message: 'Image ready. It will be uploaded with the post.',
      });
    } catch (error: any) {
      setImageState({
        ...EMPTY_IMAGE_STATE,
        message: error?.message || 'Failed to process image.',
      });
    } finally {
      event.target.value = '';
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={() => { if (!submitBusy) onClose(); }} />
      <div className="relative bg-warm-card dark:bg-surface-dark rounded-xl shadow-2xl w-full max-w-lg border border-warm-border dark:border-border-dark max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-4 border-b border-warm-border dark:border-border-dark sticky top-0 bg-warm-card dark:bg-surface-dark">
          <h2 className="text-lg font-bold text-warm-text-primary dark:text-white">Create Post</h2>
          <button
            onClick={onClose}
            disabled={submitBusy}
            className="text-warm-text-secondary hover:text-warm-text-primary disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <span className="material-icons">close</span>
          </button>
        </div>

        <div className="p-4 space-y-4">
          <div className="flex items-center justify-between rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark px-3 py-2 text-xs text-warm-text-secondary dark:text-slate-400">
            <span>{draftRestored ? 'Recovered your draft for this sub.' : `Drafts are autosaved for r/${subId || 'general'}.`}</span>
            <span>{draftSavedAt ? `Saved ${new Date(draftSavedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}` : 'Not saved yet'}</span>
          </div>

          <div className="rounded-xl border border-warm-border dark:border-border-dark p-1">
            <div className="grid grid-cols-4 gap-1">
              <button
                onClick={() => setComposerMode('text')}
                className={`rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  draft.mode === 'text'
                    ? 'bg-warm-accent text-white'
                    : 'text-warm-text-secondary dark:text-slate-300 hover:bg-warm-bg dark:hover:bg-background-dark'
                }`}
              >
                Text Post
              </button>
              <button
                onClick={() => setComposerMode('link')}
                className={`rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  draft.mode === 'link'
                    ? 'bg-warm-accent text-white'
                    : 'text-warm-text-secondary dark:text-slate-300 hover:bg-warm-bg dark:hover:bg-background-dark'
                }`}
              >
                Link Post
              </button>
              <button
                onClick={() => setActivePanel('compose')}
                className={`rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  activePanel === 'compose'
                    ? 'bg-warm-sidebar dark:bg-background-dark text-warm-text-primary dark:text-white'
                    : 'text-warm-text-secondary dark:text-slate-300 hover:bg-warm-bg dark:hover:bg-background-dark'
                }`}
              >
                Compose
              </button>
              <button
                onClick={() => setActivePanel('preview')}
                className={`rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  activePanel === 'preview'
                    ? 'bg-warm-sidebar dark:bg-background-dark text-warm-text-primary dark:text-white'
                    : 'text-warm-text-secondary dark:text-slate-300 hover:bg-warm-bg dark:hover:bg-background-dark'
                }`}
              >
                Preview
              </button>
            </div>
          </div>

          {activePanel === 'compose' ? (
            <>
          <div>
            <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
              Title *
            </label>
            <input
              type="text"
              value={draft.title}
              maxLength={MAX_TITLE_LENGTH}
              onChange={(e) => applyDraftPatch({ title: normalizeDraftInput(e.target.value) })}
              placeholder="Enter a descriptive title"
              className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
            />
            <div className="mt-1 flex justify-between text-xs text-warm-text-secondary dark:text-slate-400">
              <span>Keep it specific enough to stand on its own in feed views.</span>
              <span>{draft.title.length}/{MAX_TITLE_LENGTH}</span>
            </div>
          </div>

          {draft.mode === 'link' && (
            <div>
              <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
                Link URL *
              </label>
              <input
                type="url"
                value={draft.linkURL}
                onChange={(e) => applyDraftPatch({ linkURL: normalizeDraftInput(e.target.value) })}
                placeholder="https://example.com/article"
                className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
              />
              <div className="mt-1 flex justify-between text-xs text-warm-text-secondary dark:text-slate-400">
                <span>Use a canonical source URL. The post will render as an outbound link card.</span>
                <span>{draft.linkURL.trim() ? 'Ready' : 'Required'}</span>
              </div>
              {linkValidationMessage && (
                <p className="mt-2 text-xs text-red-500">{linkValidationMessage}</p>
              )}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
              {draft.mode === 'link' ? 'Commentary' : 'Content'}
            </label>
            <textarea
              value={draft.body}
              maxLength={MAX_BODY_LENGTH}
              onChange={(e) => applyDraftPatch({ body: normalizeDraftInput(e.target.value) })}
              placeholder={draft.mode === 'link' ? 'Add context for why this link matters.' : "What's on your mind?"}
              rows={6}
              className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none resize-none"
            />
            <div className="mt-1 flex justify-between text-xs text-warm-text-secondary dark:text-slate-400">
              <span>
                {draft.mode === 'link'
                  ? 'Commentary is optional. The source URL is encoded separately and rendered as a link card.'
                  : 'Body is optional. If empty, the title becomes the post body for protocol compatibility.'}
              </span>
              <span>{draft.body.length}/{MAX_BODY_LENGTH}</span>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
              Image (optional)
            </label>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              onChange={handleImageSelect}
              disabled={imageState.busy}
              className="w-full text-sm text-warm-text-secondary dark:text-slate-400 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-warm-accent file:text-white file:cursor-pointer file:transition-colors"
            />
            {imageState.message && (
              <p className="mt-2 text-xs text-warm-text-secondary dark:text-slate-300">{imageState.message}</p>
            )}
            {imageState.preview && (
              <div className="mt-3 relative inline-block">
                <img
                  src={imageState.preview}
                  alt="Preview"
                  className="max-h-40 rounded-lg border border-warm-border dark:border-border-dark"
                />
                <button
                  onClick={resetImageState}
                  className="absolute -top-2 -right-2 bg-red-500 text-white rounded-full p-1 hover:bg-red-600"
                >
                  <span className="material-icons text-sm">close</span>
                </button>
              </div>
            )}

            <div className="mt-3">
              <label className="block text-xs font-medium text-warm-text-secondary dark:text-slate-400 mb-2">
                Or use external image URL (recommended for lower network storage pressure)
              </label>
              <div className="flex gap-2">
                <input
                  type="url"
                  value={draft.externalImageURL}
                  onChange={(e) => applyDraftPatch({ externalImageURL: normalizeDraftInput(e.target.value) })}
                  placeholder="https://example.com/image.jpg"
                  className="flex-1 px-3 py-2 text-sm rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
                />
                <button
                  onClick={() => applyDraftPatch({ externalImageURL: '' })}
                  className="px-3 py-2 text-xs font-medium text-warm-text-secondary dark:text-slate-300 rounded-lg border border-warm-border dark:border-border-dark hover:bg-warm-bg dark:hover:bg-background-dark"
                >
                  Clear
                </button>
              </div>
              <p className="mt-1 text-xs text-warm-text-secondary dark:text-slate-400">
                If both local file and external URL are set, local file is used for post media.
              </p>
              {draft.externalImageURL.trim() && externalImageValidationMessage && (
                <p className="mt-2 text-xs text-red-500">{externalImageValidationMessage}</p>
              )}
              {imageState.preview && (
                <p className="mt-2 text-xs text-warm-text-secondary dark:text-slate-400">
                  Local image uploads are session-only and are not persisted in drafts.
                </p>
              )}
            </div>
          </div>
            </>
          ) : (
            <div className="rounded-2xl border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark p-5">
              <div className="text-xs uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">Preview</div>
              <h3 className="mt-3 text-2xl font-bold text-warm-text-primary dark:text-white">
                {draft.title.trim() || 'Untitled draft'}
              </h3>
              {draft.mode === 'link' && draft.linkURL.trim() && (
                <div className="mt-4 flex items-center justify-between gap-4 rounded-2xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark px-4 py-4">
                  <div className="min-w-0">
                    <div className="text-xs font-semibold uppercase tracking-wide text-warm-text-secondary dark:text-slate-400">
                      External Link
                    </div>
                    <div className="truncate text-base font-semibold text-warm-text-primary dark:text-white">
                      {draft.linkURL.trim()}
                    </div>
                  </div>
                  <span className="material-icons-outlined text-xl text-warm-text-secondary dark:text-slate-400">open_in_new</span>
                </div>
              )}
              {previewBody.trim() ? (
                <div className="mt-4 whitespace-pre-wrap break-words text-sm leading-7 text-warm-text-secondary dark:text-slate-300">
                  {draft.mode === 'link' ? (draft.body.trim() || 'No commentary added yet.') : previewBody}
                </div>
              ) : (
                <div className="mt-4 text-sm text-warm-text-secondary dark:text-slate-400">
                  Nothing to preview yet.
                </div>
              )}
              {imageState.preview && (
                <img
                  src={imageState.preview}
                  alt="Draft preview"
                  className="mt-4 max-h-56 rounded-xl border border-warm-border dark:border-border-dark"
                />
              )}
            </div>
          )}
        </div>

        <div className="flex items-center justify-between gap-3 p-4 border-t border-warm-border dark:border-border-dark sticky bottom-0 bg-warm-card dark:bg-surface-dark">
          <button
            onClick={discardDraft}
            disabled={submitBusy}
            className="px-3 py-2 text-xs font-medium text-warm-text-secondary dark:text-slate-300 rounded-lg border border-warm-border dark:border-border-dark hover:bg-warm-bg dark:hover:bg-background-dark disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Discard Draft
          </button>
          <div className="flex justify-end gap-3">
            <button
              onClick={onClose}
              disabled={submitBusy}
              className="px-4 py-2 text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Cancel
            </button>
            <button
              onClick={() => void handleSubmit()}
              disabled={!draft.title.trim() || submitBusy || !!validationMessage}
              className="px-4 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitBusy ? 'Posting...' : draft.mode === 'link' ? 'Post Link' : 'Post'}
            </button>
          </div>
        </div>
        {submitMessage && (
          <div className="px-4 pb-3 text-xs text-red-500">{submitMessage}</div>
        )}
      </div>
    </div>
  );
}
