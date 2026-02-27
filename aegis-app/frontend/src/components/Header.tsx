import { useState, useRef, useEffect } from 'react';
import { Profile, Sub, ForumMessage } from '../types';

interface HeaderProps {
  currentSubId: string;
  profile?: Profile;
  onCreatePost: () => void;
  onProfileClick: () => void;
  onMyPostsClick: () => void;
  onFavoritesClick: () => void;
  onSignOut: () => void;
  isDark: boolean;
  onThemeToggle: () => void;
  searchQuery: string;
  searchResults: { subs: Sub[]; posts: ForumMessage[] } | null;
  onSearch: (query: string, scope?: string) => void;
  onSearchResultClick: (type: 'sub' | 'post', id: string) => void;
  onSearchClear: () => void;
}

function getInitials(name: string): string {
  return name.slice(0, 2).toUpperCase();
}

export function Header({ 
  currentSubId, 
  profile, 
  onCreatePost, 
  onProfileClick,
  onMyPostsClick,
  onFavoritesClick,
  onSignOut,
  isDark, 
  onThemeToggle,
  searchQuery,
  searchResults,
  onSearch,
  onSearchResultClick,
  onSearchClear
}: HeaderProps) {
  const [showDropdown, setShowDropdown] = useState(false);
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [searchScope, setSearchScope] = useState<'global' | 'sub'>('global');
  const searchRef = useRef<HTMLDivElement>(null);
  const userMenuRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (searchRef.current && !searchRef.current.contains(event.target as Node)) {
        setShowDropdown(false);
      }
      if (userMenuRef.current && !userMenuRef.current.contains(event.target as Node)) {
        setShowUserMenu(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    if (searchQuery.trim() && searchResults) {
      setShowDropdown(true);
    } else {
      setShowDropdown(false);
    }
  }, [searchQuery, searchResults]);

  const handleUserMenuClick = (action: () => void) => {
    action();
    setShowUserMenu(false);
  };

  const handleSearchInput = (value: string) => {
    const scope = searchScope === 'sub' && currentSubId !== 'recommended' ? currentSubId : undefined;
    onSearch(value, scope);
  };

  const toggleSearchScope = () => {
    const newScope = searchScope === 'global' ? 'sub' : 'global';
    setSearchScope(newScope);
    if (searchQuery.trim()) {
      const scope = newScope === 'sub' && currentSubId !== 'recommended' ? currentSubId : undefined;
      onSearch(searchQuery, scope);
    }
  };

  const isScopeAvailable = currentSubId !== 'recommended';

  return (
    <header className="h-16 flex items-center justify-between px-4 lg:px-6 bg-warm-bg dark:bg-background-dark sticky top-0 z-50 border-b border-warm-border dark:border-border-dark shrink-0">
      <div className="flex items-center gap-3 min-w-0">
        <h1 className="text-base lg:text-lg font-bold text-warm-text-primary dark:text-white flex items-center gap-2 whitespace-nowrap">
          {currentSubId === 'recommended' ? (
            <>
              <span className="material-icons-outlined text-xl text-warm-accent">local_fire_department</span>
              <span className="hidden sm:inline">Recommended Feed</span>
            </>
          ) : (
            <>
              <span className="text-warm-text-secondary dark:text-slate-500 font-normal">Sub:</span> 
              {currentSubId}
            </>
          )}
        </h1>
      </div>
      
      <div className="flex items-center gap-2 lg:gap-3">
        <div className="relative flex items-center" ref={searchRef}>
          {isScopeAvailable && (
            <button
              onClick={toggleSearchScope}
              className={`flex items-center justify-center h-9 px-2 rounded-l-lg border-y border-l border-warm-border dark:border-border-dark text-xs font-medium transition-colors ${
                searchScope === 'sub'
                  ? 'bg-warm-accent text-white border-warm-accent'
                  : 'bg-warm-sidebar dark:bg-surface-dark text-warm-text-secondary dark:text-slate-400 hover:text-warm-text-primary dark:hover:text-white'
              }`}
              title={searchScope === 'sub' ? `Searching in ${currentSubId}` : 'Searching everywhere'}
            >
              {searchScope === 'sub' ? currentSubId.slice(0, 3).toUpperCase() : 'ALL'}
            </button>
          )}
          
          <div className="relative w-48 sm:w-64">
            <span className={`absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none ${
              isScopeAvailable ? '' : ''
            } text-warm-text-secondary dark:text-slate-400`}>
              <span className="material-icons-outlined text-lg">search</span>
            </span>
            <input
              ref={inputRef}
              className={`w-full bg-warm-sidebar dark:bg-surface-dark text-sm py-2 pl-9 pr-8 text-warm-text-primary dark:text-white placeholder-warm-text-secondary dark:placeholder-slate-400 border border-warm-border dark:border-border-dark focus:ring-1 focus:ring-warm-accent focus:border-warm-accent transition-all ${
                isScopeAvailable ? 'rounded-r-lg border-l-0' : 'rounded-lg'
              }`}
              placeholder={searchScope === 'sub' && isScopeAvailable ? `Search in ${currentSubId}...` : "Search..."}
              type="text"
              value={searchQuery}
              onChange={(e) => handleSearchInput(e.target.value)}
              onFocus={() => searchQuery.trim() && searchResults && setShowDropdown(true)}
            />
            {searchQuery && (
              <button
                onClick={onSearchClear}
                className="absolute inset-y-0 right-0 flex items-center pr-2 text-warm-text-secondary hover:text-warm-text-primary"
              >
                <span className="material-icons text-sm">close</span>
              </button>
            )}
          </div>

          {showDropdown && searchResults && (
            <div className="absolute top-full right-0 left-auto mt-2 w-72 bg-warm-card dark:bg-surface-dark rounded-lg shadow-lg border border-warm-border dark:border-border-dark overflow-hidden z-50 max-h-96 overflow-y-auto">
              {searchResults.subs.length > 0 && (
                <div className="p-2 border-b border-warm-border dark:border-border-dark">
                  <div className="text-xs font-semibold text-warm-text-secondary dark:text-slate-400 uppercase mb-1 px-1">
                    Subs
                  </div>
                  {searchResults.subs.slice(0, 3).map((sub) => (
                    <button
                      key={sub.id}
                      onClick={() => {
                        onSearchResultClick('sub', sub.id);
                        setShowDropdown(false);
                      }}
                      className="w-full flex items-center gap-2 px-2 py-1.5 hover:bg-warm-sidebar dark:hover:bg-surface-lighter rounded-lg transition-colors text-left"
                    >
                      <span className="material-icons-outlined text-base text-warm-accent">forum</span>
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-warm-text-primary dark:text-white truncate">{sub.id}</div>
                        <div className="text-xs text-warm-text-secondary dark:text-slate-400 truncate">{sub.title}</div>
                      </div>
                    </button>
                  ))}
                </div>
              )}
              
              {searchResults.posts.length > 0 && (
                <div className="p-2">
                  <div className="text-xs font-semibold text-warm-text-secondary dark:text-slate-400 uppercase mb-1 px-1">
                    Posts
                  </div>
                  {searchResults.posts.map((post) => (
                    <button
                      key={post.id}
                      onClick={() => {
                        onSearchResultClick('post', post.id);
                        setShowDropdown(false);
                      }}
                      className="w-full flex items-start gap-2 px-2 py-2 hover:bg-warm-sidebar dark:hover:bg-surface-lighter rounded-lg transition-colors text-left"
                    >
                      <span className="material-icons-outlined text-base text-warm-text-secondary mt-0.5">article</span>
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-warm-text-primary dark:text-white truncate">{post.title}</div>
                        <div className="text-xs text-warm-text-secondary dark:text-slate-400 truncate">{post.body}</div>
                      </div>
                    </button>
                  ))}
                </div>
              )}
              
              {searchResults.subs.length === 0 && searchResults.posts.length === 0 && (
                <div className="p-4 text-center text-warm-text-secondary dark:text-slate-400 text-sm">
                  No results found
                </div>
              )}
            </div>
          )}
        </div>
        
        <button 
          onClick={onCreatePost}
          className="flex items-center gap-1 bg-warm-accent hover:bg-warm-accent-hover text-white px-2 lg:px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors shadow-sm"
        >
          <span className="material-icons-round text-sm">edit_note</span>
          <span className="hidden lg:inline">Post</span>
        </button>
        
        <button 
          onClick={onThemeToggle}
          className="text-warm-text-secondary dark:text-slate-400 hover:text-warm-accent transition-colors p-1"
          title="Toggle Theme"
        >
          <span className="material-icons-outlined text-xl">
            {isDark ? 'light_mode' : 'dark_mode'}
          </span>
        </button>
        
        <div className="relative" ref={userMenuRef}>
          <button 
            onClick={() => setShowUserMenu(!showUserMenu)}
            className="flex items-center gap-2 p-1 rounded-full hover:bg-warm-bg dark:hover:bg-surface-lighter transition-colors"
          >
            {profile?.avatarURL ? (
              <img 
                className="w-8 h-8 lg:w-9 lg:h-9 rounded-full ring-2 ring-warm-card dark:ring-surface-lighter cursor-pointer" 
                src={profile.avatarURL} 
                alt={profile.displayName || 'User'}
              />
            ) : (
              <div className="w-8 h-8 lg:w-9 lg:h-9 rounded-full ring-2 ring-warm-card dark:ring-surface-lighter bg-warm-accent flex items-center justify-center text-white font-bold text-xs lg:text-sm cursor-pointer">
                {profile?.displayName ? getInitials(profile.displayName) : '?'}
              </div>
            )}
          </button>
          
          {showUserMenu && (
            <div className="absolute right-0 top-full mt-2 w-56 bg-warm-card dark:bg-surface-dark rounded-xl shadow-xl border border-warm-border dark:border-border-dark overflow-hidden z-[100]">
              <div className="p-3 border-b border-warm-border dark:border-border-dark">
                <div className="flex items-center gap-3">
                  {profile?.avatarURL ? (
                    <img 
                      className="w-10 h-10 rounded-full" 
                      src={profile.avatarURL} 
                      alt={profile.displayName || 'User'}
                    />
                  ) : (
                    <div className="w-10 h-10 rounded-full bg-warm-accent flex items-center justify-center text-white font-bold">
                      {profile?.displayName ? getInitials(profile.displayName) : '?'}
                    </div>
                  )}
                  <div className="min-w-0">
                    <div className="font-medium text-warm-text-primary dark:text-white text-sm truncate">
                      {profile?.displayName || 'User'}
                    </div>
                    <div className="text-xs text-warm-text-secondary dark:text-slate-400 truncate">
                      {profile?.pubkey?.slice(0, 8)}...
                    </div>
                  </div>
                </div>
              </div>
              
              <div className="p-2">
                <button 
                  onClick={() => handleUserMenuClick(onProfileClick)}
                  className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-warm-bg dark:hover:bg-surface-lighter text-warm-text-primary dark:text-white transition-colors text-sm"
                >
                  <span className="material-icons text-lg">person</span>
                  Profile / Settings
                </button>
                <button 
                  onClick={() => handleUserMenuClick(onMyPostsClick)}
                  className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-warm-bg dark:hover:bg-surface-lighter text-warm-text-primary dark:text-white transition-colors text-sm"
                >
                  <span className="material-icons text-lg">article</span>
                  My Posts
                </button>
                <button 
                  onClick={() => handleUserMenuClick(onFavoritesClick)}
                  className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-warm-bg dark:hover:bg-surface-lighter text-warm-text-primary dark:text-white transition-colors text-sm"
                >
                  <span className="material-icons text-lg">star</span>
                  Favorites
                </button>
              </div>
              
              <div className="p-2 border-t border-warm-border dark:border-border-dark">
                <button 
                  onClick={() => handleUserMenuClick(onSignOut)}
                  className="w-full flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 text-red-600 transition-colors text-sm"
                >
                  <span className="material-icons text-lg">logout</span>
                  Log Out
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
