export namespace main {
	
	export class AntiEntropyStats {
	    syncRequestsSent: number;
	    syncRequestsReceived: number;
	    syncResponsesReceived: number;
	    syncSummariesReceived: number;
	    indexInsertions: number;
	    blobFetchAttempts: number;
	    blobFetchSuccess: number;
	    blobFetchFailures: number;
	    lastSyncAt: number;
	    lastRemoteSummaryTs: number;
	    lastObservedSyncLagSec: number;
	
	    static createFrom(source: any = {}) {
	        return new AntiEntropyStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.syncRequestsSent = source["syncRequestsSent"];
	        this.syncRequestsReceived = source["syncRequestsReceived"];
	        this.syncResponsesReceived = source["syncResponsesReceived"];
	        this.syncSummariesReceived = source["syncSummariesReceived"];
	        this.indexInsertions = source["indexInsertions"];
	        this.blobFetchAttempts = source["blobFetchAttempts"];
	        this.blobFetchSuccess = source["blobFetchSuccess"];
	        this.blobFetchFailures = source["blobFetchFailures"];
	        this.lastSyncAt = source["lastSyncAt"];
	        this.lastRemoteSummaryTs = source["lastRemoteSummaryTs"];
	        this.lastObservedSyncLagSec = source["lastObservedSyncLagSec"];
	    }
	}
	export class CommentAttachment {
	    kind: string;
	    ref: string;
	    mime?: string;
	    width?: number;
	    height?: number;
	    sizeBytes?: number;
	
	    static createFrom(source: any = {}) {
	        return new CommentAttachment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.ref = source["ref"];
	        this.mime = source["mime"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.sizeBytes = source["sizeBytes"];
	    }
	}
	export class Comment {
	    id: string;
	    postId: string;
	    parentId: string;
	    pubkey: string;
	    body: string;
	    attachments?: CommentAttachment[];
	    score: number;
	    timestamp: number;
	    lamport: number;
	    deletedAt?: number;
	    deletedBy?: string;
	
