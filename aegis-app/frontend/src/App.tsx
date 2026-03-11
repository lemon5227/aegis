import { useState, useEffect, useCallback } from 'react';
import './style.css';
import {
  GetSubs,
  GetFeedIndexBySubSorted,
  GetFeedStream,
  GetSubscribedSubs,
  SubscribeSub,
  UnsubscribeSub,
  SearchSubs,
  SearchPosts,
  CreateSub,
  PublishCreateSub,
  PublishPostStructuredToSub,
  PublishPostWithImageToSub,
  PublishPostUpvote,
  PublishPostDownvote,
  LoadSavedIdentity,
  GenerateIdentity,
  ImportIdentityFromMnemonic,
  GetProfileDetails,
  GetPostsByAuthor,
  GetProfile,
  GetSubStats,
  GetTrustedAdmins,
  GetModerationState,
  GetPostIndexByID,
  GetPostBodyByID,
  GetCommentsByPost,
  PublishCommentWithAttachments,
  PublishCommentUpvote,
  PublishCommentDownvote,
  UpdateProfileDetails,
  PublishProfileUpdate,
  PublishShadowBan,
  PublishUnban,
  GetModerationLogs,
  TriggerCommentSyncNow,
  GetP2PStatus,
  GetAntiEntropyStats,
  TriggerAntiEntropySyncNow,
  PublishDeletePost,
  PublishDeleteComment,
  PublishPostUpdate,
  PublishCommentUpdate,
  IsDevMode,
  GetFavoritePostIDs,
  AddFavorite,
  RemoveFavorite,
  GetUnreadNotificationCount,
} from '../wailsjs/go/main/App';
import { Sidebar } from './components/Sidebar';
import { Header } from './components/Header';
import { Feed } from './components/Feed';
import { RightPanel } from './components/RightPanel';
import { DiscoverView } from './components/DiscoverView';
import { PostDetail } from './components/PostDetail';
import { MyPosts } from './components/MyPosts';
import { Favorites } from './components/Favorites';
import { SettingsPanel } from './components/SettingsPanel';
import { CreateSubModal } from './components/CreateSubModal';
import { CreatePostModal } from './components/CreatePostModal';
import { LoginModal } from './components/LoginModal';
import { ProfileView } from './components/ProfileView';
import { SearchResultsView } from './components/SearchResultsView';
import { DraftsView } from './components/DraftsView';
import { HistoryView } from './components/HistoryView';
import { NetworkBanner } from './components/NetworkBanner';
import { PendingSyncView } from './components/PendingSyncView';
import { NotificationsView } from './components/NotificationsView';
import { ToastContainer, useToasts } from './components/Toast';
import { Sub, Profile, ProfileDetails, Post, GovernanceAdmin, Identity, Comment, ModerationLog, ModerationState, SubStats, CreatePostInput, AntiEntropyStats, P2PStatus, PendingSyncAction, PendingSyncActionKind } from './types';
import { ClipboardSetText, EventsOn } from '../wailsjs/runtime/runtime';
import { recordRecentlyViewed } from './lib/history';
import { deriveNetworkHealth } from './lib/networkHealth';
import { listPendingSyncActions, recordPendingSyncAction, reconcilePendingSyncActions, removePendingSyncAction } from './lib/pendingSync';

type SortMode = 'hot' | 'new' | 'top-day' | 'top-week' | 'top-month' | 'top-all';
type ViewMode = 'feed' | 'discover' | 'search' | 'profile' | 'post-detail' | 'my-posts' | 'favorites' | 'drafts' | 'history' | 'pending-sync' | 'notifications';
type ConsistencyFocus = {
  entityType: 'post' | 'comment';
  entityId: string;
  nonce: number;
};

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) {
    return error.message.trim();
  }
  if (typeof error === 'string' && error.trim()) {
    return error.trim();
  }
  return fallback;
}

function mapPostIndexToPost(item: any): Post {
  return {
    id: item.id,
    pubkey: item.pubkey,
    title: item.title,
    bodyPreview: item.bodyPreview || '',
    contentCid: item.contentCid || '',
    imageCid: item.imageCid || '',
    thumbCid: item.thumbCid || '',
    imageMime: item.imageMime || '',
    imageSize: item.imageSize || 0,
    imageWidth: item.imageWidth || 0,
    imageHeight: item.imageHeight || 0,
    score: item.score || 0,
    timestamp: item.timestamp || 0,
    zone: (item.zone || 'public') as 'private' | 'public',
    subId: item.subId || 'general',
    visibility: item.visibility || 'normal',
  };
}

function mapForumMessageToPost(item: any): Post {
  return {
    id: item.id,
    pubkey: item.pubkey,
    title: item.title,
    bodyPreview: item.body || '',
    contentCid: item.contentCid || '',
    imageCid: item.imageCid || '',
    thumbCid: item.thumbCid || '',
    imageMime: item.imageMime || '',
    imageSize: item.imageSize || 0,
    imageWidth: item.imageWidth || 0,
    imageHeight: item.imageHeight || 0,
    score: item.score || 0,
    timestamp: item.timestamp || 0,
    zone: (item.zone || 'public') as 'private' | 'public',
    subId: item.subId || 'general',
    visibility: item.visibility || 'normal',
  };
}

function buildAppHash(route: string): string {
  return `#${route.startsWith('/') ? route : `/${route}`}`;
}

function buildShareLink(postId: string): string {
  return buildAppHash(`/post/${encodeURIComponent(postId)}`);
}

function buildSubShareLink(subId: string): string {
  return buildAppHash(`/r/${encodeURIComponent(subId)}`);
}

function shouldTrackPendingSync(status: P2PStatus | null): boolean {
  if (!status?.started) {
    return true;
  }
  return (status.connectedPeers?.length || 0) === 0;
}

function getWriteFeedback(status: P2PStatus | null, onlineMessage: string, deferredMessage: string) {
  if (shouldTrackPendingSync(status)) {
    return {
      title: 'Saved',
      message: deferredMessage,
      type: 'warning' as const,
    };
  }
  return {
    title: 'Done',
    message: onlineMessage,
    type: 'success' as const,
  };
}

