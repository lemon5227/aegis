# 需求文档：Aegis-App 架构级模块化重构

## 简介

Aegis-App 是一个基于 Wails (Go + React/TypeScript) 的去中心化论坛桌面应用。经过对全部源代码的深度审查，发现当前架构存在以下核心问题：

**架构层面的根本问题：**

1. **God Object 反模式**：`App` 结构体承载了 14 个 mutex、身份管理、Feed 流、P2P 网络、可观测性、告警等全部职责，是典型的 God Object。所有 Go 文件都在 `main` 包下，无法利用 Go 的包级封装来隔离关注点。
2. **数据层与业务层完全耦合**：`db.go`（7800+ 行）不仅包含 CRUD 操作，还混入了图片处理（`prepareImageAssets`、`resizeImageIfNeeded`）、Lamport 时钟逻辑、投票状态机、收藏签名验证、存储配额 LRU 淘汰等业务逻辑。数据访问和业务规则无法独立测试。
3. **P2P 层职责过载**：`p2p.go`（3300+ 行）混合了消息发布（20+ 个 Publish* 方法）、消息消费路由（`consumeP2PMessages` 中的巨型 switch）、反熵同步协议、内容/媒体获取协议、投票广播去重、速率限制、Peer 策略管理等。这些是完全不同的关注点。
4. **消息处理入口混乱**：`ProcessIncomingMessage`（450+ 行）是一个巨型 switch-case，处理 20+ 种消息类型，包含签名验证、Shadow Ban 检查、Lamport 时钟同步、各类业务逻辑分发。这是整个系统最脆弱的单点。
5. **前端 God Component**：`App.tsx`（1690+ 行）包含 40+ 个 useState、30+ 个 useCallback，所有状态通过 props 逐层传递。视图切换通过 `view` 状态变量手动管理，无路由系统。
6. **前端工具函数重复定义**：`formatTimeAgo` 在 PostDetail.tsx、CommentTree.tsx、PostCard.tsx 中各有一份；`getInitials` 在 PostDetail.tsx、CommentTree.tsx、Header.tsx 中各有一份。
7. **前端直接调用 Wails 绑定**：组件直接 import `wailsjs/go/main/App` 中的 50+ 个函数，无 API 封装层，导致后端接口变更时需要修改大量组件。

**已有的合理设计（应保留）：**
- `recommendation.go` 的策略模式设计良好，有清晰的接口和注册机制
- `authenticated_messages.go` 职责单一，专注消息签名/验签
- `message_outbox.go` 实现了可靠的消息发送队列，设计合理
- `p2p_config.go` 和 `p2p_peers.go` 已经从 p2p.go 中分离，方向正确
- 前端 `lib/` 目录下的工具模块（drafts.ts, history.ts, networkHealth.ts, pendingSync.ts, postContent.ts）划分合理
- 前端 `types/index.ts` 类型定义集中管理，结构清晰

## 术语表

- **App_Struct**: Go 后端的 `App` 结构体，当前承载所有业务逻辑和 14 个 mutex 的 God Object
- **DB_Layer**: 数据库访问层，当前集中在 `db.go` 单文件中（7800+ 行）
- **P2P_Layer**: P2P 网络层，当前集中在 `p2p.go`（3300+ 行）及 `p2p_config.go`、`p2p_peers.go`、`p2p_public_ip.go`
- **ProcessIncomingMessage**: 消息处理入口函数，450+ 行的巨型 switch-case，处理 20+ 种消息类型
- **Frontend_App**: React 前端主组件 `App.tsx`（1690+ 行），承载所有状态和路由逻辑
- **Wails_Binding**: Wails 框架的 Go-JS 绑定层，前端通过 `wailsjs/go/main/App` 调用后端方法
- **Lamport_Clock**: 分布式逻辑时钟，用于消息排序和冲突解决
- **Anti_Entropy**: 反熵同步协议，用于节点间数据一致性修复
- **Facade**: 门面模式，为 Wails 绑定提供统一的 API 入口

## 需求

### 需求 1：Go 后端数据模型与常量独立化

**用户故事：** 作为开发者，我希望所有数据类型定义和业务常量从 `db.go` 中提取到独立文件，以便快速定位数据模型、减少 `db.go` 的认知负担。

#### 验收标准

