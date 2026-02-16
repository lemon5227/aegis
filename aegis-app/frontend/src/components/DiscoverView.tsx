import { Sub } from '../types';

interface DiscoverViewProps {
  subs: Sub[];
  subscribedSubIds: Set<string>;
  onSubClick: (subId: string) => void;
  onToggleSubscription: (subId: string) => void;
}

export function DiscoverView({ subs, subscribedSubIds, onSubClick, onToggleSubscription }: DiscoverViewProps) {
  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white mb-6">
          Discover All Sub-communities
        </h1>
        
        {subs.length === 0 ? (
          <div className="text-center py-12 text-warm-text-secondary dark:text-slate-400">
            <span className="material-icons text-4xl mb-4">explore</span>
            <p>No sub-communities found. Create one to get started!</p>
          </div>
        ) : (
          <div className="grid gap-4">
            {subs.map((sub) => {
              const isSubscribed = subscribedSubIds.has(sub.id);
              return (
                <div
                  key={sub.id}
                  className="bg-warm-card dark:bg-surface-dark rounded-xl p-4 border border-warm-border dark:border-border-dark hover:border-warm-accent/40 transition-colors"
                >
                  <div className="flex items-start justify-between">
                    <div 
                      className="flex items-start gap-4 cursor-pointer flex-1"
                      onClick={() => onSubClick(sub.id)}
                    >
                      <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-warm-accent to-orange-400 flex items-center justify-center text-white shadow-lg shrink-0">
                        <span className="material-icons-outlined text-2xl">forum</span>
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <h3 className="text-lg font-bold text-warm-text-primary dark:text-white">{sub.id}</h3>
                          {isSubscribed && (
                            <span className="px-2 py-0.5 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 text-xs font-medium rounded-full">
                              Subscribed
                            </span>
                          )}
                        </div>
                        <p className="text-sm text-warm-text-secondary dark:text-slate-400 mb-2">
                          {sub.title || 'No description'}
                        </p>
                        {sub.description && (
                          <p className="text-sm text-warm-text-secondary dark:text-slate-400 line-clamp-2">
                            {sub.description}
                          </p>
                        )}
                      </div>
                    </div>
                    
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleSubscription(sub.id);
                      }}
                      className={`shrink-0 ml-4 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                        isSubscribed
                          ? 'bg-green-600 hover:bg-green-700 text-white'
                          : 'bg-warm-accent hover:bg-warm-accent-hover text-white'
                      }`}
                    >
                      {isSubscribed ? (
                        <span className="flex items-center gap-1">
                          <span className="material-icons text-sm">check</span>
                          Subscribed
                        </span>
                      ) : (
                        'Subscribe'
                      )}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
