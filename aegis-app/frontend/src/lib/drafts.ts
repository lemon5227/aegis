import { CommentDraftSummary, DraftSummary, PostComposerMode, PostDraftSummary } from '../types';

const POST_DRAFT_PREFIX = 'aegis:create-post-draft:v2:';
const COMMENT_DRAFT_PREFIX = 'aegis:comment-draft:v1:';

type StoredPostDraft = {
  version: number;
  mode: PostComposerMode;
  title: string;
  body: string;
  linkURL: string;
  externalImageURL: string;
  updatedAt: number;
};

type StoredCommentDraft = {
  version: number;
  body: string;
  replyToId: string | null;
  updatedAt: number;
};

function canUseStorage(): boolean {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

function getStorageKeys(): string[] {
  if (!canUseStorage()) {
    return [];
  }
  return Object.keys(window.localStorage);
}

function parsePostDraft(key: string): PostDraftSummary | null {
  if (!key.startsWith(POST_DRAFT_PREFIX) || !canUseStorage()) {
    return null;
  }
  const remainder = key.slice(POST_DRAFT_PREFIX.length);
  const separatorIndex = remainder.indexOf(':');
  if (separatorIndex <= 0) {
    return null;
  }

  const authorPublicKey = remainder.slice(0, separatorIndex);
  const subId = remainder.slice(separatorIndex + 1);
  const rawValue = window.localStorage.getItem(key);
  if (!rawValue) {
    return null;
  }

  try {
    const parsed = JSON.parse(rawValue) as Partial<StoredPostDraft>;
    return {
      kind: 'post',
      storageKey: key,
      subId,
      authorPublicKey,
      title: typeof parsed.title === 'string' ? parsed.title : '',
      body: typeof parsed.body === 'string' ? parsed.body : '',
      linkURL: typeof parsed.linkURL === 'string' ? parsed.linkURL : '',
      externalImageURL: typeof parsed.externalImageURL === 'string' ? parsed.externalImageURL : '',
      mode: parsed.mode === 'link' ? 'link' : 'text',
      updatedAt: typeof parsed.updatedAt === 'number' ? parsed.updatedAt : 0,
    };
  } catch {
    return null;
  }
}

function parseCommentDraft(key: string): CommentDraftSummary | null {
  if (!key.startsWith(COMMENT_DRAFT_PREFIX) || !canUseStorage()) {
    return null;
  }
  const remainder = key.slice(COMMENT_DRAFT_PREFIX.length);
  const separatorIndex = remainder.indexOf(':');
  if (separatorIndex <= 0) {
    return null;
  }

  const authorPublicKey = remainder.slice(0, separatorIndex);
  const postId = remainder.slice(separatorIndex + 1);
  const rawValue = window.localStorage.getItem(key);
  if (!rawValue) {
    return null;
  }

  try {
    const parsed = JSON.parse(rawValue) as Partial<StoredCommentDraft>;
    return {
      kind: 'comment',
      storageKey: key,
      postId,
      authorPublicKey,
      body: typeof parsed.body === 'string' ? parsed.body : '',
      replyToId: typeof parsed.replyToId === 'string' ? parsed.replyToId : null,
      updatedAt: typeof parsed.updatedAt === 'number' ? parsed.updatedAt : 0,
    };
  } catch {
    return null;
  }
}

export function listDrafts(authorPublicKey?: string): DraftSummary[] {
  const normalizedAuthor = (authorPublicKey || '').trim();
  const items = getStorageKeys()
    .map((key) => parsePostDraft(key) || parseCommentDraft(key))
    .filter((item): item is DraftSummary => item !== null)
    .filter((item) => !normalizedAuthor || item.authorPublicKey === normalizedAuthor)
    .sort((left, right) => right.updatedAt - left.updatedAt);

  return items;
}

export function removeDraft(storageKey: string): void {
  if (!canUseStorage()) {
    return;
  }
  window.localStorage.removeItem(storageKey);
}
