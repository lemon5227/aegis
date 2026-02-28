import { ChangeEvent, useEffect, useRef, useState } from 'react';
import { AntiEntropyStats, EntityOpRecord, GovernanceAdmin, ModerationLog, ModerationState, Profile, TombstoneGCResult } from '../types';
import { CheckForUpdates, GetAntiEntropyStats, GetGovernancePolicy, GetP2PConfig, GetP2PStatus, GetPrivacySettings, GetStorageUsage, GetVersionHistory, ListEntityOps, ResetLocalTestData, RunTombstoneGC, SaveP2PConfig, SetGovernancePolicy, SetPrivacySettings, StartP2P, StopP2P, UpdateProfileDetails } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

interface SettingsPanelProps {
  isOpen: boolean;
  onClose: () => void;
  profile?: Profile;
  isAdmin: boolean;
  governanceAdmins: GovernanceAdmin[];
  moderationStates?: ModerationState[];
  moderationLogs?: ModerationLog[];
  onSaveProfile: (displayName: string, avatarURL: string, bio: string) => Promise<void> | void;
  onPublishProfile: (displayName: string, avatarURL: string) => void;
  onBanUser: (targetPubkey: string, reason: string) => void;
  onUnbanUser: (targetPubkey: string, reason: string) => void;
  consistencyFocus?: {
    entityType: 'post' | 'comment';
    entityId: string;
    nonce: number;
  } | null;
  isDevMode: boolean;
}

type Tab = 'account' | 'privacy' | 'network' | 'update' | 'consistency' | 'governance';

type P2PStatusView = {
  started: boolean;
  peerId: string;
  listenAddrs: string[];
  connectedPeers: string[];
  topic: string;
};

type StorageUsageView = {
  privateUsedBytes: number;
  publicUsedBytes: number;
  privateQuota: number;
  publicQuota: number;
  totalQuota: number;
};

type UpdateStatusView = {
  currentVersion: string;
  latestVersion: string;
  hasUpdate: boolean;
  releaseURL: string;
  releaseNotes: string;
  publishedAt: number;
  checkedAt: number;
  errorMessage: string;
};

type VersionHistoryItemView = {
  version: string;
  publishedAt: number;
  summary: string;
  url: string;
};

