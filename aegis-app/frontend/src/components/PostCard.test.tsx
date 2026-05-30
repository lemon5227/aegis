import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PostCard } from './PostCard';
import type { Post, Profile } from '../types';

// Mock the wails runtime so PostCard's BrowserOpenURL import does not pull
// in actual native bindings during the test run.
const browserOpenURL = vi.fn();
vi.mock('../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: (url: string) => browserOpenURL(url),
}));

const post = (overrides: Partial<Post> = {}): Post => ({
  id: 'p-1',
  pubkey: 'deadbeef-pubkey',
  title: 'A Post Title',
  bodyPreview: 'Some body preview text.',
  contentCid: 'cid-x',
  imageCid: '',
  thumbCid: '',
  imageMime: '',
  imageSize: 0,
  imageWidth: 0,
  imageHeight: 0,
  score: 5,
  timestamp: Math.floor(Date.now() / 1000),
  zone: 'public',
  subId: 'general',
  visibility: 'normal',
  isPinned: false,
  pinnedAt: 0,
  isLocked: false,
  lockedAt: 0,
  ...overrides,
});

const profile = (overrides: Partial<Profile> = {}): Profile => ({
  pubkey: 'deadbeef-pubkey',
  displayName: 'Alice',
  avatarURL: '',
  updatedAt: 1,
  ...overrides,
});

describe('PostCard', () => {
  beforeEach(() => {
    browserOpenURL.mockClear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders title, score, body preview, and sub badge', () => {
    render(
      <PostCard
        post={post({ title: 'Hello World', score: 42, bodyPreview: 'first body' })}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
      />,
    );
    expect(screen.getByText('Hello World')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByText('first body')).toBeInTheDocument();
    expect(screen.getByText('#general')).toBeInTheDocument();
  });

  it('falls back to a truncated pubkey when no profile is provided', () => {
    render(
      <PostCard
        post={post({ pubkey: 'abcdef0123456789' })}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
      />,
    );
    expect(screen.getByText('abcdef01')).toBeInTheDocument();
  });

  it('shows the profile display name when available', () => {
    render(
      <PostCard
        post={post()}
        authorProfile={profile({ displayName: 'Alice Liddell' })}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
      />,
    );
    expect(screen.getByText('Alice Liddell')).toBeInTheDocument();
  });

  it('renders the avatar image when avatarURL is set', () => {
    render(
      <PostCard
        post={post()}
        authorProfile={profile({ displayName: 'Alice', avatarURL: 'https://x/a.png' })}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
      />,
    );
    const img = screen.getByAltText('Alice') as HTMLImageElement;
    expect(img).toBeInTheDocument();
    expect(img.src).toBe('https://x/a.png');
  });

  it('clicking the article surface invokes onClick with the post', () => {
    const onClick = vi.fn();
    render(<PostCard post={post()} onUpvote={vi.fn()} onClick={onClick} />);
    fireEvent.click(screen.getByText('A Post Title'));
    expect(onClick).toHaveBeenCalledTimes(1);
    expect(onClick.mock.calls[0][0].id).toBe('p-1');
  });

  it('clicking the upvote button invokes onUpvote and stops bubbling', () => {
    const onUpvote = vi.fn();
    const onClick = vi.fn();
    const { container } = render(
      <PostCard post={post()} onUpvote={onUpvote} onClick={onClick} />,
    );

    const upvoteBtn = container.querySelector('button')!;
    fireEvent.click(upvoteBtn);
    expect(onUpvote).toHaveBeenCalledWith('p-1');
    expect(onClick).not.toHaveBeenCalled();
  });

  it('clicking the author label calls onAuthorClick with pubkey', () => {
    const onAuthorClick = vi.fn();
    render(
      <PostCard
        post={post()}
        authorProfile={profile({ displayName: 'Alice' })}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
        onAuthorClick={onAuthorClick}
      />,
    );
    fireEvent.click(screen.getByText('Alice'));
    expect(onAuthorClick).toHaveBeenCalledWith('deadbeef-pubkey');
  });

  it('clicking the share button calls onShare with the post', () => {
    const onShare = vi.fn();
    render(
      <PostCard
        post={post()}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
        onShare={onShare}
      />,
    );
    fireEvent.click(screen.getByText('Share'));
    expect(onShare).toHaveBeenCalledTimes(1);
    expect(onShare.mock.calls[0][0].id).toBe('p-1');
  });

  it('renders an unfilled bookmark when not favorited', () => {
    render(
      <PostCard post={post()} onUpvote={vi.fn()} onClick={vi.fn()} />,
    );
    expect(screen.getByText('Save')).toBeInTheDocument();
    expect(screen.getByText('bookmark_border')).toBeInTheDocument();
  });

  it('renders a filled bookmark + Saved label when favorited', () => {
    render(
      <PostCard
        post={post()}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
        isFavorited
      />,
    );
    expect(screen.getByText('Saved')).toBeInTheDocument();
    expect(screen.getByText('bookmark')).toBeInTheDocument();
  });

  it('clicking the favorite button calls onToggleFavorite', () => {
    const onToggleFavorite = vi.fn();
    render(
      <PostCard
        post={post()}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
        onToggleFavorite={onToggleFavorite}
      />,
    );
    fireEvent.click(screen.getByText('Save'));
    expect(onToggleFavorite).toHaveBeenCalledWith('p-1');
  });

  it('shows Pinned badge when post.isPinned', () => {
    render(
      <PostCard post={post({ isPinned: true })} onUpvote={vi.fn()} onClick={vi.fn()} />,
    );
    expect(screen.getByText('Pinned')).toBeInTheDocument();
  });

  it('shows Locked badge when post.isLocked', () => {
    render(
      <PostCard post={post({ isLocked: true })} onUpvote={vi.fn()} onClick={vi.fn()} />,
    );
    expect(screen.getByText('Locked')).toBeInTheDocument();
  });

  it('shows Recommended badge when isRecommended prop is set', () => {
    render(
      <PostCard
        post={post()}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
        isRecommended
      />,
    );
    expect(screen.getByText('Recommended')).toBeInTheDocument();
  });

  it('shows "Syncing content" badge when bodyPreview empty but contentCid present', () => {
    render(
      <PostCard
        post={post({ bodyPreview: '', contentCid: 'cid-pending' })}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
      />,
    );
    expect(screen.getByText('Syncing content')).toBeInTheDocument();
  });

  it('renders an external-link card for link-mode posts and opens browser on click', () => {
    render(
      <PostCard
        post={post({ bodyPreview: 'Aegis-Link: https://example.com/path\n\ncommentary' })}
        onUpvote={vi.fn()}
        onClick={vi.fn()}
      />,
    );

    expect(screen.getByText('External Link')).toBeInTheDocument();
    expect(screen.getByText('example.com')).toBeInTheDocument();

    fireEvent.click(screen.getByText('External Link').closest('button')!);
    expect(browserOpenURL).toHaveBeenCalledWith('https://example.com/path');
  });
});
