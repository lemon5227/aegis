import { useEffect, useRef, useState } from 'react';
import './App.css';
import {
    AddLocalPostWithImageToSub,
    AddLocalPostStructuredToSub,
    AddTrustedAdmin,
    ConnectPeer,
    CreateSub,
    GenerateIdentity,
    GetCommentsByPost,
    GetFeedIndexBySubSorted,
    GetGovernancePolicy,
    GetModerationState,
    GetModerationLogs,
    GetMediaByCID,
    GetP2PStatus,
    GetPostBodyByID,
    GetPostMediaByID,
    GetProfile,
    GetPrivateFeed,
    GetStorageUsage,
    GetSubs,
    GetTrustedAdmins,
    PublishPostStructuredToSub,
    PublishPostWithImageToSub,
    PublishPostUpvote,
    PublishGovernancePolicy,
    PublishCreateSub,
    PublishComment,
    PublishCommentUpvote,
    PublishProfileUpdate,
    PublishShadowBan,
    PublishUnban,
    ImportIdentityFromMnemonic,
    LoadSavedIdentity,
    StartP2P,
    StopP2P,
    UpdateProfile,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

type ForumMessage = {
    id: string;
    pubkey: string;
    title: string;
    body: string;
    contentCid: string;
    imageCid: string;
    thumbCid: string;
    imageMime: string;
    imageSize: number;
    imageWidth: number;
    imageHeight: number;
    score: number;
    timestamp: number;
    sizeBytes: number;
    zone: 'private' | 'public';
    subId: string;
    visibility: string;
};

type Comment = {
    id: string;
    postId: string;
    parentId: string;
    pubkey: string;
    body: string;
    score: number;
    timestamp: number;
};

type ModerationState = {
    targetPubkey: string;
    action: string;
    sourceAdmin: string;
    timestamp: number;
    reason: string;
};

type ModerationLog = {
    id: number;
    targetPubkey: string;
    action: string;
    sourceAdmin: string;
    timestamp: number;
    reason: string;
    result: string;
};

type GovernancePolicy = {
    hideHistoryOnShadowBan: boolean;
};

type StorageUsage = {
    privateUsedBytes: number;
    publicUsedBytes: number;
    privateQuota: number;
    publicQuota: number;
    totalQuota: number;
};

type P2PStatus = {
    started: boolean;
    peerId: string;
    listenAddrs: string[];
    connectedPeers: string[];
    topic: string;
};

type GovernanceAdmin = {
    adminPubkey: string;
    role: string;
    active: boolean;
};

type Sub = {
    id: string;
    title: string;
    description: string;
    createdAt: number;
};

type Profile = {
    pubkey: string;
    displayName: string;
    avatarURL: string;
    updatedAt: number;
};

type PostBodyBlob = {
    contentCid: string;
    body: string;
    sizeBytes: number;
};

type MediaBlob = {
    contentCid: string;
    dataBase64: string;
    mime: string;
    sizeBytes: number;
    width: number;
    height: number;
    isThumbnail: boolean;
};

const bytesToMB = (bytes: number) => `${(bytes / (1024 * 1024)).toFixed(2)} MB`;

function App() {
    const [mnemonic, setMnemonic] = useState('');
    const [publicKey, setPublicKey] = useState('');
    const [postTitle, setPostTitle] = useState('Hello Aegis');
    const [postBody, setPostBody] = useState('Hello Aegis from local node');
    const [postImageBase64, setPostImageBase64] = useState('');
    const [postImageMime, setPostImageMime] = useState('');
    const [postImagePreviewURL, setPostImagePreviewURL] = useState('');
    const [postZone, setPostZone] = useState<'private' | 'public'>('public');
    const [moderationTarget, setModerationTarget] = useState('');
    const [moderationReason, setModerationReason] = useState('manual-test');
    const [p2pPort, setP2pPort] = useState('40100');
    const [peerAddress, setPeerAddress] = useState('');
    const [trustedAdminInput, setTrustedAdminInput] = useState('');
    const [mnemonicInput, setMnemonicInput] = useState('');
    const [p2pStatus, setP2pStatus] = useState<P2PStatus | null>(null);
    const [trustedAdmins, setTrustedAdmins] = useState<GovernanceAdmin[]>([]);
    const [subs, setSubs] = useState<Sub[]>([]);
    const [currentSubId, setCurrentSubId] = useState('general');
    const [sortMode, setSortMode] = useState<'hot' | 'new'>('hot');
    const [newSubId, setNewSubId] = useState('');
    const [newSubTitle, setNewSubTitle] = useState('');
    const [newSubDescription, setNewSubDescription] = useState('');
    const [profileDisplayName, setProfileDisplayName] = useState('');
    const [profileAvatarURL, setProfileAvatarURL] = useState('');

    const [feed, setFeed] = useState<ForumMessage[]>([]);
    const [privateFeed, setPrivateFeed] = useState<ForumMessage[]>([]);
    const [moderation, setModeration] = useState<ModerationState[]>([]);
    const [moderationLogs, setModerationLogs] = useState<ModerationLog[]>([]);
    const [governancePolicy, setGovernancePolicy] = useState<GovernancePolicy>({ hideHistoryOnShadowBan: true });
    const [storage, setStorage] = useState<StorageUsage | null>(null);

    const [error, setError] = useState('');
    const [governanceStatus, setGovernanceStatus] = useState('');
    const [loading, setLoading] = useState(false);
    const [selectedPublicPost, setSelectedPublicPost] = useState<ForumMessage | null>(null);
    const [postComments, setPostComments] = useState<Comment[]>([]);
    const [commentBody, setCommentBody] = useState('');
    const [replyToCommentId, setReplyToCommentId] = useState('');
    const [profilesByPubkey, setProfilesByPubkey] = useState<Record<string, Profile>>({});
    const [postBodyCache, setPostBodyCache] = useState<Record<string, string>>({});
    const [selectedBodyLoading, setSelectedBodyLoading] = useState(false);
    const [selectedBodyHint, setSelectedBodyHint] = useState('');
    const [thumbnailURLByCID, setThumbnailURLByCID] = useState<Record<string, string>>({});
    const [selectedImageURL, setSelectedImageURL] = useState('');
    const [selectedImageLoading, setSelectedImageLoading] = useState(false);
    const postImageInputRef = useRef<HTMLInputElement | null>(null);
    const selectedPostIdRef = useRef('');
    const selectedBodyRetryTimerRef = useRef<number | null>(null);
    const selectedBodyRetryCountRef = useRef(0);
    const identityReady = publicKey.trim().length > 0;
    const currentAdmin = trustedAdmins.find((admin) => admin.adminPubkey === publicKey);
    const isGovernanceAdmin = !!currentAdmin;
    const currentRoleLabel = currentAdmin ? (currentAdmin.role === 'genesis' ? 'Genesis Admin' : 'Trusted Admin') : 'Normal Node';

    const hasWailsRuntime = () => {
        const wailsBridge = (window as any)?.go?.main?.App;
        return !!wailsBridge;
    };

    const ensureRuntime = () => {
        if (!hasWailsRuntime()) {
            throw new Error('未检测到 Wails Runtime。请在项目根目录运行: wails dev');
        }
    };

    const toPreview = (text: string, max = 80) => {
        const normalized = (text || '').trim();
        if (normalized.length <= max) {
            return normalized;
        }
        return `${normalized.slice(0, max)}...`;
    };

    const shortPubkey = (pubkey: string) => {
        if (!pubkey) {
            return 'unknown';
        }
        return `${pubkey.slice(0, 10)}...`;
    };

    const getAuthorLabel = (pubkey: string) => {
        const profile = profilesByPubkey[pubkey];
        if (profile?.displayName?.trim()) {
            return profile.displayName.trim();
        }
        return shortPubkey(pubkey);
    };

    const getAuthorAvatar = (pubkey: string) => {
        const profile = profilesByPubkey[pubkey];
        return (profile?.avatarURL || '').trim();
    };

    const extractPostIDFromCommentsEvent = (payload: any) => {
        if (!payload) {
            return '';
        }
        if (typeof payload === 'string') {
            return payload.trim();
        }
        if (typeof payload === 'object' && typeof payload.postId === 'string') {
            return payload.postId.trim();
        }
        return '';
    };

    const getEffectiveListenPort = (status: P2PStatus | null) => {
        if (!status?.listenAddrs?.length) {
            return '';
        }

        for (const address of status.listenAddrs) {
            const match = address.match(/\/tcp\/(\d+)/);
            if (match?.[1]) {
                return match[1];
            }
        }

        return '';
    };

    const classifyListenAddress = (address: string) => {
        const normalized = (address || '').trim().toLowerCase();
        const ip4Match = normalized.match(/\/ip4\/([^/]+)/);
        if (ip4Match?.[1]) {
            const ip4 = ip4Match[1];
            if (ip4 === '127.0.0.1') {
                return 5;
            }
            if (
                ip4.startsWith('10.') ||
                ip4.startsWith('192.168.') ||
                ip4.startsWith('169.254.') ||
                /^172\.(1[6-9]|2\d|3[0-1])\./.test(ip4)
            ) {
                return 2;
            }
            return 1;
        }

        const ip6Match = normalized.match(/\/ip6\/([^/]+)/);
        if (ip6Match?.[1]) {
            const ip6 = ip6Match[1];
            if (ip6 === '::1') {
                return 5;
            }
            if (ip6.startsWith('fe80:')) {
                return 4;
            }
            return 0;
        }

        return 6;
    };

    const prioritizeListenAddresses = (addresses: string[]) => {
        const rows = [...(addresses || [])];
        rows.sort((left, right) => {
            const leftRank = classifyListenAddress(left);
            const rightRank = classifyListenAddress(right);
            if (leftRank !== rightRank) {
                return leftRank - rightRank;
            }
            return left.localeCompare(right);
        });
        return rows;
    };

    const prioritizedListenAddrs = prioritizeListenAddresses(p2pStatus?.listenAddrs || []);
    const recommendedListenAddr = prioritizedListenAddrs.find((address) => classifyListenAddress(address) <= 2) || '';
    const hasGlobalIPv6Address = prioritizedListenAddrs.some((address) => classifyListenAddress(address) === 0);

    const clearSelectedBodyRetryTimer = () => {
        if (selectedBodyRetryTimerRef.current !== null) {
            window.clearTimeout(selectedBodyRetryTimerRef.current);
            selectedBodyRetryTimerRef.current = null;
        }
    };

    const scheduleSelectedBodyRetry = (post: ForumMessage) => {
        if (selectedBodyRetryCountRef.current >= 3) {
            return;
        }

        clearSelectedBodyRetryTimer();
        selectedBodyRetryTimerRef.current = window.setTimeout(() => {
            if (selectedPostIdRef.current !== post.id) {
                return;
            }
            void hydrateSelectedPostBody(post, true);
        }, 2000);
    };

    async function loadProfiles(pubkeys: string[]) {
        const unique = Array.from(new Set(pubkeys.map((item) => item.trim()).filter((item) => item.length > 0)));
        if (unique.length === 0) {
            return;
        }

        const profileRows = await Promise.all(unique.map((pubkey) => GetProfile(pubkey)));
        setProfilesByPubkey((previous) => {
            const next = { ...previous };
            for (const row of profileRows as Profile[]) {
                if (row?.pubkey) {
                    next[row.pubkey] = row;
                }
            }
            return next;
        });
    }

    async function loadCurrentProfile(pubkey: string) {
        const trimmed = pubkey.trim();
        if (!trimmed) {
            setProfileDisplayName('');
            setProfileAvatarURL('');
            return;
        }

        const profile = await GetProfile(trimmed) as Profile;
        setProfileDisplayName(profile.displayName || '');
        setProfileAvatarURL(profile.avatarURL || '');
        setProfilesByPubkey((previous) => ({ ...previous, [trimmed]: profile }));
    }

    async function loadPublicFeedBySub(subId: string) {
        const indexRows = await GetFeedIndexBySubSorted(subId, sortMode) as any[];
        const mapped = indexRows.map((item) => ({
            id: item.id,
            pubkey: item.pubkey,
            title: item.title,
            body: item.bodyPreview || '',
            contentCid: item.contentCid || '',
            imageCid: item.imageCid || '',
            thumbCid: item.thumbCid || '',
            imageMime: item.imageMime || '',
            imageSize: item.imageSize || 0,
            imageWidth: item.imageWidth || 0,
            imageHeight: item.imageHeight || 0,
            score: item.score || 0,
            timestamp: item.timestamp || 0,
            sizeBytes: 0,
            zone: (item.zone || 'public') as 'private' | 'public',
            subId: item.subId || 'general',
            visibility: item.visibility || 'normal',
        })) as ForumMessage[];
        setFeed(mapped);
        await preloadFeedThumbnails(mapped);
    }

    async function preloadFeedThumbnails(rows: ForumMessage[]) {
        const targets = rows
            .map((item) => item.thumbCid || item.imageCid)
            .map((cid) => (cid || '').trim())
            .filter((cid) => cid.length > 0)
            .slice(0, 20);

        const missing = targets.filter((cid) => !thumbnailURLByCID[cid]);
        if (missing.length === 0) {
            return;
        }

        const fetched = await Promise.all(missing.map(async (cid) => {
            try {
                const media = await GetMediaByCID(cid) as MediaBlob;
                if (!media?.dataBase64) {
                    return { cid, url: '' };
                }
                const mime = media.mime || 'image/jpeg';
                return { cid, url: `data:${mime};base64,${media.dataBase64}` };
            } catch {
                return { cid, url: '' };
            }
        }));

        setThumbnailURLByCID((previous) => {
            const next = { ...previous };
            for (const item of fetched) {
                if (item.url) {
                    next[item.cid] = item.url;
                }
            }
            return next;
        });
    }

    async function hydrateSelectedPostBody(post: ForumMessage, backgroundRetry = false) {
        const cached = postBodyCache[post.id];
        if (cached) {
            clearSelectedBodyRetryTimer();
            selectedBodyRetryCountRef.current = 0;
            setSelectedBodyHint('');
            setSelectedPublicPost({ ...post, body: cached });
            return;
        }

        if (!backgroundRetry) {
            clearSelectedBodyRetryTimer();
            selectedBodyRetryCountRef.current = 0;
            setSelectedBodyHint('正在请求完整正文...');
            setSelectedBodyLoading(true);
        }

        try {
            const blob = await GetPostBodyByID(post.id) as PostBodyBlob;
            const fullBody = (blob?.body || '').trim();
            if (!fullBody) {
                setSelectedPublicPost(post);
                setSelectedBodyHint('正文正在同步中，会自动刷新。');
                selectedBodyRetryCountRef.current += 1;
                scheduleSelectedBodyRetry(post);
                return;
            }

            clearSelectedBodyRetryTimer();
            selectedBodyRetryCountRef.current = 0;
            setSelectedBodyHint('');
            setPostBodyCache((previous) => ({ ...previous, [post.id]: fullBody }));
            setSelectedPublicPost({ ...post, body: fullBody, contentCid: blob.contentCid || post.contentCid });
        } catch {
            setSelectedPublicPost(post);
            setSelectedBodyHint('正文正在同步中，会自动重试。');
            selectedBodyRetryCountRef.current += 1;
            scheduleSelectedBodyRetry(post);
        } finally {
            if (!backgroundRetry) {
                setSelectedBodyLoading(false);
            }
        }
    }

    async function hydrateSelectedPostImage(post: ForumMessage) {
        const mediaCID = (post.imageCid || '').trim();
        if (!mediaCID) {
            setSelectedImageURL('');
            return;
        }

        setSelectedImageLoading(true);
        try {
            const media = await GetPostMediaByID(post.id) as MediaBlob;
            if (!media?.dataBase64) {
                setSelectedImageURL('');
                return;
            }

            const mime = media.mime || 'image/jpeg';
            const url = `data:${mime};base64,${media.dataBase64}`;
            setSelectedImageURL(url);
            setThumbnailURLByCID((previous) => ({
                ...previous,
                [mediaCID]: previous[mediaCID] || url,
            }));
        } catch {
            setSelectedImageURL('');
        } finally {
            setSelectedImageLoading(false);
        }
    }

    async function loadCommentsForPost(postId: string) {
        const trimmed = postId.trim();
        if (!trimmed) {
            setPostComments([]);
            return;
        }

        const comments = await GetCommentsByPost(trimmed);
        const rows = comments as Comment[];
        setPostComments(rows);
        await loadProfiles(rows.map((item) => item.pubkey));
    }

    async function switchSortMode(mode: 'hot' | 'new') {
        setError('');
        try {
            ensureRuntime();
            setSortMode(mode);
            const indexRows = await GetFeedIndexBySubSorted(currentSubId, mode) as any[];
            const mapped = indexRows.map((item) => ({
                id: item.id,
                pubkey: item.pubkey,
                title: item.title,
                body: item.bodyPreview || '',
                contentCid: item.contentCid || '',
                imageCid: item.imageCid || '',
                thumbCid: item.thumbCid || '',
                imageMime: item.imageMime || '',
                imageSize: item.imageSize || 0,
                imageWidth: item.imageWidth || 0,
                imageHeight: item.imageHeight || 0,
                score: item.score || 0,
                timestamp: item.timestamp || 0,
                sizeBytes: 0,
                zone: (item.zone || 'public') as 'private' | 'public',
                subId: item.subId || 'general',
                visibility: item.visibility || 'normal',
            })) as ForumMessage[];
            setFeed(mapped);
            await preloadFeedThumbnails(mapped);
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function createIdentity() {
        setLoading(true);
        setError('');

        try {
            ensureRuntime();

            const identity = await GenerateIdentity();
            setMnemonic(identity.mnemonic);
            setPublicKey(identity.publicKey);
            await loadCurrentProfile(identity.publicKey);

            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        } finally {
            setLoading(false);
        }
    }

    async function loadSavedIdentity(showNotFound = false) {
        try {
            ensureRuntime();
            const identity = await LoadSavedIdentity();
            setMnemonic(identity.mnemonic);
            setPublicKey(identity.publicKey);
            await loadCurrentProfile(identity.publicKey);
            await refreshDashboard();
        } catch (exception) {
            const message = String(exception);
            if (message.toLowerCase().includes('identity not found')) {
                if (showNotFound) {
                    setError('未发现已保存身份，请先 Create Identity 或导入助记词');
                }
                return;
            }
            setError(message);
        }
    }

    async function importIdentity() {
        setError('');
        try {
            ensureRuntime();
            if (!mnemonicInput.trim()) {
                throw new Error('请输入助记词');
            }

            const identity = await ImportIdentityFromMnemonic(mnemonicInput.trim());
            setMnemonic(identity.mnemonic);
            setPublicKey(identity.publicKey);
            await loadCurrentProfile(identity.publicKey);
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function refreshDashboard() {
        setError('');
        try {
            ensureRuntime();

            const [privateMessages, moderationRows, moderationLogRows, policy, usage, admins, subRows] = await Promise.all([
                GetPrivateFeed(),
                GetModerationState(),
                GetModerationLogs(50),
                GetGovernancePolicy(),
                GetStorageUsage(),
                GetTrustedAdmins(),
                GetSubs(),
            ]);

            setPrivateFeed(privateMessages as ForumMessage[]);
            setModeration(moderationRows as ModerationState[]);
            setModerationLogs(moderationLogRows as ModerationLog[]);
            setGovernancePolicy(policy as GovernancePolicy);
            setStorage(usage as StorageUsage);
            setTrustedAdmins(admins as GovernanceAdmin[]);

            const fetchedSubs = subRows as Sub[];
            setSubs(fetchedSubs);

            let effectiveSubId = currentSubId;
            const found = fetchedSubs.some((sub) => sub.id === currentSubId);
            if (!found && fetchedSubs.length > 0) {
                const fallback = fetchedSubs.find((sub) => sub.id === 'general')?.id || fetchedSubs[0].id;
                effectiveSubId = fallback;
                setCurrentSubId(fallback);
            }

            const indexRows = await GetFeedIndexBySubSorted(effectiveSubId, sortMode) as any[];
            const publicRows = indexRows.map((item) => ({
                id: item.id,
                pubkey: item.pubkey,
                title: item.title,
                body: item.bodyPreview || '',
                contentCid: item.contentCid || '',
                imageCid: item.imageCid || '',
                thumbCid: item.thumbCid || '',
                imageMime: item.imageMime || '',
                imageSize: item.imageSize || 0,
                imageWidth: item.imageWidth || 0,
                imageHeight: item.imageHeight || 0,
                score: item.score || 0,
                timestamp: item.timestamp || 0,
                sizeBytes: 0,
                zone: (item.zone || 'public') as 'private' | 'public',
                subId: item.subId || 'general',
                visibility: item.visibility || 'normal',
            })) as ForumMessage[];
            setFeed(publicRows);
            await preloadFeedThumbnails(publicRows);

            const relatedPubkeys = [...publicRows.map((item) => item.pubkey), ...((privateMessages as ForumMessage[]).map((item) => item.pubkey))];
            if (publicKey.trim()) {
                relatedPubkeys.push(publicKey.trim());
            }
            await loadProfiles(relatedPubkeys);

            const status = await GetP2PStatus();
            setP2pStatus(status as P2PStatus);

            if (selectedPostIdRef.current) {
                await loadCommentsForPost(selectedPostIdRef.current);
                const selected = publicRows.find((item) => item.id === selectedPostIdRef.current);
                if (selected) {
                    await hydrateSelectedPostBody(selected);
                    await hydrateSelectedPostImage(selected);
                }
            }
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function switchSub(subId: string) {
        setError('');
        try {
            ensureRuntime();
            setCurrentSubId(subId);
            await loadPublicFeedBySub(subId);
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function createSub() {
        setError('');
        try {
            ensureRuntime();
            if (!newSubId.trim()) {
                throw new Error('请输入 Sub ID');
            }

            const created = await CreateSub(newSubId.trim(), newSubTitle.trim(), newSubDescription.trim());
            const sub = created as Sub;
            await PublishCreateSub(sub.id, newSubTitle.trim(), newSubDescription.trim());
            setNewSubId('');
            setNewSubTitle('');
            setNewSubDescription('');
            await refreshDashboard();
            await switchSub(sub.id);
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function addLocalPost() {
        setError('');
        try {
            ensureRuntime();

            if (!publicKey) {
                throw new Error('请先创建身份后再写入帖子');
            }

            if (postImageBase64.trim()) {
                await AddLocalPostWithImageToSub(publicKey, postTitle.trim(), postBody.trim(), postZone, currentSubId, postImageBase64.trim(), postImageMime.trim() || 'image/jpeg');
            } else {
                await AddLocalPostStructuredToSub(publicKey, postTitle.trim(), postBody.trim(), postZone, currentSubId);
            }

            setPostImageBase64('');
            setPostImageMime('');
            setPostImagePreviewURL('');
            if (postImageInputRef.current) {
                postImageInputRef.current.value = '';
            }
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function applyShadowBan() {
        setError('');
        setGovernanceStatus('');
        try {
            ensureRuntime();

            if (!publicKey) {
                throw new Error('请先创建身份作为管理节点公钥');
            }
            if (!moderationTarget.trim()) {
                throw new Error('请输入目标公钥');
            }

            await PublishShadowBan(moderationTarget.trim(), publicKey, moderationReason.trim());
            setGovernanceStatus('治理消息已发送：SHADOW_BAN');
            await refreshDashboard();
        } catch (exception) {
            const message = String(exception);
            setError(message);
            setGovernanceStatus(`治理消息失败：${message}`);
        }
    }

    async function applyUnban() {
        setError('');
        setGovernanceStatus('');
        try {
            ensureRuntime();

            if (!publicKey) {
                throw new Error('请先创建身份作为管理节点公钥');
            }
            if (!moderationTarget.trim()) {
                throw new Error('请输入目标公钥');
            }

            await PublishUnban(moderationTarget.trim(), publicKey, moderationReason.trim());
            setGovernanceStatus('治理消息已发送：UNBAN');
            await refreshDashboard();
        } catch (exception) {
            const message = String(exception);
            setError(message);
            setGovernanceStatus(`治理消息失败：${message}`);
        }
    }

    async function toggleHideHistoryOnShadowBan() {
        setError('');
        setGovernanceStatus('');
        try {
            ensureRuntime();
            if (!isGovernanceAdmin) {
                throw new Error('仅治理管理员可修改策略');
            }
            const nextValue = !governancePolicy.hideHistoryOnShadowBan;
            await PublishGovernancePolicy(nextValue);
            setGovernancePolicy({ hideHistoryOnShadowBan: nextValue });
            setGovernanceStatus(`治理策略已广播：封禁${nextValue ? '会隐藏' : '不会隐藏'}历史帖`);
            await refreshDashboard();
        } catch (exception) {
            const message = String(exception);
            setError(message);
            setGovernanceStatus(`治理策略更新失败：${message}`);
        }
    }

    async function trustCurrentIdentity() {
        setError('');
        try {
            ensureRuntime();

            if (!publicKey) {
                throw new Error('请先创建身份再加入管理员列表');
            }

            await AddTrustedAdmin(publicKey, 'appointed');
            setTrustedAdminInput(publicKey);
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function addTrustedAdminByInput() {
        setError('');
        try {
            ensureRuntime();

            if (!trustedAdminInput.trim()) {
                throw new Error('请输入管理员公钥');
            }

            await AddTrustedAdmin(trustedAdminInput.trim(), 'appointed');
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function startP2P() {
        setError('');
        try {
            ensureRuntime();

            const port = Number.parseInt(p2pPort, 10);
            const bootstrapPeers = peerAddress.trim() ? [peerAddress.trim()] : [];
            const status = await StartP2P(port, bootstrapPeers);
            setP2pStatus(status as P2PStatus);
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function stopP2P() {
        setError('');
        try {
            ensureRuntime();

            await StopP2P();
            const status = await GetP2PStatus();
            setP2pStatus(status as P2PStatus);
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function connectPeerAddress() {
        setError('');
        try {
            ensureRuntime();

            if (!peerAddress.trim()) {
                throw new Error('请输入对端 multiaddr');
            }

            await ConnectPeer(peerAddress.trim());
            const status = await GetP2PStatus();
            setP2pStatus(status as P2PStatus);
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function publishNetworkPost() {
        setError('');
        try {
            ensureRuntime();

            if (!publicKey) {
                throw new Error('请先创建身份后再广播帖子');
            }

            if (postImageBase64.trim()) {
                await PublishPostWithImageToSub(publicKey, postTitle.trim(), postBody.trim(), postImageBase64.trim(), postImageMime.trim() || 'image/jpeg', currentSubId);
            } else {
                await PublishPostStructuredToSub(publicKey, postTitle.trim(), postBody.trim(), currentSubId);
            }

            setPostImageBase64('');
            setPostImageMime('');
            setPostImagePreviewURL('');
            if (postImageInputRef.current) {
                postImageInputRef.current.value = '';
            }
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function handleSelectPostImage(event: React.ChangeEvent<HTMLInputElement>) {
        setError('');
        try {
            const file = event.target.files?.[0];
            if (!file) {
                setPostImageBase64('');
                setPostImageMime('');
                setPostImagePreviewURL('');
                return;
            }

            const mime = file.type || 'image/jpeg';
            const dataURL = await new Promise<string>((resolve, reject) => {
                const reader = new FileReader();
                reader.onerror = () => reject(new Error('读取图片失败'));
                reader.onload = () => resolve(String(reader.result || ''));
                reader.readAsDataURL(file);
            });

            const marker = ';base64,';
            const index = dataURL.indexOf(marker);
            if (index <= 0) {
                throw new Error('图片编码失败');
            }

            setPostImageMime(mime);
            setPostImageBase64(dataURL.slice(index + marker.length));
            setPostImagePreviewURL(dataURL);
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function saveProfile() {
        setError('');
        try {
            ensureRuntime();

            if (!publicKey.trim()) {
                throw new Error('请先创建或加载身份后再更新资料');
            }

            const profile = await UpdateProfile(profileDisplayName.trim(), profileAvatarURL.trim()) as Profile;
            await PublishProfileUpdate(publicKey.trim(), profile.displayName || '', profile.avatarURL || '');
            setProfilesByPubkey((previous) => ({ ...previous, [publicKey.trim()]: profile }));
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function publishCommentForSelectedPost() {
        setError('');
        try {
            ensureRuntime();

            if (!publicKey.trim()) {
                throw new Error('请先创建或加载身份后再评论');
            }
            if (!selectedPublicPost?.id) {
                throw new Error('请先选择一个帖子');
            }

            const body = commentBody.trim();
            if (!body) {
                throw new Error('请输入评论内容');
            }

            const parentId = replyToCommentId.trim();
            await PublishComment(publicKey.trim(), selectedPublicPost.id, parentId, body);

            setCommentBody('');
            setReplyToCommentId('');
            await loadCommentsForPost(selectedPublicPost.id);
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function upvoteSelectedPost() {
        setError('');
        try {
            ensureRuntime();

            if (!publicKey.trim()) {
                throw new Error('请先创建或加载身份后再投票');
            }
            if (!selectedPublicPost?.id) {
                throw new Error('请先选择一个帖子');
            }

            await PublishPostUpvote(publicKey.trim(), selectedPublicPost.id);

            setFeed((previous) => previous.map((item) => (
                item.id === selectedPublicPost.id
                    ? { ...item, score: (item.score || 0) + 1 }
                    : item
            )));
            setSelectedPublicPost((previous) => {
                if (!previous || previous.id !== selectedPublicPost.id) {
                    return previous;
                }
                return { ...previous, score: (previous.score || 0) + 1 };
            });
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function upvoteComment(comment: Comment) {
        setError('');
        try {
            ensureRuntime();

            if (!publicKey.trim()) {
                throw new Error('请先创建或加载身份后再投票');
            }
            if (!selectedPublicPost?.id) {
                throw new Error('请先选择一个帖子');
            }

            await PublishCommentUpvote(publicKey.trim(), selectedPublicPost.id, comment.id);

            setPostComments((previous) => previous.map((row) => (
                row.id === comment.id
                    ? { ...row, score: (row.score || 0) + 1 }
                    : row
            )));
        } catch (exception) {
            setError(String(exception));
        }
    }

    useEffect(() => {
        if (!hasWailsRuntime()) {
            return;
        }

        loadSavedIdentity(false);

        const unsubscribeFeed = EventsOn('feed:updated', () => {
            refreshDashboard();
        });

        const unsubscribeP2P = EventsOn('p2p:updated', () => {
            refreshDashboard();
        });

        const unsubscribeComments = EventsOn('comments:updated', (payload: any) => {
            const postID = extractPostIDFromCommentsEvent(payload);
            if (!postID) {
                return;
            }
            if (selectedPostIdRef.current === postID) {
                loadCommentsForPost(postID);
            }
        });

        return () => {
            unsubscribeFeed();
            unsubscribeP2P();
            unsubscribeComments();
        };
    }, []);

    useEffect(() => {
        selectedPostIdRef.current = selectedPublicPost?.id || '';
    }, [selectedPublicPost?.id]);

    useEffect(() => {
        if (!selectedPublicPost?.id) {
            clearSelectedBodyRetryTimer();
            selectedBodyRetryCountRef.current = 0;
            setSelectedBodyHint('');
            setSelectedImageURL('');
            setSelectedImageLoading(false);
        }
    }, [selectedPublicPost?.id]);

    useEffect(() => {
        if (!hasWailsRuntime()) {
            return;
        }

        if (!selectedPublicPost?.id) {
            setPostComments([]);
            return;
        }

        loadCommentsForPost(selectedPublicPost.id);
    }, [selectedPublicPost?.id]);

    useEffect(() => {
        return () => {
            clearSelectedBodyRetryTimer();
        };
    }, []);

    return (
        <div id="App">
            <h1>Aegis MVP Dashboard</h1>
            <p className="result">Phase 2 可视化：身份、写入、Feed、配额、治理状态</p>

            <div className="toolbar">
                <button className="btn" onClick={createIdentity} disabled={loading}>
                    {loading ? 'Generating...' : 'Create Identity'}
                </button>
                <button className="btn" onClick={() => loadSavedIdentity(true)}>Load Saved Identity</button>
                <button className="btn" onClick={refreshDashboard}>Refresh Dashboard</button>
            </div>

            <div className="panel">
                <h3>Import Identity</h3>
                <textarea
                    className="input"
                    rows={2}
                    value={mnemonicInput}
                    onChange={(event) => setMnemonicInput(event.target.value)}
                    placeholder="Paste mnemonic to login with existing identity"
                />
                <div className="row">
                    <button className="btn" onClick={importIdentity}>Import From Mnemonic</button>
                </div>
            </div>

            {!identityReady && (
                <div className="hint">请先点击 Create Identity，再进行发帖、广播和治理操作。</div>
            )}

            {mnemonic && (
                <div className="panel">
                    <h3>Mnemonic</h3>
                    <p>{mnemonic}</p>
                </div>
            )}

            {publicKey && (
                <div className="panel">
                    <h3>Public Key</h3>
                    <p>{publicKey}</p>
                    <p className="badge">Role: {currentRoleLabel}</p>
                </div>
            )}

            <div className="panel">
                <h3>Profile</h3>
                <input
                    className="input"
                    value={profileDisplayName}
                    onChange={(event) => setProfileDisplayName(event.target.value)}
                    placeholder="Display Name"
                />
                <input
                    className="input"
                    value={profileAvatarURL}
                    onChange={(event) => setProfileAvatarURL(event.target.value)}
                    placeholder="Avatar URL"
                />
                <div className="row">
                    <button className="btn" onClick={saveProfile} disabled={!identityReady}>Save Profile</button>
                </div>
                {profileAvatarURL.trim() ? (
                    <div className="row">
                        <img
                            src={profileAvatarURL.trim()}
                            alt="profile-preview"
                            style={{ width: 40, height: 40, borderRadius: 20 }}
                            onError={(event) => {
                                event.currentTarget.style.display = 'none';
                            }}
                        />
                        <span>Avatar Preview</span>
                    </div>
                ) : null}
            </div>

            <div className="panel">
                <h3>Subs</h3>
                <div className="row">
                    <select value={currentSubId} onChange={(event) => switchSub(event.target.value)}>
                        {subs.map((sub) => (
                            <option key={sub.id} value={sub.id}>{sub.id}</option>
                        ))}
                    </select>
                    <select value={sortMode} onChange={(event) => switchSortMode(event.target.value as 'hot' | 'new')}>
                        <option value="hot">hot</option>
                        <option value="new">new</option>
                    </select>
                    <span className="badge">Current: {currentSubId}</span>
                </div>
                <input
                    className="input"
                    value={newSubId}
                    onChange={(event) => setNewSubId(event.target.value)}
                    placeholder="New Sub ID (e.g. golang)"
                />
                <input
                    className="input"
                    value={newSubTitle}
                    onChange={(event) => setNewSubTitle(event.target.value)}
                    placeholder="Sub title (optional)"
                />
                <input
                    className="input"
                    value={newSubDescription}
                    onChange={(event) => setNewSubDescription(event.target.value)}
                    placeholder="Sub description (optional)"
                />
                <div className="row">
                    <button className="btn" onClick={createSub}>Create / Update Sub</button>
                </div>
            </div>

            <div className="panel">
                <h3>Add Local Post</h3>
                <input
                    className="input"
                    value={postTitle}
                    onChange={(event) => setPostTitle(event.target.value)}
                    placeholder="Post title"
                />
                <textarea
                    className="input"
                    rows={3}
                    value={postBody}
                    onChange={(event) => setPostBody(event.target.value)}
                    placeholder="Input post body"
                />
                <input
                    ref={postImageInputRef}
                    className="input"
                    type="file"
                    accept="image/*"
                    onClick={(event) => {
                        event.currentTarget.value = '';
                    }}
                    onChange={handleSelectPostImage}
                />
                {postImagePreviewURL ? (
                    <div className="row">
                        <img
                            src={postImagePreviewURL}
                            alt="post-image-preview"
                            style={{ maxWidth: 200, maxHeight: 160, borderRadius: 6 }}
                        />
                        <button
                            className="btn"
                            onClick={() => {
                                setPostImageBase64('');
                                setPostImageMime('');
                                setPostImagePreviewURL('');
                                if (postImageInputRef.current) {
                                    postImageInputRef.current.value = '';
                                }
                            }}
                        >
                            Remove Image
                        </button>
                    </div>
                ) : null}
                <div className="row">
                    <select value={postZone} onChange={(event) => setPostZone(event.target.value as 'private' | 'public')}>
                        <option value="public">public</option>
                        <option value="private">private</option>
                    </select>
                    <button className="btn" onClick={addLocalPost} disabled={!identityReady}>Insert Local Post</button>
                    <button className="btn" onClick={publishNetworkPost} disabled={!identityReady}>Publish To Network</button>
                </div>
            </div>

            <div className="panel">
                <h3>P2P Network</h3>
                <div className="row">
                    <input
                        className="input short"
                        value={p2pPort}
                        onChange={(event) => setP2pPort(event.target.value)}
                        placeholder="Listen port (e.g. 40100)"
                    />
                    <button className="btn" onClick={startP2P}>Start P2P</button>
                    <button className="btn" onClick={stopP2P}>Stop P2P</button>
                </div>

                <input
                    className="input"
                    value={peerAddress}
                    onChange={(event) => setPeerAddress(event.target.value)}
                    placeholder="Peer multiaddr, e.g. /ip6/2a02:.../tcp/40100/p2p/12D3Koo..."
                />
                <div className="row">
                    <button className="btn" onClick={connectPeerAddress}>Connect Peer</button>
                </div>

                {recommendedListenAddr ? (
                    <p className="hint">Recommended address to share: {recommendedListenAddr}</p>
                ) : null}
                {p2pStatus?.started && !hasGlobalIPv6Address ? (
                    <p className="hint">No global IPv6 detected. P2P will use IPv4 NAT traversal and relay fallback.</p>
                ) : null}

                {p2pStatus ? (
                    <ul>
                        <li>Started: {p2pStatus.started ? 'yes' : 'no'}</li>
                        <li>Effective Listen Port: {getEffectiveListenPort(p2pStatus) || 'N/A'}</li>
                        <li>Peer ID: {p2pStatus.peerId || 'N/A'}</li>
                        <li>Topic: {p2pStatus.topic || 'N/A'}</li>
                        <li>Connected Peers: {p2pStatus.connectedPeers?.length || 0}</li>
                    </ul>
                ) : <p>Click Refresh Dashboard to load status</p>}

                {p2pStatus?.listenAddrs?.length ? (
                    <div>
                        <h4>Listen Addresses (prioritized)</h4>
                        <ul>
                            {prioritizedListenAddrs.map((address) => <li key={address}>{address}</li>)}
                        </ul>
                    </div>
                ) : null}
            </div>

            <div className="panel">
                <h3>Governance Admins</h3>
                <div className="row">
                    <button className="btn" onClick={trustCurrentIdentity} disabled={!identityReady}>Trust Current Identity</button>
                </div>
                <input
                    className="input"
                    value={trustedAdminInput}
                    onChange={(event) => setTrustedAdminInput(event.target.value)}
                    placeholder="Add trusted admin pubkey"
                />
                <div className="row">
                    <button className="btn" onClick={addTrustedAdminByInput}>Add Trusted Admin</button>
                </div>
                {trustedAdmins.length === 0 ? <p>No trusted admins</p> : (
                    <ul>
                        {trustedAdmins.map((admin) => (
                            <li key={admin.adminPubkey}>{admin.role} · {admin.adminPubkey.slice(0, 18)}...</li>
                        ))}
                    </ul>
                )}
            </div>

            <div className="panel">
                <h3>Moderation Controls (Broadcast)</h3>
                <div className="row">
                    <span className="badge">Hide History On Shadow Ban: {governancePolicy.hideHistoryOnShadowBan ? 'ON' : 'OFF'}</span>
                    <button className="btn" onClick={toggleHideHistoryOnShadowBan} disabled={!isGovernanceAdmin}>Toggle Policy</button>
                </div>
                {!isGovernanceAdmin ? <p className="hint">仅 Genesis / Trusted Admin 可修改治理策略。</p> : null}
                <input
                    className="input"
                    value={moderationTarget}
                    onChange={(event) => setModerationTarget(event.target.value)}
                    placeholder="Target pubkey"
                />
                <input
                    className="input"
                    value={moderationReason}
                    onChange={(event) => setModerationReason(event.target.value)}
                    placeholder="Reason"
                />
                <div className="row">
                    <button className="btn" onClick={applyShadowBan} disabled={!identityReady}>Mock SHADOW_BAN</button>
                    <button className="btn" onClick={applyUnban} disabled={!identityReady}>Mock UNBAN</button>
                </div>
                {governanceStatus ? <p className="result">{governanceStatus}</p> : null}
            </div>

            <div className="grid">
                <div className="panel">
                    <h3>Storage Usage</h3>
                    {storage ? (
                        <ul>
                            <li>Private: {bytesToMB(storage.privateUsedBytes)} / {bytesToMB(storage.privateQuota)}</li>
                            <li>Public: {bytesToMB(storage.publicUsedBytes)} / {bytesToMB(storage.publicQuota)}</li>
                            <li>Total Quota: {bytesToMB(storage.totalQuota)}</li>
                        </ul>
                    ) : <p>Click Refresh Dashboard</p>}
                </div>

                <div className="panel">
                    <h3>Moderation State</h3>
                    {moderation.length === 0 ? <p>No moderation record</p> : (
                        <ul>
                            {moderation.slice(0, 10).map((row) => (
                                <li key={row.targetPubkey + row.timestamp}>
                                    {row.action} · {row.targetPubkey.slice(0, 12)}... · {row.reason || 'no reason'}
                                </li>
                            ))}
                        </ul>
                    )}
                </div>

                <div className="panel">
                    <h3>Moderation Logs</h3>
                    {moderationLogs.length === 0 ? <p>No moderation log</p> : (
                        <ul>
                            {moderationLogs.slice(0, 20).map((row) => (
                                <li key={`${row.id}-${row.timestamp}`}>
                                    {row.action} · {row.targetPubkey.slice(0, 12)}... · {row.result} · {row.reason || 'no reason'}
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            </div>

            <div className="grid">
                <div className="panel">
                    <h3>Public Feed · sub/{currentSubId} · {sortMode.toUpperCase()} ({feed.length})</h3>
                    {feed.length === 0 ? <p>No public post</p> : (
                        <ul>
                            {feed.slice(0, 10).map((item) => (
                                <li key={item.id} onClick={() => {
                                    hydrateSelectedPostBody(item);
                                    hydrateSelectedPostImage(item);
                                    loadCommentsForPost(item.id);
                                }}>
                                    {getAuthorAvatar(item.pubkey) ? (
                                        <img
                                            src={getAuthorAvatar(item.pubkey)}
                                            alt="avatar"
                                            style={{ width: 20, height: 20, borderRadius: 10, marginRight: 6, verticalAlign: 'middle' }}
                                            onError={(event) => {
                                                event.currentTarget.style.display = 'none';
                                            }}
                                        />
                                    ) : null}
                                    {(item.thumbCid && thumbnailURLByCID[item.thumbCid]) ? (
                                        <img
                                            src={thumbnailURLByCID[item.thumbCid]}
                                            alt="post-thumb"
                                            style={{ width: 40, height: 40, borderRadius: 6, marginRight: 6, objectFit: 'cover', verticalAlign: 'middle' }}
                                        />
                                    ) : null}
                                    [{item.subId || 'general'}] <strong>{item.title}</strong> · {toPreview(item.body)} · @{getAuthorLabel(item.pubkey)}
                                    <span className="badge" style={{ marginLeft: 8 }}>▲ {item.score || 0}</span>
                                </li>
                            ))}
                        </ul>
                    )}

                    {selectedPublicPost ? (
                        <div>
                            <h4>Selected Post</h4>
                            <p><strong>{selectedPublicPost.title}</strong></p>
                            <p>Author: @{getAuthorLabel(selectedPublicPost.pubkey)}</p>
                            <div className="row">
                                <span className="badge">Score: {selectedPublicPost.score || 0}</span>
                                <button className="btn" onClick={upvoteSelectedPost} disabled={!identityReady}>Upvote Post</button>
                            </div>
                            {getAuthorAvatar(selectedPublicPost.pubkey) ? (
                                <img
                                    src={getAuthorAvatar(selectedPublicPost.pubkey)}
                                    alt="avatar"
                                    style={{ width: 40, height: 40, borderRadius: 20 }}
                                    onError={(event) => {
                                        event.currentTarget.style.display = 'none';
                                    }}
                                />
                            ) : null}
                            <p>{selectedPublicPost.body}</p>
                            {selectedBodyLoading ? <p className="hint">Loading full body...</p> : null}
                            {!selectedBodyLoading && selectedBodyHint ? <p className="hint">{selectedBodyHint}</p> : null}
                            {selectedPublicPost.imageCid ? (
                                <div>
                                    {selectedImageLoading ? <p className="hint">Loading image...</p> : null}
                                    {!selectedImageLoading && selectedImageURL ? (
                                        <img
                                            src={selectedImageURL}
                                            alt="post-image"
                                            style={{ maxWidth: 520, maxHeight: 420, borderRadius: 8 }}
                                        />
                                    ) : null}
                                </div>
                            ) : null}

                            <div className="panel">
                                <h4>Comments ({postComments.length})</h4>
                                <textarea
                                    className="input"
                                    rows={2}
                                    value={commentBody}
                                    onChange={(event) => setCommentBody(event.target.value)}
                                    placeholder="Write a comment"
                                />
                                <input
                                    className="input"
                                    value={replyToCommentId}
                                    onChange={(event) => setReplyToCommentId(event.target.value)}
                                    placeholder="Reply to comment ID (optional)"
                                />
                                <div className="row">
                                    <button className="btn" onClick={publishCommentForSelectedPost} disabled={!identityReady}>Publish Comment</button>
                                    {replyToCommentId ? (
                                        <button className="btn" onClick={() => setReplyToCommentId('')}>Clear Reply</button>
                                    ) : null}
                                </div>

                                {postComments.length === 0 ? <p>No comments</p> : (
                                    <ul>
                                        {postComments.map((comment) => (
                                            <li key={comment.id}>
                                                {comment.parentId ? <span className="badge">reply</span> : null} @{getAuthorLabel(comment.pubkey)} · {comment.body}
                                                <div className="row">
                                                    <small>{comment.id.slice(0, 10)}...</small>
                                                    <span className="badge">▲ {comment.score || 0}</span>
                                                    <button className="btn" onClick={() => upvoteComment(comment)} disabled={!identityReady}>Upvote</button>
                                                    <button className="btn" onClick={() => setReplyToCommentId(comment.id)}>Reply</button>
                                                </div>
                                            </li>
                                        ))}
                                    </ul>
                                )}
                            </div>
                        </div>
                    ) : null}
                </div>

                <div className="panel">
                    <h3>Private Feed ({privateFeed.length})</h3>
                    {privateFeed.length === 0 ? <p>No private post</p> : (
                        <ul>
                            {privateFeed.slice(0, 10).map((item) => (
                                <li key={item.id}>
                                    {getAuthorAvatar(item.pubkey) ? (
                                        <img
                                            src={getAuthorAvatar(item.pubkey)}
                                            alt="avatar"
                                            style={{ width: 20, height: 20, borderRadius: 10, marginRight: 6, verticalAlign: 'middle' }}
                                            onError={(event) => {
                                                event.currentTarget.style.display = 'none';
                                            }}
                                        />
                                    ) : null}
                                    <strong>{item.title}</strong> · {toPreview(item.body)} · @{getAuthorLabel(item.pubkey)}
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            </div>

            {error && <p className="error">{error}</p>}
        </div>
    );
}

export default App;
