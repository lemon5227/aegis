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
  GetProfile,
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
  PublishDeletePost,
  PublishDeleteComment,
  IsDevMode,
  GetFavoritePostIDs,
  AddFavorite,
  RemoveFavorite,
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
import { ToastContainer, useToasts } from './components/Toast';
import { Sub, Profile, Post, GovernanceAdmin, Identity, Comment, ModerationLog, ModerationState } from './types';
import { EventsOn } from '../wailsjs/runtime/runtime';

type SortMode = 'hot' | 'new';
type ViewMode = 'feed' | 'discover' | 'post-detail' | 'my-posts' | 'favorites';
type ConsistencyFocus = {
  entityType: 'post' | 'comment';
  entityId: string;
  nonce: number;
};

function App() {
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [isAdmin, setIsAdmin] = useState(false);
  const [subs, setSubs] = useState<Sub[]>([]);
  const [subscribedSubs, setSubscribedSubs] = useState<Sub[]>([]);
  const [subscribedSubIds, setSubscribedSubIds] = useState<Set<string>>(new Set());
  const [currentSubId, setCurrentSubId] = useState<string>('general');
  const [view, setView] = useState<ViewMode>('feed');
  const [sortMode, setSortMode] = useState<SortMode>('hot');
  const [posts, setPosts] = useState<Array<Post & { reason?: string; isSubscribed?: boolean }>>([]);
  const [profiles, setProfiles] = useState<Record<string, Profile>>({});
  const [isDark, setIsDark] = useState(false);
  const [showLoginModal, setShowLoginModal] = useState(false);
  const [showCreateSubModal, setShowCreateSubModal] = useState(false);
  const [showCreatePostModal, setShowCreatePostModal] = useState(false);
  const [showSettingsPanel, setShowSettingsPanel] = useState(false);
  const [consistencyFocus, setConsistencyFocus] = useState<ConsistencyFocus | null>(null);
  const [loading, setLoading] = useState(false);
  const [searchResults, setSearchResults] = useState<{ subs: Sub[]; posts: any[] } | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [unreadSubs, setUnreadSubs] = useState<Set<string>>(new Set());

  const [selectedPost, setSelectedPost] = useState<Post | null>(null);
  const [postBody, setPostBody] = useState<string>('');
  const [postComments, setPostComments] = useState<Comment[]>([]);
  const [governanceAdmins, setGovernanceAdmins] = useState<GovernanceAdmin[]>([]);
  const [moderationStates, setModerationStates] = useState<ModerationState[]>([]);
  const [moderationLogs, setModerationLogs] = useState<ModerationLog[]>([]);
  const [onlineCount, setOnlineCount] = useState(0);
  const [isDevMode, setIsDevMode] = useState(false);

  const { toasts, addToast, removeToast } = useToasts();

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    IsDevMode().then(setIsDevMode).catch(console.error);
  }, []);

  const hasWailsRuntime = () => {
    return !!(window as any)?.go?.main?.App;
  };

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

  const loadIdentity = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    try {
      const id = await LoadSavedIdentity();
      setIdentity(id);
      if (id.publicKey) {
        const p = await GetProfileDetails(id.publicKey);
        setProfile(p);
        setProfiles((prev) => ({ ...prev, [id.publicKey]: p }));
        await loadGovernanceData(id.publicKey);
        await loadFavorites();
      }
    } catch (e) {
      console.log('No saved identity');
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

  const activateIdentity = useCallback(async (id: Identity) => {
    setIdentity(id);
    if (id.publicKey) {
      const p = await GetProfileDetails(id.publicKey);
      setProfile(p);
      setProfiles((prev) => ({ ...prev, [id.publicKey]: p }));
      await loadGovernanceData(id.publicKey);
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
        id: item.post.id,
        pubkey: item.post.pubkey,
        title: item.post.title,
        bodyPreview: item.post.body || '',
        contentCid: item.post.contentCid || '',
        imageCid: item.post.imageCid || '',
        thumbCid: item.post.thumbCid || '',
        imageMime: item.post.imageMime || '',
        imageSize: item.post.imageSize || 0,
        imageWidth: item.post.imageWidth || 0,
        imageHeight: item.post.imageHeight || 0,
        score: item.post.score || 0,
        timestamp: item.post.timestamp || 0,
        zone: (item.post.zone || 'public') as 'private' | 'public',
        subId: item.post.subId || 'general',
        visibility: item.post.visibility || 'normal',
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
      const mapped = (feed as any[]).map((item) => ({
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
      }));
      setPosts(mapped);
      // Clear unread status when loading the sub
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
      console.error('Failed to create sub:', e);
      addToast({
        title: 'Error',
        message: 'Failed to create sub',
        type: 'error',
      });
    }
  };

  const handleCreatePost = async (title: string, body: string, imageBase64?: string, imageMime?: string, externalImageURL?: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      const targetSubId = currentSubId === 'recommended' ? 'general' : currentSubId;
      const trimmedTitle = title.trim();
      const trimmedBody = body.trim();
      const effectiveBody = trimmedBody || trimmedTitle;
      const trimmedImage = (imageBase64 || '').trim();
      const trimmedMime = (imageMime || '').trim();
      const trimmedExternalImage = (externalImageURL || '').trim();

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
      await loadPosts(currentSubId, sortMode);
      addToast({
        title: 'Post Created',
        message: 'Your post has been published',
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to create post:', e);
      addToast({
        title: 'Error',
        message: 'Failed to create post',
        type: 'error',
      });
      throw e;
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
      await refreshPostScoreState(postId);
    } catch (e) {
      console.error('Failed to upvote:', e);
    }
  };

  const handleDownvote = async (postId: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishPostDownvote(identity.publicKey, postId);
      await refreshPostScoreState(postId);
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
    } catch (e) {
      console.error('Failed to toggle favorite:', e);
      addToast({ title: 'Error', message: 'Failed to update favorite', type: 'error' });
    }
  };

  const handlePostClick = async (post: Post) => {
    setSelectedPost(post);
    setView('post-detail');
    await loadPostDetail(post);
  };

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
    setView('feed');
  };

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
    if (!query.trim()) {
      setSearchResults(null);
      return;
    }
    if (!hasWailsRuntime()) return;
    try {
      const [subResults, postResults] = await Promise.all([
        scope ? Promise.resolve([]) : SearchSubs(query, 10),
        SearchPosts(query, scope || '', 10),
      ]);
      setSearchResults({ subs: subResults, posts: postResults });
    } catch (e) {
      console.error('Failed to search:', e);
    }
  };

  const handleSearchResultClick = async (type: 'sub' | 'post', id: string) => {
    setSearchResults(null);
    setSearchQuery('');
    if (type === 'sub') {
      setCurrentSubId(id);
      setView('feed');
    } else {
      try {
        const index = await GetPostIndexByID(id);
        const post: Post = {
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
      void refreshCommentsForSelectedPost(selectedPost.id);
      addToast({
        title: 'Reply Sent',
        message: 'Your reply has been posted',
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to post comment:', e);
      addToast({
        title: 'Error',
        message: 'Failed to post reply',
        type: 'error',
      });
      throw e;
    }
  };

  const handleDeletePost = async (postId: string) => {
    if (!hasWailsRuntime() || !identity) return;
    await PublishDeletePost(identity.publicKey, postId);
    setSelectedPost(null);
    setPostBody('');
    setPostComments([]);
    setView('feed');
    await loadPosts(currentSubId, sortMode);
    addToast({
      title: 'Post Deleted',
      message: 'Post has been deleted',
      type: 'info',
    });
  };

  const handleDeleteComment = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    await PublishDeleteComment(identity.publicKey, commentId);
    const comments = await GetCommentsByPost(selectedPost.id);
    setPostComments(comments);
    addToast({
      title: 'Comment Deleted',
      message: 'Comment has been deleted',
      type: 'info',
    });
  };

  const handleCommentUpvote = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishCommentUpvote(identity.publicKey, selectedPost.id, commentId);
      await refreshCommentsForSelectedPost(selectedPost.id);
    } catch (e) {
      console.error('Failed to upvote comment:', e);
    }
  };

  const handleCommentDownvote = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishCommentDownvote(identity.publicKey, selectedPost.id, commentId);
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
      addToast({
        title: 'Profile Published',
        message: 'Your profile update has been broadcasted',
        type: 'success',
      });
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
    loadIdentity();
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
    if (!identity) {
      setShowLoginModal(true);
    }
  }, [identity]);

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
        // Mark as unread if not currently viewing it
        if (payload.subId !== currentSubId) {
          setUnreadSubs((prev) => new Set(prev).add(payload.subId));
        }

        // Don't show toast for own posts to avoid redundancy
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
      // Just reload favorited ID list
      void loadFavorites();
    });
    return () => {
      unsubscribe();
    };
  }, [loadFavorites]);

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
      if (!identity || view !== 'feed') return;
      void loadPosts(currentSubId, sortMode);
    });
    return () => {
      unsubscribe();
    };
  }, [identity, view, currentSubId, sortMode, loadPosts]);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    if (!identity) {
      setOnlineCount(0);
      return;
    }

    let alive = true;
    const refresh = async () => {
      try {
        const status = await GetP2PStatus();
        if (!alive) return;
        if (!status?.started) {
          setOnlineCount(0);
          return;
        }
        const peers = Array.isArray(status.connectedPeers) ? status.connectedPeers.length : 0;
        setOnlineCount(peers + 1);
      } catch {
        if (alive) setOnlineCount(0);
      }
    };

    void refresh();
    const timer = window.setInterval(() => {
      void refresh();
    }, 15000);

    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [identity]);

  const currentSub = currentSubId === 'recommended'
    ? { id: 'recommended', title: 'Recommended Feed', description: 'Your personalized feed based on subscriptions and trending posts' }
    : (subs.find((s) => s.id === currentSubId) || { id: currentSubId, title: currentSubId, description: '' });

  const membersCount = new Set(posts.map((post) => post.pubkey).filter((value) => !!value)).size;

  const isCurrentSubSubscribed = currentSubId !== 'recommended' && subscribedSubIds.has(currentSubId);

  return (
    <div className={`h-screen flex flex-col ${isDark ? 'dark' : ''}`}>
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
            onMyPostsClick={() => setView('my-posts')}
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
            }}
          />

          {view === 'feed' && (
            <Feed
              posts={posts.map(p => ({ ...p, isFavorited: favoritePostIds.has(p.id) }))}
              sortMode={sortMode}
              profiles={profiles}
              onSortChange={setSortMode}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
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

          {view === 'post-detail' && selectedPost && (
            <PostDetail
              post={{ ...selectedPost, isFavorited: favoritePostIds.has(selectedPost.id) }}
              body={postBody}
              comments={postComments}
              profiles={profiles}
              currentPubkey={identity?.publicKey}
              onBack={handleBackToFeed}
              onUpvote={handleUpvote}
              onDownvote={handleDownvote}
              onReply={handleCommentReply}
              onCommentUpvote={handleCommentUpvote}
              onCommentDownvote={handleCommentDownvote}
              onDeletePost={handleDeletePost}
              onDeleteComment={handleDeleteComment}
              onViewOperationTimeline={handleViewOperationTimeline}
              isDevMode={isDevMode}
              onToggleFavorite={handleToggleFavorite}
            />
          )}

          {view === 'my-posts' && identity && (
            <MyPosts
              currentPubkey={identity.publicKey}
              profiles={profiles}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
            />
          )}

          {view === 'favorites' && (
            <Favorites
              allPosts={posts}
              profiles={profiles}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
              onToggleFavorite={handleToggleFavorite}
            />
          )}
        </div>

        {view === 'feed' && (
          <RightPanel
            sub={currentSub}
            isSubscribed={isCurrentSubSubscribed}
            membersCount={membersCount}
            onlineCount={onlineCount}
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
        onClose={() => setShowCreatePostModal(false)}
        onCreate={handleCreatePost}
      />

      <ToastContainer toasts={toasts} onClose={removeToast} />
    </div>
  );
}

export default App;
// ... (imports)
import { PublishPostUpdate } from '../wailsjs/go/main/App';

// ... (in App component)

  const handleEditPost = async (postId: string, title: string, body: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishPostUpdate(identity.publicKey, postId, title, body);
      // Optimistically update local state or just reload
      // Reloading is safer for consistency
      await loadPostDetail({ ...selectedPost!, title } as Post);
      // Also update the posts list
      setPosts(prev => prev.map(p => p.id === postId ? { ...p, title, bodyPreview: body.slice(0, 140) } : p));

      addToast({
        title: 'Post Updated',
        message: 'Your changes have been published.',
        type: 'success',
      });
    } catch (e: any) {
      console.error('Failed to update post:', e);
      addToast({
        title: 'Error',
        message: 'Failed to update post: ' + e.message,
        type: 'error',
      });
      throw e;
    }
  };

  // ... (in return JSX, passing handleEditPost to PostDetail)

          {view === 'post-detail' && selectedPost && (
            <PostDetail
              // ... existing props
              onEditPost={handleEditPost}
            />
          )}
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
  GetProfile,
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
  PublishDeletePost,
  PublishDeleteComment,
  IsDevMode,
  GetFavoritePostIDs,
  AddFavorite,
  RemoveFavorite,
  PublishPostUpdate, // Added import
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
import { ToastContainer, useToasts } from './components/Toast';
import { Sub, Profile, Post, GovernanceAdmin, Identity, Comment, ModerationLog, ModerationState } from './types';
import { EventsOn } from '../wailsjs/runtime/runtime';

type SortMode = 'hot' | 'new';
type ViewMode = 'feed' | 'discover' | 'post-detail' | 'my-posts' | 'favorites';
type ConsistencyFocus = {
  entityType: 'post' | 'comment';
  entityId: string;
  nonce: number;
};

function App() {
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [isAdmin, setIsAdmin] = useState(false);
  const [subs, setSubs] = useState<Sub[]>([]);
  const [subscribedSubs, setSubscribedSubs] = useState<Sub[]>([]);
  const [subscribedSubIds, setSubscribedSubIds] = useState<Set<string>>(new Set());
  const [currentSubId, setCurrentSubId] = useState<string>('general');
  const [view, setView] = useState<ViewMode>('feed');
  const [sortMode, setSortMode] = useState<SortMode>('hot');
  const [posts, setPosts] = useState<Array<Post & { reason?: string; isSubscribed?: boolean }>>([]);
  const [profiles, setProfiles] = useState<Record<string, Profile>>({});
  const [isDark, setIsDark] = useState(false);
  const [showLoginModal, setShowLoginModal] = useState(false);
  const [showCreateSubModal, setShowCreateSubModal] = useState(false);
  const [showCreatePostModal, setShowCreatePostModal] = useState(false);
  const [showSettingsPanel, setShowSettingsPanel] = useState(false);
  const [consistencyFocus, setConsistencyFocus] = useState<ConsistencyFocus | null>(null);
  const [loading, setLoading] = useState(false);
  const [searchResults, setSearchResults] = useState<{ subs: Sub[]; posts: any[] } | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [unreadSubs, setUnreadSubs] = useState<Set<string>>(new Set());
  const [favoritePostIds, setFavoritePostIds] = useState<Set<string>>(new Set());

  const [selectedPost, setSelectedPost] = useState<Post | null>(null);
  const [postBody, setPostBody] = useState<string>('');
  const [postComments, setPostComments] = useState<Comment[]>([]);
  const [governanceAdmins, setGovernanceAdmins] = useState<GovernanceAdmin[]>([]);
  const [moderationStates, setModerationStates] = useState<ModerationState[]>([]);
  const [moderationLogs, setModerationLogs] = useState<ModerationLog[]>([]);
  const [onlineCount, setOnlineCount] = useState(0);
  const [isDevMode, setIsDevMode] = useState(false);

  const { toasts, addToast, removeToast } = useToasts();

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    IsDevMode().then(setIsDevMode).catch(console.error);
  }, []);

  const hasWailsRuntime = () => {
    return !!(window as any)?.go?.main?.App;
  };

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

  const loadIdentity = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    try {
      const id = await LoadSavedIdentity();
      setIdentity(id);
      if (id.publicKey) {
        const p = await GetProfileDetails(id.publicKey);
        setProfile(p);
        setProfiles((prev) => ({ ...prev, [id.publicKey]: p }));
        await loadGovernanceData(id.publicKey);
        await loadFavorites();
      }
    } catch (e) {
      console.log('No saved identity');
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

  const activateIdentity = useCallback(async (id: Identity) => {
    setIdentity(id);
    if (id.publicKey) {
      const p = await GetProfileDetails(id.publicKey);
      setProfile(p);
      setProfiles((prev) => ({ ...prev, [id.publicKey]: p }));
      await loadGovernanceData(id.publicKey);
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
        id: item.post.id,
        pubkey: item.post.pubkey,
        title: item.post.title,
        bodyPreview: item.post.body || '',
        contentCid: item.post.contentCid || '',
        imageCid: item.post.imageCid || '',
        thumbCid: item.post.thumbCid || '',
        imageMime: item.post.imageMime || '',
        imageSize: item.post.imageSize || 0,
        imageWidth: item.post.imageWidth || 0,
        imageHeight: item.post.imageHeight || 0,
        score: item.post.score || 0,
        timestamp: item.post.timestamp || 0,
        zone: (item.post.zone || 'public') as 'private' | 'public',
        subId: item.post.subId || 'general',
        visibility: item.post.visibility || 'normal',
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
      const mapped = (feed as any[]).map((item) => ({
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
      }));
      setPosts(mapped);
      // Clear unread status when loading the sub
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
      console.error('Failed to create sub:', e);
      addToast({
        title: 'Error',
        message: 'Failed to create sub',
        type: 'error',
      });
    }
  };

  const handleCreatePost = async (title: string, body: string, imageBase64?: string, imageMime?: string, externalImageURL?: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      const targetSubId = currentSubId === 'recommended' ? 'general' : currentSubId;
      const trimmedTitle = title.trim();
      const trimmedBody = body.trim();
      const effectiveBody = trimmedBody || trimmedTitle;
      const trimmedImage = (imageBase64 || '').trim();
      const trimmedMime = (imageMime || '').trim();
      const trimmedExternalImage = (externalImageURL || '').trim();

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
      await loadPosts(currentSubId, sortMode);
      addToast({
        title: 'Post Created',
        message: 'Your post has been published',
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to create post:', e);
      addToast({
        title: 'Error',
        message: 'Failed to create post',
        type: 'error',
      });
      throw e;
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
      await refreshPostScoreState(postId);
    } catch (e) {
      console.error('Failed to upvote:', e);
    }
  };

  const handleDownvote = async (postId: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishPostDownvote(identity.publicKey, postId);
      await refreshPostScoreState(postId);
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
    } catch (e) {
      console.error('Failed to toggle favorite:', e);
      addToast({ title: 'Error', message: 'Failed to update favorite', type: 'error' });
    }
  };

  const handlePostClick = async (post: Post) => {
    setSelectedPost(post);
    setView('post-detail');
    await loadPostDetail(post);
  };

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
    setView('feed');
  };

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
    if (!query.trim()) {
      setSearchResults(null);
      return;
    }
    if (!hasWailsRuntime()) return;
    try {
      const [subResults, postResults] = await Promise.all([
        scope ? Promise.resolve([]) : SearchSubs(query, 10),
        SearchPosts(query, scope || '', 10),
      ]);
      setSearchResults({ subs: subResults, posts: postResults });
    } catch (e) {
      console.error('Failed to search:', e);
    }
  };

  const handleSearchResultClick = async (type: 'sub' | 'post', id: string) => {
    setSearchResults(null);
    setSearchQuery('');
    if (type === 'sub') {
      setCurrentSubId(id);
      setView('feed');
    } else {
      try {
        const index = await GetPostIndexByID(id);
        const post: Post = {
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
      void refreshCommentsForSelectedPost(selectedPost.id);
      addToast({
        title: 'Reply Sent',
        message: 'Your reply has been posted',
        type: 'success',
      });
    } catch (e) {
      console.error('Failed to post comment:', e);
      addToast({
        title: 'Error',
        message: 'Failed to post reply',
        type: 'error',
      });
      throw e;
    }
  };

  const handleDeletePost = async (postId: string) => {
    if (!hasWailsRuntime() || !identity) return;
    await PublishDeletePost(identity.publicKey, postId);
    setSelectedPost(null);
    setPostBody('');
    setPostComments([]);
    setView('feed');
    await loadPosts(currentSubId, sortMode);
    addToast({
      title: 'Post Deleted',
      message: 'Post has been deleted',
      type: 'info',
    });
  };

  const handleDeleteComment = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    await PublishDeleteComment(identity.publicKey, commentId);
    const comments = await GetCommentsByPost(selectedPost.id);
    setPostComments(comments);
    addToast({
      title: 'Comment Deleted',
      message: 'Comment has been deleted',
      type: 'info',
    });
  };

  const handleCommentUpvote = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishCommentUpvote(identity.publicKey, selectedPost.id, commentId);
      await refreshCommentsForSelectedPost(selectedPost.id);
    } catch (e) {
      console.error('Failed to upvote comment:', e);
    }
  };

  const handleCommentDownvote = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishCommentDownvote(identity.publicKey, selectedPost.id, commentId);
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
      addToast({
        title: 'Profile Published',
        message: 'Your profile update has been broadcasted',
        type: 'success',
      });
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

  const handleEditPost = async (postId: string, title: string, body: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishPostUpdate(identity.publicKey, postId, title, body);
      await loadPostDetail({ ...selectedPost!, title } as Post);
      setPosts(prev => prev.map(p => p.id === postId ? { ...p, title, bodyPreview: body.slice(0, 140) } : p));

      addToast({
        title: 'Post Updated',
        message: 'Your changes have been published.',
        type: 'success',
      });
    } catch (e: any) {
      console.error('Failed to update post:', e);
      addToast({
        title: 'Error',
        message: 'Failed to update post: ' + e.message,
        type: 'error',
      });
      throw e;
    }
  };

  useEffect(() => {
    loadIdentity();
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
    if (!identity) {
      setShowLoginModal(true);
    }
  }, [identity]);

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
        // Mark as unread if not currently viewing it
        if (payload.subId !== currentSubId) {
          setUnreadSubs((prev) => new Set(prev).add(payload.subId));
        }

        // Don't show toast for own posts to avoid redundancy
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
      // Just reload favorited ID list
      void loadFavorites();
    });
    return () => {
      unsubscribe();
    };
  }, [loadFavorites]);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
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
      if (!identity || view !== 'feed') return;
      void loadPosts(currentSubId, sortMode);
    });
    return () => {
      unsubscribe();
    };
  }, [identity, view, currentSubId, sortMode, loadPosts]);

  useEffect(() => {
    if (!hasWailsRuntime()) return;
    if (!identity) {
      setOnlineCount(0);
      return;
    }

    let alive = true;
    const refresh = async () => {
      try {
        const status = await GetP2PStatus();
        if (!alive) return;
        if (!status?.started) {
          setOnlineCount(0);
          return;
        }
        const peers = Array.isArray(status.connectedPeers) ? status.connectedPeers.length : 0;
        setOnlineCount(peers + 1);
      } catch {
        if (alive) setOnlineCount(0);
      }
    };

    void refresh();
    const timer = window.setInterval(() => {
      void refresh();
    }, 15000);

    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [identity]);

  const currentSub = currentSubId === 'recommended'
    ? { id: 'recommended', title: 'Recommended Feed', description: 'Your personalized feed based on subscriptions and trending posts' }
    : (subs.find((s) => s.id === currentSubId) || { id: currentSubId, title: currentSubId, description: '' });

  const membersCount = new Set(posts.map((post) => post.pubkey).filter((value) => !!value)).size;

  const isCurrentSubSubscribed = currentSubId !== 'recommended' && subscribedSubIds.has(currentSubId);

  return (
    <div className={`h-screen flex flex-col ${isDark ? 'dark' : ''}`}>
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
            onMyPostsClick={() => setView('my-posts')}
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
            }}
          />

          {view === 'feed' && (
            <Feed
              posts={posts.map(p => ({ ...p, isFavorited: favoritePostIds.has(p.id) }))}
              sortMode={sortMode}
              profiles={profiles}
              onSortChange={setSortMode}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
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

          {view === 'post-detail' && selectedPost && (
            <PostDetail
              post={{ ...selectedPost, isFavorited: favoritePostIds.has(selectedPost.id) }}
              body={postBody}
              comments={postComments}
              profiles={profiles}
              currentPubkey={identity?.publicKey}
              onBack={handleBackToFeed}
              onUpvote={handleUpvote}
              onDownvote={handleDownvote}
              onReply={handleCommentReply}
              onCommentUpvote={handleCommentUpvote}
              onCommentDownvote={handleCommentDownvote}
              onDeletePost={handleDeletePost}
              onDeleteComment={handleDeleteComment}
              onViewOperationTimeline={handleViewOperationTimeline}
              isDevMode={isDevMode}
              onToggleFavorite={handleToggleFavorite}
              onEditPost={handleEditPost}
            />
          )}

          {view === 'my-posts' && identity && (
            <MyPosts
              currentPubkey={identity.publicKey}
              profiles={profiles}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
            />
          )}

          {view === 'favorites' && (
            <Favorites
              allPosts={posts}
              profiles={profiles}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
              onToggleFavorite={handleToggleFavorite}
            />
          )}
        </div>

        {view === 'feed' && (
          <RightPanel
            sub={currentSub}
            isSubscribed={isCurrentSubSubscribed}
            membersCount={membersCount}
            onlineCount={onlineCount}
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
        onClose={() => setShowCreatePostModal(false)}
        onCreate={handleCreatePost}
      />

      <ToastContainer toasts={toasts} onClose={removeToast} />
    </div>
  );
}

export default App;
