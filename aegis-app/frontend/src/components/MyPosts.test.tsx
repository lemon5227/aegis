import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MyPosts } from './MyPosts';

const getMyPosts = vi.fn();
vi.mock('../../wailsjs/go/main/App', () => ({
  GetMyPosts: (limit: number, cursor: string) => getMyPosts(limit, cursor),
}));
vi.mock('../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}));

const fakePost = (id: string, overrides: Record<string, unknown> = {}) => ({
  id,
  pubkey: 'pk-1',
  title: `Title ${id}`,
  bodyPreview: `body ${id}`,
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

describe('MyPosts', () => {
  beforeEach(() => {
    getMyPosts.mockReset();
    // Make the wails-runtime check pass by setting window.go.
    (window as unknown as { go: unknown }).go = { main: { App: {} } };
  });

  afterEach(() => {
    vi.clearAllMocks();
    delete (window as unknown as { go?: unknown }).go;
  });

  it('renders the page title', async () => {
    getMyPosts.mockResolvedValue({ items: [], nextCursor: '' });

    render(
      <MyPosts
        currentPubkey="pk-1"
        profiles={{}}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    expect(screen.getByText('My Posts')).toBeInTheDocument();
  });

  it('renders empty state when no posts', async () => {
    getMyPosts.mockResolvedValue({ items: [], nextCursor: '' });

    render(
      <MyPosts
        currentPubkey="pk-empty"
        profiles={{}}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("You haven't posted anything yet.")).toBeInTheDocument();
    });
  });

  it('renders one PostCard per returned post', async () => {
    getMyPosts.mockResolvedValue({
      items: [fakePost('p-1', { title: 'First Post' }), fakePost('p-2', { title: 'Second Post' })],
      nextCursor: '',
    });

    render(
      <MyPosts
        currentPubkey="pk-multi"
        profiles={{}}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText('First Post')).toBeInTheDocument();
      expect(screen.getByText('Second Post')).toBeInTheDocument();
    });
  });

  it('shows loading state initially when no cache exists', () => {
    getMyPosts.mockResolvedValue({ items: [], nextCursor: '' });

    render(
      <MyPosts
        currentPubkey="pk-fresh"
        profiles={{}}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('falls back to empty state if GetMyPosts rejects (no Wails runtime)', async () => {
    // Remove Wails runtime to exercise the no-runtime branch.
    delete (window as unknown as { go?: unknown }).go;

    render(
      <MyPosts
        currentPubkey="pk-no-runtime"
        profiles={{}}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("You haven't posted anything yet.")).toBeInTheDocument();
    });
  });

  it('handles GetMyPosts errors gracefully (no crash)', async () => {
    getMyPosts.mockRejectedValue(new Error('boom'));

    render(
      <MyPosts
        currentPubkey="pk-error"
        profiles={{}}
        onUpvote={vi.fn()}
        onPostClick={vi.fn()}
      />,
    );

    await waitFor(() => {
      // After the error, loading flips false and we land on empty state.
      expect(screen.getByText("You haven't posted anything yet.")).toBeInTheDocument();
    });
  });
});
