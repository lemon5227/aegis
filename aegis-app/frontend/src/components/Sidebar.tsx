import { Sub } from '../types';

interface SidebarProps {
  subs: Sub[];
  subscribedSubs: Sub[];
  currentSubId: string;
  onSelectSub: (subId: string) => void;
  onDiscoverClick: () => void;
  onCreateSub: () => void;
  unreadSubs?: Set<string>;
}

export function Sidebar({ subs, subscribedSubs, currentSubId, onSelectSub, onDiscoverClick, onCreateSub, unreadSubs }: SidebarProps) {
  const isSelected = (subId: string) => currentSubId === subId;

  return (
    <aside className="w-64 bg-warm-sidebar dark:bg-surface-dark flex flex-col border-r border-warm-border dark:border-border-dark flex-shrink-0">
      <div className="h-16 flex items-center px-6 border-b border-warm-border/60 dark:border-border-dark/50">
        <div className="flex items-center gap-2 text-warm-accent">
          <span className="material-icons-round text-3xl">shield</span>
          <span className="text-xl font-bold tracking-tight text-warm-text-primary dark:text-white">Aegis</span>
        </div>
      </div>
      
      <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-1">
        <button
          onClick={() => onSelectSub('recommended')}
          className={`group flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-colors w-full text-left ${
            isSelected('recommended')
              ? 'bg-warm-accent/10 text-warm-accent border border-warm-accent/20'
              : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
          }`}
        >
          <span className="material-icons-outlined mr-3 text-lg">local_fire_department</span>
          Recommended Feed
        </button>

        {subscribedSubs.length > 0 && (
          <>
            <div className="px-3 mt-4 mb-2 text-xs font-semibold text-warm-text-secondary dark:text-slate-400 uppercase tracking-wider">
              My Subscriptions
            </div>
            {subscribedSubs.map((sub) => (
              <button
                key={sub.id}
                onClick={() => onSelectSub(sub.id)}
                className={`group flex items-center justify-between px-3 py-2.5 text-sm font-medium rounded-lg transition-colors w-full text-left ${
                  isSelected(sub.id)
                    ? 'bg-warm-accent/10 text-warm-accent border border-warm-accent/20'
                    : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                }`}
              >
                <div className="flex items-center truncate">
                  <span className="material-icons-outlined mr-3 text-lg text-green-600">check_circle</span>
                  <span className="truncate">{sub.id}</span>
                </div>
                {unreadSubs && unreadSubs.has(sub.id) && !isSelected(sub.id) && (
                  <span className="w-2 h-2 rounded-full bg-red-500 ml-2 shrink-0"></span>
                )}
              </button>
            ))}
          </>
        )}

        <button
          onClick={onDiscoverClick}
          className="group flex items-center px-3 py-2.5 text-sm font-medium rounded-lg transition-colors w-full text-left mt-4 text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white"
        >
          <span className="material-icons-outlined mr-3 text-lg">explore</span>
          Discover More
        </button>
      </nav>
      
      <div className="p-4 border-t border-warm-border/60 dark:border-border-dark/50">
        <button
          onClick={onCreateSub}
          className="w-full flex items-center justify-center gap-2 bg-warm-card dark:bg-surface-lighter hover:bg-white dark:hover:bg-border-dark text-warm-text-primary dark:text-slate-200 px-4 py-2.5 rounded-lg text-sm font-medium transition-colors border border-warm-border dark:border-border-dark shadow-sm"
        >
          <span className="material-icons-round text-sm">add_circle_outline</span>
          Create Sub
        </button>
      </div>
    </aside>
  );
}
