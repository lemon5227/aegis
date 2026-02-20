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
  UpdateProfileDetails,
  PublishProfileUpdate,
  PublishShadowBan,
  PublishUnban,
  GetModerationLogs,
  TriggerCommentSyncNow,
  GetP2PStatus,
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
import { Sub, Profile, Post, GovernanceAdmin, Identity, Comment, ModerationLog, ModerationState } from './types';
import { EventsOn } from '../wailsjs/runtime/runtime';

type SortMode = 'hot' | 'new';
type ViewMode = 'feed' | 'discover' | 'post-detail' | 'my-posts' | 'favorites';

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
  const [posts, setPosts] = useState<Post[]>([]);
  const [profiles, setProfiles] = useState<Record<string, Profile>>({});
  const [isDark, setIsDark] = useState(false);
  const [showLoginModal, setShowLoginModal] = useState(false);
  const [showCreateSubModal, setShowCreateSubModal] = useState(false);
  const [showCreatePostModal, setShowCreatePostModal] = useState(false);
  const [showSettingsPanel, setShowSettingsPanel] = useState(false);
  const [loading, setLoading] = useState(false);
  const [searchResults, setSearchResults] = useState<{ subs: Sub[]; posts: any[] } | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  
  const [selectedPost, setSelectedPost] = useState<Post | null>(null);
  const [postBody, setPostBody] = useState<string>('');
  const [postComments, setPostComments] = useState<Comment[]>([]);
  const [governanceAdmins, setGovernanceAdmins] = useState<GovernanceAdmin[]>([]);
  const [moderationStates, setModerationStates] = useState<ModerationState[]>([]);
  const [moderationLogs, setModerationLogs] = useState<ModerationLog[]>([]);
  const [onlineCount, setOnlineCount] = useState(0);

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
      }
    } catch (e) {
      console.log('No saved identity');
    }
  }, [loadGovernanceData]);

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
    }
    await loadSubs();
    await loadSubscribedSubs();
  }, [loadGovernanceData, loadSubs, loadSubscribedSubs]);

  const loadRecommendedFeed = useCallback(async () => {
    if (!hasWailsRuntime()) return;
    try {
      const stream = await GetFeedStream(50);
      const mapped: Post[] = stream.items.map((item: any) => ({
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
      const mapped: Post[] = (feed as any[]).map((item) => ({
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
    } catch (e) {
      console.error('Failed to create sub:', e);
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
    } catch (e) {
      console.error('Failed to create post:', e);
      throw e;
    }
  };

  const handleUpvote = async (postId: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishPostUpvote(identity.publicKey, postId);
      setPosts((prev) =>
        prev.map((p) => (p.id === postId ? { ...p, score: (p.score || 0) + 1 } : p))
      );
    } catch (e) {
      console.error('Failed to upvote:', e);
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
      } else {
        await SubscribeSub(subId);
        const sub = subs.find((s) => s.id === subId);
        if (sub) {
          setSubscribedSubs((prev) => [...prev, sub]);
          setSubscribedSubIds((prev) => new Set(prev).add(subId));
        }
      }
    } catch (e) {
      console.error('Failed to toggle subscription:', e);
    }
  };

  const handleSearch = async (query: string) => {
    setSearchQuery(query);
    if (!query.trim()) {
      setSearchResults(null);
      return;
    }
    if (!hasWailsRuntime()) return;
    try {
      const [subResults, postResults] = await Promise.all([
        SearchSubs(query, 10),
        SearchPosts(query, '', 10),
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
      const comments = await GetCommentsByPost(selectedPost.id);
      setPostComments(comments);
    } catch (e) {
      console.error('Failed to post comment:', e);
      throw e;
    }
  };

  const handleCommentUpvote = async (commentId: string) => {
    if (!hasWailsRuntime() || !identity || !selectedPost) return;
    try {
      await PublishCommentUpvote(identity.publicKey, selectedPost.id, commentId);
      setPostComments((prev) =>
        prev.map((c) => (c.id === commentId ? { ...c, score: (c.score || 0) + 1 } : c))
      );
    } catch (e) {
      console.error('Failed to upvote comment:', e);
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
    } catch (e) {
      console.error('Failed to save profile:', e);
      throw e;
    }
  };

  const handlePublishProfile = async (displayName: string, avatarURL: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishProfileUpdate(identity.publicKey, displayName, avatarURL);
    } catch (e) {
      console.error('Failed to publish profile:', e);
      throw e;
    }
  };

  const handleBanUser = async (targetPubkey: string, reason: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishShadowBan(targetPubkey, identity.publicKey, reason);
      await loadGovernanceData(identity.publicKey);
    } catch (e) {
      console.error('Failed to ban user:', e);
    }
  };

  const handleUnbanUser = async (targetPubkey: string, reason: string) => {
    if (!hasWailsRuntime() || !identity) return;
    try {
      await PublishUnban(targetPubkey, identity.publicKey, reason);
      await loadGovernanceData(identity.publicKey);
    } catch (e) {
      console.error('Failed to unban user:', e);
    }
  };

  const handleSignOut = () => {
    setIdentity(null);
    setProfile(null);
    setShowLoginModal(true);
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
    const unsubscribe = EventsOn('feed:updated', () => {
      if (!identity || view !== 'feed') return;
      void loadPosts(currentSubId, sortMode);
    });
    return () => {
      unsubscribe();
    };
  }, [identity, view, currentSubId, sortMode, loadPosts]);

  useEffect(() => {
    if (!hasWailsRuntime() || !identity) {
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
              posts={posts}
              sortMode={sortMode}
              profiles={profiles}
              onSortChange={setSortMode}
              onUpvote={handleUpvote}
              onPostClick={handlePostClick}
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
              post={selectedPost}
              body={postBody}
              comments={postComments}
              profiles={profiles}
              currentPubkey={identity?.publicKey}
              onBack={handleBackToFeed}
              onUpvote={handleUpvote}
              onReply={handleCommentReply}
              onCommentUpvote={handleCommentUpvote}
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
    </div>
  );
}

export default App;