1. THE Refactoring_Tool SHALL 将 `db.go` 中全部 30+ 个 `type` 定义（ForumMessage, PostIndex, PostBodyBlob, MediaBlob, Sub, SubStats, Profile, ProfileDetails, Comment, CommentAttachment, ModerationState, ModerationLog, GovernancePolicy, P2PConfig, PrivacySettings, IdentityState, StorageUsage, FeedStreamItem, FeedStream, PostIndexPage, FavoriteOpRecord, EntityOpRecord, TombstoneGCResult, GovernanceAdmin, SyncPostDigest, SyncCommentDigest, IncomingMessage, KnownPeerExchange, LamportVersion 等）提取到 `types.go` 文件中
2. THE Refactoring_Tool SHALL 将 `app.go` 中的 Identity 和 AntiEntropyStats 类型定义也移入 `types.go`，使所有数据模型集中管理
3. THE Refactoring_Tool SHALL 将所有业务常量（存储配额常量 totalQuotaBytes/privateQuotaBytes/publicQuotaBytes、操作类型常量 postOpTypeCreate/postOpTypeUpdate/postOpTypeDelete、Lamport 相关常量 lamportSchemaV2/authScopeUser、实体类型常量 entityTypePost/entityTypeComment、投票状态常量 voteStateNone/voteStateUp/voteStateDown、默认值常量 defaultOpNonceBytes）提取到 `constants.go` 文件中
4. THE Refactoring_Tool SHALL 将 `p2p.go` 中散落的消息类型常量（messageType* 系列）和速率限制常量也移入 `constants.go`
5. THE Refactoring_Tool SHALL 将 `notifications.go` 中的通知类型常量（NotifType* 系列）也移入 `constants.go`
6. THE Refactoring_Tool SHALL 确保提取后所有现有代码的编译和测试通过，无功能回归
7. IF 提取过程中发现循环依赖，THEN THE Refactoring_Tool SHALL 通过接口抽象或调整分组来解决依赖问题

### 需求 2：Go 后端数据库层按领域拆分

**用户故事：** 作为开发者，我希望 `db.go` 的 7800+ 行代码按业务领域拆分为职责单一的文件，每个文件聚焦一个数据领域。

#### 验收标准

