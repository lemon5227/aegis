import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { listDrafts, removeDraft } from './drafts';

const POST_DRAFT_PREFIX = 'aegis:create-post-draft:v2:';
const COMMENT_DRAFT_PREFIX = 'aegis:comment-draft:v1:';

function seedPostDraft(authorPublicKey: string, subId: string, payload: Record<string, unknown>) {
  const key = `${POST_DRAFT_PREFIX}${authorPublicKey}:${subId}`;
  window.localStorage.setItem(key, JSON.stringify(payload));
  return key;
}

function seedCommentDraft(authorPublicKey: string, postId: string, payload: Record<string, unknown>) {
  const key = `${COMMENT_DRAFT_PREFIX}${authorPublicKey}:${postId}`;
  window.localStorage.setItem(key, JSON.stringify(payload));
  return key;
}

describe('listDrafts', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it('returns empty when no drafts exist', () => {
    expect(listDrafts()).toEqual([]);
  });

  it('returns post drafts with parsed fields', () => {
    seedPostDraft('author-1', 'sub-a', {
      version: 2,
      mode: 'text',
      title: 'My Title',
      body: 'My body',
      linkURL: '',
      externalImageURL: '',
      updatedAt: 1700000000,
    });

    const got = listDrafts();
    expect(got).toHaveLength(1);
    expect(got[0].kind).toBe('post');
    if (got[0].kind === 'post') {
      expect(got[0].title).toBe('My Title');
      expect(got[0].body).toBe('My body');
      expect(got[0].mode).toBe('text');
      expect(got[0].subId).toBe('sub-a');
      expect(got[0].authorPublicKey).toBe('author-1');
    }
  });

  it('returns comment drafts with parsed fields', () => {
    seedCommentDraft('author-1', 'post-X', {
      version: 1,
      body: 'Reply body',
      replyToId: 'parent-comment-1',
      updatedAt: 1700000000,
    });

    const got = listDrafts();
    expect(got).toHaveLength(1);
    expect(got[0].kind).toBe('comment');
    if (got[0].kind === 'comment') {
      expect(got[0].body).toBe('Reply body');
      expect(got[0].postId).toBe('post-X');
      expect(got[0].replyToId).toBe('parent-comment-1');
    }
  });

  it('filters by author public key when provided', () => {
    seedPostDraft('author-A', 'sub-a', { title: 'A', updatedAt: 1 });
    seedPostDraft('author-B', 'sub-b', { title: 'B', updatedAt: 1 });

    const filtered = listDrafts('author-A');
    expect(filtered).toHaveLength(1);
    if (filtered[0].kind === 'post') {
      expect(filtered[0].title).toBe('A');
    }
  });

  it('returns drafts ordered by updatedAt DESC', () => {
    seedPostDraft('author-1', 'sub-a', { updatedAt: 100, title: 'old' });
    seedPostDraft('author-1', 'sub-b', { updatedAt: 300, title: 'new' });
    seedPostDraft('author-1', 'sub-c', { updatedAt: 200, title: 'mid' });

    const got = listDrafts();
    if (got[0].kind === 'post' && got[1].kind === 'post' && got[2].kind === 'post') {
      expect([got[0].title, got[1].title, got[2].title]).toEqual(['new', 'mid', 'old']);
    } else {
      throw new Error('expected three post drafts');
    }
  });

  it('skips malformed JSON payloads', () => {
    const key = `${POST_DRAFT_PREFIX}author-1:sub-a`;
    window.localStorage.setItem(key, 'not-json');
    expect(listDrafts()).toEqual([]);
  });

  it('skips keys without the expected separator', () => {
    // No ':' between author and subId — should be ignored.
    window.localStorage.setItem(`${POST_DRAFT_PREFIX}only-author`, '{}');
    expect(listDrafts()).toEqual([]);
  });

  it('uses safe defaults for missing fields', () => {
    seedPostDraft('a', 's', {}); // empty payload

    const got = listDrafts();
    expect(got).toHaveLength(1);
    if (got[0].kind === 'post') {
      expect(got[0].title).toBe('');
      expect(got[0].body).toBe('');
      expect(got[0].mode).toBe('text'); // default
      expect(got[0].updatedAt).toBe(0);
    }
  });

  it('treats mode="link" as link, anything else as text', () => {
    seedPostDraft('a', 's-link', { mode: 'link', title: 'L', updatedAt: 1 });
    seedPostDraft('a', 's-text', { mode: 'text', title: 'T', updatedAt: 1 });
    seedPostDraft('a', 's-bogus', { mode: 'unknown-mode', title: 'B', updatedAt: 1 });

    const got = listDrafts();
    const byTitle = Object.fromEntries(
      got.flatMap((d) => (d.kind === 'post' ? [[d.title, d.mode]] : [])),
    );
    expect(byTitle).toEqual({ L: 'link', T: 'text', B: 'text' });
  });

  it('ignores foreign localStorage keys', () => {
    window.localStorage.setItem('unrelated:key', JSON.stringify({ x: 1 }));
    seedPostDraft('a', 's', { title: 'mine', updatedAt: 1 });

    const got = listDrafts();
    expect(got).toHaveLength(1);
    if (got[0].kind === 'post') {
      expect(got[0].title).toBe('mine');
    }
  });
});

describe('removeDraft', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('removes a draft by storage key', () => {
    const key = seedPostDraft('a', 's', { title: 'goodbye', updatedAt: 1 });
    expect(listDrafts()).toHaveLength(1);

    removeDraft(key);
    expect(listDrafts()).toHaveLength(0);
  });

  it('is a no-op for unknown keys', () => {
    seedPostDraft('a', 's', { title: 'kept', updatedAt: 1 });
    removeDraft('does-not-exist');
    expect(listDrafts()).toHaveLength(1);
  });
});
