export namespace main {
	
	export class ForumMessage {
	    id: string;
	    pubkey: string;
	    content: string;
	    timestamp: number;
	    sizeBytes: number;
	    zone: string;
	    isProtected: number;
	    visibility: string;
	
	    static createFrom(source: any = {}) {
	        return new ForumMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pubkey = source["pubkey"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	        this.sizeBytes = source["sizeBytes"];
	        this.zone = source["zone"];
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

}

