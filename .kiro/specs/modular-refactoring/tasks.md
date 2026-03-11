# 实施计划：Aegis-App 架构级模块化重构

## 概述

按照设计文档的 4 阶段执行顺序，将 Go 后端和 React 前端的 God Object 拆分为职责单一的模块。每个阶段完成后通过编译和测试验证，并提交推送到仓库。所有代码保持在 `main` 包下（Go 端），前端引入 Context + Hook 模式。

## 任务

### 阶段一：Go 基础层提取（需求 1）

- [x] 1. 提取数据类型到 types.go
  - [x] 1.1 创建 `types.go` 文件，从 `db.go` 中提取全部 30+ 个 type 定义
    - 包括 ForumMessage, PostIndex, PostBodyBlob, MediaBlob, Sub, SubStats, Profile, ProfileDetails, Comment, CommentAttachment, ModerationState, ModerationLog, GovernancePolicy, P2PConfig, PrivacySettings, IdentityState, StorageUsage, FeedStreamItem, FeedStream, PostIndexPage, FavoriteOpRecord, EntityOpRecord, TombstoneGCResult, GovernanceAdmin, SyncPostDigest, SyncCommentDigest, IncomingMessage, KnownPeerExchange, LamportVersion 等
    - 文件顶部添加用途注释
    - _需求: 1.1_

  - [x] 1.2 从 `app.go` 中提取 Identity 和 AntiEntropyStats 类型到 `types.go`
    - _需求: 1.2_

  - [x] 1.3 创建 `constants.go` 文件，从 `db.go` 中提取全部业务常量
    - 包括存储配额常量（totalQuotaBytes, privateQuotaBytes, publicQuotaBytes）
    - 操作类型常量（postOpTypeCreate/Update/Delete）
    - Lamport 相关常量（lamportSchemaV2, authScopeUser）
    - 实体类型常量（entityTypePost/Comment）
    - 投票状态常量（voteStateNone/Up/Down）
    - 默认值常量（defaultOpNonceBytes）
    - 文件顶部添加用途注释
    - _需求: 1.3_

  - [x] 1.4 从 `p2p.go` 中提取消息类型常量（messageType* 系列）和速率限制默认值到 `constants.go`
    - _需求: 1.4_

  - [x] 1.5 从 `notifications.go` 中提取通知类型常量（NotifType* 系列）到 `constants.go`
    - _需求: 1.5_

  - [x] 1.6 验证编译和测试通过
    - 运行 `go build ./...` 确保编译通过
    - 运行 `go test ./...` 确保所有现有测试通过
    - _需求: 1.6, 14.1, 14.2_

- [x] 2. 阶段一检查点 — 提交推送
  - 确保所有测试通过，ask the user if questions arise
  - 执行 `git add -A && git commit -m "refactor: 阶段一 — 提取类型和常量到 types.go 和 constants.go" && git push`

### 阶段二：Go 数据层拆分（需求 2, 5）

- [-] 3. 拆分 db.go — Schema 和帖子
  - [x] 3.1 创建 `db_schema.go`，提取 `initDatabase` 和 `ensureSchema` 中的全部 CREATE TABLE 语句
    - 文件顶部添加用途注释
    - _需求: 2.1_

  - [x] 3.2 创建 `db_posts.go`，提取帖子相关操作
    - 包括 insertMessage, AddLocalPostStructured*, GetFeed, GetFeedBySub*, GetFeedIndexBySubSorted, GetPostIndexByID, GetMyPosts, GetPostsByAuthor, GetPrivateFeed, UpdateLocalPost, deleteLocalPostAsAuthor, postExists, SearchPosts, AddLocalPostWithImageToSub, queryPostsBySubSet, queryRecommendedPosts
    - _需求: 2.2_

  - [ ] 3.3 创建 `db_comments.go`，提取评论相关操作
    - 包括 insertComment, AddLocalComment*, GetCommentsByPost, UpdateLocalComment, deleteLocalCommentAsAuthor, upsertCommentTombstone
    - _需求: 2.3_

  - [ ] 3.4 验证编译通过
    - 运行 `go build ./...`
    - _需求: 14.1_