	    static createFrom(source: any = {}) {
	        return new Comment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.postId = source["postId"];
	        this.parentId = source["parentId"];
	        this.pubkey = source["pubkey"];
	        this.body = source["body"];
	        this.attachments = this.convertValues(source["attachments"], CommentAttachment);
	        this.score = source["score"];
	        this.timestamp = source["timestamp"];
	        this.lamport = source["lamport"];
	        this.deletedAt = source["deletedAt"];
	        this.deletedBy = source["deletedBy"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ForumMessage {
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
	    content: string;
	    score: number;
	    timestamp: number;
	    lamport: number;
	    sizeBytes: number;
	    zone: string;
	    subId: string;
	    isProtected: number;
	    visibility: string;
	
	    static createFrom(source: any = {}) {
	        return new ForumMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pubkey = source["pubkey"];
	        this.title = source["title"];
	        this.body = source["body"];
	        this.contentCid = source["contentCid"];
	        this.imageCid = source["imageCid"];
	        this.thumbCid = source["thumbCid"];
	        this.imageMime = source["imageMime"];
	        this.imageSize = source["imageSize"];
	        this.imageWidth = source["imageWidth"];
	        this.imageHeight = source["imageHeight"];
	        this.content = source["content"];
	        this.score = source["score"];
	        this.timestamp = source["timestamp"];
	        this.lamport = source["lamport"];
	        this.sizeBytes = source["sizeBytes"];
	        this.zone = source["zone"];
	        this.subId = source["subId"];
	        this.isProtected = source["isProtected"];
	        this.visibility = source["visibility"];
	    }
	}
	export class FeedStreamItem {
	    post: ForumMessage;
	    reason: string;
	    isSubscribed: boolean;
	    recommendationScore: number;
	
	    static createFrom(source: any = {}) {
	        return new FeedStreamItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.post = this.convertValues(source["post"], ForumMessage);
	        this.reason = source["reason"];
	        this.isSubscribed = source["isSubscribed"];
	        this.recommendationScore = source["recommendationScore"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FeedStream {
	    items: FeedStreamItem[];
	    algorithm: string;
	    generatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new FeedStream(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], FeedStreamItem);
	        this.algorithm = source["algorithm"];
	        this.generatedAt = source["generatedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class GovernanceAdmin {
	    adminPubkey: string;
	    role: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GovernanceAdmin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.adminPubkey = source["adminPubkey"];
	        this.role = source["role"];
	        this.active = source["active"];
	    }
	}
	export class GovernancePolicy {
	    hideHistoryOnShadowBan: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GovernancePolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hideHistoryOnShadowBan = source["hideHistoryOnShadowBan"];
	    }
	}
	export class Identity {
	    mnemonic: string;
	    publicKey: string;
	
	    static createFrom(source: any = {}) {
	        return new Identity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mnemonic = source["mnemonic"];
	        this.publicKey = source["publicKey"];
	    }
	}
	export class IdentityState {
	    pubkey: string;
	    state: string;
	    storageCommitBytes: number;
	    publicQuotaBytes: number;
	    privateQuotaBytes: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new IdentityState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pubkey = source["pubkey"];
	        this.state = source["state"];
	        this.storageCommitBytes = source["storageCommitBytes"];
	        this.publicQuotaBytes = source["publicQuotaBytes"];
	        this.privateQuotaBytes = source["privateQuotaBytes"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class MediaBlob {
	    contentCid: string;
	    dataBase64: string;
	    mime: string;
	    sizeBytes: number;
	    width: number;
	    height: number;
	    isThumbnail: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MediaBlob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.contentCid = source["contentCid"];
	        this.dataBase64 = source["dataBase64"];
	        this.mime = source["mime"];
	        this.sizeBytes = source["sizeBytes"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.isThumbnail = source["isThumbnail"];
	    }
	}
	export class ModerationLog {
	    id: number;
	    targetPubkey: string;
	    action: string;
	    sourceAdmin: string;
	    timestamp: number;
	    lamport: number;
	    reason: string;
	    result: string;
	
	    static createFrom(source: any = {}) {
	        return new ModerationLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.targetPubkey = source["targetPubkey"];
	        this.action = source["action"];
	        this.sourceAdmin = source["sourceAdmin"];
	        this.timestamp = source["timestamp"];
	        this.lamport = source["lamport"];
	        this.reason = source["reason"];
	        this.result = source["result"];
	    }
	}
	export class ModerationState {
	    targetPubkey: string;
	    action: string;
	    sourceAdmin: string;
	    timestamp: number;
	    lamport: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ModerationState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetPubkey = source["targetPubkey"];
	        this.action = source["action"];
	        this.sourceAdmin = source["sourceAdmin"];
	        this.timestamp = source["timestamp"];
	        this.lamport = source["lamport"];
	        this.reason = source["reason"];
	    }
	}
	export class P2PConfig {
	    listenPort: number;
	    relayPeers: string[];
	    autoStart: boolean;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new P2PConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listenPort = source["listenPort"];
	        this.relayPeers = source["relayPeers"];
	        this.autoStart = source["autoStart"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class P2PStatus {
	    started: boolean;
	    peerId: string;
	    listenAddrs: string[];
	    announceAddrs: string[];
	    connectedPeers: string[];
	    topic: string;
	
	    static createFrom(source: any = {}) {
	        return new P2PStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.started = source["started"];
	        this.peerId = source["peerId"];
	        this.listenAddrs = source["listenAddrs"];
	        this.announceAddrs = source["announceAddrs"];
	        this.connectedPeers = source["connectedPeers"];
	        this.topic = source["topic"];
	    }
	}
	export class PostBodyBlob {
	    contentCid: string;
	    body: string;
	    sizeBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new PostBodyBlob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.contentCid = source["contentCid"];
	        this.body = source["body"];
	        this.sizeBytes = source["sizeBytes"];
	    }
	}
	export class PostIndex {
	    id: string;
	    pubkey: string;
	    title: string;
	    bodyPreview: string;
	    contentCid: string;
	    imageCid: string;
	    thumbCid: string;
	    imageMime: string;
	    imageSize: number;
	    imageWidth: number;
	    imageHeight: number;
	    score: number;
	    timestamp: number;
	    zone: string;
	    subId: string;
	    visibility: string;
	
	    static createFrom(source: any = {}) {
	        return new PostIndex(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pubkey = source["pubkey"];
	        this.title = source["title"];
	        this.bodyPreview = source["bodyPreview"];
	        this.contentCid = source["contentCid"];
	        this.imageCid = source["imageCid"];
	        this.thumbCid = source["thumbCid"];
	        this.imageMime = source["imageMime"];
	        this.imageSize = source["imageSize"];
	        this.imageWidth = source["imageWidth"];
	        this.imageHeight = source["imageHeight"];
	        this.score = source["score"];
	        this.timestamp = source["timestamp"];
	        this.zone = source["zone"];
	        this.subId = source["subId"];
	        this.visibility = source["visibility"];
	    }
	}
	export class PostIndexPage {
	    items: PostIndex[];
	    nextCursor: string;
	
	    static createFrom(source: any = {}) {
	        return new PostIndexPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], PostIndex);
	        this.nextCursor = source["nextCursor"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PrivacySettings {
	    showOnlineStatus: boolean;
	    allowSearch: boolean;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new PrivacySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.showOnlineStatus = source["showOnlineStatus"];
	        this.allowSearch = source["allowSearch"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Profile {
	    pubkey: string;
	    displayName: string;
	    avatarURL: string;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pubkey = source["pubkey"];
	        this.displayName = source["displayName"];
	        this.avatarURL = source["avatarURL"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ProfileDetails {
	    pubkey: string;
	    displayName: string;
	    avatarURL: string;
	    bio: string;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new ProfileDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pubkey = source["pubkey"];
	        this.displayName = source["displayName"];
	        this.avatarURL = source["avatarURL"];
	        this.bio = source["bio"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ReleaseAlert {
	    key: string;
	    metric: string;
	    level: string;
	    value: number;
	    threshold: number;
	    windowSec: number;
	    triggeredAt: number;
	
	    static createFrom(source: any = {}) {
	        return new ReleaseAlert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.metric = source["metric"];
	        this.level = source["level"];
	        this.value = source["value"];
	        this.threshold = source["threshold"];
	        this.windowSec = source["windowSec"];
	        this.triggeredAt = source["triggeredAt"];
	    }
	}
	export class ReleaseMetrics {
	    content_fetch_success_rate: number;
	    content_fetch_latency_p95: number;
	    blob_cache_hit_rate: number;
	    sync_lag_seconds: number;
	    content_fetch_attempts: number;
	    content_fetch_success: number;
	    content_fetch_failures: number;
	    blob_cache_hits: number;
	    blob_cache_misses: number;
	
	    static createFrom(source: any = {}) {
	        return new ReleaseMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content_fetch_success_rate = source["content_fetch_success_rate"];
	        this.content_fetch_latency_p95 = source["content_fetch_latency_p95"];
	        this.blob_cache_hit_rate = source["blob_cache_hit_rate"];
	        this.sync_lag_seconds = source["sync_lag_seconds"];
	        this.content_fetch_attempts = source["content_fetch_attempts"];
	        this.content_fetch_success = source["content_fetch_success"];
	        this.content_fetch_failures = source["content_fetch_failures"];
	        this.blob_cache_hits = source["blob_cache_hits"];
	        this.blob_cache_misses = source["blob_cache_misses"];
	    }
	}
	export class StorageUsage {
	    privateUsedBytes: number;
	    publicUsedBytes: number;
	    privateQuota: number;
	    publicQuota: number;
	    totalQuota: number;
	
	    static createFrom(source: any = {}) {
	        return new StorageUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.privateUsedBytes = source["privateUsedBytes"];
	        this.publicUsedBytes = source["publicUsedBytes"];
	        this.privateQuota = source["privateQuota"];
	        this.publicQuota = source["publicQuota"];
	        this.totalQuota = source["totalQuota"];
	    }
	}
	export class Sub {
	    id: string;
	    title: string;
	    description: string;
	    createdAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Sub(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class UpdateStatus {
	    currentVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	    releaseURL: string;
	    releaseNotes: string;
	    publishedAt: number;
	    checkedAt: number;
	    errorMessage: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	        this.releaseURL = source["releaseURL"];
	        this.releaseNotes = source["releaseNotes"];
	        this.publishedAt = source["publishedAt"];
	        this.checkedAt = source["checkedAt"];
	        this.errorMessage = source["errorMessage"];
	    }
	}
	export class VersionHistoryItem {
	    version: string;
	    publishedAt: number;
	    summary: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new VersionHistoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.publishedAt = source["publishedAt"];
	        this.summary = source["summary"];
	        this.url = source["url"];
	    }
	}

}

