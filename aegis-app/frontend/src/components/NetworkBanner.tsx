import { PendingSyncAction } from '../types';
import { NetworkHealthSnapshot } from '../lib/networkHealth';

interface NetworkBannerProps {
  health: NetworkHealthSnapshot;
  pendingSyncActions: PendingSyncAction[];
  busy?: boolean;
  onSyncNow?: () => void;
  onOpenNetworkSettings?: () => void;
}

export function NetworkBanner({
  health,
  pendingSyncActions,
  busy = false,
  onSyncNow,
  onOpenNetworkSettings,
}: NetworkBannerProps) {
  const shouldShow = health.level !== 'healthy' || pendingSyncActions.length > 0;
  if (!shouldShow) {
    return null;
  }

  const palette = {
    offline: 'border-slate-300 bg-slate-100 text-slate-800 dark:border-slate-700 dark:bg-slate-900/80 dark:text-slate-200',
    degraded: 'border-red-300 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-950/60 dark:text-red-200',
    syncing: 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-800 dark:bg-amber-950/50 dark:text-amber-200',
    healthy: 'border-green-300 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-950/40 dark:text-green-200',
  }[health.level];

  return (
    <div className={`border-b px-4 py-3 ${palette}`}>
      <div className="mx-auto flex max-w-7xl items-center justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-center gap-2 text-sm font-semibold">
            <span className="material-icons-outlined text-base">
              {health.level === 'offline' ? 'wifi_off' : health.level === 'degraded' ? 'sync_problem' : 'sync'}
            </span>
            <span>{health.label}</span>
            {pendingSyncActions.length > 0 && health.level === 'healthy' && (
              <span className="rounded-full border border-current/20 px-2 py-0.5 text-[11px] font-semibold">
                {pendingSyncActions.length} saved
              </span>
            )}
          </div>
          <div className="mt-1 text-xs opacity-90">
            {health.summary}
            {pendingSyncActions.length > 0 && ' Your recent actions are safely saved.'}
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          <button
            onClick={onOpenNetworkSettings}
            className="rounded-lg border border-current/20 px-3 py-1.5 text-xs font-medium hover:bg-black/5 dark:hover:bg-white/5"
          >
            Network
          </button>
          <button
            onClick={onSyncNow}
            disabled={busy}
            className="rounded-lg bg-warm-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-warm-accent-hover disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {busy ? 'Syncing...' : 'Sync Now'}
          </button>
        </div>
      </div>
    </div>
  );
}