- [ ] 4. 拆分 db.go — 投票、收藏、治理
  - [ ] 4.1 创建 `db_votes.go`，提取投票状态机逻辑
    - 包括 applyPostVoteState, applyCommentVoteState, applyPostUpvote/Downvote, applyCommentUpvote/Downvote, currentPostVoteStateTx, currentCommentVoteStateTx, getPostVoteState, getCommentVoteState, voteDelta, UpvotePost, DownvotePost, UpvoteComment, DownvoteComment
    - _需求: 2.4_

  - [ ] 4.2 创建 `db_favorites.go`，提取收藏逻辑和收藏辅助函数
    - 包括 AddFavorite, RemoveFavorite, IsFavorited, GetFavoritePostIDs, GetFavorites, isFavoritedByPubkey, buildLocalFavoriteOperation, applyFavoriteOperation, verifyFavoriteOperationSignature, emitFavoritesUpdated
    - 同时将收藏辅助函数（normalizeFavoriteOperation, favoriteStateForOperation, favoriteOperationWins, buildFavoriteSignaturePayload, encodeFavoriteCursor, decodeFavoriteCursor, encodeMyPostsCursor, decodeMyPostsCursor）移入
    - _需求: 2.5, 5.4_

  - [ ] 4.3 创建 `db_governance.go`，提取治理和审核逻辑
    - 包括 ApplyShadowBan, ApplyUnban, AddTrustedAdmin, GetTrustedAdmins, isTrustedAdmin, upsertModeration, isShadowBanned, getModerationSnapshot, shouldAcceptPublicContent, GetModerationState, getLatestModerationTimestamp, listModerationSince, GetModerationLogs, insertModerationLogIfAbsent, getLatestAppliedModerationLogTimestamp, listAppliedModerationLogsSince
    - _需求: 2.6_

  - [ ] 4.4 验证编译通过
    - 运行 `go build ./...`
    - _需求: 14.1_

- [ ] 5. 拆分 db.go — 身份、Sub、Blob
  - [ ] 5.1 创建 `db_identity.go`，提取身份和个人资料逻辑
    - 包括 UpdateProfile, UpdateProfileDetails, GetProfile, GetProfileDetails, upsertProfile, saveLocalIdentity, getLocalIdentity, GetIdentityState
    - _需求: 2.7_

  - [ ] 5.2 创建 `db_subs.go`，提取 Sub 管理逻辑
    - 包括 CreateSub, GetSubs, SubscribeSub, UnsubscribeSub, GetSubscribedSubs, SearchSubs, upsertSub, GetSubStats, isSubSubscribed, listSubscribedSubIDs, emitSubscribedSubUpdate
    - _需求: 2.8_

  - [ ] 5.3 创建 `db_blobs.go`，提取内容和媒体 Blob 存储逻辑
    - 包括 GetPostBodyByCID, GetPostBodyByID, GetMediaByCID, GetPostMediaByID, getMediaBlobRawLocal, getMediaBlobLocal, upsertMediaBlobRaw, getContentBlobLocal, canServeContentBlobToNetwork, canServeMediaBlobToNetwork, upsertContentBlob, hasContentBlobLocal, hasMediaBlobLocal, StoreCommentImageDataURL
    - _需求: 2.9_

  - [ ] 5.4 验证编译通过
    - 运行 `go build ./...`
    - _需求: 14.1_

- [ ] 6. 拆分 db.go — 同步、操作日志、设置、图片处理、辅助函数
  - [ ] 6.1 创建 `db_sync.go`，提取同步摘要查询逻辑
    - 包括 listRecentPublicPostDigests, getLatestPublicPostTimestamp, getLatestPublicCommentTimestamp, listPublicPostDigestsSince, listPublicCommentDigestsSince, listPublicCommentDigestsByPostSince, upsertPublicPostIndexFromDigest, getLatestFavoriteOpTimestamp, listFavoriteOpsSince
    - _需求: 2.11_

  - [ ] 6.2 创建 `db_operations.go`，提取 Lamport 时钟和实体操作日志逻辑
    - 包括 nextLamport, observeLamport, normalizeIncomingLamport, compareLamportVersion, generateOperationID, fallbackOperationID, resolveOperationID, resolveCurrentVersion, appendEntityOperationTx, appendEntityOperation, ListEntityOps, RunTombstoneGC
    - 同时将归一化函数（normalizeOperationType, normalizeVoteState, normalizeAuthScope）移入
    - _需求: 2.12, 5.3_

  - [ ] 6.3 创建 `db_settings.go`，提取配额、隐私、治理策略
    - 包括 ensureZoneQuota, ensureBlobQuotaWithLRU, GetStorageUsage, GetPrivacySettings, SetPrivacySettings, GetGovernancePolicy, SetGovernancePolicy
    - _需求: 2.13_

  - [ ] 6.4 创建 `media.go`，提取图片处理逻辑（非数据库层）
    - 包括 prepareImageAssets, normalizedImageMIME, encodeImageForStorage, resizeImageIfNeeded, hasTransparency
    - _需求: 2.10_

  - [ ] 6.5 创建 `helpers.go`，集中通用辅助函数
    - 包括 normalizeSubID, deriveTitle, deriveBodyPreview, buildMessageID, buildContentCID, buildBinaryCID, normalizeCommentAttachments, encodeCommentAttachmentsJSON, decodeCommentAttachmentsJSON, mediaCIDsFromAttachments, computeHotScore, normalizeFeedStreamLimit, normalizeFeedStreamAlgorithm, scoreFeedRecommendation, countFeedItemsByReason, normalizeSearchLimit, normalizeFavoriteLimit, normalizeMyPostsLimit, normalizeFeedSortMode, topWindowStartUnix
    - _需求: 5.1, 5.2_

  - [ ] 6.6 确保 `db.go` 仅保留通用数据库辅助函数
    - 保留 makeSQLPlaceholders, queryForumMessages, ResetLocalTestData, isDevModeEnabled, IsDevMode, marshalOperationPayload
    - 删除已迁移到其他文件的函数
    - _需求: 2.14_

  - [ ] 6.7 验证编译和全部测试通过
    - 运行 `go build ./...` 确保编译通过
    - 运行 `go test ./...` 确保所有现有测试通过
    - 确保无重复的辅助函数定义
    - _需求: 5.5, 14.1, 14.2_

  - [ ]* 6.8 编写属性测试验证函数放置正确性和 Wails 绑定签名
    - **Property 1: 函数/类型放置正确性** — 通过 go/ast 解析验证每个函数恰好存在于一个文件中
    - **Property 3: Wails 绑定 API 签名保持不变** — 通过反射验证 App 公开方法签名与基线一致
    - **验证: 需求 1.1, 1.2, 1.3, 2.1-2.14, 14.3**

