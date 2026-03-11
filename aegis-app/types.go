// types.go — 全部数据模型类型定义，集中管理 Go 后端的结构体类型。
package main

// ForumMessage represents a forum post message stored in the database.
type ForumMessage struct {
	ID          string `json:"id"`
	Pubkey      string `json:"pubkey"`
	OpID        string `json:"opId,omitempty"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	ContentCID  string `json:"contentCid"`
	ImageCID    string `json:"imageCid"`
	ThumbCID    string `json:"thumbCid"`
	ImageMIME   string `json:"imageMime"`
	ImageSize   int64  `json:"imageSize"`
	ImageWidth  int    `json:"imageWidth"`
	ImageHeight int    `json:"imageHeight"`
	Content     string `json:"content"`
	Score       int64  `json:"score"`
	Timestamp   int64  `json:"timestamp"`
	Lamport     int64  `json:"lamport"`
	SizeBytes   int64  `json:"sizeBytes"`
	Zone        string `json:"zone"`
	SubID       string `json:"subId"`
	IsProtected int    `json:"isProtected"`
	Visibility  string `json:"visibility"`
	DeletedAt   int64  `json:"deletedAt,omitempty"`
	DeletedBy   string `json:"deletedBy,omitempty"`
}

// PostIndex represents a lightweight post index entry for feed display.
type PostIndex struct {
	ID          string `json:"id"`
	Pubkey      string `json:"pubkey"`
	Title       string `json:"title"`
	BodyPreview string `json:"bodyPreview"`
	ContentCID  string `json:"contentCid"`
	ImageCID    string `json:"imageCid"`
	ThumbCID    string `json:"thumbCid"`
	ImageMIME   string `json:"imageMime"`
	ImageSize   int64  `json:"imageSize"`
	ImageWidth  int    `json:"imageWidth"`
	ImageHeight int    `json:"imageHeight"`
	Score       int64  `json:"score"`
	Timestamp   int64  `json:"timestamp"`
	Zone        string `json:"zone"`
	SubID       string `json:"subId"`
	Visibility  string `json:"visibility"`
}

// PostBodyBlob represents the full body content of a post.
type PostBodyBlob struct {
	ContentCID string `json:"contentCid"`
	Body       string `json:"body"`
	SizeBytes  int64  `json:"sizeBytes"`
}

// MediaBlob represents a media attachment blob.
type MediaBlob struct {
	ContentCID  string `json:"contentCid"`
	DataBase64  string `json:"dataBase64"`
	Mime        string `json:"mime"`
	SizeBytes   int64  `json:"sizeBytes"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	IsThumbnail bool   `json:"isThumbnail"`
}

// Sub represents a forum sub-community.
type Sub struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"createdAt"`
}

// SubStats represents statistics for a sub-community.
type SubStats struct {
	SubID           string `json:"subId"`
	SubscriberCount int64  `json:"subscriberCount"`
	PostCount       int64  `json:"postCount"`
	ActiveAuthors   int64  `json:"activeAuthors"`
	RecentPosts24h  int64  `json:"recentPosts24h"`
	CreatedAt       int64  `json:"createdAt"`
}

// Profile represents a user's public profile.
type Profile struct {
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarURL"`
	UpdatedAt   int64  `json:"updatedAt"`
}

// ProfileDetails represents a user's detailed profile including bio.
type ProfileDetails struct {
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarURL"`
	Bio         string `json:"bio"`
	UpdatedAt   int64  `json:"updatedAt"`
}

// Comment represents a comment on a post.
type Comment struct {
	ID          string              `json:"id"`
	PostID      string              `json:"postId"`
	ParentID    string              `json:"parentId"`
	Pubkey      string              `json:"pubkey"`
	OpID        string              `json:"opId,omitempty"`
	Body        string              `json:"body"`
	Attachments []CommentAttachment `json:"attachments,omitempty"`
	Score       int64               `json:"score"`
	Timestamp   int64               `json:"timestamp"`
	Lamport     int64               `json:"lamport"`
	DeletedAt   int64               `json:"deletedAt,omitempty"`
	DeletedBy   string              `json:"deletedBy,omitempty"`
}