1. THE Refactoring_Tool SHALL 将数据库初始化（`initDatabase`）和全部 Schema DDL（`ensureSchema` 中的 CREATE TABLE 语句）提取到 `db_schema.go` 文件中
2. THE Refactoring_Tool SHALL 将帖子相关操作（`insertMessage`, `AddLocalPostStructured*`, `GetFeed`, `GetFeedBySub*`, `GetFeedIndexBySubSorted`, `GetPostIndexByID`, `GetMyPosts`, `GetPostsByAuthor`, `GetPrivateFeed`, `UpdateLocalPost`, `deleteLocalPostAsAuthor`, `postExists`, `SearchPosts`）提取到 `db_posts.go` 文件中
3. THE Refactoring_Tool SHALL 将评论相关操作（`insertComment`, `AddLocalComment*`, `GetCommentsByPost`, `UpdateLocalComment`, `deleteLocalCommentAsAuthor`, `upsertCommentTombstone`）提取到 `db_comments.go` 文件中
4. THE Refactoring_Tool SHALL 将投票状态机逻辑（`applyPostVoteState`, `applyCommentVoteState`, `applyPostUpvote`, `applyPostDownvote`, `applyCommentUpvote`, `applyCommentDownvote`, `currentPostVoteStateTx`, `currentCommentVoteStateTx`, `getPostVoteState`, `getCommentVoteState`, `voteDelta`, `UpvotePost`, `DownvotePost`, `UpvoteComment`, `DownvoteComment`）提取到 `db_votes.go` 文件中
5. THE Refactoring_Tool SHALL 将收藏逻辑（`AddFavorite`, `RemoveFavorite`, `IsFavorited`, `GetFavoritePostIDs`, `GetFavorites`, `isFavoritedByPubkey`, `buildLocalFavoriteOperation`, `applyFavoriteOperation`, `verifyFavoriteOperationSignature`, `emitFavoritesUpdated`, 以及收藏相关的 cursor 编解码和归一化函数）提取到 `db_favorites.go` 文件中
6. THE Refactoring_Tool SHALL 将治理相关逻辑（`ApplyShadowBan`, `ApplyUnban`, `AddTrustedAdmin`, `GetTrustedAdmins`, `isTrustedAdmin`, `upsertModeration`, `isShadowBanned`, `getModerationSnapshot`, `shouldAcceptPublicContent`, `GetModerationState`, `getLatestModerationTimestamp`, `listModerationSince`, `GetModerationLogs`, `insertModerationLogIfAbsent`, `getLatestAppliedModerationLogTimestamp`, `listAppliedModerationLogsSince`）提取到 `db_governance.go` 文件中
7. THE Refactoring_Tool SHALL 将用户身份和个人资料逻辑（`UpdateProfile`, `UpdateProfileDetails`, `GetProfile`, `GetProfileDetails`, `upsertProfile`, `saveLocalIdentity`, `getLocalIdentity`, `GetIdentityState`）提取到 `db_identity.go` 文件中
8. THE Refactoring_Tool SHALL 将 Sub 管理逻辑（`CreateSub`, `GetSubs`, `SubscribeSub`, `UnsubscribeSub`, `GetSubscribedSubs`, `SearchSubs`, `upsertSub`, `GetSubStats`, `isSubSubscribed`, `listSubscribedSubIDs`, `emitSubscribedSubUpdate`）提取到 `db_subs.go` 文件中
9. THE Refactoring_Tool SHALL 将内容 Blob 和媒体 Blob 存储逻辑（`GetPostBodyByCID`, `GetPostBodyByID`, `GetMediaByCID`, `GetPostMediaByID`, `getMediaBlobRawLocal`, `getMediaBlobLocal`, `upsertMediaBlobRaw`, `getContentBlobLocal`, `canServeContentBlobToNetwork`, `canServeMediaBlobToNetwork`, `upsertContentBlob`, `hasContentBlobLocal`, `hasMediaBlobLocal`, `StoreCommentImageDataURL`）提取到 `db_blobs.go` 文件中
10. THE Refactoring_Tool SHALL 将图片处理逻辑（`prepareImageAssets`, `normalizedImageMIME`, `encodeImageForStorage`, `resizeImageIfNeeded`, `hasTransparency`）提取到 `media.go` 文件中，因为图片处理不属于数据库层职责
11. THE Refactoring_Tool SHALL 将同步摘要查询逻辑（`listRecentPublicPostDigests`, `getLatestPublicPostTimestamp`, `getLatestPublicCommentTimestamp`, `listPublicPostDigestsSince`, `listPublicCommentDigestsSince`, `listPublicCommentDigestsByPostSince`, `upsertPublicPostIndexFromDigest`, `getLatestFavoriteOpTimestamp`, `listFavoriteOpsSince`）提取到 `db_sync.go` 文件中
12. THE Refactoring_Tool SHALL 将 Lamport 时钟逻辑（`nextLamport`, `observeLamport`, `normalizeIncomingLamport`, `compareLamportVersion`, `generateOperationID`, `fallbackOperationID`, `resolveOperationID`, `resolveCurrentVersion`）和实体操作日志逻辑（`appendEntityOperationTx`, `appendEntityOperation`, `ListEntityOps`, `RunTombstoneGC`）提取到 `db_operations.go` 文件中
13. THE Refactoring_Tool SHALL 将存储配额管理（`ensureZoneQuota`, `ensureBlobQuotaWithLRU`, `GetStorageUsage`）和隐私设置（`GetPrivacySettings`, `SetPrivacySettings`）和治理策略（`GetGovernancePolicy`, `SetGovernancePolicy`）提取到 `db_settings.go` 文件中
14. THE Refactoring_Tool SHALL 确保拆分后 `db.go` 仅保留通用数据库辅助函数（`makeSQLPlaceholders`, `queryForumMessages`, `ResetLocalTestData`, `isDevModeEnabled`, `IsDevMode`, `marshalOperationPayload`）
15. WHEN 所有拆分完成后，THE Refactoring_Tool SHALL 确保所有现有测试通过

### 需求 3：Go 后端 P2P 层按协议职责拆分

**用户故事：** 作为开发者，我希望 `p2p.go` 的 3300+ 行代码按协议职责拆分，使每个文件聚焦单一通信协议。

#### 验收标准