- [ ] 7. 阶段二检查点 — 提交推送
  - 确保所有测试通过，ask the user if questions arise
  - 执行 `git add -A && git commit -m "refactor: 阶段二 — 按领域拆分 db.go 和提取辅助函数" && git push`

### 阶段三：Go P2P 层和业务层拆分（需求 3, 4）

- [ ] 8. 拆分 p2p.go — 发布和消费
  - [ ] 8.1 创建 `p2p_publish.go`，提取全部 20+ 个消息发布方法
    - 包括 PublishPostStructured*, PublishComment*, PublishDeletePost, PublishDeleteComment, PublishPostUpvote, PublishPostDownvote, PublishCommentUpvote, PublishCommentDownvote, PublishProfileUpdate, PublishShadowBan, PublishUnban, PublishGovernancePolicy, PublishCreateSub, PublishPostUpdate, PublishCommentUpdate, publishFavoriteOperation, publishLocalProfileUpdateLocked, publishGovernanceMessage, publishPayloadAsync, signAndQueueOutgoingMessage
    - _需求: 3.2_

  - [ ] 8.2 创建 `p2p_consume.go`，提取消息消费和路由逻辑
    - 提取 consumeP2PMessages 及其内部的消息类型分发
    - 将 ProcessIncomingMessage（450+ 行巨型 switch-case）从 `db.go` 移入 `p2p_consume.go`
    - _需求: 3.3, 3.4_

  - [ ] 8.3 验证编译通过
    - 运行 `go build ./...`
    - _需求: 14.1_

- [ ] 9. 拆分 p2p.go — 同步、获取、投票、速率限制
  - [ ] 9.1 创建 `p2p_sync.go`，提取反熵同步协议
    - 包括 runAntiEntropySyncWorker, publishSyncSummaryRequest, publishCommentSyncRequest, publishGovernanceSyncRequest, publishFavoriteSyncRequest, handle*Sync*Request/Response, TriggerAntiEntropySyncNow, TriggerCommentSyncNow, updateAntiEntropyStats 及所有 resolve*AntiEntropy* 配置函数
    - _需求: 3.5_

  - [ ] 9.2 创建 `p2p_fetch.go`，提取内容/媒体获取协议
    - 包括 fetchContentBlobFromNetwork, fetchMediaBlobFromNetwork, handleContentFetchRequest/Response, handleMediaFetchRequest/Response, publishContentFetchNotFound, publishMediaFetchNotFound 及 fetch 速率限制函数
    - _需求: 3.6_

  - [ ] 9.3 创建 `p2p_votes.go`，提取投票广播去重逻辑
    - 包括 scheduleVoteStateBroadcast, resolveVoteBroadcastDebounce
    - _需求: 3.7_

  - [ ] 9.4 创建 `p2p_ratelimit.go`，提取速率限制和 Peer 策略逻辑
    - 包括 allowIncomingMessage, allowFetchRequest, refreshPeerPoliciesFromEnv, markPeerGreylisted, isPeerBlocked, parsePeerIDSet 及 resolve*RateLimit* 函数
    - _需求: 3.8_

  - [ ] 9.5 将网络地址解析逻辑合并到 `p2p_config.go`
    - 包括 resolveP2PListenAddrs, resolveP2PAnnounceAddrs, resolveRelayPeerInfos, resolveRelayPeers, resolveMaxConnectedPeers, resolveGreylistTTLSeconds, resolveRelayServiceEnabled
    - _需求: 3.9_

  - [ ] 9.6 精简 `p2p.go`，仅保留 P2P 入口函数
    - 保留 StartP2P, StopP2P, startP2POnPortLocked, ConnectPeer, connectPeerLocked, connectBootstrapPeersAsync, GetP2PStatus, getP2PStatusLocked, mdnsNotifee.HandlePeerFound
    - _需求: 3.1_

  - [ ] 9.7 验证编译通过
    - 运行 `go build ./...`
    - _需求: 14.1_