// CommentAttachment represents a media attachment on a comment.
type CommentAttachment struct {
	Kind      string `json:"kind"`
	Ref       string `json:"ref"`
	Mime      string `json:"mime,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
}

// ModerationState represents the current moderation state for a user.
type ModerationState struct {
	TargetPubkey string `json:"targetPubkey"`
	Action       string `json:"action"`
	SourceAdmin  string `json:"sourceAdmin"`
	Timestamp    int64  `json:"timestamp"`
	Lamport      int64  `json:"lamport"`
	Reason       string `json:"reason"`
}

// ModerationLog represents a moderation action log entry.
type ModerationLog struct {
	ID           int64  `json:"id"`
	TargetPubkey string `json:"targetPubkey"`
	Action       string `json:"action"`
	SourceAdmin  string `json:"sourceAdmin"`
	Timestamp    int64  `json:"timestamp"`
	Lamport      int64  `json:"lamport"`
	Reason       string `json:"reason"`
	Result       string `json:"result"`
}

// GovernancePolicy represents the governance policy settings.
type GovernancePolicy struct {
	HideHistoryOnShadowBan bool `json:"hideHistoryOnShadowBan"`
}

// P2PConfig represents the P2P network configuration.
type P2PConfig struct {
	ListenPort int      `json:"listenPort"`
	RelayPeers []string `json:"relayPeers"`
	AutoStart  bool     `json:"autoStart"`
	UpdatedAt  int64    `json:"updatedAt"`
}

// PrivacySettings represents user privacy preferences.
type PrivacySettings struct {
	ShowOnlineStatus bool  `json:"showOnlineStatus"`
	AllowSearch      bool  `json:"allowSearch"`
	UpdatedAt        int64 `json:"updatedAt"`
}

// IdentityState represents the state of a user identity.
type IdentityState struct {
	Pubkey             string `json:"pubkey"`
	State              string `json:"state"`
	StorageCommitBytes int64  `json:"storageCommitBytes"`
	PublicQuotaBytes   int64  `json:"publicQuotaBytes"`
	PrivateQuotaBytes  int64  `json:"privateQuotaBytes"`
	UpdatedAt          int64  `json:"updatedAt"`
}

// StorageUsage represents current storage usage statistics.
type StorageUsage struct {
	PrivateUsedBytes int64 `json:"privateUsedBytes"`
	PublicUsedBytes  int64 `json:"publicUsedBytes"`
	PrivateQuota     int64 `json:"privateQuota"`
	PublicQuota      int64 `json:"publicQuota"`
	TotalQuota       int64 `json:"totalQuota"`
}

// FeedStreamItem represents a single item in the feed stream.
type FeedStreamItem struct {
	Post                ForumMessage `json:"post"`
	Reason              string       `json:"reason"`
	IsSubscribed        bool         `json:"isSubscribed"`
	RecommendationScore float64      `json:"recommendationScore"`
}

// FeedStream represents a paginated feed stream with algorithm metadata.
type FeedStream struct {
	Items       []FeedStreamItem `json:"items"`
	Algorithm   string           `json:"algorithm"`
	GeneratedAt int64            `json:"generatedAt"`
}

// PostIndexPage represents a paginated page of post index entries.
type PostIndexPage struct {
	Items      []PostIndex `json:"items"`
	NextCursor string      `json:"nextCursor"`
}

// FavoriteOpRecord represents a favorite operation record.
type FavoriteOpRecord struct {
	OpID      string `json:"opId"`
	Pubkey    string `json:"pubkey"`
	PostID    string `json:"postId"`
	Op        string `json:"op"`
	CreatedAt int64  `json:"createdAt"`
	Signature string `json:"signature"`
}

// EntityOpRecord represents an entity operation log record.
type EntityOpRecord struct {
	OpID          string `json:"opId"`
	EntityType    string `json:"entityType"`
	EntityID      string `json:"entityId"`
	OpType        string `json:"opType"`
	AuthorPubkey  string `json:"authorPubkey"`
	Lamport       int64  `json:"lamport"`
	Timestamp     int64  `json:"timestamp"`
	SchemaVersion int    `json:"schemaVersion"`
	AuthScope     string `json:"authScope"`
	PayloadJSON   string `json:"payloadJson"`
}

// TombstoneGCResult represents the result of a tombstone garbage collection run.
type TombstoneGCResult struct {
	ScannedPosts    int `json:"scannedPosts"`
	DeletedPosts    int `json:"deletedPosts"`
	ScannedComments int `json:"scannedComments"`
	DeletedComments int `json:"deletedComments"`
}

// GovernanceAdmin represents a trusted governance administrator.
type GovernanceAdmin struct {
	AdminPubkey string `json:"adminPubkey"`
	Role        string `json:"role"`
	Active      bool   `json:"active"`
}

// SyncPostDigest represents a post digest used in anti-entropy sync.
type SyncPostDigest struct {
	ID               string `json:"id"`
	Pubkey           string `json:"pubkey"`
	OpID             string `json:"op_id,omitempty"`
	OpType           string `json:"op_type,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
	DeletedAtLamport int64  `json:"deleted_at_lamport,omitempty"`
	Title            string `json:"title"`
	ContentCID       string `json:"content_cid"`
	ImageCID         string `json:"image_cid"`
	ThumbCID         string `json:"thumb_cid"`
	ImageMIME        string `json:"image_mime"`
	ImageSize        int64  `json:"image_size"`
	ImageWidth       int    `json:"image_width"`
	ImageHeight      int    `json:"image_height"`
	Timestamp        int64  `json:"timestamp"`
	Lamport          int64  `json:"lamport"`
	SubID            string `json:"sub_id"`
}

