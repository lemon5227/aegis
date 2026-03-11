// constants.go — 全部业务常量定义，包括存储配额、操作类型、Lamport 相关、消息类型、通知类型等。
package main

// Storage quota constants.
const (
	totalQuotaBytes   int64 = 100 * 1024 * 1024
	privateQuotaBytes int64 = 20 * 1024 * 1024
	publicQuotaBytes  int64 = 80 * 1024 * 1024
)

// Default sub ID.
const defaultSubID = "general"

// Operation type, Lamport, entity type, vote state, and default value constants.
const (
	postOpTypeCreate    = "CREATE"
	postOpTypeUpdate    = "UPDATE"
	postOpTypeDelete    = "DELETE"
	lamportSchemaV2     = 2
	authScopeUser       = "user"
	entityTypePost      = "post"
	entityTypeComment   = "comment"
	voteStateNone       = "NONE"
	voteStateUp         = "UP"
	voteStateDown       = "DOWN"
	defaultOpNonceBytes = 8
)

// P2P message type constants.
const (
	messageTypeContentFetchRequest    = "CONTENT_FETCH_REQUEST"
	messageTypeContentFetchResponse   = "CONTENT_FETCH_RESPONSE"
	messageTypeMediaFetchRequest      = "MEDIA_FETCH_REQUEST"
	messageTypeMediaFetchResponse     = "MEDIA_FETCH_RESPONSE"
	messageTypeSyncSummaryRequest     = "SYNC_SUMMARY_REQUEST"
	messageTypeSyncSummaryResponse    = "SYNC_SUMMARY_RESPONSE"
	messageTypeCommentSyncRequest     = "COMMENT_SYNC_REQUEST"
	messageTypeCommentSyncResponse    = "COMMENT_SYNC_RESPONSE"
	messageTypeGovernanceSyncRequest  = "GOVERNANCE_SYNC_REQUEST"
	messageTypeGovernanceSyncResponse = "GOVERNANCE_SYNC_RESPONSE"
	messageTypeFavoriteOp             = "FAVORITE_OP"
	messageTypeFavoriteSyncRequest    = "FAVORITE_SYNC_REQUEST"
	messageTypeFavoriteSyncResponse   = "FAVORITE_SYNC_RESPONSE"
	messageTypePeerExchangeRequest    = "PEER_EXCHANGE_REQUEST"
	messageTypePeerExchangeResponse   = "PEER_EXCHANGE_RESPONSE"
)

// Notification type constants.
const (
	NotifTypePostComment    = "post_comment"
	NotifTypeCommentReply   = "comment_reply"
	NotifTypePostUpvote     = "post_upvote"
	NotifTypePostDownvote   = "post_downvote"
	NotifTypeCommentUpvote  = "comment_upvote"
	NotifTypeCommentDownvote = "comment_downvote"
	NotifTypeGovernance     = "governance_action"
)
