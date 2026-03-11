import { useState, useEffect, useCallback, useRef } from 'react';
import { Profile, Notification as NotifType } from '../types';
import { GetNotifications, MarkNotificationRead, MarkAllNotificationsRead } from '../../wailsjs/go/main/App';

interface NotificationsViewProps {
  profiles: Record<string, Profile>;
  onNavigateToPost: (postId: string, commentId?: string) => void;
  onNavigateToProfile: () => void;
}

const PAGE_SIZE = 20;

const NOTIF_META: Record<string, { icon: string; label: string; color: string }> = {
  post_comment: { icon: 'chat_bubble', label: 'commented on your post', color: 'text-blue-500' },
  comment_reply: { icon: 'reply', label: 'replied to your comment', color: 'text-indigo-500' },
  post_upvote: { icon: 'thumb_up', label: 'upvoted your post', color: 'text-green-500' },
  post_downvote: { icon: 'thumb_down', label: 'downvoted your post', color: 'text-red-400' },
  comment_upvote: { icon: 'thumb_up', label: 'upvoted your comment', color: 'text-green-500' },
  comment_downvote: { icon: 'thumb_down', label: 'downvoted your comment', color: 'text-red-400' },
  governance_action: { icon: 'gavel', label: 'governance action', color: 'text-amber-500' },
};

function formatRelativeTime(ts: number): string {
  const now = Math.floor(Date.now() / 1000);
  const diff = now - ts;
  if (diff < 60) return 'just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
  return new Date(ts * 1000).toLocaleDateString();
}

function displayName(profiles: Record<string, Profile>, pubkey: string): string {
  const p = profiles[pubkey];
  if (p?.displayName) return p.displayName;
  return pubkey.slice(0, 8) + '…';
}

export function NotificationsView({ profiles, onNavigateToPost, onNavigateToProfile }: NotificationsViewProps) {
  const [notifications, setNotifications] = useState<NotifType[]>([]);
  const [nextCursor, setNextCursor] = useState('');
  const [loading, setLoading] = useState(false);
  const [initialLoaded, setInitialLoaded] = useState(false);
  const sentinelRef = useRef<HTMLDivElement>(null);

  const loadPage = useCallback(async (cursor: string, append: boolean) => {
    setLoading(true);
    try {
      const page = await GetNotifications(PAGE_SIZE, cursor);
      const items = page.items ?? [];
      setNotifications((prev) => append ? [...prev, ...items] : items);
      setNextCursor(page.nextCursor ?? '');
    } catch (e) {
      console.error('Failed to load notifications:', e);
    } finally {
      setLoading(false);
      setInitialLoaded(true);
    }
  }, []);

  useEffect(() => {
    void loadPage('', false);
  }, [loadPage]);

  // Infinite scroll observer
  useEffect(() => {
    if (!nextCursor || loading) return;
    const el = sentinelRef.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && nextCursor) {
          void loadPage(nextCursor, true);
        }
      },
      { threshold: 0.1 },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [nextCursor, loading, loadPage]);

  const handleMarkAllRead = async () => {
    try {
      await MarkAllNotificationsRead();
      setNotifications((prev) => prev.map((n) => ({ ...n, isRead: true })));
    } catch (e) {
      console.error('Failed to mark all read:', e);
    }
  };

  const handleClick = async (n: NotifType) => {
    if (!n.isRead) {
      try {
        await MarkNotificationRead(n.id);
        setNotifications((prev) => prev.map((x) => (x.id === n.id ? { ...x, isRead: true } : x)));
      } catch (e) {
        console.error('Failed to mark notification read:', e);
      }
    }
    if (n.type === 'governance_action') {
      onNavigateToProfile();
      return;
    }
    if (n.postId) {
      const commentId = n.targetType === 'comment' ? n.targetEntityId : undefined;
      onNavigateToPost(n.postId, commentId);
    }
  };

  const hasUnread = notifications.some((n) => !n.isRead);

  return (
    <div className="flex-1 overflow-y-auto bg-warm-bg dark:bg-bg-dark">
      <div className="max-w-2xl mx-auto py-6 px-4">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-bold text-warm-text-primary dark:text-white">Notifications</h2>
          {hasUnread && (
            <button
              onClick={() => void handleMarkAllRead()}
              className="text-sm text-warm-accent hover:text-warm-accent/80 transition-colors"
            >
              Mark all as read
            </button>
          )}
        </div>

        {initialLoaded && notifications.length === 0 && (
          <div className="text-center py-16 text-warm-text-secondary dark:text-slate-400">
            <span className="material-icons-outlined text-5xl mb-3 block opacity-40">notifications_none</span>
            <p>No notifications yet</p>
          </div>
        )}

        <div className="space-y-1">
          {notifications.map((n) => {
            const meta = NOTIF_META[n.type] ?? { icon: 'info', label: n.type, color: 'text-gray-400' };
            return (
              <button
                key={n.id}
                onClick={() => void handleClick(n)}
                className={`w-full flex items-start gap-3 px-4 py-3 rounded-lg text-left transition-colors ${
                  n.isRead
                    ? 'bg-transparent hover:bg-warm-card dark:hover:bg-surface-lighter'
                    : 'bg-warm-accent/5 dark:bg-warm-accent/10 hover:bg-warm-accent/10 dark:hover:bg-warm-accent/15'
                }`}
              >
                <span className={`material-icons-outlined text-xl mt-0.5 ${meta.color}`}>{meta.icon}</span>
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-warm-text-primary dark:text-slate-200">
                    <span className="font-medium">{displayName(profiles, n.sourcePubkey)}</span>
                    {' '}{meta.label}
                  </p>
                  <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-0.5">
                    {formatRelativeTime(n.createdAt)}
                  </p>
                </div>
                {!n.isRead && (
                  <span className="w-2 h-2 rounded-full bg-warm-accent mt-2 shrink-0" />
                )}
              </button>
            );
          })}
        </div>

        <div ref={sentinelRef} className="h-8" />
        {loading && (
          <div className="text-center py-4 text-warm-text-secondary dark:text-slate-400 text-sm">Loading…</div>
        )}
      </div>
    </div>
  );
}