// SyncCommentDigest represents a comment digest used in anti-entropy sync.
type SyncCommentDigest struct {
	ID               string              `json:"id"`
	PostID           string              `json:"post_id"`
	ParentID         string              `json:"parent_id"`
	Pubkey           string              `json:"pubkey"`
	OpID             string              `json:"op_id,omitempty"`
	OpType           string              `json:"op_type,omitempty"`
	Deleted          bool                `json:"deleted,omitempty"`
	DeletedAtLamport int64               `json:"deleted_at_lamport,omitempty"`
	DisplayName      string              `json:"display_name"`
	AvatarURL        string              `json:"avatar_url"`
	Body             string              `json:"body"`
	Attachments      []CommentAttachment `json:"attachments,omitempty"`
	Score            int64               `json:"score"`
	Timestamp        int64               `json:"timestamp"`
	Lamport          int64               `json:"lamport"`
}

// IncomingMessage represents a message received from the P2P network.
type IncomingMessage struct {
	Type                   string              `json:"type"`
	OpType                 string              `json:"op_type,omitempty"`
	OpID                   string              `json:"op_id,omitempty"`
	SchemaVersion          int                 `json:"schema_version,omitempty"`
	AuthScope              string              `json:"auth_scope,omitempty"`
	ID                     string              `json:"id"`
	Pubkey                 string              `json:"pubkey"`
	VoterPubkey            string              `json:"voter_pubkey"`
	VoteState              string              `json:"vote_state,omitempty"`
	PostID                 string              `json:"post_id"`
	CommentID              string              `json:"comment_id"`
	ParentID               string              `json:"parent_id"`
	DisplayName            string              `json:"display_name"`
	AvatarURL              string              `json:"avatar_url"`
	Title                  string              `json:"title"`
	Body                   string              `json:"body"`
	CommentAttachments     []CommentAttachment `json:"comment_attachments,omitempty"`
	ContentCID             string              `json:"content_cid"`
	ImageCID               string              `json:"image_cid"`
	ThumbCID               string              `json:"thumb_cid"`
	ImageMIME              string              `json:"image_mime"`
	ImageSize              int64               `json:"image_size"`
	ImageWidth             int                 `json:"image_width"`
	ImageHeight            int                 `json:"image_height"`
	ImageDataBase64        string              `json:"image_data_base64,omitempty"`
	IsThumbnail            bool                `json:"is_thumbnail,omitempty"`
	RequestID              string              `json:"request_id"`
	RequesterPeerID        string              `json:"requester_peer_id"`
	ResponderPeerID        string              `json:"responder_peer_id"`
	SyncSinceTimestamp     int64               `json:"sync_since_timestamp,omitempty"`
	SyncWindowSeconds      int64               `json:"sync_window_seconds,omitempty"`
	SyncBatchSize          int                 `json:"sync_batch_size,omitempty"`
	CommentSinceTs         int64               `json:"comment_since_ts,omitempty"`
	CommentBatchSize       int                 `json:"comment_batch_size,omitempty"`
	GovernanceSinceTs      int64               `json:"governance_since_ts,omitempty"`
	GovernanceBatchSize    int                 `json:"governance_batch_size,omitempty"`
	GovernanceLogSinceTs   int64               `json:"governance_log_since_ts,omitempty"`
	GovernanceLogLimit     int                 `json:"governance_log_limit,omitempty"`
	GovernanceStates       []ModerationState   `json:"governance_states,omitempty"`
	GovernanceLogs         []ModerationLog     `json:"governance_logs,omitempty"`
	FavoriteOpID           string              `json:"favorite_op_id,omitempty"`
	FavoriteOp             string              `json:"favorite_op,omitempty"`
	FavoriteSinceTs        int64               `json:"favorite_since_ts,omitempty"`
	FavoriteBatchSize      int                 `json:"favorite_batch_size,omitempty"`
	FavoriteOps            []FavoriteOpRecord  `json:"favorite_ops,omitempty"`
	Found                  bool                `json:"found"`
	SizeBytes              int64               `json:"size_bytes"`
	Content                string              `json:"content"`
	SubID                  string              `json:"sub_id"`
	SubTitle               string              `json:"sub_title"`
	SubDesc                string              `json:"sub_desc"`
	Timestamp              int64               `json:"timestamp"`
	Lamport                int64               `json:"lamport,omitempty"`
	DeletedAtLamport       int64               `json:"deleted_at_lamport,omitempty"`
	Signature              string              `json:"signature"`
	TargetPubkey           string              `json:"target_pubkey"`
	AdminPubkey            string              `json:"admin_pubkey"`
	Reason                 string              `json:"reason"`
	Summaries              []SyncPostDigest    `json:"summaries,omitempty"`
	CommentSummaries       []SyncCommentDigest `json:"comment_summaries,omitempty"`
	KnownPeers             []KnownPeerExchange `json:"known_peers,omitempty"`
	RelayCapable           bool                `json:"relay_capable,omitempty"`
	PublicReachable        bool                `json:"public_reachable,omitempty"`
	HideHistoryOnShadowBan bool                `json:"hide_history_on_shadowban"`
}