- [ ] 10. App 结构体瘦身（需求 4）
  - [ ] 10.1 创建 `identity.go`，从 `app.go` 提取身份管理逻辑
    - 包括 GenerateIdentity, LoadSavedIdentity, ImportIdentityFromMnemonic, SignMessage, VerifyMessage, deriveKeypairFromMnemonic
    - _需求: 4.1_

  - [ ] 10.2 创建 `feed.go`，从 `app.go` 提取 Feed 流逻辑
    - 包括 GetFeedStream, GetFeedStreamWithStrategy, queryRecommendedCandidates
    - _需求: 4.2_

  - [ ] 10.3 将 P2P 自动启动配置解析逻辑移入 `p2p_config.go`
    - 包括 shouldAutoStartP2P, resolveAutoStartP2PPort, resolveBootstrapPeers, resolveAutoStartPortCandidates, isTCPPortAvailable
    - _需求: 4.3_

  - [ ] 10.4 精简 `app.go`，为 App 结构体字段添加分组注释
    - 保留 App 结构体定义、NewApp、startup、shutdown、SetDatabasePath、GetAntiEntropyStats
    - 按设计文档添加分组注释（数据库、P2P 网络、速率限制、内容获取、可观测性、告警、投票广播、消息发送队列）
    - _需求: 4.4, 4.5_

  - [ ] 10.5 验证编译和全部测试通过
    - 运行 `go build ./...` 确保编译通过
    - 运行 `go test ./...` 确保所有现有测试通过
    - _需求: 14.1, 14.2_

  - [ ]* 10.6 编写属性测试验证文件大小限制
    - **Property 5: 文件大小限制** — 验证所有 Go 文件不超过 500 行（db_schema.go 可放宽到 800 行）
    - **验证: 需求 15.1**

- [ ] 11. 阶段三检查点 — 提交推送
  - 确保所有测试通过，ask the user if questions arise
  - 执行 `git add -A && git commit -m "refactor: 阶段三 — 拆分 p2p.go 和 App 结构体瘦身" && git push`

### 阶段四：React 前端重构（需求 6-13）

#### 4a: API 封装层（需求 13）

- [ ] 12. 创建前端 API 封装层
  - [ ] 12.1 创建 `src/api/identity.ts`，封装身份相关 Wails 绑定
    - 包括 LoadSavedIdentity, GenerateIdentity, ImportIdentityFromMnemonic, SignMessage, VerifyMessage
    - 添加统一错误处理和 TypeScript 类型标注
    - _需求: 13.1_

  - [ ] 12.2 创建 `src/api/posts.ts`，封装帖子相关 Wails 绑定
    - 包括 GetFeedStream, GetFeedIndexBySubSorted, GetPostIndexByID, GetPostBodyByID, GetMyPosts, GetPostsByAuthor, PublishPostStructuredToSub, PublishPostWithImageToSub, PublishPostUpdate, PublishDeletePost, SearchPosts
    - _需求: 13.1_

  - [ ] 12.3 创建 `src/api/comments.ts`，封装评论相关 Wails 绑定
    - 包括 GetCommentsByPost, PublishCommentWithAttachments, PublishCommentUpdate, PublishDeleteComment, PublishCommentUpvote, PublishCommentDownvote, TriggerCommentSyncNow
    - _需求: 13.1_

  - [ ] 12.4 创建 `src/api/subs.ts`，封装 Sub 相关 Wails 绑定
    - 包括 GetSubs, CreateSub, PublishCreateSub, SubscribeSub, UnsubscribeSub, GetSubscribedSubs, SearchSubs, GetSubStats
    - _需求: 13.1_

  - [ ] 12.5 创建 `src/api/network.ts`、`src/api/governance.ts`、`src/api/profile.ts`、`src/api/favorites.ts`、`src/api/notifications.ts`
    - 按设计文档封装各领域的 Wails 绑定
    - _需求: 13.1_

  - [ ] 12.6 创建 `src/api/index.ts` 统一导出
    - _需求: 13.1_

  - [ ] 12.7 验证 TypeScript 编译通过
    - 运行 `npx tsc --noEmit`
    - _需求: 13.2, 14.6_