1. THE Refactoring_Tool SHALL 将 `p2p.go` 精简为 P2P 层入口，仅保留启动/停止（`StartP2P`, `StopP2P`, `startP2POnPortLocked`）、连接管理（`ConnectPeer`, `connectPeerLocked`, `connectBootstrapPeersAsync`）、状态查询（`GetP2PStatus`, `getP2PStatusLocked`）和 mDNS 发现（`mdnsNotifee`）
2. THE Refactoring_Tool SHALL 将全部 20+ 个消息发布方法（`PublishPostStructured*`, `PublishComment*`, `PublishDeletePost`, `PublishDeleteComment`, `PublishPostUpvote`, `PublishPostDownvote`, `PublishCommentUpvote`, `PublishCommentDownvote`, `PublishProfileUpdate`, `PublishShadowBan`, `PublishUnban`, `PublishGovernancePolicy`, `PublishCreateSub`, `PublishPostUpdate`, `PublishCommentUpdate`, `publishFavoriteOperation`, `publishLocalProfileUpdateLocked`, `publishGovernanceMessage`, `publishPayloadAsync`, `signAndQueueOutgoingMessage`）提取到 `p2p_publish.go` 文件中
3. THE Refactoring_Tool SHALL 将消息消费和路由逻辑（`consumeP2PMessages` 及其内部的消息类型分发）提取到 `p2p_consume.go` 文件中
4. THE Refactoring_Tool SHALL 将 `ProcessIncomingMessage`（450+ 行巨型 switch-case）从 `db.go` 移入 `p2p_consume.go`，因为消息处理属于 P2P 消费层职责而非数据库层
5. THE Refactoring_Tool SHALL 将反熵同步协议（`runAntiEntropySyncWorker`, `publishSyncSummaryRequest`, `publishCommentSyncRequest`, `publishGovernanceSyncRequest`, `publishFavoriteSyncRequest`, `handleSyncSummaryRequest`, `handleSyncSummaryResponse`, `handleCommentSyncRequest`, `handleCommentSyncResponse`, `handleGovernanceSyncRequest`, `handleGovernanceSyncResponse`, `handleFavoriteSyncRequest`, `handleFavoriteSyncResponse`, `TriggerAntiEntropySyncNow`, `TriggerCommentSyncNow`, `updateAntiEntropyStats` 及所有 resolve*AntiEntropy* 配置函数）提取到 `p2p_sync.go` 文件中
6. THE Refactoring_Tool SHALL 将内容/媒体获取协议（`fetchContentBlobFromNetwork`, `fetchMediaBlobFromNetwork`, `handleContentFetchRequest`, `handleContentFetchResponse`, `publishContentFetchNotFound`, `handleMediaFetchRequest`, `handleMediaFetchResponse`, `publishMediaFetchNotFound` 及 fetch 速率限制函数）提取到 `p2p_fetch.go` 文件中
7. THE Refactoring_Tool SHALL 将投票广播去重逻辑（`scheduleVoteStateBroadcast`, `resolveVoteBroadcastDebounce`）提取到 `p2p_votes.go` 文件中
8. THE Refactoring_Tool SHALL 将速率限制和 Peer 策略逻辑（`allowIncomingMessage`, `allowFetchRequest`, `refreshPeerPoliciesFromEnv`, `markPeerGreylisted`, `isPeerBlocked`, `parsePeerIDSet`, 及所有 resolve*RateLimit* 函数）提取到 `p2p_ratelimit.go` 文件中
9. THE Refactoring_Tool SHALL 将网络地址解析逻辑（`resolveP2PListenAddrs`, `resolveP2PAnnounceAddrs`, `resolveRelayPeerInfos`, `resolveRelayPeers`, `resolveMaxConnectedPeers`, `resolveGreylistTTLSeconds`, `resolveRelayServiceEnabled`）合并到 `p2p_config.go` 中
10. WHEN 所有拆分完成后，THE Refactoring_Tool SHALL 确保 P2P 功能正常工作且所有测试通过

### 需求 4：Go 后端 App 结构体瘦身与职责分离

**用户故事：** 作为开发者，我希望 `App` 结构体不再是 God Object，而是作为 Wails 绑定的 Facade 层，将具体业务逻辑委托给专门的服务文件。

#### 验收标准

1. THE Refactoring_Tool SHALL 将 `app.go` 中的身份管理逻辑（`GenerateIdentity`, `LoadSavedIdentity`, `ImportIdentityFromMnemonic`, `SignMessage`, `VerifyMessage`, `deriveKeypairFromMnemonic`）提取到 `identity.go` 文件中
2. THE Refactoring_Tool SHALL 将 `app.go` 中的 Feed 流逻辑（`GetFeedStream`, `GetFeedStreamWithStrategy`, `queryRecommendedCandidates`）提取到 `feed.go` 文件中
3. THE Refactoring_Tool SHALL 将 `app.go` 中的 P2P 自动启动配置解析逻辑（`shouldAutoStartP2P`, `resolveAutoStartP2PPort`, `resolveBootstrapPeers`, `resolveAutoStartPortCandidates`, `isTCPPortAvailable`）移入 `p2p_config.go` 中
4. THE Refactoring_Tool SHALL 确保 `app.go` 仅保留：App 结构体定义、`NewApp` 构造函数、`startup`/`shutdown` 生命周期方法、`SetDatabasePath`、`GetAntiEntropyStats`
5. THE Refactoring_Tool SHALL 为 `App` 结构体的字段添加分组注释（数据库、P2P 网络、速率限制、内容获取、可观测性、告警、投票广播、消息发送队列）

### 需求 5：Go 后端通用辅助函数集中管理

**用户故事：** 作为开发者，我希望散落在各文件中的通用辅助函数按职责集中管理，消除重复定义和查找困难。

#### 验收标准

