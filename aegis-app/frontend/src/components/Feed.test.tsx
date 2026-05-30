import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Feed } from './Feed';
import type { Post, SortMode } from '../types';

// Feed renders PostCard, which imports BrowserOpenURL from wailsjs.
vi.mock('../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}));

const post = (overrides: Partial<Post> = {}): Post => ({
  id: 'p-1',
  pubkey: 'pk-1',
  title: 'Hello',
  bodyPreview: 'body',
  contentCid: '',
  imageCid: '',
  thumbCid: '',
  imageMime: '',
  imageSize: 0,
  imageWidth: 0,
  imageHeight: 0,
  score: 0,
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

describe('Feed', () => {
  it('renders sort buttons', () => {
    render(
      <Feed
        posts={[]}
        sortMode={'hot' as SortMode}
        profiles={{}}
        onSortChange={vi.fn()}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    expect(screen.getByRole('button', { name: 'Hot' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'New' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Top' })).toBeInTheDocument();
  });

  it('shows the empty state when posts is empty', () => {
    render(
      <Feed
        posts={[]}
        sortMode={'hot' as SortMode}
        profiles={{}}
        onSortChange={vi.fn()}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    expect(screen.getByText('No posts yet. Be the first to post!')).toBeInTheDocument();
  });

  it('renders one PostCard per post', () => {
    render(
      <Feed
        posts={[
          post({ id: 'p-1', title: 'Post One' }),
          post({ id: 'p-2', title: 'Post Two' }),
          post({ id: 'p-3', title: 'Post Three' }),
        ]}
        sortMode={'hot' as SortMode}
        profiles={{}}
        onSortChange={vi.fn()}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    expect(screen.getByText('Post One')).toBeInTheDocument();
    expect(screen.getByText('Post Two')).toBeInTheDocument();
    expect(screen.getByText('Post Three')).toBeInTheDocument();
  });

  it('clicking a sort button calls onSortChange', () => {
    const onSortChange = vi.fn();
    render(
      <Feed
        posts={[]}
        sortMode={'hot' as SortMode}
        profiles={{}}
        onSortChange={onSortChange}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'New' }));
    expect(onSortChange).toHaveBeenCalledWith('new');

    fireEvent.click(screen.getByRole('button', { name: 'Hot' }));
    expect(onSortChange).toHaveBeenCalledWith('hot');
  });

  it('clicking Top defaults to top-week when not already in top mode', () => {
    const onSortChange = vi.fn();
    render(
      <Feed
        posts={[]}
        sortMode={'hot' as SortMode}
        profiles={{}}
        onSortChange={onSortChange}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Top' }));
    expect(onSortChange).toHaveBeenCalledWith('top-week');
  });

  it('renders top-window options only when in top mode', () => {
    const { rerender } = render(
      <Feed
        posts={[]}
        sortMode={'hot' as SortMode}
        profiles={{}}
        onSortChange={vi.fn()}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    expect(screen.queryByRole('button', { name: 'Day' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Week' })).not.toBeInTheDocument();

    rerender(
      <Feed
        posts={[]}
        sortMode={'top-week' as SortMode}
        profiles={{}}
        onSortChange={vi.fn()}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    expect(screen.getByRole('button', { name: 'Day' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Week' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Month' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'All' })).toBeInTheDocument();
  });

  it('clicking a top-window button calls onSortChange with that window', () => {
    const onSortChange = vi.fn();
    render(
      <Feed
        posts={[]}
        sortMode={'top-week' as SortMode}
        profiles={{}}
        onSortChange={onSortChange}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Day' }));
    expect(onSortChange).toHaveBeenCalledWith('top-day');

    fireEvent.click(screen.getByRole('button', { name: 'All' }));
    expect(onSortChange).toHaveBeenCalledWith('top-all');
  });

  it('passes isRecommended=true only for recommended-but-not-subscribed posts', () => {
    render(
      <Feed
        posts={[
          { ...post({ id: 'p-rec', title: 'Recommended Post' }), reason: 'recommended_hot', isSubscribed: false },
          { ...post({ id: 'p-sub', title: 'Subscribed Post' }), reason: 'recommended_hot', isSubscribed: true },
        ]}
        sortMode={'hot' as SortMode}
        profiles={{}}
        onSortChange={vi.fn()}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    // Only the not-subscribed recommended post should show the 'Recommended' badge.
    const badges = screen.getAllByText('Recommended');
    expect(badges).toHaveLength(1);
  });

  it('shows isFavorited bookmark when post.isFavorited is true', () => {
    render(
      <Feed
        posts={[{ ...post({ id: 'p-fav', title: 'Fav' }), isFavorited: true }]}
        sortMode={'hot' as SortMode}
        profiles={{}}
        onSortChange={vi.fn()}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    expect(screen.getByText('Saved')).toBeInTheDocument();
  });
});