#### 4b: 共享工具函数和组件（需求 11）

- [ ] 13. 提取共享工具函数
  - [ ] 13.1 创建 `src/lib/time.ts`，统一 `formatTimeAgo` 和 `formatCreatedAt`
    - 从 PostDetail.tsx、CommentTree.tsx、PostCard.tsx 中提取 formatTimeAgo
    - 从 RightPanel.tsx 中提取 formatCreatedAt
    - 更新所有引用组件的 import
    - _需求: 11.1, 11.3_

  - [ ] 13.2 创建 `src/lib/string.ts`，统一 `getInitials`
    - 从 PostDetail.tsx、CommentTree.tsx、Header.tsx 中提取 getInitials
    - 更新所有引用组件的 import
    - _需求: 11.2_

  - [ ] 13.3 创建 `src/lib/richText.ts`，统一 `linkifyAndMarkdown`/`renderRichText` 和 `buildQuotedReply`
    - 从 PostDetail.tsx 和 CommentTree.tsx 中提取并统一为单一实现
    - _需求: 11.7_

  - [ ] 13.4 创建 `src/lib/mappers.ts`，提取 `mapPostIndexToPost` 和 `mapForumMessageToPost`
    - 从 App.tsx 中提取
    - _需求: 8.3_

  - [ ] 13.5 创建 `src/lib/errors.ts`，提取 `getErrorMessage`
    - 从 App.tsx 中提取
    - _需求: 8.4_

  - [ ] 13.6 创建 `src/lib/routing.ts`，提取 `buildAppHash`、`buildShareLink`、`buildSubShareLink`
    - 从 App.tsx 中提取
    - _需求: 8.5_

  - [ ] 13.7 创建 `src/lib/syncFeedback.ts`，提取 `shouldTrackPendingSync` 和 `getWriteFeedback`
    - 从 App.tsx 中提取
    - _需求: 8.6_

  - [ ] 13.8 创建 `src/lib/imageUtils.ts`，提取 SettingsPanel 中的图片压缩逻辑
    - 提取 compressAvatarFileIfNeeded 及相关函数
    - _需求: 9.8_

  - [ ] 13.9 验证 TypeScript 编译通过
    - 运行 `npx tsc --noEmit`
    - _需求: 14.6_

- [ ] 14. 提取共享 UI 组件
  - [ ] 14.1 创建 `src/components/shared/Avatar.tsx`
    - 统一头像渲染逻辑（avatarUrl 判断 + initials fallback）
    - 替换 PostCard.tsx、PostDetail.tsx、CommentTree.tsx、Header.tsx 中的重复实现
    - _需求: 11.4_

  - [ ] 14.2 创建 `src/components/shared/ConfirmDialog.tsx`
    - 统一删除确认对话框模式
    - 替换 PostDetail 和 CommentTree 中的重复实现
    - _需求: 11.5_

  - [ ] 14.3 创建 `src/components/shared/ImagePreview.tsx`
    - 统一图片预览 Lightbox 模式
    - 替换 PostDetail 和 CommentTree 中的重复实现
    - _需求: 11.6_

  - [ ] 14.4 验证 TypeScript 编译通过，确认无重复函数定义
    - 运行 `npx tsc --noEmit`
    - 确认 formatTimeAgo、getInitials、renderRichText 各只有一份定义
    - _需求: 11.8, 14.6_

  - [ ]* 14.5 编写属性测试验证无重复定义
    - **Property 2: 无重复定义** — 扫描所有 .ts/.tsx 文件验证 formatTimeAgo, getInitials, renderRichText 各只出现一次
    - **验证: 需求 11.8**

#### 4c: Context 状态管理（需求 6）