1. THE Refactoring_Tool SHALL 将字符串处理辅助函数（`normalizeSubID`, `deriveTitle`, `deriveBodyPreview`, `buildMessageID`, `buildContentCID`, `buildBinaryCID`, `normalizeCommentAttachments`, `encodeCommentAttachmentsJSON`, `decodeCommentAttachmentsJSON`, `mediaCIDsFromAttachments`）提取到 `helpers.go` 文件中
2. THE Refactoring_Tool SHALL 将数值和排序辅助函数（`computeHotScore`, `normalizeFeedStreamLimit`, `normalizeFeedStreamAlgorithm`, `scoreFeedRecommendation`, `countFeedItemsByReason`, `normalizeSearchLimit`, `normalizeFavoriteLimit`, `normalizeMyPostsLimit`, `normalizeFeedSortMode`, `topWindowStartUnix`）提取到 `helpers.go` 文件中
3. THE Refactoring_Tool SHALL 将 Lamport 版本比较和操作类型归一化函数（`normalizeOperationType`, `normalizeVoteState`, `normalizeAuthScope`）移入 `db_operations.go` 中，因为它们与 Lamport 操作紧密相关
4. THE Refactoring_Tool SHALL 将收藏相关辅助函数（`normalizeFavoriteOperation`, `favoriteStateForOperation`, `favoriteOperationWins`, `buildFavoriteSignaturePayload`, `encodeFavoriteCursor`, `decodeFavoriteCursor`, `encodeMyPostsCursor`, `decodeMyPostsCursor`）移入 `db_favorites.go` 中
5. THE Refactoring_Tool SHALL 确保无重复的辅助函数定义存在于项目中

### 需求 6：React 前端状态管理层引入

**用户故事：** 作为前端开发者，我希望引入轻量级状态管理方案，消除 App.tsx 中 40+ 个 useState 和严重的 prop drilling 问题。

#### 验收标准

1. THE Refactoring_Tool SHALL 引入 React Context + useReducer 模式作为全局状态管理层
2. THE Refactoring_Tool SHALL 将身份相关状态（identity, profile, isAdmin, identityChecked）和身份操作（loadIdentity, createIdentity, importIdentity）提取到 `contexts/AuthContext.tsx` 中
3. THE Refactoring_Tool SHALL 将网络健康状态（p2pStatus, antiEntropyStats, onlineCount, networkBusy, networkHealth）和网络操作（loadNetworkHealth, triggerSyncNow）提取到 `contexts/NetworkContext.tsx` 中
4. THE Refactoring_Tool SHALL 将 UI 状态（isDark, view, currentSubId, sortMode, showLoginModal, showCreateSubModal, showCreatePostModal, showSettingsPanel, consistencyFocus, viewSyncToken）提取到 `contexts/UIContext.tsx` 中
5. THE Refactoring_Tool SHALL 将帖子和 Feed 相关状态（posts, profiles, favoritePostIds, subscribedSubs, subscribedSubIds, subs, unreadSubs, currentSubStats）提取到 `contexts/FeedContext.tsx` 中
6. THE Refactoring_Tool SHALL 将通知相关状态（unreadNotificationCount）和待同步操作状态（pendingSyncActions）提取到 `contexts/NotificationContext.tsx` 中
7. WHEN Context 引入完成后，THE Refactoring_Tool SHALL 确保组件不再需要超过 3 层的 props 传递

### 需求 7：React 前端自定义 Hook 抽象

**用户故事：** 作为前端开发者，我希望将 App.tsx 中的业务逻辑回调提取为可复用的自定义 Hook，使组件只关注渲染。

#### 验收标准

1. THE Refactoring_Tool SHALL 将帖子操作逻辑提取到 `hooks/usePosts.ts` 中（loadPosts, handleCreatePost, handleUpvote, handleDownvote, handleToggleFavorite, handleSharePost, loadFavorites）
2. THE Refactoring_Tool SHALL 将帖子详情逻辑提取到 `hooks/usePostDetail.ts` 中（loadPostDetail, handlePostClick, handleBackToFeed, refreshCommentsForSelectedPost, handleRefreshSelectedPost, selectedPost/postBody/postComments 状态管理）
3. THE Refactoring_Tool SHALL 将 Sub 管理逻辑提取到 `hooks/useSubs.ts` 中（loadSubs, loadSubscribedSubs, handleCreateSub, handleToggleSubscription, loadSubStats）
4. THE Refactoring_Tool SHALL 将搜索逻辑提取到 `hooks/useSearch.ts` 中（handleSearch, searchResults/searchQuery/searchScopeSubId 状态管理）
5. THE Refactoring_Tool SHALL 将治理数据加载逻辑提取到 `hooks/useGovernance.ts` 中（loadGovernanceData, governanceAdmins/moderationStates/moderationLogs 状态管理）
6. THE Refactoring_Tool SHALL 将待同步操作逻辑提取到 `hooks/usePendingSync.ts` 中（trackPendingSyncAction, handleDismissPendingSyncAction, reconcilePendingSyncActions）
7. THE Refactoring_Tool SHALL 将用户资料查看逻辑提取到 `hooks/useProfileView.ts` 中（selectedProfile/selectedProfilePosts 状态管理和加载逻辑）
8. WHEN 所有 Hook 提取完成后，THE Refactoring_Tool SHALL 确保 App.tsx 的行数减少到 300 行以内