export function SettingsPanel({
  isOpen,
  onClose,
  profile,
  isAdmin,
  governanceAdmins,
  moderationStates = [],
  moderationLogs = [],
  onSaveProfile,
  onPublishProfile,
  onBanUser,
  onUnbanUser,
  consistencyFocus,
  isDevMode,
}: SettingsPanelProps) {
  const [activeTab, setActiveTab] = useState<Tab>('account');
  const [displayName, setDisplayName] = useState(profile?.displayName || '');
  const [avatarURL, setAvatarURL] = useState(profile?.avatarURL || '');
  const [bio, setBio] = useState(profile?.bio || '');
  const [publicStatus, setPublicStatus] = useState(true);
  const [allowSearch, setAllowSearch] = useState(true);
  const [privacyBusy, setPrivacyBusy] = useState(false);
  const [privacyMessage, setPrivacyMessage] = useState('');
  const [showPrivateKey, setShowPrivateKey] = useState(false);
  const [accountBusy, setAccountBusy] = useState(false);
  const [accountMessage, setAccountMessage] = useState('');
  const [accountToast, setAccountToast] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [avatarExternalURL, setAvatarExternalURL] = useState('');
  const [banTarget, setBanTarget] = useState('');
  const [banReason, setBanReason] = useState('');
  const [governanceTab, setGovernanceTab] = useState<'banned' | 'appeals' | 'logs'>('banned');
  const [hideHistoryOnShadowBan, setHideHistoryOnShadowBan] = useState(true);
  const [governancePolicyBusy, setGovernancePolicyBusy] = useState(false);
  const [governancePolicyMessage, setGovernancePolicyMessage] = useState('');
  const [p2pListenPort, setP2PListenPort] = useState('40100');
  const [p2pRelayPeersInput, setP2PRelayPeersInput] = useState('');
  const [p2pAutoStart, setP2PAutoStart] = useState(true);
  const [p2pStatus, setP2PStatus] = useState<P2PStatusView | null>(null);
  const [p2pBusy, setP2PBusy] = useState(false);
  const [p2pMessage, setP2PMessage] = useState('');
  const [storageUsage, setStorageUsage] = useState<StorageUsageView | null>(null);
  const [storageLoading, setStorageLoading] = useState(false);
  const [storageMessage, setStorageMessage] = useState('');
  const [resetBusy, setResetBusy] = useState(false);
  const [resetConfirmArmed, setResetConfirmArmed] = useState(false);
  const [updateStatus, setUpdateStatus] = useState<UpdateStatusView | null>(null);
  const [versionHistory, setVersionHistory] = useState<VersionHistoryItemView[]>([]);
  const [updateBusy, setUpdateBusy] = useState(false);
  const [updateMessage, setUpdateMessage] = useState('');
  const [antiEntropyStats, setAntiEntropyStats] = useState<AntiEntropyStats | null>(null);
  const [consistencyBusy, setConsistencyBusy] = useState(false);
  const [consistencyMessage, setConsistencyMessage] = useState('');
  const [entityOps, setEntityOps] = useState<EntityOpRecord[]>([]);
  const [entityOpTypeFilter, setEntityOpTypeFilter] = useState<'post' | 'comment' | ''>('');
  const [entityOpIDFilter, setEntityOpIDFilter] = useState('');
  const [entityOpLimit, setEntityOpLimit] = useState('200');
  const [gcRetentionDays, setGCRetentionDays] = useState('30');
  const [gcStablePasses, setGCStablePasses] = useState('2');
  const [gcBatchSize, setGCBatchSize] = useState('200');
  const [gcResult, setGCResult] = useState<TombstoneGCResult | null>(null);
  const [gcConfirmArmed, setGCConfirmArmed] = useState(false);
  const avatarFileInputRef = useRef<HTMLInputElement | null>(null);
  const accountToastTimerRef = useRef<number | null>(null);

  const hasWailsRuntime = () => {
    return !!(window as any)?.go?.main?.App;
  };

  const loadAntiEntropyState = async () => {
    if (!hasWailsRuntime()) return;
    try {
      const stats = await GetAntiEntropyStats();
      setAntiEntropyStats(stats);
      setConsistencyMessage('');
    } catch (error) {
      console.error('Failed to load anti-entropy stats:', error);
      setConsistencyMessage('Failed to load anti-entropy stats.');
    }
  };

  const loadEntityOps = async () => {
    await loadEntityOpsWithFilters(entityOpTypeFilter, entityOpIDFilter, entityOpLimit);
  };

  const loadEntityOpsWithFilters = async (entityType: 'post' | 'comment' | '', entityID: string, limitValue: string) => {
    const limit = Number.parseInt(limitValue, 10);
    const effectiveLimit = Number.isFinite(limit) && limit > 0 ? Math.min(limit, 5000) : 200;

    setConsistencyBusy(true);
    try {
      const rows = await ListEntityOps(entityType, entityID.trim(), effectiveLimit);
      setEntityOps(Array.isArray(rows) ? rows : []);
      setConsistencyMessage('');
    } catch (error: any) {
      const errMsg = String(error).toLowerCase();
      if (errMsg.includes('dev mode') || errMsg.includes('dev-only')) {
        setEntityOps([]);
      } else {
        console.error('Failed to load entity ops:', error);
        setConsistencyMessage('Failed to load operation timeline.');
      }
    } finally {
      setConsistencyBusy(false);
    }
  };

  const handleRunTombstoneGC = async () => {
    const retentionDays = Number.parseInt(gcRetentionDays, 10);
    const stablePasses = Number.parseInt(gcStablePasses, 10);
    const batchSize = Number.parseInt(gcBatchSize, 10);
    if (!Number.isFinite(retentionDays) || retentionDays <= 0) {
      setConsistencyMessage('Retention days must be a positive integer.');
      return;
    }
    if (!Number.isFinite(stablePasses) || stablePasses <= 0) {
      setConsistencyMessage('Stable passes must be a positive integer.');
      return;
    }
    if (!Number.isFinite(batchSize) || batchSize <= 0 || batchSize > 2000) {
      setConsistencyMessage('Batch size must be in 1..2000.');
      return;
    }

    if (!gcConfirmArmed) {
      setGCConfirmArmed(true);
      setConsistencyMessage('Click Run Tombstone GC again within 8 seconds to confirm.');
      window.setTimeout(() => {
        setGCConfirmArmed(false);
      }, 8000);
      return;
    }

    setConsistencyBusy(true);
    try {
      const result = await RunTombstoneGC(retentionDays, stablePasses, batchSize);
      setGCResult(result);
      setGCConfirmArmed(false);
      setConsistencyMessage('Tombstone GC completed.');
      await loadEntityOps();
    } catch (error) {
      console.error('Failed to run tombstone GC:', error);
      setConsistencyMessage('Failed to run tombstone GC.');
    } finally {
      setConsistencyBusy(false);
    }
  };

  const parsePeerInput = (raw: string): string[] => {
    return Array.from(
      new Set(
        raw
          .split(/[\n,;]+/)
          .map((item) => item.trim())
          .filter((item) => item.length > 0)
      )
    );
  };

  const loadP2PState = async () => {
    if (!hasWailsRuntime()) return;
    try {
      const [cfg, status] = await Promise.all([GetP2PConfig(), GetP2PStatus()]);
      setP2PListenPort(String(cfg.listenPort || 40100));
      setP2PRelayPeersInput((cfg.relayPeers || []).join('\n'));
      setP2PAutoStart(!!cfg.autoStart);
      setP2PStatus(status);
    } catch (error) {
      console.error('Failed to load p2p settings:', error);
      setP2PMessage('Failed to load P2P configuration.');
    }
  };

  const loadStorageUsage = async () => {
    if (!hasWailsRuntime()) return;
    setStorageLoading(true);
    try {
      const usage = await GetStorageUsage();
      setStorageUsage(usage);
      setStorageMessage('');
    } catch (error) {
      console.error('Failed to load storage usage:', error);
      setStorageMessage('Failed to load storage usage.');
    } finally {
      setStorageLoading(false);
    }
  };

  const loadUpdateData = async () => {
    if (!hasWailsRuntime()) return;
    setUpdateBusy(true);
    try {
      const [status, history] = await Promise.all([CheckForUpdates(), GetVersionHistory(8)]);
      setUpdateStatus(status);
      setVersionHistory(history || []);
      setUpdateMessage(status?.errorMessage || '');
    } catch (error) {
      console.error('Failed to load update data:', error);
      setUpdateMessage('Failed to load update data.');
    } finally {
      setUpdateBusy(false);
    }
  };

  const loadPrivacySettings = async () => {
    if (!hasWailsRuntime()) return;
    try {
      const settings = await GetPrivacySettings();
      setPublicStatus(!!settings.showOnlineStatus);
      setAllowSearch(!!settings.allowSearch);
      setPrivacyMessage('');
    } catch (error) {
      console.error('Failed to load privacy settings:', error);
      setPrivacyMessage('Failed to load privacy settings.');
    }
  };

  const loadGovernancePolicy = async () => {
    if (!isAdmin || !hasWailsRuntime()) return;
    try {
      const policy = await GetGovernancePolicy();
      setHideHistoryOnShadowBan(!!policy.hideHistoryOnShadowBan);
      setGovernancePolicyMessage('');
    } catch (error) {
      console.error('Failed to load governance policy:', error);
      setGovernancePolicyMessage('Failed to load governance policy.');
    }
  };

  useEffect(() => {
    if (!isOpen || !hasWailsRuntime()) {
      return;
    }

    void loadP2PState();
    void loadStorageUsage();
    void loadPrivacySettings();
    void loadGovernancePolicy();
    void loadUpdateData();
    void loadAntiEntropyState();
    void loadEntityOps();
    const unsubscribe = EventsOn('p2p:updated', () => {
      void loadP2PState();
    });
    return () => {
      unsubscribe();
    };
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen || !consistencyFocus || !consistencyFocus.entityId.trim()) {
      return;
    }
    setActiveTab('consistency');
    setEntityOpTypeFilter(consistencyFocus.entityType);
    setEntityOpIDFilter(consistencyFocus.entityId);
    void loadEntityOpsWithFilters(consistencyFocus.entityType, consistencyFocus.entityId, entityOpLimit);
  }, [isOpen, consistencyFocus?.nonce]);

  const formatBytes = (bytes: number): string => {
    if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    let value = bytes;
    let idx = 0;
    while (value >= 1024 && idx < units.length - 1) {
      value /= 1024;
      idx += 1;
    }
    return `${value.toFixed(value >= 100 || idx === 0 ? 0 : 1)} ${units[idx]}`;
  };

  const formatDateTime = (ts: number): string => {
    if (!ts) return 'Unknown';
    try {
      return new Date(ts * 1000).toLocaleString();
    } catch {
      return 'Unknown';
    }
  };

  const usagePercent = (used: number, quota: number): number => {
    if (!Number.isFinite(quota) || quota <= 0) return 0;
    return Math.max(0, Math.min(100, (used / quota) * 100));
  };

  const handleResetLocalTestData = async () => {
    if (!hasWailsRuntime()) return;

    if (!resetConfirmArmed) {
      setResetConfirmArmed(true);
      setStorageMessage('Click "Reset Local Data" again within 8 seconds to confirm.');
      window.setTimeout(() => {
        setResetConfirmArmed(false);
      }, 8000);
      return;
    }

    setResetBusy(true);
    try {
      await ResetLocalTestData();
      await Promise.all([loadStorageUsage(), loadP2PState(), loadGovernancePolicy()]);
      setStorageMessage('Local test data has been reset.');
      setResetConfirmArmed(false);
    } catch (error) {
      console.error('Failed to reset local test data:', error);
      setStorageMessage('Failed to reset local test data.');
    } finally {
      setResetBusy(false);
    }
  };

  useEffect(() => {
    setDisplayName(profile?.displayName || '');
    setAvatarURL(profile?.avatarURL || '');
    setBio(profile?.bio || '');
  }, [profile?.displayName, profile?.avatarURL, profile?.bio]);

  useEffect(() => {
    return () => {
      if (accountToastTimerRef.current !== null) {
        window.clearTimeout(accountToastTimerRef.current);
      }
    };
  }, []);

  if (!isOpen) return null;

  const showAccountToast = (type: 'success' | 'error', text: string) => {
    if (accountToastTimerRef.current !== null) {
      window.clearTimeout(accountToastTimerRef.current);
    }
    setAccountToast({ type, text });
    accountToastTimerRef.current = window.setTimeout(() => {
      setAccountToast(null);
      accountToastTimerRef.current = null;
    }, 2200);
  };

  const handleSave = async () => {
    const startedAt = Date.now();
    setAccountBusy(true);
    setAccountMessage('');
    try {
      await UpdateProfileDetails(displayName, avatarURL, bio);
      await onPublishProfile(displayName, avatarURL);
      showAccountToast('success', 'Profile updated successfully.');

      onSaveProfile(displayName, avatarURL, bio);
    } catch (error) {
      console.error('Failed to save profile:', error);
      showAccountToast('error', 'Failed to save profile.');
    } finally {
      const elapsed = Date.now() - startedAt;
      if (elapsed < 600) {
        await new Promise((resolve) => window.setTimeout(resolve, 600 - elapsed));
      }
      setAccountBusy(false);
    }
  };

  const handleSelectAvatarFile = () => {
    avatarFileInputRef.current?.click();
  };

  const readFileAsDataURL = (file: File): Promise<string> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = typeof reader.result === 'string' ? reader.result : '';
        if (!result) {
          reject(new Error('empty file result'));
          return;
        }
        resolve(result);
      };
      reader.onerror = () => reject(reader.error || new Error('read file failed'));
      reader.readAsDataURL(file);
    });
  };

  const loadImageFromDataURL = (dataURL: string): Promise<HTMLImageElement> => {
    return new Promise((resolve, reject) => {
      const image = new Image();
      image.onload = () => resolve(image);
      image.onerror = () => reject(new Error('load image failed'));
      image.src = dataURL;
    });
  };

  const canvasToBlob = (canvas: HTMLCanvasElement, type: string, quality?: number): Promise<Blob> => {
    return new Promise((resolve, reject) => {
      canvas.toBlob(
        (blob) => {
          if (!blob) {
            reject(new Error('convert canvas failed'));
            return;
          }
          resolve(blob);
        },
        type,
        quality
      );
    });
  };

  const compressAvatarFileIfNeeded = async (file: File): Promise<string> => {
    const MAX_BYTES = 256 * 1024;
    const MAX_DIMENSION = 512;

    const originalDataURL = await readFileAsDataURL(file);
    const image = await loadImageFromDataURL(originalDataURL);
    const needResize = image.width > MAX_DIMENSION || image.height > MAX_DIMENSION;
    if (!needResize && file.size <= MAX_BYTES) {
      return originalDataURL;
    }

    const scale = Math.min(1, MAX_DIMENSION / Math.max(image.width, image.height));
    const targetWidth = Math.max(1, Math.round(image.width * scale));
    const targetHeight = Math.max(1, Math.round(image.height * scale));

    const canvas = document.createElement('canvas');
    canvas.width = targetWidth;
    canvas.height = targetHeight;
    const ctx = canvas.getContext('2d');
    if (!ctx) {
      throw new Error('canvas context unavailable');
    }
    ctx.drawImage(image, 0, 0, targetWidth, targetHeight);

    const qualityCandidates = [0.9, 0.82, 0.74, 0.66, 0.58, 0.5, 0.42, 0.36];
    for (const quality of qualityCandidates) {
      const blob = await canvasToBlob(canvas, 'image/jpeg', quality);
      if (blob.size <= MAX_BYTES) {
        return await readFileAsDataURL(new File([blob], 'avatar.jpg', { type: 'image/jpeg' }));
      }
    }

    throw new Error('compressed image still too large');
  };

  const handleAvatarFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    if (!file.type.startsWith('image/')) {
      setAccountMessage('Please choose an image file.');
      event.target.value = '';
      return;
    }
    try {
      const avatarDataURL = await compressAvatarFileIfNeeded(file);
      setAvatarURL(avatarDataURL);
      setAccountMessage('Avatar selected. Click Save Changes to persist.');
    } catch (error) {
      console.error('Failed to process avatar image:', error);
      setAccountMessage('Image processing failed. Try a smaller file or use an external image URL.');
    } finally {
      event.target.value = '';
    }
  };

  const handleApplyAvatarExternalURL = () => {
    const raw = avatarExternalURL.trim();
    if (!raw) {
      setAccountMessage('Please enter an image URL.');
      return;
    }

    let parsedURL: URL;
    try {
      parsedURL = new URL(raw);
    } catch {
      setAccountMessage('Invalid image URL.');
      return;
    }

    const protocol = parsedURL.protocol.toLowerCase();
    if (protocol !== 'http:' && protocol !== 'https:') {
      setAccountMessage('Only http/https image URLs are supported.');
      return;
    }

    setAvatarURL(parsedURL.toString());
    setAccountMessage('External avatar link set. Click Save Changes to persist.');
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

  const activeBannedUsers = moderationStates.filter((state) => state.action.toUpperCase() === 'SHADOW_BAN');
  const pendingAppeals = moderationLogs.filter((log) => log.action.toUpperCase().includes('UNBAN_REQUEST'));

  const handleSaveGovernancePolicy = async () => {
    if (!isAdmin || !hasWailsRuntime()) return;
    setGovernancePolicyBusy(true);
    setGovernancePolicyMessage('');
    try {
      const policy = await SetGovernancePolicy(hideHistoryOnShadowBan);
      setHideHistoryOnShadowBan(!!policy.hideHistoryOnShadowBan);
      setGovernancePolicyMessage('Governance policy saved.');
    } catch (error) {
      console.error('Failed to save governance policy:', error);
      setGovernancePolicyMessage('Failed to save governance policy.');
    } finally {
      setGovernancePolicyBusy(false);
    }
  };

  const handleSavePrivacySettings = async () => {
    if (!hasWailsRuntime()) return;

    setPrivacyBusy(true);
    setPrivacyMessage('');
    try {
      const settings = await SetPrivacySettings(publicStatus, allowSearch);
      setPublicStatus(!!settings.showOnlineStatus);
      setAllowSearch(!!settings.allowSearch);
      setPrivacyMessage('Privacy settings saved.');
    } catch (error) {
      console.error('Failed to save privacy settings:', error);
      setPrivacyMessage('Failed to save privacy settings.');
    } finally {
      setPrivacyBusy(false);
    }
  };

  const parseListenPort = (): number | null => {
    const port = Number.parseInt(p2pListenPort, 10);
    if (!Number.isInteger(port) || port <= 0 || port > 65535) {
      setP2PMessage('Listen port must be between 1 and 65535.');
      return null;
    }
    return port;
  };

  const handleSaveP2PConfig = async () => {
    if (!hasWailsRuntime()) return;
    const port = parseListenPort();
    if (port === null) return;

    setP2PBusy(true);
    setP2PMessage('');
    try {
      const relayPeers = parsePeerInput(p2pRelayPeersInput);
      const cfg = await SaveP2PConfig(port, relayPeers, p2pAutoStart);
      setP2PListenPort(String(cfg.listenPort || port));
      setP2PRelayPeersInput((cfg.relayPeers || []).join('\n'));
      setP2PAutoStart(!!cfg.autoStart);
      setP2PMessage('P2P configuration saved.');
      await loadP2PState();
    } catch (error) {
      console.error('Failed to save p2p config:', error);
      setP2PMessage('Failed to save P2P configuration.');
    } finally {
      setP2PBusy(false);
    }
  };

  const handleStartP2P = async () => {
    if (!hasWailsRuntime()) return;
    const port = parseListenPort();
    if (port === null) return;

    setP2PBusy(true);
    setP2PMessage('');
    try {
      const relayPeers = parsePeerInput(p2pRelayPeersInput);
      await SaveP2PConfig(port, relayPeers, p2pAutoStart);
      const status = await StartP2P(port, relayPeers);
      setP2PStatus(status);
      setP2PMessage('P2P started successfully.');
    } catch (error) {
      console.error('Failed to start p2p:', error);
      setP2PMessage('Failed to start P2P network.');
    } finally {
      setP2PBusy(false);
    }
  };

  const handleStopP2P = async () => {
    if (!hasWailsRuntime()) return;

    setP2PBusy(true);
    setP2PMessage('');
    try {
      await StopP2P();
      setP2PStatus((prev) =>
        prev
          ? { ...prev, started: false, connectedPeers: [] }
          : { started: false, peerId: '', listenAddrs: [], connectedPeers: [], topic: '' }
      );
      setP2PMessage('P2P stopped.');
      await loadP2PState();
    } catch (error) {
      console.error('Failed to stop p2p:', error);
      setP2PMessage('Failed to stop P2P network.');
    } finally {
      setP2PBusy(false);
    }
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
                className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${activeTab === 'account'
                  ? 'bg-warm-accent/10 text-warm-accent'
                  : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                  }`}
              >
                <span className="material-icons text-[20px]">manage_accounts</span>
                Account
              </button>
              <button
                onClick={() => setActiveTab('privacy')}
                className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${activeTab === 'privacy'
                  ? 'bg-warm-accent/10 text-warm-accent'
                  : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                  }`}
              >
                <span className="material-icons text-[20px]">security</span>
                Privacy &amp; Keys
              </button>
              <button
                onClick={() => setActiveTab('update')}
                className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${activeTab === 'update'
                  ? 'bg-warm-accent/10 text-warm-accent'
                  : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                  }`}
              >
                <span className="material-icons text-[20px]">system_update</span>
                Updates
              </button>
              <button
                onClick={() => setActiveTab('network')}
                className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${activeTab === 'network'
                  ? 'bg-warm-accent/10 text-warm-accent'
                  : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                  }`}
              >
                <span className="material-icons text-[20px]">hub</span>
                Network &amp; P2P
              </button>
              {isDevMode && (
                <button
                  onClick={() => setActiveTab('consistency')}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${activeTab === 'consistency'
                    ? 'bg-warm-accent/10 text-warm-accent'
                    : 'text-warm-text-secondary dark:text-slate-400 hover:bg-warm-card dark:hover:bg-surface-lighter hover:text-warm-text-primary dark:hover:text-white'
                    }`}
                >
                  <span className="material-icons text-[20px]">lan</span>
                  Consistency
                </button>
              )}
              {isAdmin && (
                <button
                  onClick={() => setActiveTab('governance')}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 text-sm font-medium rounded-lg transition-colors ${activeTab === 'governance'
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

        <div className="relative flex-1 flex flex-col h-full overflow-hidden bg-warm-bg dark:bg-background-dark">
          {accountToast && (
            <div
              className={`absolute top-4 right-6 z-20 rounded-lg px-3 py-2 text-sm shadow-lg border ${accountToast.type === 'success'
                ? 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:border-emerald-900'
                : 'bg-red-50 text-red-700 border-red-200 dark:bg-red-950/40 dark:text-red-300 dark:border-red-900'
                }`}
            >
              {accountToast.text}
            </div>
          )}
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
                      <button
                        onClick={handleSelectAvatarFile}
                        className="absolute bottom-0 right-0 bg-warm-accent hover:bg-warm-accent-hover text-white p-2 rounded-full shadow-md transition-colors border-2 border-warm-surface dark:border-warm-bg"
                      >
                        <span className="material-icons text-sm">edit</span>
                      </button>
                    </div>
                    <div className="pt-2">
                      <h3 className="text-lg font-medium text-warm-text-primary dark:text-white">Profile Photo</h3>
                      <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-1 mb-3">This will be displayed on your posts and comments.</p>
                      <p className="text-xs text-warm-text-secondary dark:text-slate-400 mb-3">Local uploads are auto-compressed for network sync. External image URL is recommended to reduce storage and payload size.</p>
                      <div className="flex gap-3">
                        <button
                          onClick={() => {
                            setAvatarURL('');
                            setAccountMessage('Avatar removed. Click Save Changes to persist.');
                          }}
                          className="px-4 py-2 text-sm font-medium text-warm-text-primary dark:text-white bg-white dark:bg-surface-dark border border-warm-border dark:border-border-dark rounded-lg hover:bg-warm-card dark:hover:bg-surface-lighter transition-colors"
                        >
                          Remove
                        </button>
                        <button
                          onClick={handleSelectAvatarFile}
                          className="px-4 py-2 text-sm font-medium text-warm-accent bg-warm-accent/10 border border-transparent rounded-lg hover:bg-warm-accent/20 transition-colors"
                        >
                          Upload New
                        </button>
                      </div>
                      <div className="mt-3 flex gap-2">
                        <input
                          type="url"
                          value={avatarExternalURL}
                          onChange={(e) => setAvatarExternalURL(e.target.value)}
                          placeholder="https://example.com/avatar.jpg"
                          className="flex-1 px-3 py-2 text-sm rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
                        />
                        <button
                          onClick={handleApplyAvatarExternalURL}
                          className="px-3 py-2 text-sm font-medium text-warm-accent bg-warm-accent/10 border border-transparent rounded-lg hover:bg-warm-accent/20 transition-colors flex items-center gap-1"
                        >
                          <span className="material-icons text-base">link</span>
                          Use URL
                        </button>
                      </div>
                      <input
                        ref={avatarFileInputRef}
                        type="file"
                        accept="image/*"
                        className="hidden"
                        onChange={handleAvatarFileChange}
                      />
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

                <div className="pt-4 space-y-2">
                  <div className="flex justify-end">
                    <button
                      onClick={handleSave}
                      disabled={accountBusy}
                      className="min-w-[132px] px-6 py-2.5 bg-warm-accent hover:bg-warm-accent-hover text-white font-medium rounded-lg shadow-lg shadow-warm-accent/30 transition-all transform active:scale-95 disabled:opacity-70 disabled:cursor-not-allowed"
                    >
                      {accountBusy ? 'Saving...' : 'Save Changes'}
                    </button>
                  </div>
                  <div className="min-h-[20px]">
                    {accountMessage && (
                      <p className="text-sm text-warm-text-secondary dark:text-slate-300 text-right">{accountMessage}</p>
                    )}
                  </div>
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
                          <input
                            type="checkbox"
                            checked={allowSearch}
                            onChange={(e) => setAllowSearch(e.target.checked)}
                            className="sr-only peer"
                          />
                          <div className="w-11 h-6 bg-gray-300 dark:bg-slate-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-warm-accent"></div>
                        </label>
                      </div>
                    </div>
                    <div className="pt-4 flex items-center justify-between gap-3">
                      {privacyMessage && (
                        <p className="text-sm text-warm-text-secondary dark:text-slate-300">{privacyMessage}</p>
                      )}
                      <button
                        onClick={handleSavePrivacySettings}
                        disabled={privacyBusy}
                        className="ml-auto px-4 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Save Privacy Settings
                      </button>
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
                        <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white">
                          Aegis {updateStatus?.currentVersion || 'v0.1.0-dev'}
                        </h3>
                        <p className="text-sm text-warm-text-secondary dark:text-slate-400">Current version</p>
                      </div>
                    </div>
                    <div className="space-y-2">
                      {updateStatus?.hasUpdate ? (
                        <div className="flex items-center gap-2 text-sm text-amber-600">
                          <span className="material-icons text-sm">new_releases</span>
                          <span>Update available: {updateStatus.latestVersion}</span>
                        </div>
                      ) : (
                        <div className="flex items-center gap-2 text-sm text-green-600">
                          <span className="material-icons text-sm">check_circle</span>
                          <span>Latest version installed</span>
                        </div>
                      )}
                      <p className="text-xs text-warm-text-secondary dark:text-slate-400">
                        Last checked: {formatDateTime(updateStatus?.checkedAt || 0)}
                      </p>
                    </div>
                  </div>

                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6">
                    <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white mb-4">What's New</h3>
                    <ul className="space-y-3 text-sm text-warm-text-secondary dark:text-slate-400">
                      {versionHistory.length === 0 ? (
                        <li className="flex items-start gap-2">
                          <span className="material-icons text-warm-accent text-sm mt-0.5">new_releases</span>
                          <span>No release history available yet.</span>
                        </li>
                      ) : (
                        versionHistory.map((item) => (
                          <li key={`${item.version}-${item.publishedAt}`} className="flex items-start gap-2">
                            <span className="material-icons text-warm-accent text-sm mt-0.5">new_releases</span>
                            <span>
                              <strong>{item.version}</strong> - {item.summary} ({formatDateTime(item.publishedAt)})
                            </span>
                          </li>
                        ))
                      )}
                    </ul>
                  </div>

                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6">
                    <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white mb-4">Check for Updates</h3>
                    <p className="text-sm text-warm-text-secondary dark:text-slate-400 mb-4">
                      {updateStatus?.hasUpdate
                        ? `A newer version (${updateStatus.latestVersion}) is available.`
                        : "You're running the latest version. We'll automatically check for updates in the background."}
                    </p>
                    <div className="flex flex-wrap items-center gap-3">
                      <button
                        onClick={() => void loadUpdateData()}
                        disabled={updateBusy}
                        className="px-4 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {updateBusy ? 'Checking...' : 'Check for Updates'}
                      </button>
                      {updateStatus?.releaseURL && (
                        <a
                          href={updateStatus.releaseURL}
                          target="_blank"
                          rel="noreferrer"
                          className="px-4 py-2 rounded-lg border border-warm-border dark:border-border-dark text-sm font-medium text-warm-text-primary dark:text-white hover:bg-warm-bg dark:hover:bg-background-dark"
                        >
                          Open Release Notes
                        </a>
                      )}
                    </div>
                    {updateMessage && (
                      <p className="mt-3 text-xs text-warm-text-secondary dark:text-slate-400">{updateMessage}</p>
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'network' && (
            <div className="flex flex-col h-full">
              <header className="px-8 py-6 border-b border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark shrink-0">
                <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white">Network &amp; P2P</h1>
                <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-1">
                  Configure node relay peers and control P2P runtime.
                </p>
              </header>
              <div className="flex-1 overflow-y-auto p-8">
                <div className="max-w-3xl space-y-6">
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6 space-y-5">
                    <div>
                      <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white">Runtime Settings</h3>
                      <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-1">
                        These settings are stored in local SQLite and used for auto-start.
                      </p>
                    </div>

                    <div className="grid gap-5 md:grid-cols-2">
                      <div>
                        <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
                          Listen Port
                        </label>
                        <input
                          type="number"
                          min={1}
                          max={65535}
                          value={p2pListenPort}
                          onChange={(e) => setP2PListenPort(e.target.value)}
                          className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
                        />
                      </div>
                      <div className="flex items-end">
                        <label className="relative inline-flex items-center cursor-pointer">
                          <input
                            type="checkbox"
                            checked={p2pAutoStart}
                            onChange={(e) => setP2PAutoStart(e.target.checked)}
                            className="sr-only peer"
                          />
                          <div className="w-11 h-6 bg-gray-300 dark:bg-slate-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-warm-accent"></div>
                          <span className="ml-3 text-sm font-medium text-warm-text-primary dark:text-white">Auto-start P2P on app launch</span>
                        </label>
                      </div>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
                        Relay / Bootstrap Peers
                      </label>
                      <textarea
                        value={p2pRelayPeersInput}
                        onChange={(e) => setP2PRelayPeersInput(e.target.value)}
                        rows={4}
                        placeholder="One multiaddr per line"
                        className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-warm-text-primary dark:text-white font-mono text-sm focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none resize-y"
                      />
                      <p className="mt-2 text-xs text-warm-text-secondary dark:text-slate-400">
                        Example: /ip4/51.107.6.171/tcp/40100/p2p/12D3KooWAFxb45HZaK3rtSRAy6wTR7PN8nUaZrXgBNuBPJoNKqwg
                      </p>
                    </div>

                    <div className="flex flex-wrap gap-3">
                      <button
                        onClick={handleSaveP2PConfig}
                        disabled={p2pBusy}
                        className="px-4 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Save Config
                      </button>
                      <button
                        onClick={handleStartP2P}
                        disabled={p2pBusy}
                        className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Start / Restart P2P
                      </button>
                      <button
                        onClick={handleStopP2P}
                        disabled={p2pBusy || !p2pStatus?.started}
                        className="px-4 py-2 bg-slate-700 hover:bg-slate-800 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Stop P2P
                      </button>
                    </div>

                    {p2pMessage && (
                      <div className="text-sm text-warm-text-secondary dark:text-slate-300 bg-warm-bg dark:bg-background-dark border border-warm-border dark:border-border-dark rounded-lg px-3 py-2">
                        {p2pMessage}
                      </div>
                    )}
                  </div>

                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6 space-y-4">
                    <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white">Current Runtime Status</h3>
                    <div className="grid gap-4 md:grid-cols-2">
                      <div>
                        <p className="text-xs text-warm-text-secondary dark:text-slate-400 uppercase">State</p>
                        <p className={`text-sm font-semibold ${p2pStatus?.started ? 'text-emerald-600' : 'text-slate-500 dark:text-slate-400'}`}>
                          {p2pStatus?.started ? 'Running' : 'Stopped'}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-warm-text-secondary dark:text-slate-400 uppercase">Connected Peers</p>
                        <p className="text-sm font-semibold text-warm-text-primary dark:text-white">
                          {p2pStatus?.connectedPeers?.length || 0}
                        </p>
                      </div>
                    </div>

                    <div>
                      <p className="text-xs text-warm-text-secondary dark:text-slate-400 uppercase mb-1">Peer ID</p>
                      <div className="bg-warm-bg dark:bg-background-dark rounded-lg p-3 font-mono text-xs text-warm-text-secondary dark:text-slate-400 break-all">
                        {p2pStatus?.peerId || 'N/A'}
                      </div>
                    </div>

                    <div>
                      <p className="text-xs text-warm-text-secondary dark:text-slate-400 uppercase mb-2">Listen Addresses</p>
                      <div className="space-y-1">
                        {(p2pStatus?.listenAddrs || []).length === 0 ? (
                          <p className="text-sm text-warm-text-secondary dark:text-slate-400">No listen addresses available.</p>
                        ) : (
                          (p2pStatus?.listenAddrs || []).map((addr) => (
                            <div key={addr} className="bg-warm-bg dark:bg-background-dark rounded-lg p-2 font-mono text-xs text-warm-text-secondary dark:text-slate-400 break-all">
                              {addr}
                            </div>
                          ))
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6 space-y-4">
                    <div className="flex items-center justify-between">
                      <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white">Storage Usage</h3>
                      <button
                        onClick={() => void loadStorageUsage()}
                        disabled={storageLoading}
                        className="px-3 py-1.5 text-xs font-medium rounded-lg border border-warm-border dark:border-border-dark text-warm-text-primary dark:text-white hover:bg-warm-card dark:hover:bg-surface-lighter disabled:opacity-50"
                      >
                        {storageLoading ? 'Refreshing...' : 'Refresh'}
                      </button>
                    </div>

                    {storageMessage && (
                      <p className="text-sm text-warm-text-secondary dark:text-slate-300">{storageMessage}</p>
                    )}

                    <div className="grid gap-4 md:grid-cols-3">
                      <div className="rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark p-3">
                        <p className="text-xs uppercase text-warm-text-secondary dark:text-slate-400">Shared Responsibility</p>
                        <p className="mt-1 text-sm font-semibold text-warm-text-primary dark:text-white">
                          {formatBytes(storageUsage?.publicUsedBytes || 0)} / {formatBytes(storageUsage?.publicQuota || 0)}
                        </p>
                        <div className="mt-2 h-2 rounded bg-slate-200 dark:bg-slate-700 overflow-hidden">
                          <div
                            className="h-full bg-warm-accent"
                            style={{ width: `${usagePercent(storageUsage?.publicUsedBytes || 0, storageUsage?.publicQuota || 0)}%` }}
                          />
                        </div>
                      </div>
                      <div className="rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark p-3">
                        <p className="text-xs uppercase text-warm-text-secondary dark:text-slate-400">Private Responsibility</p>
                        <p className="mt-1 text-sm font-semibold text-warm-text-primary dark:text-white">
                          {formatBytes(storageUsage?.privateUsedBytes || 0)} / {formatBytes(storageUsage?.privateQuota || 0)}
                        </p>
                        <div className="mt-2 h-2 rounded bg-slate-200 dark:bg-slate-700 overflow-hidden">
                          <div
                            className="h-full bg-emerald-500"
                            style={{ width: `${usagePercent(storageUsage?.privateUsedBytes || 0, storageUsage?.privateQuota || 0)}%` }}
                          />
                        </div>
                      </div>
                      <div className="rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark p-3">
                        <p className="text-xs uppercase text-warm-text-secondary dark:text-slate-400">Total Quota</p>
                        <p className="mt-1 text-sm font-semibold text-warm-text-primary dark:text-white">
                          {formatBytes((storageUsage?.publicUsedBytes || 0) + (storageUsage?.privateUsedBytes || 0))} / {formatBytes(storageUsage?.totalQuota || 0)}
                        </p>
                        <div className="mt-2 h-2 rounded bg-slate-200 dark:bg-slate-700 overflow-hidden">
                          <div
                            className="h-full bg-indigo-500"
                            style={{ width: `${usagePercent((storageUsage?.publicUsedBytes || 0) + (storageUsage?.privateUsedBytes || 0), storageUsage?.totalQuota || 0)}%` }}
                          />
                        </div>
                      </div>
                    </div>

                    <p className="text-xs text-warm-text-secondary dark:text-slate-400">
                      Shared = data your node may serve to peers. Private = local-only responsibility (including policy-blocked shadow-ban data).
                    </p>

                    <div className="pt-2 border-t border-warm-border dark:border-border-dark">
                      <div className="flex flex-wrap items-center justify-between gap-3">
                        <div>
                          <p className="text-sm font-semibold text-red-600 dark:text-red-400">Danger Zone: Reset Local Test Data</p>
                          <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-1">
                            Keeps identity and app settings, clears local forum/test data for a clean test environment.
                          </p>
                        </div>
                        <button
                          onClick={() => void handleResetLocalTestData()}
                          disabled={resetBusy}
                          className={`px-4 py-2 rounded-lg text-sm font-medium text-white disabled:opacity-50 disabled:cursor-not-allowed ${resetConfirmArmed ? 'bg-red-700 hover:bg-red-800' : 'bg-red-600 hover:bg-red-700'}`}
                        >
                          {resetBusy ? 'Resetting...' : (resetConfirmArmed ? 'Confirm Reset' : 'Reset Local Data')}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'consistency' && (
            <div className="flex flex-col h-full">
              <header className="px-8 py-6 border-b border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark shrink-0">
                <h1 className="text-2xl font-bold text-warm-text-primary dark:text-white">Consistency</h1>
                <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-1">
                  Lamport convergence diagnostics, operation timeline, and tombstone GC controls.
                </p>
              </header>
              <div className="flex-1 overflow-y-auto p-8">
                <div className="max-w-4xl space-y-6">
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6 space-y-4">
                    <div className="flex items-center justify-between">
                      <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white">Anti-Entropy Stats</h3>
                      <button
                        onClick={() => void loadAntiEntropyState()}
                        disabled={consistencyBusy}
                        className="px-3 py-1.5 text-xs font-medium rounded-lg border border-warm-border dark:border-border-dark text-warm-text-primary dark:text-white hover:bg-warm-card dark:hover:bg-surface-lighter disabled:opacity-50"
                      >
                        Refresh
                      </button>
                    </div>
                    <div className="grid gap-3 md:grid-cols-3 text-sm">
                      <div className="rounded-lg border border-warm-border dark:border-border-dark p-3">
                        <p className="text-xs text-warm-text-secondary dark:text-slate-400 uppercase">Sync Requests</p>
                        <p className="mt-1 font-semibold text-warm-text-primary dark:text-white">{antiEntropyStats?.syncRequestsSent || 0} sent / {antiEntropyStats?.syncRequestsReceived || 0} recv</p>
                      </div>
                      <div className="rounded-lg border border-warm-border dark:border-border-dark p-3">
                        <p className="text-xs text-warm-text-secondary dark:text-slate-400 uppercase">Blob Fetch</p>
                        <p className="mt-1 font-semibold text-warm-text-primary dark:text-white">{antiEntropyStats?.blobFetchSuccess || 0} ok / {antiEntropyStats?.blobFetchFailures || 0} fail</p>
                      </div>
                      <div className="rounded-lg border border-warm-border dark:border-border-dark p-3">
                        <p className="text-xs text-warm-text-secondary dark:text-slate-400 uppercase">Observed Lag</p>
                        <p className="mt-1 font-semibold text-warm-text-primary dark:text-white">{antiEntropyStats?.lastObservedSyncLagSec || 0}s</p>
                      </div>
                    </div>
                    <p className="text-xs text-warm-text-secondary dark:text-slate-400">
                      Last Sync: {formatDateTime(antiEntropyStats?.lastSyncAt || 0)} · Last Remote Summary: {formatDateTime(antiEntropyStats?.lastRemoteSummaryTs || 0)}
                    </p>
                  </div>

                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6 space-y-4">
                    <div className="flex flex-wrap items-end gap-3">
                      <div>
                        <label className="block text-xs text-warm-text-secondary dark:text-slate-400 mb-1">Entity Type</label>
                        <div className="relative">
                          <select
                            value={entityOpTypeFilter}
                            onChange={(e) => setEntityOpTypeFilter(e.target.value as 'post' | 'comment' | '')}
                            className="appearance-none w-full px-3 py-2 pr-8 rounded-lg border border-warm-border dark:border-border-dark bg-white hover:bg-warm-bg dark:bg-surface-dark dark:hover:bg-surface-lighter text-warm-text-primary dark:text-white text-sm focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none transition-colors cursor-pointer shadow-soft"
                          >
                            <option value="">All</option>
                            <option value="post">Post</option>
                            <option value="comment">Comment</option>
                          </select>
                          <span className="material-icons absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none text-warm-text-secondary dark:text-slate-400 text-base">
                            expand_more
                          </span>
                        </div>
                      </div>
                      <div className="flex-1 min-w-[220px]">
                        <label className="block text-xs text-warm-text-secondary dark:text-slate-400 mb-1">Entity ID</label>
                        <input
                          value={entityOpIDFilter}
                          onChange={(e) => setEntityOpIDFilter(e.target.value)}
                          placeholder="Optional exact entity id"
                          className="w-full px-3 py-2 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-warm-text-secondary dark:text-slate-400 mb-1">Limit</label>
                        <input
                          value={entityOpLimit}
                          onChange={(e) => setEntityOpLimit(e.target.value)}
                          className="w-24 px-3 py-2 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-sm"
                        />
                      </div>
                      <button
                        onClick={() => void loadEntityOps()}
                        disabled={consistencyBusy}
                        className="px-4 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white rounded-lg font-medium transition-colors disabled:opacity-50"
                      >
                        {consistencyBusy ? 'Loading...' : 'Load Operations'}
                      </button>
                    </div>

                    <div className="rounded-lg border border-warm-border dark:border-border-dark overflow-hidden">
                      <div className="max-h-72 overflow-auto">
                        <table className="w-full text-xs">
                          <thead className="bg-warm-bg dark:bg-background-dark">
                            <tr>
                              <th className="text-left px-3 py-2">Lamport</th>
                              <th className="text-left px-3 py-2">Entity</th>
                              <th className="text-left px-3 py-2">Op</th>
                              <th className="text-left px-3 py-2">Author</th>
                              <th className="text-left px-3 py-2">Time</th>
                            </tr>
                          </thead>
                          <tbody>
                            {entityOps.length === 0 ? (
                              <tr>
                                <td colSpan={5} className="px-3 py-4 text-center text-warm-text-secondary dark:text-slate-400">No operations found.</td>
                              </tr>
                            ) : (
                              entityOps.map((item) => (
                                <tr key={item.opId} className="border-t border-warm-border dark:border-border-dark">
                                  <td className="px-3 py-2 font-mono">{item.lamport}</td>
                                  <td className="px-3 py-2 font-mono">{item.entityType}:{item.entityId.slice(0, 16)}</td>
                                  <td className="px-3 py-2">{item.opType}</td>
                                  <td className="px-3 py-2 font-mono">{item.authorPubkey.slice(0, 12)}</td>
                                  <td className="px-3 py-2">{formatDateTime(item.timestamp)}</td>
                                </tr>
                              ))
                            )}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  </div>

                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark p-6 space-y-4">
                    <h3 className="text-lg font-semibold text-warm-text-primary dark:text-white">Tombstone GC</h3>
                    <p className="text-xs text-warm-text-secondary dark:text-slate-400">
                      Safe cleanup requires both retention horizon and stable-pass checks. Use only in operator workflows.
                    </p>
                    <div className="grid gap-3 md:grid-cols-3">
                      <div>
                        <label className="block text-xs text-warm-text-secondary dark:text-slate-400 mb-1">Retention Days</label>
                        <input value={gcRetentionDays} onChange={(e) => setGCRetentionDays(e.target.value)} className="w-full px-3 py-2 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-sm" />
                      </div>
                      <div>
                        <label className="block text-xs text-warm-text-secondary dark:text-slate-400 mb-1">Stable Passes</label>
                        <input value={gcStablePasses} onChange={(e) => setGCStablePasses(e.target.value)} className="w-full px-3 py-2 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-sm" />
                      </div>
                      <div>
                        <label className="block text-xs text-warm-text-secondary dark:text-slate-400 mb-1">Batch Size</label>
                        <input value={gcBatchSize} onChange={(e) => setGCBatchSize(e.target.value)} className="w-full px-3 py-2 rounded-lg border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark text-sm" />
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      <button
                        onClick={() => void handleRunTombstoneGC()}
                        disabled={consistencyBusy}
                        className={`px-4 py-2 text-white rounded-lg font-medium transition-colors disabled:opacity-50 ${gcConfirmArmed ? 'bg-red-700 hover:bg-red-800' : 'bg-red-600 hover:bg-red-700'}`}
                      >
                        {gcConfirmArmed ? 'Confirm Run Tombstone GC' : 'Run Tombstone GC'}
                      </button>
                      {gcResult && (
                        <span className="text-xs text-warm-text-secondary dark:text-slate-400">
                          scanned(post/comment): {gcResult.scannedPosts}/{gcResult.scannedComments}, deleted(post/comment): {gcResult.deletedPosts}/{gcResult.deletedComments}
                        </span>
                      )}
                    </div>
                  </div>

                  {consistencyMessage && (
                    <div className="text-sm text-warm-text-secondary dark:text-slate-300 bg-warm-bg dark:bg-background-dark border border-warm-border dark:border-border-dark rounded-lg px-3 py-2">
                      {consistencyMessage}
                    </div>
                  )}
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
                <div className="mb-4 rounded-xl border border-warm-border dark:border-border-dark bg-white dark:bg-surface-dark p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <h3 className="font-semibold text-warm-text-primary dark:text-white">Governance Policy</h3>
                      <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-1">
                        Control whether shadow-ban hides historical posts and comments from other users.
                      </p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer shrink-0">
                      <input
                        type="checkbox"
                        checked={hideHistoryOnShadowBan}
                        onChange={(e) => setHideHistoryOnShadowBan(e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-gray-300 dark:bg-slate-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-warm-accent"></div>
                    </label>
                  </div>
                  <div className="mt-3 flex items-center justify-between gap-3">
                    <span className="text-xs text-warm-text-secondary dark:text-slate-400">
                      {hideHistoryOnShadowBan ? 'History hidden for shadow-banned users' : 'Only future posts/comments affected'}
                    </span>
                    <button
                      onClick={handleSaveGovernancePolicy}
                      disabled={governancePolicyBusy}
                      className="px-3 py-1.5 text-xs font-medium bg-warm-accent hover:bg-warm-accent-hover text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {governancePolicyBusy ? 'Saving...' : 'Save Policy'}
                    </button>
                  </div>
                  {governancePolicyMessage && (
                    <p className="mt-2 text-xs text-warm-text-secondary dark:text-slate-400">{governancePolicyMessage}</p>
                  )}
                </div>
                <div className="flex space-x-6">
                  <button
                    onClick={() => setGovernanceTab('banned')}
                    className={`pb-3 text-sm font-medium transition-colors ${governanceTab === 'banned'
                      ? 'border-b-2 border-warm-accent text-warm-accent'
                      : 'border-b-2 border-transparent text-warm-text-secondary dark:text-slate-400 hover:text-warm-text-primary dark:hover:text-white'
                      }`}
                  >
                    Banned Users
                  </button>
                  <button
                    onClick={() => setGovernanceTab('appeals')}
                    className={`pb-3 text-sm font-medium transition-colors ${governanceTab === 'appeals'
                      ? 'border-b-2 border-warm-accent text-warm-accent'
                      : 'border-b-2 border-transparent text-warm-text-secondary dark:text-slate-400 hover:text-warm-text-primary dark:hover:text-white'
                      }`}
                  >
                    Unban Requests
                    <span className="ml-2 bg-warm-accent text-white text-[10px] px-1.5 py-0.5 rounded-full">{pendingAppeals.length}</span>
                  </button>
                  <button
                    onClick={() => setGovernanceTab('logs')}
                    className={`pb-3 text-sm font-medium transition-colors ${governanceTab === 'logs'
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
                      {activeBannedUsers.length === 0 ? (
                        <div className="p-4 text-center text-warm-text-secondary dark:text-slate-400">
                          No banned users
                        </div>
                      ) : (
                        <div className="divide-y divide-warm-border dark:divide-border-dark">
                          {activeBannedUsers.map((item) => (
                            <div key={`${item.targetPubkey}-${item.timestamp}`} className="p-4 flex items-start justify-between gap-4">
                              <div>
                                <p className="font-mono text-sm text-warm-text-primary dark:text-white break-all">{item.targetPubkey}</p>
                                <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-1">Reason: {item.reason || 'No reason provided'}</p>
                                <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-1">
                                  By {item.sourceAdmin.slice(0, 12)}... · {new Date(item.timestamp * 1000).toLocaleString()}
                                </p>
                              </div>
                              <button
                                onClick={() => handleUnban(item.targetPubkey)}
                                className="px-3 py-1.5 text-xs font-medium bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg"
                              >
                                Unban
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {governanceTab === 'appeals' && (
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark overflow-hidden">
                    <div className="p-4 border-b border-warm-border dark:border-border-dark flex justify-between items-center">
                      <h3 className="font-semibold text-warm-text-primary dark:text-white">Active Unban Requests</h3>
                      <span className="text-xs text-warm-text-secondary dark:text-slate-400">Auto-collected from moderation logs</span>
                    </div>
                    {pendingAppeals.length === 0 ? (
                      <div className="p-4 text-center text-warm-text-secondary dark:text-slate-400">
                        No pending appeals
                      </div>
                    ) : (
                      <div className="divide-y divide-warm-border dark:divide-border-dark">
                        {pendingAppeals.map((item) => (
                          <div key={item.id} className="p-4 flex items-start justify-between gap-4">
                            <div>
                              <p className="font-mono text-sm text-warm-text-primary dark:text-white break-all">{item.targetPubkey}</p>
                              <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-1">Reason: {item.reason || 'No reason provided'}</p>
                              <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-1">
                                Requested at {new Date(item.timestamp * 1000).toLocaleString()}
                              </p>
                            </div>
                            <button
                              onClick={() => handleUnban(item.targetPubkey)}
                              className="px-3 py-1.5 text-xs font-medium bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg"
                            >
                              Approve Unban
                            </button>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}

                {governanceTab === 'logs' && (
                  <div className="bg-white dark:bg-surface-dark rounded-xl border border-warm-border dark:border-border-dark overflow-hidden">
                    <div className="p-4 border-b border-warm-border dark:border-border-dark">
                      <h3 className="font-semibold text-warm-text-primary dark:text-white">Operation Log</h3>
                    </div>
                    {moderationLogs.length === 0 ? (
                      <div className="p-4 text-center text-warm-text-secondary dark:text-slate-400">
                        No logs yet
                      </div>
                    ) : (
                      <div className="divide-y divide-warm-border dark:divide-border-dark">
                        {moderationLogs.map((log) => (
                          <div key={log.id} className="p-4">
                            <div className="flex items-center justify-between gap-3">
                              <div className="flex items-center gap-2">
                                <span className={`px-2 py-0.5 rounded text-xs font-medium ${log.action.toUpperCase() === 'UNBAN' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'}`}>
                                  {log.action}
                                </span>
                                <span className="font-mono text-xs text-warm-text-secondary dark:text-slate-400 break-all">{log.targetPubkey}</span>
                              </div>
                              <span className="text-xs text-warm-text-secondary dark:text-slate-400">{new Date(log.timestamp * 1000).toLocaleString()}</span>
                            </div>
                            <p className="text-sm text-warm-text-primary dark:text-white mt-2">{log.reason || 'No reason provided'}</p>
                            <p className="text-xs text-warm-text-secondary dark:text-slate-400 mt-1">By {log.sourceAdmin.slice(0, 12)}... · Result: {log.result || 'applied'}</p>
                          </div>
                        ))}
                      </div>
                    )}
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
