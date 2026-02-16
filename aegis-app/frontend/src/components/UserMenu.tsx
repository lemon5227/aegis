import { useState, useRef, useEffect } from 'react';
import { Profile } from '../types';

interface UserMenuProps {
  profile?: Profile;
  isAdmin: boolean;
  onProfileClick: () => void;
  onMyPostsClick: () => void;
  onFavoritesClick: () => void;
  onSettingsClick: () => void;
  onGovernanceClick: () => void;
  onSignOut: () => void;
}

function getInitials(name: string): string {
  return name.slice(0, 2).toUpperCase();
}

export function UserMenu({ 
  profile, 
  isAdmin,
  onProfileClick, 
  onMyPostsClick, 
  onFavoritesClick,
  onSettingsClick, 
  onGovernanceClick,
  onSignOut 
}: UserMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const displayName = profile?.displayName || 'User';

  return (
    <div className="relative" ref={menuRef}>
      <button 
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-3 p-1 pl-3 rounded-full hover:bg-warm-bg dark:hover:bg-surface-lighter transition-colors border border-transparent hover:border-warm-border"
      >
        <div className="text-right hidden sm:block">
          <p className="text-sm font-semibold text-warm-text-primary dark:text-white">{displayName}</p>
          <p className="text-xs text-warm-accent">Level 1 Member</p>
        </div>
        <div className="relative">
          {profile?.avatarURL ? (
            <img 
              className="w-10 h-10 rounded-full object-cover border-2 border-white dark:border-warm-surface shadow-sm" 
              src={profile.avatarURL} 
              alt={displayName}
            />
          ) : (
            <div className="w-10 h-10 rounded-full bg-warm-accent flex items-center justify-center text-white font-bold border-2 border-white dark:border-warm-surface">
              {getInitials(displayName)}
            </div>
          )}
          <div className="absolute bottom-0 right-0 w-3 h-3 bg-green-500 rounded-full border-2 border-warm-surface dark:border-warm-bg"></div>
        </div>
        <span className="material-icons text-warm-text-secondary dark:text-slate-400 transition-transform">
          {isOpen ? 'expand_less' : 'expand_more'}
        </span>
      </button>
      
      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-64 bg-warm-surface dark:bg-surface-dark rounded-xl shadow-soft border border-warm-border dark:border-border-dark overflow-hidden z-50">
          <div className="p-4 border-b border-warm-border dark:border-border-dark">
            <p className="text-xs font-bold text-warm-text-secondary dark:text-slate-400 uppercase tracking-wider mb-2">
              My Account
            </p>
            <button 
              onClick={() => { onProfileClick(); setIsOpen(false); }}
              className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-warm-bg dark:hover:bg-surface-lighter text-warm-text-primary dark:text-white transition-colors"
            >
              <span className="material-icons text-xl text-warm-accent/80">person</span>
              <span className="text-sm font-medium">Profile</span>
            </button>
            <button 
              onClick={() => { onMyPostsClick(); setIsOpen(false); }}
              className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-warm-bg dark:hover:bg-surface-lighter text-warm-text-primary dark:text-white transition-colors"
            >
              <span className="material-icons text-xl text-warm-accent/80">article</span>
              <span className="text-sm font-medium">My Posts</span>
            </button>
            <button 
              onClick={() => { onFavoritesClick(); setIsOpen(false); }}
              className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-warm-bg dark:hover:bg-surface-lighter text-warm-text-primary dark:text-white transition-colors"
            >
              <span className="material-icons text-xl text-warm-accent/80">star</span>
              <span className="text-sm font-medium">My Favorites</span>
            </button>
          </div>
          
          <div className="p-4 bg-warm-bg/50 dark:bg-background-dark/50">
            <p className="text-xs font-bold text-warm-text-secondary dark:text-slate-400 uppercase tracking-wider mb-2">
              System
            </p>
            {isAdmin && (
              <button 
                onClick={() => { onGovernanceClick(); setIsOpen(false); }}
                className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-warm-border/30 text-warm-text-primary dark:text-white transition-colors"
              >
                <span className="material-icons text-xl">build</span>
                <span className="text-sm font-medium">Governance Panel</span>
              </button>
            )}
            <button 
              onClick={() => { onSettingsClick(); setIsOpen(false); }}
              className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-warm-border/30 text-warm-text-primary dark:text-white transition-colors"
            >
              <span className="material-icons text-xl">settings</span>
              <span className="text-sm font-medium">Settings</span>
            </button>
            <div className="mt-2 pt-2 border-t border-warm-border dark:border-border-dark">
              <button 
                onClick={() => { onSignOut(); setIsOpen(false); }}
                className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-red-50 text-red-600 transition-colors"
              >
                <span className="material-icons text-xl">logout</span>
                <span className="text-sm font-medium">Log Out</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