// KnownPeerExchange represents a peer entry exchanged during peer discovery.
type KnownPeerExchange struct {
	PeerID          string   `json:"peer_id"`
	Addrs           []string `json:"addrs"`
	RelayCapable    bool     `json:"relay_capable"`
	PublicReachable bool     `json:"public_reachable"`
	LastSeen        int64    `json:"last_seen"`
}

// LamportVersion represents a Lamport clock version for conflict resolution.
type LamportVersion struct {
	Lamport int64
	Author  string
	OpID    string
}

// Identity represents a user's cryptographic identity (mnemonic + public key).
type Identity struct {
	Mnemonic  string `json:"mnemonic"`
	PublicKey string `json:"publicKey"`
}

// AntiEntropyStats tracks statistics for the anti-entropy sync protocol.
type AntiEntropyStats struct {
	SyncRequestsSent       int64 `json:"syncRequestsSent"`
	SyncRequestsReceived   int64 `json:"syncRequestsReceived"`
	SyncResponsesReceived  int64 `json:"syncResponsesReceived"`
	SyncSummariesReceived  int64 `json:"syncSummariesReceived"`
	IndexInsertions        int64 `json:"indexInsertions"`
	BlobFetchAttempts      int64 `json:"blobFetchAttempts"`
	BlobFetchSuccess       int64 `json:"blobFetchSuccess"`
	BlobFetchFailures      int64 `json:"blobFetchFailures"`
	LastSyncAt             int64 `json:"lastSyncAt"`
	LastRemoteSummaryTs    int64 `json:"lastRemoteSummaryTs"`
	LastObservedSyncLagSec int64 `json:"lastObservedSyncLagSec"`
}
