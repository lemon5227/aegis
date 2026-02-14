import { useEffect, useState } from 'react';
import './App.css';
import {
    AddLocalPost,
    AddTrustedAdmin,
    ConnectPeer,
    GenerateIdentity,
    GetFeed,
    GetModerationState,
    GetP2PStatus,
    GetPrivateFeed,
    GetStorageUsage,
    GetTrustedAdmins,
    PublishPost,
    PublishShadowBan,
    PublishUnban,
    ImportIdentityFromMnemonic,
    LoadSavedIdentity,
    StartP2P,
    StopP2P,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

type ForumMessage = {
    id: string;
    pubkey: string;
    content: string;
    timestamp: number;
    sizeBytes: number;
    zone: 'private' | 'public';
    visibility: string;
};

type ModerationState = {
    targetPubkey: string;
    action: string;
    sourceAdmin: string;
    timestamp: number;
    reason: string;
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

const bytesToMB = (bytes: number) => `${(bytes / (1024 * 1024)).toFixed(2)} MB`;

function App() {
    const [mnemonic, setMnemonic] = useState('');
    const [publicKey, setPublicKey] = useState('');
    const [postContent, setPostContent] = useState('Hello Aegis from local node');
    const [postZone, setPostZone] = useState<'private' | 'public'>('public');
    const [moderationTarget, setModerationTarget] = useState('');
    const [moderationReason, setModerationReason] = useState('manual-test');
    const [p2pPort, setP2pPort] = useState('40100');
    const [peerAddress, setPeerAddress] = useState('');
    const [trustedAdminInput, setTrustedAdminInput] = useState('');
    const [mnemonicInput, setMnemonicInput] = useState('');
    const [p2pStatus, setP2pStatus] = useState<P2PStatus | null>(null);
    const [trustedAdmins, setTrustedAdmins] = useState<GovernanceAdmin[]>([]);

    const [feed, setFeed] = useState<ForumMessage[]>([]);
    const [privateFeed, setPrivateFeed] = useState<ForumMessage[]>([]);
    const [moderation, setModeration] = useState<ModerationState[]>([]);
    const [storage, setStorage] = useState<StorageUsage | null>(null);

    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const identityReady = publicKey.trim().length > 0;
    const currentAdmin = trustedAdmins.find((admin) => admin.adminPubkey === publicKey);
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

    async function createIdentity() {
        setLoading(true);
        setError('');

        try {
            ensureRuntime();

            const identity = await GenerateIdentity();
            setMnemonic(identity.mnemonic);
            setPublicKey(identity.publicKey);

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
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function refreshDashboard() {
        setError('');
        try {
            ensureRuntime();

            const [publicMessages, privateMessages, moderationRows, usage] = await Promise.all([
                GetFeed(),
                GetPrivateFeed(),
                GetModerationState(),
                GetStorageUsage(),
            ]);

            setFeed(publicMessages as ForumMessage[]);
            setPrivateFeed(privateMessages as ForumMessage[]);
            setModeration(moderationRows as ModerationState[]);
            setStorage(usage as StorageUsage);

            const status = await GetP2PStatus();
            setP2pStatus(status as P2PStatus);

            const admins = await GetTrustedAdmins();
            setTrustedAdmins(admins as GovernanceAdmin[]);
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

            await AddLocalPost(publicKey, postContent.trim(), postZone);
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function applyShadowBan() {
        setError('');
        try {
            ensureRuntime();

            if (!publicKey) {
                throw new Error('请先创建身份作为管理节点公钥');
            }
            if (!moderationTarget.trim()) {
                throw new Error('请输入目标公钥');
            }

            await PublishShadowBan(moderationTarget.trim(), publicKey, moderationReason.trim());
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    async function applyUnban() {
        setError('');
        try {
            ensureRuntime();

            if (!publicKey) {
                throw new Error('请先创建身份作为管理节点公钥');
            }
            if (!moderationTarget.trim()) {
                throw new Error('请输入目标公钥');
            }

            await PublishUnban(moderationTarget.trim(), publicKey, moderationReason.trim());
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
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

            await PublishPost(publicKey, postContent.trim());
            await refreshDashboard();
        } catch (exception) {
            setError(String(exception));
        }
    }

    useEffect(() => {
        if (!hasWailsRuntime()) {
            return;
        }

        loadSavedIdentity(false);

        const unsubscribe = EventsOn('feed:updated', () => {
            refreshDashboard();
        });

        return () => {
            unsubscribe();
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
                <h3>Add Local Post</h3>
                <textarea
                    className="input"
                    rows={3}
                    value={postContent}
                    onChange={(event) => setPostContent(event.target.value)}
                    placeholder="Input post content"
                />
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
                    placeholder="Peer multiaddr, e.g. /ip4/192.168.1.10/tcp/40100/p2p/12D3Koo..."
                />
                <div className="row">
                    <button className="btn" onClick={connectPeerAddress}>Connect Peer</button>
                </div>

                {p2pStatus ? (
                    <ul>
                        <li>Started: {p2pStatus.started ? 'yes' : 'no'}</li>
                        <li>Peer ID: {p2pStatus.peerId || 'N/A'}</li>
                        <li>Topic: {p2pStatus.topic || 'N/A'}</li>
                        <li>Connected Peers: {p2pStatus.connectedPeers?.length || 0}</li>
                    </ul>
                ) : <p>Click Refresh Dashboard to load status</p>}

                {p2pStatus?.listenAddrs?.length ? (
                    <div>
                        <h4>Listen Addresses</h4>
                        <ul>
                            {p2pStatus.listenAddrs.map((address) => <li key={address}>{address}</li>)}
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
            </div>

            <div className="grid">
                <div className="panel">
                    <h3>Public Feed ({feed.length})</h3>
                    {feed.length === 0 ? <p>No public post</p> : (
                        <ul>
                            {feed.slice(0, 10).map((item) => (
                                <li key={item.id}>{item.content}</li>
                            ))}
                        </ul>
                    )}
                </div>

                <div className="panel">
                    <h3>Private Feed ({privateFeed.length})</h3>
                    {privateFeed.length === 0 ? <p>No private post</p> : (
                        <ul>
                            {privateFeed.slice(0, 10).map((item) => (
                                <li key={item.id}>{item.content}</li>
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
