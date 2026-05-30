import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { NotificationsView } from './NotificationsView';
import type { Notification, Profile } from '../types';

// Mock the Wails Go bindings.
const getNotifications = vi.fn();
const markRead = vi.fn();
const markAllRead = vi.fn();
vi.mock('../../wailsjs/go/main/App', () => ({
  GetNotifications: (limit: number, cursor: string) => getNotifications(limit, cursor),
  MarkNotificationRead: (id: string) => markRead(id),
  MarkAllNotificationsRead: () => markAllRead(),
}));

// Stub IntersectionObserver — happy-dom doesn't ship it but the component
// uses it for infinite scroll. We only need a no-op so render succeeds.
class MockIntersectionObserver {
  observe = vi.fn();
  disconnect = vi.fn();
  unobserve = vi.fn();
  takeRecords = vi.fn(() => []);
  root = null;
  rootMargin = '';
  thresholds = [];
}

beforeEach(() => {
  // happy-dom does not provide IntersectionObserver; install a no-op stub
  // so the component's infinite-scroll wiring doesn't blow up on render.
  (globalThis as unknown as { IntersectionObserver: typeof IntersectionObserver }).IntersectionObserver =
    MockIntersectionObserver as unknown as typeof IntersectionObserver;
  getNotifications.mockReset();
  markRead.mockReset();
  markAllRead.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

const notif = (overrides: Partial<Notification> = {}): Notification => ({
  id: 'n-1',
  type: 'post_comment',
  sourcePubkey: 'pk-source',
  targetEntityId: 'p-1',
  targetType: 'post',
  postId: 'p-1',
  isRead: false,
  createdAt: Math.floor(Date.now() / 1000) - 30,
  ...overrides,
});

const profileMap = (entries: Record<string, Partial<Profile>>): Record<string, Profile> => {
  const out: Record<string, Profile> = {};
  for (const [pubkey, p] of Object.entries(entries)) {
    out[pubkey] = {
      pubkey,
      displayName: p.displayName ?? '',
      avatarURL: p.avatarURL ?? '',
      updatedAt: p.updatedAt ?? 0,
    };
  }
  return out;
};

describe('NotificationsView', () => {
  it('renders the page title', async () => {
    getNotifications.mockResolvedValue({ items: [], nextCursor: '' });

    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={vi.fn()}
        onNavigateToProfile={vi.fn()}
      />,
    );
    expect(screen.getByText('Notifications')).toBeInTheDocument();
  });

  it('shows empty state when no notifications', async () => {
    getNotifications.mockResolvedValue({ items: [], nextCursor: '' });

    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={vi.fn()}
        onNavigateToProfile={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText('No notifications yet')).toBeInTheDocument();
    });
  });

  it('renders one row per notification with action label', async () => {
    getNotifications.mockResolvedValue({
      items: [
        notif({ id: 'a', type: 'post_comment', sourcePubkey: 'pk-alice' }),
        notif({ id: 'b', type: 'post_upvote', sourcePubkey: 'pk-bob' }),
      ],
      nextCursor: '',
    });

    render(
      <NotificationsView
        profiles={profileMap({ 'pk-alice': { displayName: 'Alice' }, 'pk-bob': { displayName: 'Bob' } })}
        onNavigateToPost={vi.fn()}
        onNavigateToProfile={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText('Alice')).toBeInTheDocument();
      expect(screen.getByText('Bob')).toBeInTheDocument();
      expect(screen.getByText(/commented on your post/)).toBeInTheDocument();
      expect(screen.getByText(/upvoted your post/)).toBeInTheDocument();
    });
  });

  it('falls back to truncated pubkey when no profile', async () => {
    getNotifications.mockResolvedValue({
      items: [notif({ sourcePubkey: 'abcdef0123456789' })],
      nextCursor: '',
    });

    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={vi.fn()}
        onNavigateToProfile={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText('abcdef01…')).toBeInTheDocument();
    });
  });

  it('"Mark all as read" hidden when no unread', async () => {
    getNotifications.mockResolvedValue({
      items: [notif({ isRead: true }), notif({ id: 'n2', isRead: true })],
      nextCursor: '',
    });

    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={vi.fn()}
        onNavigateToProfile={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Mark all as read' })).not.toBeInTheDocument();
    });
  });

  it('"Mark all as read" visible when at least one unread', async () => {
    getNotifications.mockResolvedValue({
      items: [notif({ isRead: false })],
      nextCursor: '',
    });

    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={vi.fn()}
        onNavigateToProfile={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Mark all as read' })).toBeInTheDocument();
    });
  });

  it('clicking "Mark all as read" calls the binding and updates UI state', async () => {
    getNotifications.mockResolvedValue({
      items: [notif({ id: 'a', isRead: false }), notif({ id: 'b', isRead: false })],
      nextCursor: '',
    });
    markAllRead.mockResolvedValue(undefined);

    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={vi.fn()}
        onNavigateToProfile={vi.fn()}
      />,
    );

    const btn = await screen.findByRole('button', { name: 'Mark all as read' });
    fireEvent.click(btn);

    await waitFor(() => {
      expect(markAllRead).toHaveBeenCalledTimes(1);
      // After all are marked read, the button should disappear.
      expect(screen.queryByRole('button', { name: 'Mark all as read' })).not.toBeInTheDocument();
    });
  });

  it('clicking a post-type notification calls onNavigateToPost(postId)', async () => {
    getNotifications.mockResolvedValue({
      items: [
        notif({
          id: 'n-post',
          type: 'post_comment',
          postId: 'post-X',
          targetType: 'post',
          targetEntityId: 'post-X',
        }),
      ],
      nextCursor: '',
    });
    markRead.mockResolvedValue(undefined);

    const onNavigateToPost = vi.fn();
    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={onNavigateToPost}
        onNavigateToProfile={vi.fn()}
      />,
    );

    const row = await screen.findByText(/commented on your post/);
    fireEvent.click(row);

    await waitFor(() => {
      expect(markRead).toHaveBeenCalledWith('n-post');
      expect(onNavigateToPost).toHaveBeenCalledWith('post-X', undefined);
    });
  });

  it('clicking a comment-type notification passes commentId from targetEntityId', async () => {
    getNotifications.mockResolvedValue({
      items: [
        notif({
          id: 'n-cmt',
          type: 'comment_reply',
          postId: 'post-Y',
          targetType: 'comment',
          targetEntityId: 'cmt-Z',
        }),
      ],
      nextCursor: '',
    });
    markRead.mockResolvedValue(undefined);

    const onNavigateToPost = vi.fn();
    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={onNavigateToPost}
        onNavigateToProfile={vi.fn()}
      />,
    );

    const row = await screen.findByText(/replied to your comment/);
    fireEvent.click(row);

    await waitFor(() => {
      expect(onNavigateToPost).toHaveBeenCalledWith('post-Y', 'cmt-Z');
    });
  });

  it('clicking a governance_action notification calls onNavigateToProfile', async () => {
    getNotifications.mockResolvedValue({
      items: [notif({ id: 'g-1', type: 'governance_action' })],
      nextCursor: '',
    });
    markRead.mockResolvedValue(undefined);

    const onNavigateToProfile = vi.fn();
    const onNavigateToPost = vi.fn();
    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={onNavigateToPost}
        onNavigateToProfile={onNavigateToProfile}
      />,
    );

    const row = await screen.findByText(/governance action/);
    fireEvent.click(row);

    await waitFor(() => {
      expect(onNavigateToProfile).toHaveBeenCalledTimes(1);
      expect(onNavigateToPost).not.toHaveBeenCalled();
    });
  });

  it('does not re-call MarkNotificationRead when clicking an already-read notification', async () => {
    getNotifications.mockResolvedValue({
      items: [notif({ id: 'already-read', isRead: true, postId: 'post-X' })],
      nextCursor: '',
    });

    render(
      <NotificationsView
        profiles={{}}
        onNavigateToPost={vi.fn()}
        onNavigateToProfile={vi.fn()}
      />,
    );

    const row = await screen.findByText(/commented on your post/);
    fireEvent.click(row);

    // Allow any pending microtasks.
    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(markRead).not.toHaveBeenCalled();
  });
});