### 需求 8：React 前端 App.tsx 瘦身与路由重构

**用户故事：** 作为前端开发者，我希望 App.tsx 仅作为应用入口和布局容器，视图切换通过声明式方式管理。

#### 验收标准

1. THE Refactoring_Tool SHALL 引入基于 hash 的轻量路由系统（利用现有的 `buildAppHash` 模式或引入 `react-router`），替代当前的 `view` 状态变量和 `ViewMode` 类型手动切换
2. THE Refactoring_Tool SHALL 将 App.tsx 中的 Wails 事件监听逻辑（`EventsOn` 订阅 `p2p:updated`, `feed:updated`, `notifications:updated`, `favorites:updated` 等）提取到对应的 Context 中
3. THE Refactoring_Tool SHALL 将 `mapPostIndexToPost`、`mapForumMessageToPost` 提取到 `lib/mappers.ts` 中
4. THE Refactoring_Tool SHALL 将 `getErrorMessage` 提取到 `lib/errors.ts` 中
5. THE Refactoring_Tool SHALL 将 `buildAppHash`、`buildShareLink`、`buildSubShareLink` 提取到 `lib/routing.ts` 中
6. THE Refactoring_Tool SHALL 将 `shouldTrackPendingSync`、`getWriteFeedback` 提取到 `lib/syncFeedback.ts` 中
7. WHEN 重构完成后，THE Refactoring_Tool SHALL 确保 App.tsx 仅包含 Context Provider 组装、布局结构和路由定义

### 需求 9：React 前端 SettingsPanel 按 Tab 拆分

**用户故事：** 作为前端开发者，我希望 SettingsPanel 的 1690+ 行代码按 Tab 页面拆分为独立组件，每个 Tab 可独立开发和测试。

#### 验收标准

1. THE Refactoring_Tool SHALL 将 Account Tab（个人资料编辑、头像上传）逻辑提取到 `components/settings/AccountTab.tsx` 中
2. THE Refactoring_Tool SHALL 将 Privacy & Keys Tab（助记词显示、隐私设置）逻辑提取到 `components/settings/PrivacyTab.tsx` 中
3. THE Refactoring_Tool SHALL 将 Network & P2P Tab（P2P 状态、连接管理、端口配置）逻辑提取到 `components/settings/NetworkTab.tsx` 中
4. THE Refactoring_Tool SHALL 将 Updates Tab（版本检查、更新历史）逻辑提取到 `components/settings/UpdatesTab.tsx` 中
5. THE Refactoring_Tool SHALL 将 Consistency Tab（Lamport 操作日志、Tombstone GC）逻辑提取到 `components/settings/ConsistencyTab.tsx` 中
6. THE Refactoring_Tool SHALL 将 Governance Tab（Shadow Ban 管理、管理员管理、审核日志）逻辑提取到 `components/settings/GovernanceTab.tsx` 中
7. THE Refactoring_Tool SHALL 确保 `SettingsPanel.tsx` 仅保留 Tab 导航框架和布局，行数减少到 150 行以内
8. THE Refactoring_Tool SHALL 将 SettingsPanel 中的图片压缩逻辑（compressAvatarFileIfNeeded 及相关）提取到 `lib/imageUtils.ts` 中

### 需求 10：React 前端 PostDetail 按功能区域拆分

**用户故事：** 作为前端开发者，我希望 PostDetail 的 920+ 行代码按功能区域拆分，帖子展示、评论输入、编辑模式各自独立。

#### 验收标准

