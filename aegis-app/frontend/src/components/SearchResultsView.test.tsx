import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SearchResultsView } from './SearchResultsView';
import type { Post, Profile, Sub } from '../types';

vi.mock('../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}));

const sub = (overrides: Partial<Sub> = {}): Sub => ({
  id: 'rust',
  title: 'Rust Community',
  description: 'rusty things',
  createdAt: 1,
  ...overrides,
});

const post = (overrides: Partial<Post> = {}): Post => ({
  id: 'p-1',
  pubkey: 'pk-1',
  title: 'Title',
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

describe('SearchResultsView', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-30T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const NOW_SEC = Math.floor(new Date('2026-05-30T12:00:00Z').getTime() / 1000);

  it('renders the query header and total counts', () => {
    render(
      <SearchResultsView
        query="rust"
        subs={[sub({ id: 'rust' })]}
        posts={[post({ id: 'p1' }), post({ id: 'p2' })]}
        profiles={{}}
        onSubClick={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('Results for "rust"')).toBeInTheDocument();
    expect(screen.getByText(/2 posts · 1 subs/)).toBeInTheDocument();
  });

  it('shows "Searching across all subs" when no scopeSubId', () => {
    render(
      <SearchResultsView
        query="x"
        subs={[]}
        posts={[]}
        profiles={{}}
        onSubClick={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText(/Searching across all subs/)).toBeInTheDocument();
  });

  it('shows "Scoped to <sub>" when scopeSubId is set', () => {
    render(
      <SearchResultsView
        query="x"
        subs={[]}
        posts={[]}
        profiles={{}}
        scopeSubId="golang"
        onSubClick={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText(/Scoped to golang/)).toBeInTheDocument();
  });

  it('hides the Matching Subs section when scopeSubId is active', () => {
    render(
      <SearchResultsView
        query="x"
        subs={[sub({ id: 'rust' })]}
        posts={[]}
        profiles={{}}
        scopeSubId="rust"
        onSubClick={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.queryByText('Matching Subs')).not.toBeInTheDocument();
  });

  it('renders one tile per matching sub and clicking calls onSubClick', () => {
    const onSubClick = vi.fn();
    render(
      <SearchResultsView
        query="rust"
        subs={[sub({ id: 'rust' }), sub({ id: 'go' })]}
        posts={[]}
        profiles={{}}
        onSubClick={onSubClick}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('Matching Subs')).toBeInTheDocument();
    expect(screen.getByText('rust')).toBeInTheDocument();
    expect(screen.getByText('go')).toBeInTheDocument();

    fireEvent.click(screen.getByText('rust'));
    expect(onSubClick).toHaveBeenCalledWith('rust');
  });

  it('shows empty-posts placeholder when no posts match filters', () => {
    render(
      <SearchResultsView
        query="x"
        subs={[]}
        posts={[]}
        profiles={{}}
        onSubClick={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('No posts match the current filters.')).toBeInTheDocument();
  });

  it('filters posts by author name (case-insensitive)', () => {
    const profiles: Record<string, Profile> = {
      'pk-alice': { pubkey: 'pk-alice', displayName: 'Alice', avatarURL: '', updatedAt: 1 },
      'pk-bob': { pubkey: 'pk-bob', displayName: 'Bob', avatarURL: '', updatedAt: 1 },
    };
    render(
      <SearchResultsView
        query="x"
        subs={[]}
        posts={[
          post({ id: 'p-alice', pubkey: 'pk-alice', title: 'Alice Post' }),
          post({ id: 'p-bob', pubkey: 'pk-bob', title: 'Bob Post' }),
        ]}
        profiles={profiles}
        onSubClick={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByPlaceholderText('Filter by author'), { target: { value: 'aLi' } });

    expect(screen.getByText('Alice Post')).toBeInTheDocument();
    expect(screen.queryByText('Bob Post')).not.toBeInTheDocument();
  });

  it('filters posts by time range (Past day cuts off older posts)', () => {
    render(
      <SearchResultsView
        query="x"
        subs={[]}
        posts={[
          post({ id: 'fresh', title: 'Fresh', timestamp: NOW_SEC - 60 * 60 }), // 1h ago
          post({ id: 'old', title: 'Old', timestamp: NOW_SEC - 7 * 24 * 60 * 60 }), // 7d ago
        ]}
        profiles={{}}
        onSubClick={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByDisplayValue('All time'), { target: { value: 'day' } });

    expect(screen.getByText('Fresh')).toBeInTheDocument();
    expect(screen.queryByText('Old')).not.toBeInTheDocument();
  });

  it('sorts posts by Newest descending', () => {
    const { container } = render(
      <SearchResultsView
        query="x"
        subs={[]}
        posts={[
          post({ id: 'a', title: 'Older', timestamp: 100 }),
          post({ id: 'b', title: 'Newest', timestamp: 300 }),
          post({ id: 'c', title: 'Middle', timestamp: 200 }),
        ]}
        profiles={{}}
        onSubClick={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByDisplayValue('Relevance'), { target: { value: 'new' } });

    const titles = Array.from(container.querySelectorAll('h2')).map((el) => el.textContent);
    // h2s are: 'Results for ...', section headers, then PostCard h2s.
    const postTitles = titles.filter((t) => ['Older', 'Newest', 'Middle'].includes(t || ''));
    expect(postTitles).toEqual(['Newest', 'Middle', 'Older']);
  });

  it('sorts posts by Top score descending', () => {
    const { container } = render(
      <SearchResultsView
        query="x"
        subs={[]}
        posts={[
          post({ id: 'a', title: 'Score 5', score: 5 }),
          post({ id: 'b', title: 'Score 100', score: 100 }),
          post({ id: 'c', title: 'Score 50', score: 50 }),
        ]}
        profiles={{}}
        onSubClick={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByDisplayValue('Relevance'), { target: { value: 'top' } });

    const titles = Array.from(container.querySelectorAll('h2'))
      .map((el) => el.textContent)
      .filter((t) => t?.startsWith('Score'));
    expect(titles).toEqual(['Score 100', 'Score 50', 'Score 5']);
  });
});