- [ ] 15. 引入 Context 状态管理层
  - [ ] 15.1 创建 `src/contexts/AuthContext.tsx`
    - 管理 identity, profile, isAdmin, identityChecked 状态
    - 提供 loadIdentity, createIdentity, importIdentity 操作
    - 使用 useReducer 管理复杂状态，useMemo 优化 value
    - 通过 `src/api/identity.ts` 调用后端
    - _需求: 6.2_

  - [ ] 15.2 创建 `src/contexts/NetworkContext.tsx`
    - 管理 p2pStatus, antiEntropyStats, onlineCount, networkBusy, networkHealth 状态
    - 提供 loadNetworkHealth, triggerSyncNow 操作
    - 订阅 EventsOn(`p2p:updated`) 事件
    - 通过 `src/api/network.ts` 调用后端
    - _需求: 6.3_

  - [ ] 15.3 创建 `src/contexts/UIContext.tsx`
    - 管理 isDark, view, currentSubId, sortMode, showLoginModal, showCreateSubModal, showCreatePostModal, showSettingsPanel, consistencyFocus, viewSyncToken 状态
    - 提供 setView, toggleDark, openModal, closeModal 操作
    - _需求: 6.4_

  - [ ] 15.4 创建 `src/contexts/FeedContext.tsx`
    - 管理 posts, profiles, favoritePostIds, subscribedSubs, subscribedSubIds, subs, unreadSubs, currentSubStats 状态
    - 订阅 EventsOn(`feed:updated`, `favorites:updated`) 事件
    - 通过 `src/api/posts.ts`、`src/api/subs.ts`、`src/api/favorites.ts` 调用后端
    - _需求: 6.5_

  - [ ] 15.5 创建 `src/contexts/NotificationContext.tsx`
    - 管理 unreadNotificationCount, pendingSyncActions 状态
    - 订阅 EventsOn(`notifications:updated`) 事件
    - 通过 `src/api/notifications.ts` 调用后端
    - _需求: 6.6_

  - [ ] 15.6 创建 `src/contexts/index.ts` 统一导出
    - _需求: 6.1_

  - [ ] 15.7 验证 TypeScript 编译通过
    - 运行 `npx tsc --noEmit`
    - _需求: 14.6_

#### 4d: 自定义 Hook（需求 7）

- [ ] 16. 提取自定义 Hook
  - [ ] 16.1 创建 `src/hooks/usePosts.ts`
    - 提取帖子操作逻辑：loadPosts, handleCreatePost, handleUpvote, handleDownvote, handleToggleFavorite, handleSharePost, loadFavorites
    - 依赖 FeedContext, AuthContext
    - 通过 api 层调用后端
    - _需求: 7.1_

  - [ ] 16.2 创建 `src/hooks/usePostDetail.ts`
    - 提取帖子详情逻辑：loadPostDetail, handlePostClick, handleBackToFeed, refreshCommentsForSelectedPost, handleRefreshSelectedPost
    - 管理 selectedPost/postBody/postComments 状态
    - _需求: 7.2_

  - [ ] 16.3 创建 `src/hooks/useSubs.ts`
    - 提取 Sub 管理逻辑：loadSubs, loadSubscribedSubs, handleCreateSub, handleToggleSubscription, loadSubStats
    - _需求: 7.3_

  - [ ] 16.4 创建 `src/hooks/useSearch.ts`
    - 提取搜索逻辑：handleSearch, searchResults/searchQuery/searchScopeSubId 状态管理
    - _需求: 7.4_

  - [ ] 16.5 创建 `src/hooks/useGovernance.ts`
    - 提取治理数据加载逻辑：loadGovernanceData, governanceAdmins/moderationStates/moderationLogs 状态管理
    - _需求: 7.5_

  - [ ] 16.6 创建 `src/hooks/usePendingSync.ts`
    - 提取待同步操作逻辑：trackPendingSyncAction, handleDismissPendingSyncAction, reconcilePendingSyncActions
    - _需求: 7.6_

  - [ ] 16.7 创建 `src/hooks/useProfileView.ts`
    - 提取用户资料查看逻辑：selectedProfile/selectedProfilePosts 状态管理和加载逻辑
    - _需求: 7.7_

  - [ ] 16.8 创建 `src/hooks/useCommentDraft.ts`
    - 提取评论草稿管理逻辑：getCommentDraftStorageKey, loadCommentDraft, saveCommentDraft, clearCommentDraft, parseCommentDraft, canUseLocalStorage
    - _需求: 10.5_

  - [ ] 16.9 创建 `src/hooks/index.ts` 统一导出
    - _需求: 7.1-7.7_

  - [ ] 16.10 验证 TypeScript 编译通过
    - 运行 `npx tsc --noEmit`
    - _需求: 14.6_

#### 4e: SettingsPanel 和 PostDetail 拆分（需求 9, 10）

