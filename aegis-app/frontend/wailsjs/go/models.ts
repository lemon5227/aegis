export namespace main {
	
	export class Comment {
	    id: string;
	    postId: string;
	    parentId: string;
	    pubkey: string;
	    body: string;
	    score: number;
	    timestamp: number;
	
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
	        this.score = source["score"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class ForumMessage {
	    id: string;
	    pubkey: string;
	    title: string;
	    body: string;
	    contentCid: string;
	    content: string;
	    score: number;
	    timestamp: number;
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
	        this.content = source["content"];
	        this.score = source["score"];
	        this.timestamp = source["timestamp"];
	        this.sizeBytes = source["sizeBytes"];
	        this.zone = source["zone"];
	        this.subId = source["subId"];
	        this.isProtected = source["isProtected"];
	        this.visibility = source["visibility"];
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
	export class ModerationLog {
	    id: number;
	    targetPubkey: string;
	    action: string;
	    sourceAdmin: string;
	    timestamp: number;
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
	        this.reason = source["reason"];
	        this.result = source["result"];
	    }
	}
	export class ModerationState {
	    targetPubkey: string;
	    action: string;
	    sourceAdmin: string;
	    timestamp: number;
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
	        this.reason = source["reason"];
	    }
	}
	export class P2PStatus {
	    started: boolean;
	    peerId: string;
	    listenAddrs: string[];
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
	        this.score = source["score"];
	        this.timestamp = source["timestamp"];
	        this.zone = source["zone"];
	        this.subId = source["subId"];
	        this.visibility = source["visibility"];
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

}

