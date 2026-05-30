import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { HistoryView } from './HistoryView';

// Mock the wails Go binding for GetPostIndexByID and the runtime BrowserOpenURL.
const getPostIndexByID = vi.fn();
vi.mock('../../wailsjs/go/main/App', () => ({
  GetPostIndexByID: (id: string) => getPostIndexByID(id),
}));
vi.mock('../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}));

const fakePostIndex = (id: string, overrides: Record<string, unknown> = {}) => ({
  id,
  pubkey: `pk-${id}`,
  title: `Title ${id}`,
  bodyPreview: `body-${id}`,
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

describe('HistoryView', () => {
  beforeEach(() => {
    window.localStorage.clear();
    getPostIndexByID.mockReset();
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
  });

  it('renders the page title', async () => {
    getPostIndexByID.mockResolvedValue(null);
    render(
      <HistoryView profiles={{}} onPostClick={vi.fn()} onUpvote={vi.fn()} />,
    );
    expect(screen.getByText('Recently Viewed')).toBeInTheDocument();
  });

  it('shows empty state when no recently-viewed entries', async () => {
    render(
      <HistoryView profiles={{}} onPostClick={vi.fn()} onUpvote={vi.fn()} />,
    );
    await waitFor(() => {
      expect(screen.getByText('No recently viewed posts yet.')).toBeInTheDocument();
    });
  });

  it('renders one PostCard per resolved history entry', async () => {
    window.localStorage.setItem(
      'aegis:recently-viewed:v1',
      JSON.stringify([
        { postId: 'post-1', viewedAt: 200 },
        { postId: 'post-2', viewedAt: 100 },
      ]),
    );
    getPostIndexByID.mockImplementation(async (id: string) => fakePostIndex(id));

    render(
      <HistoryView profiles={{}} onPostClick={vi.fn()} onUpvote={vi.fn()} />,
    );

    await waitFor(() => {
      expect(screen.getByText('Title post-1')).toBeInTheDocument();
      expect(screen.getByText('Title post-2')).toBeInTheDocument();
    });
  });

  it('skips entries that GetPostIndexByID rejects', async () => {
    window.localStorage.setItem(
      'aegis:recently-viewed:v1',
      JSON.stringify([
        { postId: 'good', viewedAt: 200 },
        { postId: 'broken', viewedAt: 100 },
      ]),
    );
    getPostIndexByID.mockImplementation(async (id: string) => {
      if (id === 'broken') {
        throw new Error('not found');
      }
      return fakePostIndex(id);
    });

    render(
      <HistoryView profiles={{}} onPostClick={vi.fn()} onUpvote={vi.fn()} />,
    );

    await waitFor(() => {
      expect(screen.getByText('Title good')).toBeInTheDocument();
    });
    expect(screen.queryByText('Title broken')).not.toBeInTheDocument();
  });

  it('orders posts by viewedAt DESC', async () => {
    window.localStorage.setItem(
      'aegis:recently-viewed:v1',
      JSON.stringify([
        { postId: 'oldest', viewedAt: 100 },
        { postId: 'newest', viewedAt: 300 },
        { postId: 'middle', viewedAt: 200 },
      ]),
    );
    getPostIndexByID.mockImplementation(async (id: string) => fakePostIndex(id));

    const { container } = render(
      <HistoryView profiles={{}} onPostClick={vi.fn()} onUpvote={vi.fn()} />,
    );

    await waitFor(() => {
      expect(screen.getByText('Title oldest')).toBeInTheDocument();
      expect(screen.getByText('Title newest')).toBeInTheDocument();
      expect(screen.getByText('Title middle')).toBeInTheDocument();
    });

    const titles = Array.from(container.querySelectorAll('h2')).map((h) => h.textContent);
    expect(titles).toEqual(['Title newest', 'Title middle', 'Title oldest']);
  });

  it('shows "Clear History" only when there are posts', async () => {
    // Empty case.
    const { rerender } = render(
      <HistoryView profiles={{}} onPostClick={vi.fn()} onUpvote={vi.fn()} />,
    );
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Clear History' })).not.toBeInTheDocument();
    });

    // Add a post and refresh.
    window.localStorage.setItem(
      'aegis:recently-viewed:v1',
      JSON.stringify([{ postId: 'p-1', viewedAt: 1 }]),
    );
    getPostIndexByID.mockResolvedValue(fakePostIndex('p-1'));

    rerender(
      <HistoryView profiles={{}} onPostClick={vi.fn()} onUpvote={vi.fn()} refreshToken={1} />,
    );

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Clear History' })).toBeInTheDocument();
    });
  });

  it('clicking Clear History wipes localStorage AND empties the list', async () => {
    window.localStorage.setItem(
      'aegis:recently-viewed:v1',
      JSON.stringify([{ postId: 'p-1', viewedAt: 1 }]),
    );
    getPostIndexByID.mockResolvedValue(fakePostIndex('p-1'));

    render(
      <HistoryView profiles={{}} onPostClick={vi.fn()} onUpvote={vi.fn()} />,
    );

    await waitFor(() => {
      expect(screen.getByText('Title p-1')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Clear History' }));

    expect(window.localStorage.getItem('aegis:recently-viewed:v1')).toBeNull();
    expect(screen.queryByText('Title p-1')).not.toBeInTheDocument();
    expect(screen.getByText('No recently viewed posts yet.')).toBeInTheDocument();
  });
});