1. THE Refactoring_Tool SHALL 将帖子内容展示区域（标题、正文 Markdown 渲染、图片展示、链接卡片）提取到 `components/post/PostContent.tsx` 中
2. THE Refactoring_Tool SHALL 将帖子操作栏（投票按钮、评论数、分享、收藏、编辑/删除）提取到 `components/post/PostActions.tsx` 中
3. THE Refactoring_Tool SHALL 将帖子编辑模式逻辑（编辑表单、保存/取消）提取到 `components/post/PostEditor.tsx` 中
4. THE Refactoring_Tool SHALL 将评论输入区域（包含草稿自动保存、图片附件上传、代码块插入、引用回复）提取到 `components/post/CommentComposer.tsx` 中
5. THE Refactoring_Tool SHALL 将评论草稿管理逻辑（`getCommentDraftStorageKey`, `loadCommentDraft`, `saveCommentDraft`, `clearCommentDraft`, `parseCommentDraft`, `canUseLocalStorage`）提取到 `hooks/useCommentDraft.ts` 中
6. THE Refactoring_Tool SHALL 将 `linkifyAndMarkdown` 富文本渲染函数和 `buildQuotedReply` 提取到 `lib/richText.ts` 中
7. WHEN 拆分完成后，THE Refactoring_Tool SHALL 确保 PostDetail.tsx 仅作为布局容器，行数减少到 200 行以内

### 需求 11：React 前端重复工具函数消除与共享组件提取

**用户故事：** 作为前端开发者，我希望消除跨组件的重复代码（`formatTimeAgo` 出现 3 次、`getInitials` 出现 3 次），并将重复 UI 模式提取为共享组件。

#### 验收标准

1. THE Refactoring_Tool SHALL 将 `formatTimeAgo` 函数（当前在 PostDetail.tsx、CommentTree.tsx、PostCard.tsx 中各有一份独立实现）统一提取到 `lib/time.ts` 中，所有组件引用同一份实现
2. THE Refactoring_Tool SHALL 将 `getInitials` 函数（当前在 PostDetail.tsx、CommentTree.tsx、Header.tsx 中各有一份独立实现）统一提取到 `lib/string.ts` 中
3. THE Refactoring_Tool SHALL 将 `formatCreatedAt`（RightPanel.tsx 中）也合并到 `lib/time.ts` 中
4. THE Refactoring_Tool SHALL 将重复出现的头像渲染逻辑（avatarUrl 判断 + initials fallback，在 PostCard.tsx、PostDetail.tsx、CommentTree.tsx、Header.tsx 中各有实现）提取到 `components/shared/Avatar.tsx` 中
5. THE Refactoring_Tool SHALL 将确认删除对话框模式（PostDetail 和 CommentTree 中的删除确认）提取到 `components/shared/ConfirmDialog.tsx` 中
6. THE Refactoring_Tool SHALL 将图片预览 Lightbox 模式（PostDetail 和 CommentTree 中的图片放大预览）提取到 `components/shared/ImagePreview.tsx` 中
7. THE Refactoring_Tool SHALL 将 CommentTree.tsx 中的 `renderRichText` 函数与 PostDetail.tsx 中的 `linkifyAndMarkdown` 函数统一为 `lib/richText.ts` 中的单一实现
8. WHEN 提取完成后，THE Refactoring_Tool SHALL 确保项目中不存在重复的 `formatTimeAgo`、`getInitials`、`renderRichText` 函数定义

### 需求 12：React 前端目录结构按功能领域组织

**用户故事：** 作为前端开发者，我希望前端目录结构按功能领域组织，而非将 23 个组件平铺在 `components/` 下。

#### 验收标准

1. THE Refactoring_Tool SHALL 将前端源码按以下目录结构组织：
   - `src/components/layout/` — 布局组件（Header, Sidebar, RightPanel）
   - `src/components/feed/` — Feed 相关组件（Feed, PostCard）
   - `src/components/post/` — 帖子详情相关组件（PostDetail, PostContent, PostActions, PostEditor, CommentComposer, CommentTree）
   - `src/components/settings/` — 设置面板各 Tab 组件（SettingsPanel, AccountTab, PrivacyTab, NetworkTab, UpdatesTab, ConsistencyTab, GovernanceTab）
   - `src/components/shared/` — 共享 UI 组件（Avatar, ConfirmDialog, ImagePreview, Toast, NetworkBanner）
   - `src/components/modals/` — 模态框组件（CreatePostModal, CreateSubModal, LoginModal）
   - `src/components/views/` — 独立视图组件（DiscoverView, ProfileView, MyPosts, Favorites, DraftsView, HistoryView, SearchResultsView, PendingSyncView, NotificationsView, UserMenu）
   - `src/contexts/` — Context 定义
   - `src/hooks/` — 自定义 Hook
   - `src/lib/` — 工具函数库
   - `src/types/` — 类型定义
2. THE Refactoring_Tool SHALL 确保所有 import 路径在重组后正确更新
3. THE Refactoring_Tool SHALL 为每个子目录创建 `index.ts` 导出文件以简化导入路径

### 需求 13：前端 API 封装层

**用户故事：** 作为开发者，我希望前端对后端 API 的调用有统一的封装层，而非在 App.tsx 中直接 import 50+ 个 Wails 绑定函数。

