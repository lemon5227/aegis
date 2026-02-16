import { useState } from 'react';
import { Profile, GovernanceAdmin, ModerationLog } from '../types';

interface SettingsPanelProps {
  isOpen: boolean;
  onClose: () => void;
  profile?: Profile;
  isAdmin: boolean;
  governanceAdmins: GovernanceAdmin[];
  moderationLogs?: ModerationLog[];
  onSaveProfile: (displayName: string, avatarURL: string) => void;
  onPublishProfile: (displayName: string, avatarURL: string) => void;
  onBanUser: (targetPubkey: string, reason: string) => void;
  onUnbanUser: (targetPubkey: string, reason: string) => void;
}

type Tab = 'account' | 'privacy' | 'update' | 'governance';

export function SettingsPanel({ 
  isOpen, 
  onClose,
  profile,
  isAdmin,
  governanceAdmins,
  moderationLogs = [],
  onSaveProfile,
  onPublishProfile,
  onBanUser,
  onUnbanUser
}: SettingsPanelProps) {
  const [activeTab, setActiveTab] = useState<Tab>('account');
  const [displayName, setDisplayName] = useState(profile?.displayName || '');
  const [avatarURL, setAvatarURL] = useState(profile?.avatarURL || '');
  const [bio, setBio] = useState('');
  const [publicStatus, setPublicStatus] = useState(true);
  const [showPrivateKey, setShowPrivateKey] = useState(false);
  const [banTarget, setBanTarget] = useState('');
  const [banReason, setBanReason] = useState('');
  const [governanceTab, setGovernanceTab] = useState<'banned' | 'appeals' | 'logs'>('banned');

  if (!isOpen) return null;

  const handleSave = () => {
    onSaveProfile(displayName, avatarURL);
    onPublishProfile(displayName, avatarURL);
  };

  const handleBan = () => {
    if (!banTarget.trim() || !banReason.trim()) return;
    onBanUser(banTarget.trim(), banReason.trim());
    setBanTarget('');
    setBanReason('');
  };

  const handleUnban = (pubkey: string) => {
    onUnbanUser(pubkey, 'Approved unban request');
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative w-full max-w-5xl h-[85vh] bg-warm-bg dark:bg-background-dark rounded-xl shadow-2xl overflow-hidden flex flex-col md:flex-row border border-warm-border dark:border-border-dark">
        <button 
          onClick={onClose}
          className="absolute top-4 right-4 z-50 text-warm-text-secondary dark:text-slate-400 hover:text-warm-text-primary dark:hover:text-white transition-colors"
        >
          <span className="material-icons">close</span>
        </button>
        
        <div className="w-full md:w-64 bg-warm-sidebar dark:bg-surface-dark border-r border-warm-border dark:border-border-dark flex flex-col justify-between shrink-0">
          <div>
            <div className="p-6">
              <h2 className="text-xl font-bold text-warm-text-primary dark:text-white flex items-center gap-2">
                <span className="w-8 h-8 rounded bg-warm-accent flex items-center justify-center text-white">
                  <span className="material-icons text-lg">shield</span>
                </span>
                Aegis
              </h2>
              <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-1 pl-10">Decentralized Forum</p>
            </div>
            <nav className="px-3 space-y-1">
              <button 
                onClick={() => setActiveTab('account')}
                className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${
                  activeTab === 'account'
                    ? 'bg-warm-accent/10 text-warm-accent'
                    : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                }`}
              >
                <span className="material-icons text-[20px]">manage_accounts</span>
                Account
              </button>
              <button 
                onClick={() => setActiveTab('privacy')}
                className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${
                  activeTab === 'privacy'
                    ? 'bg-warm-accent/10 text-warm-accent'
                    : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                }`}
              >
                <span className="material-icons text-[20px]">security</span>
                Privacy &amp; Keys
              </button>
              <button 
                onClick={() => setActiveTab('update')}
                className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${
                  activeTab === 'update'
                    ? 'bg-warm-accent/10 text-warm-accent'
                    : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                }`}
              >
                <span className="material-icons text-[20px]">system_update</span>
                Updates
              </button>
              {isAdmin && (
                <button 
                  onClick={() => setActiveTab('governance')}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${
                    activeTab === 'governance'
                      ? 'bg-warm-accent/10 text-warm-accent'
                      : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                  }`}
                >
                  <span className="material-icons text-[20px]">admin_panel_settings</span>
                  Governance
                </button>
              )}
            </nav>
          </div>
          <div className="p-4 border-t border-warm-border dark:border-border-dark">
            <button 
              onClick={onClose}
              className="w-full flex items-center gap-3 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
            >
              <span className="material-icons text-[20px]">logout</span>
              Sign Out
            </button>
          </div>
        </div>
        
        <div className="flex-1 flex flex-col h-full overflow-hidden bg-warm-bg dark:bg-background-dark">
          {activeTab === 'account' && (
            <div className="flex flex-col h-full">
              <header className="px-8 py-6 border-b border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark shrink-0">
                <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white">Account Settings</h1>
                <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-1">Manage your public profile and presence on Aegis.</p>
              </header>
              <div className="flex-1 overflow-y-auto p-8">
                <div className="max-w-2xl space-y-8">
                  <div className="flex items-start gap-6">
                    <div className="relative group">
                      {avatarURL ? (
                        <img 
                          alt="Profile avatar" 
                          className="w-24 h-24 rounded-full object-cover border-4 border-white dark:border-warm-surface shadow-sm" 
                          src={avatarURL} 
                        />
                      ) : (
                        <div className="w-24 h-24 rounded-full bg-warm-accent flex items-center justify-center text-white text-3xl font-bold border-4 border-white dark:border-warm-surface">
                          {displayName ? displayName.slice(0, 2).toUpperCase() : '?'}
                        </div>
                      )}
                      <button className="absolute bottom-0 right-0 bg-warm-accent hover:bg-warm-accent-hover text-white p-2 rounded-full shadow-md transition-colors border-2 border-warm-surface dark:border-warm-bg">
                        <span className="material-icons text-sm">edit</span>
                      </button>
                    </div>
                    <div className="pt-2">
                      <h3 className="text-lg font-medium text-warm-text-primary dark:text-white">Profile Photo</h3>
                      <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-1 mb-3">This will be displayed on your posts and comments.</p>
                      <div className="flex gap-3">
                        <button 
                          onClick={() => setAvatarURL('')}
                          className="px-4 py-2 text-sm font-medium text-warm-text-primary dark:text-white bg-white dark:bg-surface-dark border border-warm-border dark:border-border-dark rounded-lg hover:bg-warm-card dark:hover:bg-surface-lighter transition-colors"
                        >
                          Remove
                        </button>
                        <button className="px-4 py-2 text-sm font-medium text-warm-accent bg-warm-accent/10 border border-transparent rounded-lg hover:bg-warm-accent/20 transition-colors">
                          Upload New
                        </button>
                      </div>
                    </div>
                  </div>
                  
                  <div className="space-y-6">
                    <div className="grid gap-6 md:grid-cols-2">
                      <div>
                        <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
                          Nickname
                        </label>
                        <input
                          type="text"
                          value={displayName}
                          onChange={(e) => setDisplayName(e.target.value)}
                          className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none transition-shadow"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
                          Unique Handle
                        </label>
                        <div className="relative">
                          <span className="absolute inset-y-0 left-0 pl-3 flex items-center text-warm-text-secondary dark:text-slate-400">@</span>
                          <input
                            type="text"
                            value={profile?.pubkey.slice(0, 8) || ''}
                            disabled
                            className="w-full pl-8 pr-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-lighter/50 text-warm-text-secondary dark:text-slate-400 cursor-not-allowed"
                          />
                        </div>
                      </div>
                    </div>
                    
                    <div>
                      <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
                        Bio
                      </label>
                      <textarea
                        value={bio}
                        onChange={(e) => setBio(e.target.value)}
                        placeholder="Tell the community about yourself..."
                        rows={4}
                        className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none transition-shadow resize-none"
                      />
                      <p className="mt-1 text-xs text-warm-text-secondary dark:text-slate-400 text-right">{bio.length}/160 characters</p>
                    </div>
                    
                    <div className="flex items-center justify-between py-4 border-t border-warm-border dark:border-border-dark">
                      <div>
                        <h4 className="text-sm font-medium text-warm-text-primary dark:text-white">Public Status</h4>
                        <p className="text-xs text-warm-text-secondary dark:text-slate-400">Allow others to see when you are online.</p>
                      </div>
                      <label className="relative inline-flex items-center cursor-pointer">
                        <input 
                          type="checkbox" 
                          checked={publicStatus}
                          onChange={(e) => setPublicStatus(e.target.checked)}
                          className="sr-only peer" 
                        />
                        <div className="w-11 h-6 bg-gray-300 dark:bg-slate-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-warm-accent"></div>
                      </label>
                    </div>
                  </div>
                </div>
                
                <div className="pt-4 flex justify-end">
                  <button 
                    onClick={handleSave}
                    className="px-6 py-2.5 bg-warm-accent hover:bg-warm-accent-hover text-white font-medium rounded-lg shadow-lg shadow-warm-accent/30 transition-all transform active:scale-95"
                  >
                    Save Changes
                  </button>
                </div>
              </div>
            </div>
          )}
          
          {activeTab === 'privacy' && (
            <div className="flex flex-col h-full">
              <header className="px-8 py-6 border-b border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark shrink-0">
                <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white">Privacy &amp; Keys</h1>
                <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-1">Manage your privacy settings and view your keys.</p>
              </header>
              <div className="flex-1 overflow-y-auto p-8">
                <div className="max-w-2xl space-y-6">
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6">
                    <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white mb-4">Your Public Key</h3>
                    <div className="bg-warm-bg dark:bg-background-dark rounded-lg p-4 font-mono text-sm text-warm-text-secondary dark:text-slate-400 break-all">
                      {profile?.pubkey || 'No public key'}
                    </div>
                    <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-2">This is your unique identifier on the network.</p>
                  </div>
                  
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6">
                    <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white mb-4">Mnemonic Phrase (Backup)</h3>
                    <div className="bg-warm-bg dark:bg-background-dark rounded-lg p-4 font-mono text-sm text-warm-text-secondary dark:text-slate-400 break-all">
                      {profile?.pubkey ? '•••••••••••••••••••••••' : 'No mnemonic available'}
                    </div>
                    <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-2">Keep this secret! It's used to restore your account.</p>
                  </div>
                  
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6">
                    <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white mb-4">Privacy Settings</h3>
                    <div className="space-y-4">
                      <div className="flex items-center justify-between">
                        <div>
                          <h4 className="text-sm font-medium text-warm-text-primary dark:text-white">Show Online Status</h4>
                          <p className="text-xs text-warm-text-secondary dark:text-slate-400">Let others see when you're online</p>
                        </div>
                        <label className="relative inline-flex items-center cursor-pointer">
                          <input type="checkbox" checked={publicStatus} onChange={(e) => setPublicStatus(e.target.checked)} className="sr-only peer" />
                          <div className="w-11 h-6 bg-gray-300 dark:bg-slate-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-warm-accent"></div>
                        </label>
                      </div>
                      <div className="flex items-center justify-between">
                        <div>
                          <h4 className="text-sm font-medium text-warm-text-primary dark:text-white">Allow Search</h4>
                          <p className="text-xs text-warm-text-secondary dark:text-slate-400">Allow your profile to appear in search results</p>
                        </div>
                        <label className="relative inline-flex items-center cursor-pointer">
                          <input type="checkbox" defaultChecked className="sr-only peer" />
                          <div className="w-11 h-6 bg-gray-300 dark:bg-slate-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-warm-accent"></div>
                        </label>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
          
          {activeTab === 'update' && (
            <div className="flex flex-col h-full">
              <header className="px-8 py-6 border-b border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark shrink-0">
                <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white">Updates</h1>
                <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-1">Check for updates and view version history.</p>
              </header>
              <div className="flex-1 overflow-y-auto p-8">
                <div className="max-w-2xl space-y-6">
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6">
                    <div className="flex items-center gap-4 mb-4">
                      <div className="w-12 h-12 rounded-lg bg-warm-accent/10 flex items-center justify-center">
                        <span className="material-icons text-2xl text-warm-accent">system_update</span>
                      </div>
                      <div>
                        <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white">Aegis v1.0.0</h3>
                        <p className="text-sm text-warm-text-secondary dark:text-slate-400">Current version</p>
                      </div>
                    </div>
                    <div className="space-y-2">
                      <div className="flex items-center gap-2 text-sm text-green-600">
                        <span className="material-icons text-sm">check_circle</span>
                        <span>Latest version installed</span>
                      </div>
                    </div>
                  </div>
                  
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6">
                    <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white mb-4">What's New</h3>
                    <ul className="space-y-3 text-sm text-warm-text-secondary dark:text-slate-400">
                      <li className="flex items-start gap-2">
                        <span className="material-icons text-warm-accent text-sm mt-0.5">new_releases</span>
                        <span>Initial release with core features</span>
                      </li>
                      <li className="flex items-start gap-2">
                        <span className="material-icons text-warm-accent text-sm mt-0.5">new_releases</span>
                        <span>Decentralized forum functionality</span>
                      </li>
                      <li className="flex items-start gap-2">
                        <span className="material-icons text-warm-accent text-sm mt-0.5">new_releases</span>
                        <span>P2P networking and sync</span>
                      </li>
                      <li className="flex items-start gap-2">
                        <span className="material-icons text-warm-accent text-sm mt-0.5">new_releases</span>
                        <span>Sub communities and moderation</span>
                      </li>
                    </ul>
                  </div>
                  
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6">
                    <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white mb-4">Check for Updates</h3>
                    <p className="text-sm text-warm-text-secondary dark:text-slate-400 mb-4">
                      You're running the latest version. We'll automatically check for updates in the background.
                    </p>
                    <button className="px-4 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white rounded-lg font-medium transition-colors">
                      Check for Updates
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}
          
          {activeTab === 'governance' && isAdmin && (
            <div className="flex flex-col h-full">
              <header className="px-8 py-6 border-b border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark shrink-0 flex justify-between items-center">
                <div>
                  <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white flex items-center gap-2">
                    Governance Dashboard
                    <span className="px-2 py-0.5 rounded text-xs font-medium bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 border border-red-200 dark:border-red-800">
                      Admin
                    </span>
                  </h1>
                  <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-1">Manage community standards and user access.</p>
                </div>
              </header>
              
              <div className="px-8 pt-4 border-b border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark">
                <div className="flex space-x-6">
                  <button 
                    onClick={() => setGovernanceTab('banned')}
                    className={`pb-3 text-sm font-medium transition-colors ${
                      governanceTab === 'banned'
                        ? 'border-b-2 border-warm-accent text-warm-accent'
                        : 'border-b-2 border-transparent text-warm-text-secondary dark:text-slate-400 hover:text-warm-text-primary dark:hover:text-white'
                    }`}
                  >
                    Banned Users
                  </button>
                  <button 
                    onClick={() => setGovernanceTab('appeals')}
                    className={`pb-3 text-sm font-medium transition-colors ${
                      governanceTab === 'appeals'
                        ? 'border-b-2 border-warm-accent text-warm-accent'
                        : 'border-b-2 border-transparent text-warm-text-secondary dark:text-slate-400 hover:text-warm-text-primary dark:hover:text-white'
                    }`}
                  >
                    Unban Requests 
                    <span className="ml-2 bg-warm-accent text-white text-[10px] px-1.5 py-0.5 rounded-full">3</span>
                  </button>
                  <button 
                    onClick={() => setGovernanceTab('logs')}
                    className={`pb-3 text-sm font-medium transition-colors ${
                      governanceTab === 'logs'
                        ? 'border-b-2 border-warm-accent text-warm-accent'
                        : 'border-b-2 border-transparent text-warm-text-secondary dark:text-slate-400 hover:text-warm-text-primary dark:hover:text-white'
                    }`}
                  >
                    Operation Log
                  </button>
                </div>
              </div>
              
              <div className="flex-1 overflow-y-auto p-8">
                {governanceTab === 'banned' && (
                  <div className="space-y-6">
                    <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-4">
                      <h3 className="font-semibold text-warm-text-primary dark:text-white mb-4">Ban User</h3>
                      <div className="flex gap-3">
                        <input
                          type="text"
                          value={banTarget}
                          onChange={(e) => setBanTarget(e.target.value)}
                          placeholder="User public key"
                          className="flex-1 px-4 py-2 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
                        />
                        <input
                          type="text"
                          value={banReason}
                          onChange={(e) => setBanReason(e.target.value)}
                          placeholder="Reason"
                          className="flex-1 px-4 py-2 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
                        />
                        <button 
                          onClick={handleBan}
                          disabled={!banTarget.trim() || !banReason.trim()}
                          className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          Ban
                        </button>
                      </div>
                    </div>
                    
                    <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark overflow-hidden">
                      <div className="p-4 border-b border-warm-border dark:border-border-dark">
                        <h3 className="font-semibold text-warm-text-primary dark:text-white">Banned Users</h3>
                      </div>
                      <div className="p-4 text-center text-warm-text-secondary dark:text-slate-400">
                        No banned users
                      </div>
                    </div>
                  </div>
                )}
                
                {governanceTab === 'appeals' && (
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark overflow-hidden">
                    <div className="p-4 border-b border-warm-border dark:border-border-dark flex justify-between items-center">
                      <h3 className="font-semibold text-warm-text-primary dark:text-white">Active Unban Requests</h3>
                      <button className="text-xs text-warm-accent hover:underline">View All History</button>
                    </div>
                    <div className="p-4 text-center text-warm-text-secondary dark:text-slate-400">
                      No pending appeals
                    </div>
                  </div>
                )}
                
                {governanceTab === 'logs' && (
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark overflow-hidden">
                    <div className="p-4 border-b border-warm-border dark:border-border-dark">
                      <h3 className="font-semibold text-warm-text-primary dark:text-white">Operation Log</h3>
                    </div>
                    <div className="p-4 text-center text-warm-text-secondary dark:text-slate-400">
                      No logs yet
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