function App() {
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [isAdmin, setIsAdmin] = useState(false);
  const [subs, setSubs] = useState<Sub[]>([]);
  const [subscribedSubs, setSubscribedSubs] = useState<Sub[]>([]);
  const [subscribedSubIds, setSubscribedSubIds] = useState<Set<string>>(new Set());
  const [currentSubId, setCurrentSubId] = useState<string>('general');
  const [view, setView] = useState<ViewMode>('feed');
  const [postReturnView, setPostReturnView] = useState<Exclude<ViewMode, 'post-detail'>>('feed');
  const [sortMode, setSortMode] = useState<SortMode>('hot');
  const [posts, setPosts] = useState<Array<Post & { reason?: string; isSubscribed?: boolean; isFavorited?: boolean }>>([]);
  const [profiles, setProfiles] = useState<Record<string, Profile>>({});
  const [isDark, setIsDark] = useState(false);
  const [showLoginModal, setShowLoginModal] = useState(false);
  const [identityChecked, setIdentityChecked] = useState(false);
  const [showCreateSubModal, setShowCreateSubModal] = useState(false);
  const [showCreatePostModal, setShowCreatePostModal] = useState(false);
  const [showSettingsPanel, setShowSettingsPanel] = useState(false);
  const [consistencyFocus, setConsistencyFocus] = useState<ConsistencyFocus | null>(null);
  const [loading, setLoading] = useState(false);
  const [searchResults, setSearchResults] = useState<{ subs: Sub[]; posts: any[] } | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchScopeSubId, setSearchScopeSubId] = useState<string | null>(null);
  const [unreadSubs, setUnreadSubs] = useState<Set<string>>(new Set());
  const [favoritePostIds, setFavoritePostIds] = useState<Set<string>>(new Set());
  const [unreadNotificationCount, setUnreadNotificationCount] = useState(0);
  const [draftSyncToken, setDraftSyncToken] = useState(0);
  const [historySyncToken, setHistorySyncToken] = useState(0);

  const [selectedPost, setSelectedPost] = useState<Post | null>(null);
  const [postBody, setPostBody] = useState<string>('');
  const [postComments, setPostComments] = useState<Comment[]>([]);
  const [hasRemotePostUpdate, setHasRemotePostUpdate] = useState(false);
  const [selectedProfile, setSelectedProfile] = useState<ProfileDetails | null>(null);
  const [selectedProfilePosts, setSelectedProfilePosts] = useState<Post[]>([]);
  const [currentSubStats, setCurrentSubStats] = useState<SubStats | null>(null);
  const [governanceAdmins, setGovernanceAdmins] = useState<GovernanceAdmin[]>([]);
  const [moderationStates, setModerationStates] = useState<ModerationState[]>([]);
  const [moderationLogs, setModerationLogs] = useState<ModerationLog[]>([]);
  const [onlineCount, setOnlineCount] = useState(0);
  const [p2pStatus, setP2PStatus] = useState<P2PStatus | null>(null);
  const [antiEntropyStats, setAntiEntropyStats] = useState<AntiEntropyStats | null>(null);
  const [pendingSyncActions, setPendingSyncActions] = useState<PendingSyncAction[]>([]);
  const [networkBusy, setNetworkBusy] = useState(false);
  const [isDevMode, setIsDevMode] = useState(false);
  const [viewSyncToken, setViewSyncToken] = useState(0);

  const { toasts, addToast, removeToast } = useToasts();
  const networkHealth = deriveNetworkHealth(p2pStatus, antiEntropyStats);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    IsDevMode().then(setIsDevMode).catch(console.error);
  }, []);

  const hasWailsRuntime = () => {
    return !!(window as any)?.go?.main?.App;
  };

  const bumpViewSyncToken = useCallback(() => {
    setViewSyncToken((prev) => prev + 1);
  }, []);

  const bumpDraftSyncToken = useCallback(() => {
    setDraftSyncToken((prev) => prev + 1);
  }, []);

  const bumpHistorySyncToken = useCallback(() => {
    setHistorySyncToken((prev) => prev + 1);
  }, []);

  const loadGovernanceData = useCallback(async (publicKey?: string) => {
    if (!hasWailsRuntime()) return;
    try {
      const [admins, states, logs] = await Promise.all([
        GetTrustedAdmins(),
        GetModerationState(),
        GetModerationLogs(200),
      ]);
      setGovernanceAdmins(admins || []);
      setModerationStates(states || []);
      setModerationLogs(logs || []);
      if (publicKey) {
        setIsAdmin((admins || []).some((a: GovernanceAdmin) => a.adminPubkey === publicKey && a.active));
      }
    } catch (error) {
      console.error('Failed to load governance data:', error);
    }
  }, []);

  const loadFavorites = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    try {
      const ids = await GetFavoritePostIDs();
      setFavoritePostIds(new Set(ids));
    } catch (e) {
      console.error('Failed to load favorites:', e);
    }
  }, []);

  const loadNetworkHealth = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    try {
      const [status, stats] = await Promise.all([
        GetP2PStatus(),
        GetAntiEntropyStats(),
      ]);
      setP2PStatus(status);
      setAntiEntropyStats(stats);
      setPendingSyncActions(reconcilePendingSyncActions(deriveNetworkHealth(status, stats)));
      const peers = Array.isArray(status?.connectedPeers) ? status.connectedPeers.length : 0;
      setOnlineCount(status?.started ? peers + 1 : 0);
    } catch (e) {
      console.error('Failed to load network health:', e);
      setP2PStatus(null);
      setAntiEntropyStats(null);
      setPendingSyncActions(listPendingSyncActions());
      setOnlineCount(0);
    }
  }, []);

  const loadIdentity = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    let id: Identity | null = null;
    try {
      id = await LoadSavedIdentity();
    } catch (e) {
      console.log('No saved identity');
      setIdentityChecked(true);
      return;
    }
    if (!id) {
      setIdentityChecked(true);
      return;
    }
    setIdentity(id);
    setIdentityChecked(true);
    const pubKey = id.publicKey;
    if (pubKey) {
      try {
        const p = await GetProfileDetails(pubKey);
        setProfile(p);
        setProfiles((prev) => ({ ...prev, [pubKey]: p }));
      } catch (e) {
        console.error('Failed to load profile details:', e);
      }
      await loadGovernanceData(pubKey);
      await loadFavorites();
    }
  }, [loadGovernanceData, loadFavorites]);

  const loadSubs = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    try {
      const s = await GetSubs();
      setSubs(s);
    } catch (e) {
      console.error('Failed to load subs:', e);
    }
  }, []);

  const loadSubscribedSubs = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    try {
      const subscribed = await GetSubscribedSubs();
      setSubscribedSubs(subscribed);
      setSubscribedSubIds(new Set(subscribed.map((s: Sub) => s.id)));
    } catch (e) {
      console.error('Failed to load subscribed subs:', e);
    }
  }, []);

  const loadSubStats = useCallback(async (subId: string) => {
    if (!hasWailsRuntime()) return;
    if (!subId || subId === 'recommended') {
      setCurrentSubStats(null);
      return;
    }
    try {
      const stats = await GetSubStats(subId);
      setCurrentSubStats(stats);
    } catch (e) {
      console.error('Failed to load sub stats:', e);
      setCurrentSubStats(null);
    }
  }, []);

  const openProfile = useCallback(async (pubkey: string) => {
    if (!hasWailsRuntime()) return;
    const normalized = pubkey.trim();
    if (!normalized) return;

    try {
      const [details, authoredPosts] = await Promise.all([
        GetProfileDetails(normalized),
        GetPostsByAuthor(normalized, 40),
      ]);
      const mappedPosts = (authoredPosts || []).map((item: any) => mapPostIndexToPost(item));
      setSelectedProfile(details);
      setSelectedProfilePosts(mappedPosts);
      setProfiles((prev) => ({ ...prev, [normalized]: details }));
      setView('profile');
    } catch (e) {
      console.error('Failed to open profile:', e);
      addToast({
        title: 'Error',
        message: 'Failed to load profile.',
        type: 'error',
      });
    }
  }, [addToast]);

  const activateIdentity = useCallback(async (id: Identity) => {
    setIdentity(id);
    const pubKey = id.publicKey;
    if (pubKey) {
      try {
        const p = await GetProfileDetails(pubKey);
        setProfile(p);
        setProfiles((prev) => ({ ...prev, [pubKey]: p }));
      } catch (e) {
        console.error('Failed to load profile details:', e);
      }
      await loadGovernanceData(pubKey);
      await loadFavorites();
    }
    await loadSubs();
    await loadSubscribedSubs();
  }, [loadGovernanceData, loadSubs, loadSubscribedSubs, loadFavorites]);

  const loadRecommendedFeed = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    try {
      const stream = await GetFeedStream(50);
      const mapped = stream.items.map((item: any) => ({
        ...mapForumMessageToPost(item.post),
        reason: item.reason,
        isSubscribed: item.isSubscribed,
      }));
      setPosts(mapped);
    } catch (e) {
      console.error('Failed to load recommended feed:', e);
    }
  }, []);

  const loadPosts = useCallback(async (subId: string, mode: SortMode) => {
    if (!hasWailsRuntime()) return;
    if (subId === 'recommended') {
      await loadRecommendedFeed();
      return;
    }
    try {
      const feed = await GetFeedIndexBySubSorted(subId, mode);
      const mapped = (feed as any[]).map((item) => mapPostIndexToPost(item));
      setPosts(mapped);
      setUnreadSubs((prev) => {
        const next = new Set(prev);
        next.delete(subId);
        return next;
      });
    } catch (e) {
      console.error('Failed to load posts:', e);
    }
  }, [loadRecommendedFeed]);

  const loadPostDetail = useCallback(async (post: Post) => {
    if (!hasWailsRuntime()) return;
    try {
      await TriggerCommentSyncNow(post.id);
      const body = await GetPostBodyByID(post.id);
      setPostBody(body.body || '');
      const comments = await GetCommentsByPost(post.id);
      setPostComments(comments);

      const uniquePubkeys = Array.from(new Set([post.pubkey, ...comments.map((c: Comment) => c.pubkey)]));
      const resolvedProfiles = await Promise.all(
        uniquePubkeys.map(async (pk) => {
          try {
            const profile = await GetProfile(pk);
            return [pk, profile] as const;
          } catch {
            return null;
          }
        })
      );
      const mergedProfiles: Record<string, Profile> = {};
      for (const entry of resolvedProfiles) {
        if (!entry) continue;
        mergedProfiles[entry[0]] = entry[1];
      }
      if (Object.keys(mergedProfiles).length > 0) {
        setProfiles((prev) => ({ ...prev, ...mergedProfiles }));
      }
    } catch (e) {
      console.error('Failed to load post detail:', e);
    }
  }, []);

  const createIdentity = async (): Promise<Identity | null> => {
    if (!hasWailsRuntime()) return null;
    setLoading(true);
    try {
      const id = await GenerateIdentity();
      return id;
    } catch (e) {
      console.error('Failed to create identity:', e);
      return null;
    } finally {
      setLoading(false);
    }
  };

  const importIdentity = async (mnemonic: string) => {
    if (!hasWailsRuntime()) return;
    setLoading(true);
    try {
      const id = await ImportIdentityFromMnemonic(mnemonic);
      await activateIdentity(id);
    } catch (e) {
      console.error('Failed to import identity:', e);
      throw e;
    } finally {
      setLoading(false);
    }
  };

  const handleCreateSub = async (id: string, title: string, description: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await CreateSub(id, title, description);
      await PublishCreateSub(id, title, description);
      await loadSubs();
      setCurrentSubId(id);
      addToast({
        title: 'Sub Created',
        message: `Successfully created sub ${id}`,
        type: 'success',
      });
    } catch (e) {
      const detail = getErrorMessage(e, 'Failed to create sub');
      console.error('Failed to create sub:', e);
      addToast({
        title: 'Error',
        message: detail,
        type: 'error',
      });
      throw new Error(detail);
    }
  };

  const handleCreatePost = async (input: CreatePostInput) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      const targetSubId = currentSubId === 'recommended' ? 'general' : currentSubId;
      const trimmedTitle = input.title.trim();
      const trimmedBody = input.body.trim();
      const effectiveBody = trimmedBody || trimmedTitle;
      const trimmedImage = (input.imageBase64 || '').trim();
      const trimmedMime = (input.imageMime || '').trim();
      const trimmedExternalImage = (input.externalImageURL || '').trim();

      if (trimmedImage && trimmedMime) {
        await PublishPostWithImageToSub(identity.publicKey, trimmedTitle, effectiveBody, trimmedImage, trimmedMime, targetSubId);
      } else {
        let finalBody = effectiveBody;
        if (trimmedExternalImage) {
          const markdownImage = `![image](${trimmedExternalImage})`;
          finalBody = finalBody ? `${finalBody}\n\n${markdownImage}` : markdownImage;
        }
        await PublishPostStructuredToSub(identity.publicKey, trimmedTitle, finalBody, targetSubId);
      }
      trackPendingSyncAction('post-create', trimmedTitle, `Post created in r/${targetSubId}: ${trimmedTitle}`);
      await loadPosts(currentSubId, sortMode);
      bumpViewSyncToken();
      addToast(getWriteFeedback(p2pStatus, 'Your post is live.', 'Your post is saved. We will finish sending it in the background.'));
    } catch (e) {
      const detail = getErrorMessage(e, 'Failed to create post');
      console.error('Failed to create post:', e);
      addToast({
        title: 'Error',
        message: detail,
        type: 'error',
      });
      throw new Error(detail);
    }
  };

  const refreshPostScoreState = useCallback(async (postId: string) => {
    if (!hasWailsRuntime()) return;
    try {
      const index = await GetPostIndexByID(postId);
      const updated: Post = {
        id: index.id,
        pubkey: index.pubkey,
        title: index.title,
        bodyPreview: index.bodyPreview || '',
        contentCid: index.contentCid || '',
        imageCid: index.imageCid || '',
        thumbCid: index.thumbCid || '',
        imageMime: index.imageMime || '',
        imageSize: index.imageSize || 0,
        imageWidth: index.imageWidth || 0,
        imageHeight: index.imageHeight || 0,
        score: index.score || 0,
        timestamp: index.timestamp || 0,
        zone: (index.zone || 'public') as 'private' | 'public',
        subId: index.subId || 'general',
        visibility: index.visibility || 'normal',
      };

      setPosts((prev) => prev.map((p) => (p.id === postId ? { ...p, score: updated.score } : p)));
      setSelectedPost((prev) => (prev && prev.id === postId ? { ...prev, score: updated.score } : prev));
    } catch (error) {
      console.error('Failed to refresh post score:', error);
    }
  }, []);

  const handleUpvote = async (postId: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishPostUpvote(identity.publicKey, postId);
      trackPendingSyncAction('post-vote', postId, `Post vote queued for ${postId.slice(0, 8)}`);
      await refreshPostScoreState(postId);
      bumpViewSyncToken();
      if (shouldTrackPendingSync(p2pStatus)) {
        addToast(getWriteFeedback(p2pStatus, '', 'Your reaction is saved and will finish in the background.'));
      }
    } catch (e) {
      console.error('Failed to upvote:', e);
    }
  };

  const handleDownvote = async (postId: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishPostDownvote(identity.publicKey, postId);
      trackPendingSyncAction('post-vote', postId, `Post vote queued for ${postId.slice(0, 8)}`);
      await refreshPostScoreState(postId);
      bumpViewSyncToken();
      if (shouldTrackPendingSync(p2pStatus)) {
        addToast(getWriteFeedback(p2pStatus, '', 'Your reaction is saved and will finish in the background.'));
      }
    } catch (e) {
      console.error('Failed to downvote:', e);
    }
  };

  const handleToggleFavorite = async (postId: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      if (favoritePostIds.has(postId)) {
        await RemoveFavorite(postId);
        setFavoritePostIds((prev) => {
          const next = new Set(prev);
          next.delete(postId);
          return next;
        });
        addToast({ title: 'Removed', message: 'Removed from favorites', type: 'info' });
      } else {
        await AddFavorite(postId);
        setFavoritePostIds((prev) => new Set(prev).add(postId));
        addToast({ title: 'Saved', message: 'Added to favorites', type: 'success' });
      }
      bumpViewSyncToken();
    } catch (e) {
      console.error('Failed to toggle favorite:', e);
      addToast({ title: 'Error', message: 'Failed to update favorite', type: 'error' });
    }
  };

  const handlePostClick = async (post: Post) => {
    if (view !== 'post-detail') {
      setPostReturnView(view);
    }
    recordRecentlyViewed(post.id);
    bumpHistorySyncToken();
    setHasRemotePostUpdate(false);
    setSelectedPost(post);
    setView('post-detail');
    await loadPostDetail(post);
  };

  const handleSharePost = useCallback(async (post: Post) => {
    try {
      const shareLink = buildShareLink(post.id);
      await ClipboardSetText(shareLink);
      addToast({
        title: 'Link Copied',
        message: shareLink,
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to copy share link:', e);
      addToast({
        title: 'Error',
        message: 'Failed to copy share link',
        type: 'error',
      });
    }
  }, [addToast]);

  const handleShareSub = useCallback(async (subId: string) => {
    try {
      const shareLink = buildSubShareLink(subId);
      await ClipboardSetText(shareLink);
      addToast({
        title: 'Sub Link Copied',
        message: shareLink,
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to copy sub share link:', e);
      addToast({
        title: 'Error',
        message: 'Failed to copy sub link',
        type: 'error',
      });
    }
  }, [addToast]);

  const trackPendingSyncAction = useCallback((kind: PendingSyncActionKind, entityId: string, summary: string) => {
    if (!shouldTrackPendingSync(p2pStatus)) {
      return;
    }
    const next = recordPendingSyncAction(kind, entityId, summary);
    setPendingSyncActions(next);
  }, [p2pStatus]);

  const handleDismissPendingSyncAction = useCallback((actionId: string) => {
    setPendingSyncActions(removePendingSyncAction(actionId));
  }, []);

  const handleTriggerSyncNow = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    setNetworkBusy(true);
    try {
      await TriggerAntiEntropySyncNow();
      await loadNetworkHealth();
      addToast({
        title: 'Sync Triggered',
        message: 'Manual anti-entropy sync started.',
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to trigger sync:', e);
      addToast({
        title: 'Sync Failed',
        message: getErrorMessage(e, 'Could not trigger sync right now.'),
        type: 'error',
      });
    } finally {
      setNetworkBusy(false);
    }
  }, [addToast, loadNetworkHealth]);

  const handleRepairPost = useCallback(async (postId: string) => {
    if (!hasWailsRuntime()) return;
    setNetworkBusy(true);
    try {
      await TriggerAntiEntropySyncNow();
      const index = await GetPostIndexByID(postId);
      const repairedPost = mapPostIndexToPost(index);
      setPosts((prev) => prev.map((item) => (item.id === postId ? { ...item, ...repairedPost } : item)));
      if (selectedPost?.id === postId) {
        setSelectedPost((prev) => (prev ? { ...prev, ...repairedPost } : prev));
        await loadPostDetail(repairedPost);
      }
      await loadNetworkHealth();
      addToast({
        title: 'Recovery Requested',
        message: 'Your node requested content repair for this post.',
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to repair post:', e);
      addToast({
        title: 'Recovery Failed',
        message: getErrorMessage(e, 'Could not recover this post right now.'),
        type: 'error',
      });
    } finally {
      setNetworkBusy(false);
    }
  }, [addToast, loadNetworkHealth, loadPostDetail, selectedPost]);

  const refreshCommentsForSelectedPost = useCallback(async (postId: string) => {
    if (!hasWailsRuntime()) return;
    if (!selectedPost || selectedPost.id !== postId) return;
    try {
      const comments = await GetCommentsByPost(postId);
      setPostComments(comments);
    } catch (e) {
      console.error('Failed to refresh comments:', e);
    }
  }, [selectedPost]);

  const handleBackToFeed = () => {
    setSelectedPost(null);
    setPostBody('');
    setPostComments([]);
    setHasRemotePostUpdate(false);
    setView(postReturnView);
  };

  const openPostById = useCallback(async (postId: string) => {
    const index = await GetPostIndexByID(postId);
    const post = mapPostIndexToPost(index);
    setCurrentSubId(post.subId || 'general');
    await handlePostClick(post);
  }, [handlePostClick]);

  const handleOpenPendingSyncAction = useCallback(async (action: PendingSyncAction) => {
    if (action.kind.startsWith('post-') || action.kind === 'comment-create' || action.kind === 'comment-edit' || action.kind === 'comment-delete' || action.kind === 'comment-vote') {
      await openPostById(action.entityId);
      return;
    }
    if (action.kind === 'profile-publish' && identity?.publicKey) {
      await openProfile(identity.publicKey);
    }
  }, [identity?.publicKey, openPostById, openProfile]);

  const handleRefreshSelectedPost = useCallback(async () => {
    if (!selectedPost) return;
    const index = await GetPostIndexByID(selectedPost.id);
    const updatedPost = mapPostIndexToPost(index);
    setSelectedPost(updatedPost);
    await loadPostDetail(updatedPost);
    setHasRemotePostUpdate(false);
  }, [loadPostDetail, selectedPost]);

  const handleSubSelect = (subId: string) => {
    if (subId === 'recommended') {
      setCurrentSubId('recommended');
      setView('feed');
    } else {
      setCurrentSubId(subId);
      setView('feed');
    }
  };

  const handleDiscoverClick = () => {
    setView('discover');
  };

  const handleToggleSubscription = async (subId: string) => {
    if (!hasWailsRuntime()) return;
    try {
      if (subscribedSubIds.has(subId)) {
        await UnsubscribeSub(subId);
        setSubscribedSubIds((prev) => {
          const next = new Set(prev);
          next.delete(subId);
          return next;
        });
        setSubscribedSubs((prev) => prev.filter((s) => s.id !== subId));
        addToast({
          title: 'Unsubscribed',
          message: `You have unsubscribed from ${subId}`,
          type: 'info',
        });
      } else {
        await SubscribeSub(subId);
        const sub = subs.find((s) => s.id === subId);
        if (sub) {
          setSubscribedSubs((prev) => [...prev, sub]);
          setSubscribedSubIds((prev) => new Set(prev).add(subId));
        }
        addToast({
          title: 'Subscribed',
          message: `You are now subscribed to ${subId}`,
          type: 'success',
        });
      }
    } catch (e) {
      console.error('Failed to toggle subscription:', e);
      addToast({
        title: 'Error',
        message: 'Failed to update subscription',
        type: 'error',
      });
    }
  };

  const handleSearch = async (query: string, scope?: string) => {
    setSearchQuery(query);
    setSearchScopeSubId(scope ? scope : null);
    if (!query.trim()) {
      setSearchResults(null);
      if (view === 'search') {
        setView('feed');
      }
      return;
    }
    if (!hasWailsRuntime()) return;
    try {
      const [subResults, postResults] = await Promise.all([
        scope ? Promise.resolve([]) : SearchSubs(query, 10),
        SearchPosts(query, scope || '', 100),
      ]);
      const uniquePubkeys = Array.from(new Set((postResults || []).map((post: any) => post.pubkey).filter(Boolean)));
      if (uniquePubkeys.length > 0) {
        const resolvedProfiles = await Promise.all(
          uniquePubkeys.map(async (pk) => {
            try {
              const resolved = await GetProfile(pk);
              return [pk, resolved] as const;
            } catch {
              return null;
            }
          })
        );
        const mergedProfiles: Record<string, Profile> = {};
        for (const entry of resolvedProfiles) {
          if (!entry) continue;
          mergedProfiles[entry[0]] = entry[1];
        }
        if (Object.keys(mergedProfiles).length > 0) {
          setProfiles((prev) => ({ ...prev, ...mergedProfiles }));
        }
      }
      setSearchResults({ subs: subResults, posts: postResults });
      if (query.trim().length >= 2) {
        setView('search');
      }
    } catch (e) {
      console.error('Failed to search:', e);
    }
  };

  const handleSearchResultClick = async (type: 'sub' | 'post', id: string) => {
    setSearchResults(null);
    setSearchQuery('');
    setSearchScopeSubId(null);
    if (type === 'sub') {
      setCurrentSubId(id);
      setView('feed');
    } else {
      try {
        const index = await GetPostIndexByID(id);
        const post: Post = mapPostIndexToPost(index);
        setCurrentSubId(post.subId || 'general');
        await handlePostClick(post);
      } catch (e) {
        console.error('Failed to open post from search result:', e);
      }
    }
  };

  const handleCommentReply = async (parentId: string, body: string, localImageDataURLs: string[] = [], externalImageURLs: string[] = []) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishCommentWithAttachments(identity.publicKey, selectedPost.id, parentId, body, localImageDataURLs, externalImageURLs);
      trackPendingSyncAction('comment-create', selectedPost.id, `Comment saved locally on post ${selectedPost.id.slice(0, 8)}`);
      void refreshCommentsForSelectedPost(selectedPost.id);
      addToast(getWriteFeedback(p2pStatus, 'Your reply is posted.', 'Your reply is saved. We will finish sending it in the background.'));
    } catch (e) {
      const detail = getErrorMessage(e, 'Failed to post reply');
      console.error('Failed to post comment:', e);
      addToast({
        title: 'Error',
        message: detail,
        type: 'error',
      });
      throw new Error(detail);
    }
  };

  const handleDeletePost = async (postId: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishDeletePost(identity.publicKey, postId);
      trackPendingSyncAction('post-delete', postId, `Post delete queued for ${postId.slice(0, 8)}`);
      setSelectedPost(null);
      setPostBody('');
      setPostComments([]);
      setView('feed');
      await loadPosts(currentSubId, sortMode);
      bumpViewSyncToken();
      addToast(getWriteFeedback(p2pStatus, 'Post has been deleted.', 'That change is saved and will finish in the background.'));
    } catch (e) {
      const detail = getErrorMessage(e, 'Failed to delete post');
      console.error('Failed to delete post:', e);
      addToast({ title: 'Error', message: detail, type: 'error' });
      throw new Error(detail);
    }
  };

  const handleDeleteComment = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishDeleteComment(identity.publicKey, commentId);
      trackPendingSyncAction('comment-delete', selectedPost.id, `Comment delete queued for ${commentId.slice(0, 8)}`);
      const comments = await GetCommentsByPost(selectedPost.id);
      setPostComments(comments);
      bumpViewSyncToken();
      addToast(getWriteFeedback(p2pStatus, 'Comment has been deleted.', 'That change is saved and will finish in the background.'));
    } catch (e) {
      const detail = getErrorMessage(e, 'Failed to delete comment');
      console.error('Failed to delete comment:', e);
      addToast({ title: 'Error', message: detail, type: 'error' });
      throw new Error(detail);
    }
  };

  const handleUpdatePost = async (postId: string, title: string, body: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishPostUpdate(identity.publicKey, postId, title, body);
      trackPendingSyncAction('post-edit', postId, `Post edit queued for ${postId.slice(0, 8)}`);
      const index = await GetPostIndexByID(postId);
      const updatedPost = mapPostIndexToPost(index);
      setPosts((prev) => prev.map((item) => (item.id === postId ? { ...item, ...updatedPost } : item)));
      setSelectedPost((prev) => (prev && prev.id === postId ? { ...prev, ...updatedPost } : prev));
      const latestBody = await GetPostBodyByID(postId);
      setPostBody(latestBody.body || body);
      bumpViewSyncToken();
      addToast(getWriteFeedback(p2pStatus, 'Your post has been updated.', 'Your changes are saved and will finish in the background.'));
    } catch (e) {
      const detail = getErrorMessage(e, 'Failed to update post');
      console.error('Failed to update post:', e);
      addToast({ title: 'Error', message: detail, type: 'error' });
      throw new Error(detail);
    }
  };

  const handleUpdateComment = async (commentId: string, body: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishCommentUpdate(identity.publicKey, commentId, body);
      trackPendingSyncAction('comment-edit', selectedPost.id, `Comment edit queued for ${commentId.slice(0, 8)}`);
      const comments = await GetCommentsByPost(selectedPost.id);
      setPostComments(comments);
      bumpViewSyncToken();
      addToast(getWriteFeedback(p2pStatus, 'Your comment has been updated.', 'Your changes are saved and will finish in the background.'));
    } catch (e) {
      const detail = getErrorMessage(e, 'Failed to update comment');
      console.error('Failed to update comment:', e);
      addToast({ title: 'Error', message: detail, type: 'error' });
      throw new Error(detail);
    }
  };

  const handleCommentUpvote = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishCommentUpvote(identity.publicKey, selectedPost.id, commentId);
      trackPendingSyncAction('comment-vote', selectedPost.id, `Comment vote queued for ${commentId.slice(0, 8)}`);
      await refreshCommentsForSelectedPost(selectedPost.id);
    } catch (e) {
      console.error('Failed to upvote comment:', e);
    }
  };

  const handleCommentDownvote = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishCommentDownvote(identity.publicKey, selectedPost.id, commentId);
      trackPendingSyncAction('comment-vote', selectedPost.id, `Comment vote queued for ${commentId.slice(0, 8)}`);
      await refreshCommentsForSelectedPost(selectedPost.id);
    } catch (e) {
      console.error('Failed to downvote comment:', e);
    }
  };

  const handleSaveProfile = async (displayName: string, avatarURL: string, bio: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      const p = await UpdateProfileDetails(displayName, avatarURL, bio);
      setProfile(p);
      if (p.pubkey) {
        setProfiles((prev) => ({ ...prev, [p.pubkey]: p }));
      }
      addToast({
        title: 'Profile Saved',
        message: 'Your profile has been updated locally',
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to save profile:', e);
      addToast({
        title: 'Error',
        message: 'Failed to save profile',
        type: 'error',
      });
      throw e;
    }
  };

  const handlePublishProfile = async (displayName: string, avatarURL: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishProfileUpdate(identity.publicKey, displayName, avatarURL);
      trackPendingSyncAction('profile-publish', identity.publicKey, 'Profile publish queued for replication');
      addToast(getWriteFeedback(p2pStatus, 'Your profile has been updated.', 'Your profile changes are saved and will finish in the background.'));
    } catch (e) {
      console.error('Failed to publish profile:', e);
      addToast({
        title: 'Error',
        message: 'Failed to publish profile',
        type: 'error',
      });
      throw e;
    }
  };

  const handleBanUser = async (targetPubkey: string, reason: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishShadowBan(targetPubkey, identity.publicKey, reason);
      await loadGovernanceData(identity.publicKey);
      addToast({
        title: 'User Banned',
        message: 'Shadow ban has been applied',
        type: 'warning',
      });
    } catch (e) {
      console.error('Failed to ban user:', e);
      addToast({
        title: 'Error',
        message: 'Failed to ban user',
        type: 'error',
      });
    }
  };

  const handleUnbanUser = async (targetPubkey: string, reason: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishUnban(targetPubkey, identity.publicKey, reason);
      await loadGovernanceData(identity.publicKey);
      addToast({
        title: 'User Unbanned',
        message: 'Ban has been lifted',
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to unban user:', e);
      addToast({
        title: 'Error',
        message: 'Failed to unban user',
        type: 'error',
      });
    }
  };

  const handleSignOut = () => {
    setIdentity(null);
    setProfile(null);
    setSelectedProfile(null);
    setSelectedProfilePosts([]);
    setCurrentSubStats(null);
    setShowLoginModal(true);
  };

  const handleViewOperationTimeline = (entityType: 'post' | 'comment', entityId: string) => {
    const normalizedID = entityId.trim();
    if (!normalizedID) return;
    setConsistencyFocus({ entityType, entityId: normalizedID, nonce: Date.now() });
    setShowSettingsPanel(true);
  };

  const toggleTheme = () => {
    const newDark = !isDark;
    setIsDark(newDark);
    if (newDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  };

  useEffect(() => {
    if (hasWailsRuntime()) {
      loadIdentity();
      return;
    }
    // Wails runtime may not be injected yet; poll until ready
    const timer = window.setInterval(() => {
      if (hasWailsRuntime()) {
        window.clearInterval(timer);
        loadIdentity();
      }
    }, 50);
    return () => window.clearInterval(timer);
  }, [loadIdentity]);

  useEffect(() => {
    if (identity) {
      loadSubs();
      loadSubscribedSubs();
    }
  }, [identity, loadSubs, loadSubscribedSubs]);

  useEffect(() => {
    if (identity && view === 'feed') {
      loadPosts(currentSubId, sortMode);
    }
  }, [identity, currentSubId, sortMode, loadPosts, view]);

  useEffect(() => {
    if (!identity) return;
    void loadSubStats(currentSubId);
  }, [identity, currentSubId, loadSubStats]);

  useEffect(() => {
    if (identityChecked && !identity) {
      setShowLoginModal(true);
    }
  }, [identity, identityChecked]);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    const unsubscribe = EventsOn('comments:updated', (payload: { postId?: string } | undefined) => {
      const postId = payload?.postId;
      if (!postId) return;
      void refreshCommentsForSelectedPost(postId);
    });
    return () => {
      unsubscribe();
    };
  }, [refreshCommentsForSelectedPost]);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    const unsubscribe = EventsOn('sub:updated', (payload: any) => {
      if (payload && payload.subId && subscribedSubIds.has(payload.subId)) {
        if (payload.subId !== currentSubId) {
          setUnreadSubs((prev) => new Set(prev).add(payload.subId));
        }

        if (payload.pubkey !== identity?.publicKey) {
          addToast({
            title: `New Post in ${payload.subId}`,
            message: payload.title || 'New content available',
            type: 'info',
            duration: 6000,
            onClick: () => {
              if (currentSubId !== payload.subId) {
                setCurrentSubId(payload.subId);
              } else {
                void loadPosts(payload.subId, sortMode);
              }
            }
          });
        }
      }
    });
    return () => unsubscribe();
  }, [identity, subscribedSubIds, currentSubId, sortMode, addToast, loadPosts]);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    const unsubscribe = EventsOn('subs:updated', () => {
      void loadSubs();
      if (identity) {
        void loadSubscribedSubs();
      }
    });
    return () => {
      unsubscribe();
    };
  }, [identity, loadSubs, loadSubscribedSubs]);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    const unsubscribe = EventsOn('favorites:updated', (payload: { postId?: string } | undefined) => {
      void loadFavorites();
      bumpViewSyncToken();
    });
    return () => {
      unsubscribe();
    };
  }, [loadFavorites, bumpViewSyncToken]);

  useEffect(() => {
    if (!hasWailsRuntime() || !identity) return;
    const refreshCount = () => {
      GetUnreadNotificationCount().then((c) => setUnreadNotificationCount(c)).catch(console.error);
    };
    refreshCount();
    const unsubscribe = EventsOn('notifications:updated', refreshCount);
    return () => {
      unsubscribe();
    };
  }, [identity]);

  useEffect(() => {
    if (!hasWailsRuntime() || !identity) return;
    const timer = window.setInterval(() => {
      void loadSubs();
      void loadSubscribedSubs();
    }, 20000);
    return () => {
      window.clearInterval(timer);
    };
  }, [identity, loadSubs, loadSubscribedSubs]);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    const unsubscribe = EventsOn('feed:updated', () => {
      bumpViewSyncToken();
      if (!identity) return;
      if (view === 'feed') {
        void loadPosts(currentSubId, sortMode);
      } else if (view === 'post-detail' && selectedPost) {
        setHasRemotePostUpdate(true);
      }
      void loadNetworkHealth();
    });
    return () => {
      unsubscribe();
    };
  }, [identity, view, currentSubId, sortMode, loadPosts, bumpViewSyncToken, loadNetworkHealth, selectedPost]);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    if (!identity) {
      setOnlineCount(0);
      setP2PStatus(null);
      setAntiEntropyStats(null);
      return;
    }

    let alive = true;
    const refresh = async () => {
      if (!alive) return;
      await loadNetworkHealth();
    };

    void refresh();
    const timer = window.setInterval(() => {
      void refresh();
    }, 15000);

    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [identity, loadNetworkHealth]);

  useEffect(() => {
    const nextHash = (() => {
      if (view === 'post-detail' && selectedPost) {
        return buildAppHash(`/post/${encodeURIComponent(selectedPost.id)}`);
      }
      if (view === 'profile' && selectedProfile?.pubkey) {
        return buildAppHash(`/u/${encodeURIComponent(selectedProfile.pubkey)}`);
      }
      if (view === 'search' && searchQuery.trim()) {
        return buildAppHash(`/search?q=${encodeURIComponent(searchQuery.trim())}${searchScopeSubId ? `&sub=${encodeURIComponent(searchScopeSubId)}` : ''}`);
      }
      if (view === 'feed') {
        return buildAppHash(`/r/${encodeURIComponent(currentSubId)}`);
      }
      return buildAppHash(`/${view}`);
    })();

    if (window.location.hash !== nextHash) {
      window.history.replaceState(null, '', nextHash);
    }
  }, [currentSubId, searchQuery, searchScopeSubId, selectedPost, selectedProfile, view]);

  useEffect(() => {
    if (!identity || !hasWailsRuntime()) return;

    let alive = true;
    const applyHashRoute = async () => {
      const hash = window.location.hash.replace(/^#/, '').trim();
      if (!hash) return;

      const [pathPart, queryString = ''] = hash.split('?');
      const normalizedPath = pathPart.startsWith('/') ? pathPart : `/${pathPart}`;
      const segments = normalizedPath.split('/').filter(Boolean);

      try {
        if (segments[0] === 'post' && segments[1]) {
          const postId = decodeURIComponent(segments[1]);
          await openPostById(postId);
          return;
        }

        if (segments[0] === 'u' && segments[1]) {
          const pubkey = decodeURIComponent(segments[1]);
          await openProfile(pubkey);
          return;
        }

        if (segments[0] === 'r' && segments[1]) {
          const subId = decodeURIComponent(segments[1]);
          if (!alive) return;
          setSelectedPost(null);
          setView('feed');
          setCurrentSubId(subId);
          return;
        }

        if (segments[0] === 'search') {
          const params = new URLSearchParams(queryString);
          const query = params.get('q') || '';
          const scope = params.get('sub') || undefined;
          if (query.trim()) {
            await handleSearch(query, scope);
          }
        }
      } catch (error) {
        console.error('Failed to resolve deep link:', error);
      }
    };

    void applyHashRoute();
    const onHashChange = () => {
      void applyHashRoute();
    };
    window.addEventListener('hashchange', onHashChange);
    return () => {
      alive = false;
      window.removeEventListener('hashchange', onHashChange);
    };
  }, [handleSearch, identity, openPostById, openProfile]);

  const currentSub = currentSubId === 'recommended'
    ? { id: 'recommended', title: 'Recommended Feed', description: 'Your personalized feed based on subscriptions and trending posts' }
    : (subs.find((s) => s.id === currentSubId) || { id: currentSubId, title: currentSubId, description: '' });

  const membersCount = new Set(posts.map((post) => post.pubkey).filter((value) => !!value)).size;

  const isCurrentSubSubscribed = currentSubId !== 'recommended' && subscribedSubIds.has(currentSubId);

  return (
    <div className={`h-screen flex flex-col ${isDark ? 'dark' : ''}`}>
      <NetworkBanner
        health={networkHealth}
        pendingSyncActions={pendingSyncActions}
        busy={networkBusy}
        onSyncNow={() => void handleTriggerSyncNow()}
        onOpenNetworkSettings={() => setShowSettingsPanel(true)}
      />
      <div className="flex-1 flex overflow-hidden" style={{ minWidth: '900px' }}>
        <Sidebar
          subs={subs}
          subscribedSubs={subscribedSubs}
          currentSubId={currentSubId}
          onSelectSub={handleSubSelect}
          onDiscoverClick={handleDiscoverClick}
          onCreateSub={() => setShowCreateSubModal(true)}
          unreadSubs={unreadSubs}
        />

        <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
          <Header
            currentSubId={currentSubId}
            profile={profile || undefined}
            onCreatePost={() => setShowCreatePostModal(true)}
            onProfileClick={() => setShowSettingsPanel(true)}
            onViewOwnProfile={() => {
              if (identity?.publicKey) {
                void openProfile(identity.publicKey);
              }
            }}
            onMyPostsClick={() => setView('my-posts')}
            onDraftsClick={() => {
              bumpDraftSyncToken();
              setView('drafts');
            }}
            onHistoryClick={() => {
              bumpHistorySyncToken();
              setView('history');
            }}
            onPendingSyncClick={() => setView('pending-sync')}
            onFavoritesClick={() => setView('favorites')}
            onSignOut={handleSignOut}
            isDark={isDark}
            onThemeToggle={toggleTheme}
            searchQuery={searchQuery}
            searchResults={searchResults}
            onSearch={handleSearch}
            onSearchResultClick={handleSearchResultClick}
            onSearchClear={() => {
              setSearchQuery('');
              setSearchResults(null);
              setSearchScopeSubId(null);
              if (view === 'search') {
                setView('feed');
              }
            }}
            unreadNotificationCount={unreadNotificationCount}
            onNotificationsClick={() => setView('notifications')}
          />

          {view === 'feed' && (
            <Feed
              posts={posts.map(p => ({ ...p, isFavorited: favoritePostIds.has(p.id) }))}
              sortMode={sortMode}
              profiles={profiles}
              onSortChange={setSortMode}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
              onAuthorClick={(pubkey) => void openProfile(pubkey)}
              onShare={(post) => void handleSharePost(post)}
              onToggleFavorite={handleToggleFavorite}
            />
          )}

          {view === 'discover' && (
            <DiscoverView
              subs={subs}
              subscribedSubIds={subscribedSubIds}
              onSubClick={handleSubSelect}
              onToggleSubscription={handleToggleSubscription}
            />
          )}

          {view === 'search' && (
            <SearchResultsView
              query={searchQuery}
              subs={searchResults?.subs || []}
              posts={(searchResults?.posts || []).map((post) => ({
                ...mapForumMessageToPost(post),
                isFavorited: favoritePostIds.has(post.id),
              }))}
              profiles={profiles}
              scopeSubId={searchScopeSubId || undefined}
              onSubClick={handleSubSelect}
              onPostClick={handlePostClick}
              onAuthorClick={(pubkey) => void openProfile(pubkey)}
              onShare={(post) => void handleSharePost(post)}
              onUpvote={handleUpvote}
              onToggleFavorite={handleToggleFavorite}
            />
          )}

          {view === 'profile' && (
            <ProfileView
              profile={selectedProfile}
              posts={selectedProfilePosts.map((post) => ({
                ...post,
                isFavorited: favoritePostIds.has(post.id),
              }))}
              profiles={profiles}
              isOwnProfile={selectedProfile?.pubkey === identity?.publicKey}
              onEditProfile={() => setShowSettingsPanel(true)}
              onPostClick={handlePostClick}
              onUpvote={handleUpvote}
              onShare={(post) => void handleSharePost(post)}
              onToggleFavorite={handleToggleFavorite}
            />
          )}

          {view === 'post-detail' && selectedPost && (
            <PostDetail
              post={{ ...selectedPost, isFavorited: favoritePostIds.has(selectedPost.id) }}
              body={postBody}
              comments={postComments}
              profiles={profiles}
              currentPubkey={identity?.publicKey}
              onAuthorClick={(pubkey) => void openProfile(pubkey)}
              onBack={handleBackToFeed}
              onUpvote={handleUpvote}
              onDownvote={handleDownvote}
              onShare={(post) => void handleSharePost(post)}
              onRepairPost={(postId) => void handleRepairPost(postId)}
              hasRemoteUpdate={hasRemotePostUpdate}
              onRefreshRemoteUpdate={() => void handleRefreshSelectedPost()}
              onReply={handleCommentReply}
              onCommentUpvote={handleCommentUpvote}
              onCommentDownvote={handleCommentDownvote}
              onDeletePost={handleDeletePost}
              onDeleteComment={handleDeleteComment}
              onEditPost={handleUpdatePost}
              onEditComment={handleUpdateComment}
              onViewOperationTimeline={handleViewOperationTimeline}
              isDevMode={isDevMode}
              onToggleFavorite={handleToggleFavorite}
            />
          )}

          {view === 'my-posts' && identity && (
            <MyPosts
              currentPubkey={identity.publicKey}
              refreshToken={viewSyncToken}
              profiles={profiles}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
              onAuthorClick={(pubkey) => void openProfile(pubkey)}
              onShare={(post) => void handleSharePost(post)}
            />
          )}

          {view === 'favorites' && (
            <Favorites
              allPosts={posts}
              refreshToken={viewSyncToken}
              currentPubkey={identity?.publicKey}
              profiles={profiles}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
              onAuthorClick={(pubkey) => void openProfile(pubkey)}
              onShare={(post) => void handleSharePost(post)}
              onToggleFavorite={handleToggleFavorite}
            />
          )}

          {view === 'drafts' && (
            <DraftsView
              authorPublicKey={identity?.publicKey}
              refreshToken={draftSyncToken}
              onOpenPostDraft={(subId) => {
                setCurrentSubId(subId);
                setView('feed');
                setShowCreatePostModal(true);
              }}
              onOpenCommentDraft={(postId) => {
                void openPostById(postId);
              }}
            />
          )}

          {view === 'history' && (
            <HistoryView
              refreshToken={historySyncToken}
              profiles={profiles}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
              onAuthorClick={(pubkey) => void openProfile(pubkey)}
              onShare={(post) => void handleSharePost(post)}
            />
          )}

          {view === 'pending-sync' && (
            <PendingSyncView
              actions={pendingSyncActions}
              onOpenAction={(action) => void handleOpenPendingSyncAction(action)}
              onDismissAction={handleDismissPendingSyncAction}
              onSyncNow={() => void handleTriggerSyncNow()}
            />
          )}

          {view === 'notifications' && (
            <NotificationsView
              profiles={profiles}
              onNavigateToPost={(postId) => void openPostById(postId)}
              onNavigateToProfile={() => {
                if (identity?.publicKey) {
                  void openProfile(identity.publicKey);
                }
              }}
            />
          )}
        </div>

        {view === 'feed' && (
          <RightPanel
            sub={currentSub}
            isSubscribed={isCurrentSubSubscribed}
            stats={currentSubStats || undefined}
            membersCount={membersCount}
            onlineCount={onlineCount}
            onCreatePost={() => setShowCreatePostModal(true)}
            onShareSub={() => void handleShareSub(currentSubId)}
            p2pStatus={p2pStatus}
            antiEntropyStats={antiEntropyStats}
            pendingSyncActions={pendingSyncActions}
            networkBusy={networkBusy}
            onSyncNow={() => void handleTriggerSyncNow()}
            onOpenNetworkSettings={() => setShowSettingsPanel(true)}
            onDismissPendingSyncAction={handleDismissPendingSyncAction}
            onToggleSubscription={() => handleToggleSubscription(currentSubId)}
          />
        )}
      </div>

      <SettingsPanel
        isOpen={showSettingsPanel}
        onClose={() => setShowSettingsPanel(false)}
        profile={profile || undefined}
        isAdmin={isAdmin}
        governanceAdmins={governanceAdmins}
        moderationStates={moderationStates}
        moderationLogs={moderationLogs}
        onSaveProfile={handleSaveProfile}
        onPublishProfile={handlePublishProfile}
        onBanUser={handleBanUser}
        onUnbanUser={handleUnbanUser}
        consistencyFocus={consistencyFocus}
        isDevMode={isDevMode}
      />

      <LoginModal
        isOpen={showLoginModal && !identity}
        onClose={() => setShowLoginModal(false)}
        onCreateIdentity={createIdentity}
        onActivateIdentity={activateIdentity}
        onLoadIdentity={loadIdentity}
        onImportMnemonic={importIdentity}
      />

      <CreateSubModal
        isOpen={showCreateSubModal}
        onClose={() => setShowCreateSubModal(false)}
        onCreate={handleCreateSub}
      />

      <CreatePostModal
        isOpen={showCreatePostModal}
        onClose={() => {
          setShowCreatePostModal(false);
          bumpDraftSyncToken();
        }}
        subId={currentSubId === 'recommended' ? 'general' : currentSubId}
        authorPublicKey={identity?.publicKey}
        onCreate={handleCreatePost}
      />

      <ToastContainer toasts={toasts} onClose={removeToast} />
    </div>
  );
}

export default App;