#### 验收标准

1. THE Refactoring_Tool SHALL 创建 `src/api/` 目录，按业务领域封装后端调用：
   - `src/api/identity.ts` — 身份相关 API（LoadSavedIdentity, GenerateIdentity, ImportIdentityFromMnemonic, SignMessage, VerifyMessage）
   - `src/api/posts.ts` — 帖子相关 API（GetFeedStream, GetFeedIndexBySubSorted, GetPostIndexByID, GetPostBodyByID, GetMyPosts, GetPostsByAuthor, PublishPostStructuredToSub, PublishPostWithImageToSub, PublishPostUpdate, PublishDeletePost, SearchPosts）
   - `src/api/comments.ts` — 评论相关 API（GetCommentsByPost, PublishCommentWithAttachments, PublishCommentUpdate, PublishDeleteComment, PublishCommentUpvote, PublishCommentDownvote, TriggerCommentSyncNow）
   - `src/api/subs.ts` — Sub 相关 API（GetSubs, CreateSub, PublishCreateSub, SubscribeSub, UnsubscribeSub, GetSubscribedSubs, SearchSubs, GetSubStats）
   - `src/api/network.ts` — 网络相关 API（GetP2PStatus, GetAntiEntropyStats, TriggerAntiEntropySyncNow）
   - `src/api/governance.ts` — 治理相关 API（GetTrustedAdmins, GetModerationState, GetModerationLogs, PublishShadowBan, PublishUnban）
   - `src/api/profile.ts` — 个人资料相关 API（GetProfile, GetProfileDetails, UpdateProfileDetails, PublishProfileUpdate）
   - `src/api/favorites.ts` — 收藏相关 API（GetFavoritePostIDs, AddFavorite, RemoveFavorite）
   - `src/api/notifications.ts` — 通知相关 API（GetUnreadNotificationCount, GetNotifications, MarkNotificationRead, MarkAllNotificationsRead）
2. THE Refactoring_Tool SHALL 确保每个 API 封装函数包含统一的错误处理和 TypeScript 类型标注
3. THE Refactoring_Tool SHALL 确保组件和 Hook 通过 `api/` 层调用后端，而非直接导入 `wailsjs/go/main/App`

### 需求 14：重构过程的功能完整性保障

**用户故事：** 作为产品负责人，我希望重构过程中所有现有功能保持完整，不引入任何功能回归。

#### 验收标准

1. WHILE 重构进行中，THE Refactoring_Tool SHALL 确保每个重构步骤后 Go 代码可编译通过（`go build ./...`）
2. WHILE 重构进行中，THE Refactoring_Tool SHALL 确保每个重构步骤后所有现有 Go 测试通过（`go test ./...`）
3. THE Refactoring_Tool SHALL 确保所有 Wails 绑定的公开方法签名保持不变（方法名、参数类型、返回类型），前端调用不受影响
4. THE Refactoring_Tool SHALL 确保重构后的代码保持与原代码相同的运行时行为
5. IF 重构过程中发现现有 Bug，THEN THE Refactoring_Tool SHALL 以代码注释形式记录但不在本次重构中修复，避免混淆重构与功能变更
6. THE Refactoring_Tool SHALL 确保前端 TypeScript 编译通过（`npx tsc --noEmit`）

### 需求 15：代码质量标准

**用户故事：** 作为开发者，我希望重构后的代码符合高标准的工程规范，每个文件职责单一、命名清晰。

#### 验收标准

1. THE Refactoring_Tool SHALL 确保重构后每个 Go 文件不超过 500 行（Schema 定义文件 `db_schema.go` 可放宽到 800 行）
2. THE Refactoring_Tool SHALL 确保重构后每个 React 组件文件不超过 300 行
3. THE Refactoring_Tool SHALL 确保每个文件顶部包含一行简要的文件用途注释
4. THE Refactoring_Tool SHALL 确保所有公开函数和类型包含 GoDoc/JSDoc 注释
5. THE Refactoring_Tool SHALL 确保 Go 文件命名使用 snake_case（如 `db_posts.go`），React 组件文件使用 PascalCase（如 `PostDetail.tsx`），Hook 文件使用 camelCase（如 `usePosts.ts`），工具函数文件使用 camelCase（如 `mappers.ts`）
6. THE Refactoring_Tool SHALL 确保无未使用的导入、变量或函数存在于重构后的代码中
7. THE Refactoring_Tool SHALL 确保已有的良好设计保持不变：`recommendation.go` 的策略模式、`authenticated_messages.go` 的签名逻辑、`message_outbox.go` 的发送队列、前端 `lib/` 下的工具模块
