import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DraftsView } from './DraftsView';

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

describe('DraftsView', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it('renders the page title and description', () => {
    render(
      <DraftsView
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={vi.fn()}
      />,
    );
    expect(screen.getByText('Saved Drafts')).toBeInTheDocument();
    expect(screen.getByText('Drafts')).toBeInTheDocument();
  });

  it('shows empty state when no drafts exist', () => {
    render(
      <DraftsView
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={vi.fn()}
      />,
    );
    expect(screen.getByText('No drafts saved yet.')).toBeInTheDocument();
  });

  it('renders a post draft with title, sub badge, and post-mode label', () => {
    seedPostDraft('alice', 'rust', {
      title: 'My Rust Post',
      body: 'Body content',
      mode: 'text',
      updatedAt: Date.parse('2026-05-30T12:00:00Z'),
    });

    render(
      <DraftsView
        authorPublicKey="alice"
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={vi.fn()}
      />,
    );

    expect(screen.getByText('Post Draft')).toBeInTheDocument();
    expect(screen.getByText('My Rust Post')).toBeInTheDocument();
    expect(screen.getByText(/r\/rust/)).toBeInTheDocument();
    expect(screen.getByText(/Text post/)).toBeInTheDocument();
    expect(screen.getByText('Body content')).toBeInTheDocument();
  });

  it('uses "Untitled post draft" when post draft has no title', () => {
    seedPostDraft('alice', 'rust', { body: 'no title', updatedAt: 1 });

    render(
      <DraftsView
        authorPublicKey="alice"
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={vi.fn()}
      />,
    );
    expect(screen.getByText('Untitled post draft')).toBeInTheDocument();
  });

  it('shows "Link post" label for link-mode drafts', () => {
    seedPostDraft('alice', 'rust', {
      title: 'Link draft',
      mode: 'link',
      linkURL: 'https://example.com',
      updatedAt: 1,
    });

    render(
      <DraftsView
        authorPublicKey="alice"
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={vi.fn()}
      />,
    );
    expect(screen.getByText(/Link post/)).toBeInTheDocument();
  });

  it('renders a comment draft with reply context when replyToId set', () => {
    seedCommentDraft('alice', 'post-X12345678abcd', {
      body: 'My reply body',
      replyToId: 'parent-comment-yzx789',
      updatedAt: 1,
    });

    render(
      <DraftsView
        authorPublicKey="alice"
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={vi.fn()}
      />,
    );

    expect(screen.getByText('Comment Draft')).toBeInTheDocument();
    expect(screen.getByText('Reply draft')).toBeInTheDocument();
    expect(screen.getByText('My reply body')).toBeInTheDocument();
    expect(screen.getByText(/Post post-X12/)).toBeInTheDocument();
    expect(screen.getByText(/replying to parent-c/)).toBeInTheDocument();
  });

  it('uses "Comment draft" label when no replyToId', () => {
    seedCommentDraft('alice', 'post-X', {
      body: 'top-level comment',
      replyToId: null,
      updatedAt: 1,
    });

    render(
      <DraftsView
        authorPublicKey="alice"
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={vi.fn()}
      />,
    );
    expect(screen.getByText('Comment draft')).toBeInTheDocument();
  });

  it('clicking Resume on a post draft calls onOpenPostDraft(subId)', () => {
    seedPostDraft('alice', 'rust', { title: 'T', updatedAt: 1 });
    const onOpenPostDraft = vi.fn();

    render(
      <DraftsView
        authorPublicKey="alice"
        onOpenPostDraft={onOpenPostDraft}
        onOpenCommentDraft={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Resume' }));
    expect(onOpenPostDraft).toHaveBeenCalledWith('rust');
  });

  it('clicking Resume on a comment draft calls onOpenCommentDraft(postId)', () => {
    seedCommentDraft('alice', 'post-X', { body: 'b', updatedAt: 1 });
    const onOpenCommentDraft = vi.fn();

    render(
      <DraftsView
        authorPublicKey="alice"
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={onOpenCommentDraft}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Resume' }));
    expect(onOpenCommentDraft).toHaveBeenCalledWith('post-X');
  });

  it('clicking Delete removes the draft from view AND localStorage', () => {
    const key = seedPostDraft('alice', 'rust', { title: 'goodbye', updatedAt: 1 });

    render(
      <DraftsView
        authorPublicKey="alice"
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={vi.fn()}
      />,
    );

    expect(screen.getByText('goodbye')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    expect(screen.queryByText('goodbye')).not.toBeInTheDocument();
    expect(window.localStorage.getItem(key)).toBeNull();
  });

  it('filters drafts by authorPublicKey prop', () => {
    seedPostDraft('alice', 's', { title: 'alice draft', updatedAt: 1 });
    seedPostDraft('bob', 's', { title: 'bob draft', updatedAt: 1 });

    render(
      <DraftsView
        authorPublicKey="alice"
        onOpenPostDraft={vi.fn()}
        onOpenCommentDraft={vi.fn()}
      />,
    );
    expect(screen.getByText('alice draft')).toBeInTheDocument();
    expect(screen.queryByText('bob draft')).not.toBeInTheDocument();
  });
});