- [ ] 17. 拆分 SettingsPanel
  - [ ] 17.1 创建 `src/components/settings/AccountTab.tsx`
    - 提取个人资料编辑、头像上传逻辑
    - 使用 AuthContext 获取身份状态，通过 api 层调用后端
    - _需求: 9.1_

  - [ ] 17.2 创建 `src/components/settings/PrivacyTab.tsx`
    - 提取助记词显示、隐私设置逻辑
    - _需求: 9.2_

  - [ ] 17.3 创建 `src/components/settings/NetworkTab.tsx`
    - 提取 P2P 状态、连接管理、端口配置逻辑
    - 使用 NetworkContext 获取网络状态
    - _需求: 9.3_

  - [ ] 17.4 创建 `src/components/settings/UpdatesTab.tsx`
    - 提取版本检查、更新历史逻辑
    - _需求: 9.4_

  - [ ] 17.5 创建 `src/components/settings/ConsistencyTab.tsx`
    - 提取 Lamport 操作日志、Tombstone GC 逻辑
    - _需求: 9.5_

  - [ ] 17.6 创建 `src/components/settings/GovernanceTab.tsx`
    - 提取 Shadow Ban 管理、管理员管理、审核日志逻辑
    - 使用 useGovernance Hook
    - _需求: 9.6_

  - [ ] 17.7 精简 `SettingsPanel.tsx`，仅保留 Tab 导航框架和布局
    - 确保行数减少到 150 行以内
    - _需求: 9.7_

  - [ ] 17.8 验证 TypeScript 编译通过
    - 运行 `npx tsc --noEmit`
    - _需求: 14.6_

- [ ] 18. 拆分 PostDetail
  - [ ] 18.1 创建 `src/components/post/PostContent.tsx`
    - 提取标题、正文 Markdown 渲染、图片展示、链接卡片
    - 使用 lib/richText.ts 进行富文本渲染
    - _需求: 10.1_

  - [ ] 18.2 创建 `src/components/post/PostActions.tsx`
    - 提取投票按钮、评论数、分享、收藏、编辑/删除操作栏
    - 使用 shared/ConfirmDialog 进行删除确认
    - _需求: 10.2_

  - [ ] 18.3 创建 `src/components/post/PostEditor.tsx`
    - 提取编辑表单、保存/取消逻辑
    - _需求: 10.3_

  - [ ] 18.4 创建 `src/components/post/CommentComposer.tsx`
    - 提取草稿自动保存、图片附件上传、代码块插入、引用回复
    - 使用 useCommentDraft Hook 管理草稿
    - _需求: 10.4_

  - [ ] 18.5 精简 `PostDetail.tsx`，仅作为布局容器
    - 确保行数减少到 200 行以内
    - _需求: 10.7_

  - [ ] 18.6 验证 TypeScript 编译通过
    - 运行 `npx tsc --noEmit`
    - _需求: 14.6_

#### 4f: App.tsx 瘦身与路由重构（需求 8）

- [ ] 19. App.tsx 瘦身
  - [ ] 19.1 将 Wails 事件监听逻辑迁移到对应的 Context 中
    - EventsOn(`p2p:updated`) → NetworkContext
    - EventsOn(`feed:updated`, `favorites:updated`) → FeedContext
    - EventsOn(`notifications:updated`) → NotificationContext
    - _需求: 8.2_

  - [ ] 19.2 引入基于 hash 的轻量路由系统
    - 创建 `src/hooks/useHashRouter.ts` 自定义 Hook
    - 替代当前的 view 状态变量和 ViewMode 类型手动切换
    - 基于现有的 buildAppHash 模式
    - _需求: 8.1_

  - [ ] 19.3 重构 App.tsx 为 Context Provider 组装 + 布局 + 路由
    - 将 App.tsx 中的 useState/useCallback 替换为 Context 和 Hook 调用
    - 组装所有 Context Provider
    - 使用 useHashRouter 进行视图切换
    - 确保组件不再需要超过 3 层的 props 传递
    - _需求: 6.7, 8.7_

  - [ ] 19.4 更新所有组件的 import，通过 Context/Hook 获取状态和操作
    - 替换 props 传递为 useContext 调用
    - 确保通过 api 层调用后端，不直接导入 wailsjs/go/main/App
    - _需求: 13.3_

  - [ ] 19.5 验证 TypeScript 编译通过，确认 App.tsx 行数 ≤ 300
    - 运行 `npx tsc --noEmit`
    - _需求: 7.8, 14.6_

  - [ ]* 19.6 编写属性测试验证前端组件不直接导入 Wails 绑定
    - **Property 7: 前端组件不直接导入 Wails 绑定** — 扫描 src/components/, src/hooks/, src/contexts/ 验证无 wailsjs/go/main/App 导入
    - **验证: 需求 13.3**

#### 4g: 目录结构重组（需求 12）

