import { Sub } from '../types';

interface RightPanelProps {
  sub: Sub | { id: string; title: string; description: string } | undefined;
  isSubscribed: boolean;
  onToggleSubscription: () => void;
}

export function RightPanel({ sub, isSubscribed, onToggleSubscription }: RightPanelProps) {
  const subTitle = sub?.title || sub?.id || 'General';
  const subDescription = sub?.description || 'Welcome to this community!';

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
              <div className="text-xs text-warm-text-secondary dark:text-slate-400">Created recently</div>
            </div>
          </div>
          
          <p className="text-sm text-warm-text-secondary dark:text-slate-300 mb-4 leading-relaxed">
            {subDescription}
          </p>
          
          <div className="grid grid-cols-2 gap-4 border-t border-warm-border/50 dark:border-border-dark pt-4 mb-4">
            <div>
              <div className="text-lg font-bold text-warm-text-primary dark:text-white">-</div>
              <div className="text-xs text-warm-text-secondary dark:text-slate-400">Members</div>
            </div>
            <div>
              <div className="text-lg font-bold text-warm-text-primary dark:text-white flex items-center gap-1">
                <span className="w-2 h-2 rounded-full bg-green-500"></span> -
              </div>
              <div className="text-xs text-warm-text-secondary dark:text-slate-400">Online</div>
            </div>
          </div>
          
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
        </div>
        
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
