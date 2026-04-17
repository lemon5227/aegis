import { AntiEntropyStats, P2PStatus, PendingSyncAction, Sub, SubSettings, SubStats } from '../types';
import { deriveNetworkHealth, formatRelativeNetworkTime } from '../lib/networkHealth';

interface RightPanelProps {
  sub: Sub | { id: string; title: string; description: string } | undefined;
  isSubscribed: boolean;
  stats?: SubStats;
  settings?: SubSettings;
  membersCount?: number;
  onlineCount?: number;
  onCreatePost?: () => void;
  onShareSub?: () => void;
  p2pStatus?: P2PStatus | null;
  antiEntropyStats?: AntiEntropyStats | null;
  pendingSyncActions?: PendingSyncAction[];
  networkBusy?: boolean;
  onSyncNow?: () => void;
  onOpenNetworkSettings?: () => void;
  onDismissPendingSyncAction?: (actionId: string) => void;
  onToggleSubscription: () => void;
}

function formatCreatedAt(timestamp: number): string {
  if (!timestamp) return 'Created recently';
  return `Created ${new Date(timestamp * 1000).toLocaleDateString()}`;
}

export function RightPanel({
  sub,
  isSubscribed,
  stats,
  settings,
  membersCount = 0,
  onlineCount = 0,
  onCreatePost,
  onShareSub,
  p2pStatus = null,
  antiEntropyStats = null,
  pendingSyncActions = [],
  networkBusy = false,
  onSyncNow,
  onOpenNetworkSettings,
  onDismissPendingSyncAction,
  onToggleSubscription,
}: RightPanelProps) {
  const subTitle = sub?.title || sub?.id || 'General';
  const subDescription = sub?.description || 'Welcome to this community!';
  const fallbackCreatedAt = sub && 'createdAt' in sub ? sub.createdAt : 0;
  const visibleMembers = stats?.subscriberCount || membersCount;
  const postCount = stats?.postCount || 0;
  const activeAuthors = stats?.activeAuthors || membersCount;
  const recentPosts = stats?.recentPosts24h || 0;
  const subRules = settings?.rules || [];
  const announcement = settings?.announcement?.trim() || '';
  const networkHealth = deriveNetworkHealth(p2pStatus, antiEntropyStats);
  const networkBadgeClasses = {
    healthy: 'bg-green-100 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800',
    syncing: 'bg-amber-100 text-amber-700 border-amber-200 dark:bg-amber-900/20 dark:text-amber-300 dark:border-amber-800',
    degraded: 'bg-red-100 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800',
    offline: 'bg-slate-200 text-slate-700 border-slate-300 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-700',
  }[networkHealth.level];

  return (
    <aside className="w-80 bg-warm-sidebar dark:bg-surface-dark border-l border-warm-border dark:border-border-dark flex-shrink-0">
      <div className="p-6 overflow-y-auto h-full">
        <div className="bg-warm-card dark:bg-surface-lighter rounded-xl p-4 mb-6 border border-warm-border dark:border-border-dark">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-bold text-warm-text-primary dark:text-white uppercase tracking-wider">
              About Community
            </h3>
            <button className="text-warm-text-secondary hover:text-warm-text-primary transition-colors">
              <span className="material-icons-outlined text-base">more_horiz</span>
            </button>
          </div>
          
          <div className="flex items-center gap-3 mb-4">
            <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-warm-accent to-orange-400 flex items-center justify-center text-white shadow-lg shadow-orange-500/20">
              <span className="material-icons-outlined text-2xl">forum</span>
            </div>
            <div>
              <div className="font-bold text-warm-text-primary dark:text-white">{subTitle}</div>
              <div className="text-xs text-warm-text-secondary dark:text-slate-400">{formatCreatedAt(stats?.createdAt || fallbackCreatedAt)}</div>
            </div>
          </div>
          
          <p className="text-sm text-warm-text-secondary dark:text-slate-300 mb-4 leading-relaxed">
            {subDescription}
          </p>
          
          <div className="grid grid-cols-2 gap-4 border-t border-warm-border/50 dark:border-border-dark pt-4 mb-4">
            <div>
              <div className="text-lg font-bold text-warm-text-primary dark:text-white">{visibleMembers.toLocaleString()}</div>
              <div className="text-xs text-warm-text-secondary dark:text-slate-400">Members</div>
            </div>
            <div>
              <div className="text-lg font-bold text-warm-text-primary dark:text-white flex items-center gap-1">
                <span className="w-2 h-2 rounded-full bg-green-500"></span> {onlineCount.toLocaleString()}
              </div>
              <div className="text-xs text-warm-text-secondary dark:text-slate-400">Online</div>
            </div>
            <div>
              <div className="text-lg font-bold text-warm-text-primary dark:text-white">{postCount.toLocaleString()}</div>
              <div className="text-xs text-warm-text-secondary dark:text-slate-400">Posts</div>
            </div>
            <div>
              <div className="text-lg font-bold text-warm-text-primary dark:text-white">{activeAuthors.toLocaleString()}</div>
              <div className="text-xs text-warm-text-secondary dark:text-slate-400">Authors</div>
            </div>
          </div>

          <div className="mb-4 rounded-xl bg-warm-sidebar/60 dark:bg-surface-dark px-3 py-2 border border-warm-border/50 dark:border-border-dark">
            <div className="text-[11px] uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">Activity</div>
            <div className="mt-1 text-sm font-semibold text-warm-text-primary dark:text-white">
              {recentPosts.toLocaleString()} posts in the last 24 hours
            </div>
          </div>

          {announcement && (
            <div className="mb-4 rounded-xl border border-warm-border/60 dark:border-border-dark bg-warm-bg/80 dark:bg-surface-dark px-3 py-3">
              <div className="text-[11px] uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">Announcement</div>
              <div className="mt-1 text-sm text-warm-text-primary dark:text-white leading-relaxed">
                {announcement}
              </div>
            </div>
          )}

          {subRules.length > 0 && (
            <div className="mb-4 rounded-xl border border-warm-border/60 dark:border-border-dark bg-warm-bg/80 dark:bg-surface-dark px-3 py-3">
              <div className="text-[11px] uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">Community Rules</div>
              <ul className="mt-2 space-y-2">
                {subRules.map((rule, index) => (
                  <li key={`${index}-${rule}`} className="flex gap-2 text-sm text-warm-text-primary dark:text-white">
                    <span className="mt-[2px] inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-warm-card dark:bg-surface-lighter text-[11px] font-semibold text-warm-text-secondary dark:text-slate-300">
                      {index + 1}
                    </span>
                    <span className="leading-relaxed">{rule}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
          
          <button 
            onClick={onToggleSubscription}
            className={`w-full py-2 rounded-lg text-sm font-medium transition-colors shadow-sm ${
              isSubscribed
                ? 'bg-green-600 hover:bg-green-700 text-white'
                : 'bg-warm-accent hover:bg-warm-accent-hover text-white'
            }`}
          >
            {isSubscribed ? (
              <span className="flex items-center justify-center gap-2">
                <span className="material-icons text-sm">check</span>
                Subscribed
              </span>
            ) : (
              'Subscribe'
            )}
          </button>
          <div className="mt-3 grid grid-cols-2 gap-2">
            <button
              onClick={onCreatePost}
              className="rounded-lg border border-warm-border dark:border-border-dark px-3 py-2 text-xs font-medium text-warm-text-primary dark:text-white hover:bg-warm-bg dark:hover:bg-surface-dark transition-colors"
            >
              New Post
            </button>
            <button
              onClick={onShareSub}
              className="rounded-lg border border-warm-border dark:border-border-dark px-3 py-2 text-xs font-medium text-warm-text-primary dark:text-white hover:bg-warm-bg dark:hover:bg-surface-dark transition-colors"
            >
              Share Sub
            </button>
          </div>
        </div>

        <div className="bg-warm-card dark:bg-surface-lighter rounded-xl p-4 mb-6 border border-warm-border dark:border-border-dark">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-bold text-warm-text-primary dark:text-white uppercase tracking-wider">
                App Status
              </h3>
              <p className="mt-1 text-xs text-warm-text-secondary dark:text-slate-400">
                {networkHealth.summary}
              </p>
            </div>
            <span className={`shrink-0 rounded-full border px-2.5 py-1 text-[11px] font-semibold ${networkBadgeClasses}`}>
              {networkHealth.label}
            </span>
          </div>

          <div className="mt-4 rounded-xl bg-warm-sidebar/60 dark:bg-surface-dark px-3 py-3 border border-warm-border/50 dark:border-border-dark">
            <div className="text-[11px] uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">Recent activity</div>
            <div className="mt-1 text-sm font-semibold text-warm-text-primary dark:text-white">
              {formatRelativeNetworkTime(networkHealth.lastSyncAt)}
            </div>
          </div>

          <div className="mt-4 grid grid-cols-2 gap-2">
            <button
              onClick={onSyncNow}
              disabled={networkBusy}
              className="rounded-lg bg-warm-accent px-3 py-2 text-xs font-medium text-white hover:bg-warm-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {networkBusy ? 'Syncing...' : 'Sync Now'}
            </button>
            <button
              onClick={onOpenNetworkSettings}
              className="rounded-lg border border-warm-border dark:border-border-dark px-3 py-2 text-xs font-medium text-warm-text-primary dark:text-white hover:bg-warm-bg dark:hover:bg-surface-dark transition-colors"
            >
              Network Settings
            </button>
          </div>
        </div>

        {pendingSyncActions.length > 0 && (
          <div className="bg-warm-card dark:bg-surface-lighter rounded-xl p-4 mb-6 border border-amber-300 dark:border-amber-800">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-bold text-warm-text-primary dark:text-white uppercase tracking-wider">
                  Saved Actions
                </h3>
                <p className="mt-1 text-xs text-warm-text-secondary dark:text-slate-400">
                  Your recent changes are saved and will finish in the background.
                </p>
              </div>
              <span className="rounded-full bg-amber-100 px-2.5 py-1 text-[11px] font-semibold text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
                {pendingSyncActions.length}
              </span>
            </div>

            <div className="mt-4 space-y-2">
              {pendingSyncActions.slice(0, 4).map((action) => (
                <div key={action.id} className="rounded-lg border border-warm-border/70 dark:border-border-dark bg-warm-bg/70 dark:bg-surface-dark px-3 py-3">
                  <div className="text-sm font-medium text-warm-text-primary dark:text-white">{action.summary}</div>
                  <div className="mt-1 text-xs text-warm-text-secondary dark:text-slate-400">{formatRelativeNetworkTime(Math.floor(action.createdAt / 1000))}</div>
                  {onDismissPendingSyncAction && (
                    <button
                      onClick={() => onDismissPendingSyncAction(action.id)}
                      className="mt-2 text-xs font-medium text-warm-accent hover:underline"
                    >
                      Dismiss
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
        
        <div className="mb-6">
          <h3 className="text-xs font-bold text-warm-text-secondary dark:text-slate-400 uppercase tracking-wider mb-3">
            Rules
          </h3>
          <ul className="space-y-2">
            <li className="flex gap-3 text-sm text-warm-text-secondary dark:text-slate-300 p-2 hover:bg-warm-card dark:hover:bg-surface-lighter rounded-lg transition-colors cursor-default">
              <span className="font-bold text-warm-accent">1.</span>
              <span>Be respectful to others</span>
            </li>
            <li className="flex gap-3 text-sm text-warm-text-secondary dark:text-slate-300 p-2 hover:bg-warm-card dark:hover:bg-surface-lighter rounded-lg transition-colors cursor-default">
              <span className="font-bold text-warm-accent">2.</span>
              <span>No spam or self-promotion</span>
            </li>
            <li className="flex gap-3 text-sm text-warm-text-secondary dark:text-slate-300 p-2 hover:bg-warm-card dark:hover:bg-surface-lighter rounded-lg transition-colors cursor-default">
              <span className="font-bold text-warm-accent">3.</span>
              <span>Use appropriate flairs</span>
            </li>
          </ul>
        </div>
        
        <div>
          <h3 className="text-xs font-bold text-warm-text-secondary dark:text-slate-400 uppercase tracking-wider mb-3">
            Trending Tags
          </h3>
          <div className="flex flex-wrap gap-2">
            <span className="px-2 py-1 bg-warm-card dark:bg-surface-lighter text-xs text-warm-text-secondary dark:text-slate-300 rounded border border-warm-border dark:border-transparent hover:bg-warm-border dark:hover:bg-border-dark cursor-pointer transition-colors">
              #javascript
            </span>
            <span className="px-2 py-1 bg-warm-card dark:bg-surface-lighter text-xs text-warm-text-secondary dark:text-slate-300 rounded border border-warm-border dark:border-transparent hover:bg-warm-border dark:hover:bg-border-dark cursor-pointer transition-colors">
              #rustlang
            </span>
            <span className="px-2 py-1 bg-warm-card dark:bg-surface-lighter text-xs text-warm-text-secondary dark:text-slate-300 rounded border border-warm-border dark:border-transparent hover:bg-warm-border dark:hover:bg-border-dark cursor-pointer transition-colors">
              #ai
            </span>
            <span className="px-2 py-1 bg-warm-card dark:bg-surface-lighter text-xs text-warm-text-secondary dark:text-slate-300 rounded border border-warm-border dark:border-transparent hover:bg-warm-border dark:hover:bg-border-dark cursor-pointer transition-colors">
              #web3
            </span>
          </div>
        </div>
      </div>
    </aside>
  );
}