- [ ] 20. 前端目录结构重组
  - [ ] 20.1 创建目录结构并移动布局组件
    - 移动 Header.tsx → src/components/layout/Header.tsx
    - 移动 Sidebar.tsx → src/components/layout/Sidebar.tsx
    - 移动 RightPanel.tsx → src/components/layout/RightPanel.tsx
    - 创建 src/components/layout/index.ts
    - _需求: 12.1_

  - [ ] 20.2 移动 Feed 相关组件
    - 移动 Feed.tsx → src/components/feed/Feed.tsx
    - 移动 PostCard.tsx → src/components/feed/PostCard.tsx
    - 创建 src/components/feed/index.ts
    - _需求: 12.1_

  - [ ] 20.3 移动帖子详情相关组件到 src/components/post/
    - 确保 PostDetail.tsx, PostContent.tsx, PostActions.tsx, PostEditor.tsx, CommentComposer.tsx, CommentTree.tsx 都在 post/ 目录下
    - 创建 src/components/post/index.ts
    - _需求: 12.1_

  - [ ] 20.4 移动模态框组件
    - 移动 CreatePostModal.tsx → src/components/modals/CreatePostModal.tsx
    - 移动 CreateSubModal.tsx → src/components/modals/CreateSubModal.tsx
    - 移动 LoginModal.tsx → src/components/modals/LoginModal.tsx
    - 创建 src/components/modals/index.ts
    - _需求: 12.1_

  - [ ] 20.5 移动独立视图组件到 src/components/views/
    - 移动 DiscoverView.tsx, ProfileView.tsx, MyPosts.tsx, Favorites.tsx, DraftsView.tsx, HistoryView.tsx, SearchResultsView.tsx, PendingSyncView.tsx, NotificationsView.tsx, UserMenu.tsx
    - 创建 src/components/views/index.ts
    - _需求: 12.1_

  - [ ] 20.6 确保 shared/ 和 settings/ 目录的 index.ts 导出文件完整
    - 创建 src/components/shared/index.ts
    - 创建 src/components/settings/index.ts
    - _需求: 12.3_

  - [ ] 20.7 更新所有 import 路径
    - 确保所有组件间的 import 路径在重组后正确
    - _需求: 12.2_

  - [ ] 20.8 验证 TypeScript 编译通过
    - 运行 `npx tsc --noEmit`
    - _需求: 14.6_

  - [ ]* 20.9 编写属性测试验证文件命名规范和文件大小限制
    - **Property 5: 文件大小限制** — 验证所有 React 组件文件不超过 300 行
    - **Property 6: 文件命名规范** — 验证组件文件 PascalCase、Hook 文件 camelCase、工具函数文件 camelCase
    - **验证: 需求 15.1, 15.2, 15.5**

- [ ] 21. 阶段四检查点 — 提交推送
  - 确保所有测试通过，ask the user if questions arise
  - 执行 `git add -A && git commit -m "refactor: 阶段四 — React 前端重构（API封装/Context/Hook/组件拆分/目录重组）" && git push`

### 最终验证

- [ ] 22. 全面验证和最终检查
  - [ ] 22.1 运行 Go 后端全部验证
    - `go build ./...` 编译通过
    - `go test ./...` 全部测试通过
    - 确认 Wails 绑定公开方法签名未变更
    - _需求: 14.1, 14.2, 14.3_

  - [ ] 22.2 运行前端全部验证
    - `npx tsc --noEmit` 编译通过
    - 确认无未使用的导入、变量或函数
    - _需求: 14.6, 15.6_

  - [ ] 22.3 验证已有良好设计保持不变
    - 确认 recommendation.go、authenticated_messages.go、message_outbox.go 内容未变（或仅有 import 调整）
    - 确认前端 lib/drafts.ts、lib/history.ts、lib/networkHealth.ts、lib/pendingSync.ts、lib/postContent.ts、types/index.ts 内容未变
    - _需求: 15.7_

  - [ ] 22.4 验证文件用途注释
    - 确认所有新创建或修改的源文件顶部包含一行简要用途注释
    - _需求: 15.3_

  - [ ]* 22.5 运行全部属性测试
    - **Property 1-9** 全部通过
    - **验证: 需求 1-15 全覆盖**

- [ ] 23. 最终提交推送
  - 执行 `git add -A && git commit -m "refactor: 最终验证通过 — Aegis-App 模块化重构完成" && git push`

## 备注

- 标记 `*` 的任务为可选任务，可跳过以加速 MVP
- 每个任务引用了具体的需求编号以确保可追溯性
- 检查点任务确保增量验证
- 属性测试验证跨所有输入的通用正确性属性
- 单元测试验证具体示例和边界情况
- 每个阶段结束后执行 git commit + push，确保增量交付
