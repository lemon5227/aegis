import { PendingSyncAction } from '../types';

interface PendingSyncViewProps {
  actions: PendingSyncAction[];
  onOpenAction?: (action: PendingSyncAction) => void;
  onDismissAction?: (actionId: string) => void;
  onSyncNow?: () => void;
}

export function PendingSyncView({ actions, onOpenAction, onDismissAction, onSyncNow }: PendingSyncViewProps) {
  const getActionLabel = (action: PendingSyncAction) => {
    switch (action.kind) {
      case 'post-create':
      case 'post-edit':
      case 'post-delete':
        return 'Post';
      case 'comment-create':
      case 'comment-edit':
      case 'comment-delete':
      case 'comment-vote':
        return 'Comment';
      case 'post-vote':
        return 'Reaction';
      case 'profile-publish':
        return 'Profile';
      default:
        return 'Saved action';
    }
  };

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6">
      <div className="mx-auto max-w-4xl space-y-6">
        <section className="rounded-3xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark p-6">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-xs uppercase tracking-[0.18em] text-warm-text-secondary dark:text-slate-400">Background</p>
              <h1 className="mt-2 text-3xl font-bold text-warm-text-primary dark:text-white">Saved Actions</h1>
              <p className="mt-3 text-sm text-warm-text-secondary dark:text-slate-300">
                These changes are already saved on this device. The app will finish sending them in the background when it can.
              </p>
            </div>
            <button
              onClick={onSyncNow}
              className="rounded-lg bg-warm-accent px-4 py-2 text-sm font-medium text-white hover:bg-warm-accent-hover"
            >
              Sync Now
            </button>
          </div>
        </section>

        {actions.length === 0 ? (
          <div className="rounded-3xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark py-16 text-center text-warm-text-secondary dark:text-slate-400">
            <span className="material-icons-outlined text-4xl mb-3">cloud_done</span>
            <p>No saved background actions.</p>
          </div>
        ) : (
          <div className="space-y-4">
            {actions.map((action) => (
              <article
                key={action.id}
                className="rounded-2xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark p-5"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <div className="text-xs uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">{getActionLabel(action)}</div>
                    <h2 className="mt-1 text-lg font-semibold text-warm-text-primary dark:text-white">{action.summary}</h2>
                    <div className="mt-2 text-sm text-warm-text-secondary dark:text-slate-400">
                      Created {new Date(action.createdAt).toLocaleString()}
                    </div>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    {onOpenAction && (
                      <button
                        onClick={() => onOpenAction(action)}
                        className="rounded-lg bg-warm-accent px-4 py-2 text-sm font-medium text-white hover:bg-warm-accent-hover"
                      >
                        Open
                      </button>
                    )}
                    {onDismissAction && (
                      <button
                        onClick={() => onDismissAction(action.id)}
                        className="rounded-lg border border-warm-border dark:border-border-dark px-4 py-2 text-sm font-medium text-warm-text-secondary dark:text-slate-300 hover:bg-warm-bg dark:hover:bg-background-dark"
                      >
                        Dismiss
                      </button>
                    )}
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
