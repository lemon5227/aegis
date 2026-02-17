# Aegis 前后端 API 缺口瀑布计划（2026-02-16）

## 1. 目标
在不破坏已完成能力（订阅/推送/搜索/FeedStream）的前提下，完成“前端已做 UI 与后端 API 契约”的对齐。

本文件是后端执行清单，不包含前端实现细节。

---

## 2. 对账来源
- 前端代码：`aegis-app/frontend/src`
- 前端 API 需求：`docs/FRONTEND_API_REQUIREMENTS.md`
- 当前后端导出 API：`aegis-app/frontend/wailsjs/go/main/App.d.ts`

---

## 3. 缺口清单（按前端现状核对）

## 3.1 需要新增后端 API（真实缺口）
1. 收藏持久化缺口（高）
- 证据：`aegis-app/frontend/src/components/Favorites.tsx:18`、`aegis-app/frontend/src/components/Favorites.tsx:36`
- 现状：使用 localStorage 存储收藏 ID。
- 需要：`AddFavorite/RemoveFavorite/GetFavorites/IsFavorited`。

2. 我的帖子缺口（高）
- 证据：`aegis-app/frontend/src/components/MyPosts.tsx:18`
- 现状：使用 localStorage 存储帖子，非权威数据源。
- 需要：`GetMyPosts(limit, cursor)`（本机身份语义）。

3. 隐私设置与 Bio 缺口（中）
- 证据：`aegis-app/frontend/src/components/SettingsPanel.tsx:215`、`aegis-app/frontend/src/components/SettingsPanel.tsx:282`
- 现状：Bio/Privacy 仅本地状态，未接后端。
- 需要：`GetPrivacySettings/SetPrivacySettings`，以及 `UpdateProfile` 扩展（或新增 `UpdateProfileDetails`）。

4. 更新检查缺口（中低）
- 证据：`aegis-app/frontend/src/components/SettingsPanel.tsx:360`
- 现状：Updates 页面为静态展示。
- 需要：`CheckForUpdates/GetVersionHistory`（V1 可先返回本地版本与“未配置更新源”状态）。

5. 帖子下踩缺口（中）
- 证据：`aegis-app/frontend/src/components/PostDetail.tsx:120`
- 现状：UI 有下踩按钮，后端无对应 API。
- 需要：`DownvotePost`（以及后续 vote 模型升级）。

## 3.2 可复用现有后端 API（前端未接，不属于后端缺口）
1. 治理“封禁状态”可复用
- 现有 API：`GetModerationState`（`aegis-app/frontend/wailsjs/go/main/App.d.ts:49`）
- 前端现状：Banned Users 区块仍是静态文案（`aegis-app/frontend/src/components/SettingsPanel.tsx:453`）。
- 结论：优先前端接现有 API，无需先加新后端接口。

2. 订阅与评论事件可复用
- 现有事件：`sub:updated`、`subs:subscriptions_updated`、`comments:updated`
- 证据：`aegis-app/db.go:1595`、`aegis-app/db.go:3553`、`aegis-app/p2p.go:1218`
- 前端现状：未监听 runtime 事件。
- 结论：先接现有事件；不急于新增新事件。

## 3.3 结构性问题（建议后端补一个最小 API）
1. 搜索结果跳转到帖子详情不稳定（中）
- 证据：`aegis-app/frontend/src/App.tsx:363`
- 问题：点击搜索帖子时仅在当前 `posts` 列表中查找，可能找不到。
- 建议新增：`GetPostIndexByID(postID)`，复用现有索引模型，降低前端状态耦合。

---

## 4. API 设计原则（复用优先）
1. 本机身份语义优先
- 与 `UpvotePost(postID)` 一致，收藏与我的帖子接口默认从本地 identity 推导 `pubkey`。
- 同助记词（同 `pubkey`）多设备之间通过索引同步收敛。

2. 加法兼容
- 只新增 API，不改旧签名；旧接口至少保留两个阶段。

3. 复用已有数据表和查询模型
- 帖子读取优先复用 `messages`/`PostIndex` 查询逻辑。
- 治理读取优先复用 `moderation`/`moderation_logs` 现有能力。

4. 事件沿用现有通道
- 优先复用 `runtime.EventsEmit` 与既有事件名；新增事件需评审命名冲突与 payload 兼容性。

5. 索引同步范式统一
- 收藏索引同步与帖子索引同步使用同一范式（增量 + 反熵 + 批次窗口）。
- 同步仅传索引操作，不传正文/媒体。
- 收藏详情命中后的正文缓存只写本机私有区，不占共享区。

6. Relay 拓扑统一
- 每个节点默认具备 relay 能力（可作为中继服务端/客户端）。
- 具备公网 IP 的节点可承担 relay 角色，私网节点打洞失败时自动回退 relay 链路。
- Relay 仅负责转发连接与消息，不改变收藏/帖子索引一致性语义。

---

## 5. 冻结 API 草案（后端）

## 5.1 Favorites（P0）
- `AddFavorite(postID string) error`
- `RemoveFavorite(postID string) error`
- `GetFavorites(limit int, cursor string) (PostIndexPage, error)`
- `IsFavorited(postID string) (bool, error)`
- 可选批量：`GetFavoritePostIDs() ([]string, error)`

事件：
- `favorites:updated`（payload: `{ postId: string }`）

内部同步协议（非前端 API）：
- `FAVORITE_OP`
- `FAVORITE_SYNC_REQUEST`
- `FAVORITE_SYNC_RESPONSE`

## 5.2 My Posts（P0）
- `GetMyPosts(limit int, cursor string) (PostIndexPage, error)`

## 5.3 Profile/Privacy（P1）
- `UpdateProfileDetails(displayName string, avatarURL string, bio string) (ProfileDetails, error)`
- `GetProfileDetails(pubkey string) (ProfileDetails, error)`
- `SetPrivacySettings(showOnlineStatus bool, allowSearch bool) (PrivacySettings, error)`
- `GetPrivacySettings() (PrivacySettings, error)`

## 5.4 Governance（P1，复用优先）
- 先复用：`GetModerationState()`
- 后补（仅在产品确认后）：`GetUnbanRequests()`

## 5.5 Update（P2）
- `CheckForUpdates() (UpdateStatus, error)`
- `GetVersionHistory(limit int) ([]VersionEntry, error)`

## 5.6 搜索跳转辅助（P1）
- `GetPostIndexByID(postID string) (PostIndex, error)`

## 5.7 P2P 配置（P1）
- `GetP2PConfig() (P2PConfig, error)`
- `SaveP2PConfig(listenPort int, relayPeers []string, autoStart bool) (P2PConfig, error)`

`P2PConfig` 建议字段：
- `listenPort int`
- `relayPeers []string`
- `autoStart bool`
- `updatedAt int64`

说明：
- 保留现有 `StartP2P/StopP2P/GetP2PStatus/ConnectPeer`，新接口仅负责配置持久化。
- 启动配置优先级冻结：`StartP2P 显式参数 > SQLite 配置 > ENV > 默认值`。
- relay 拓扑沿用既有语义：每个节点可启用 relay 能力，公网节点可承担中继角色。
- 初始中继可通过 ENV 种子注入（示例）：
  - `AEGIS_BOOTSTRAP_PEERS=/ip4/51.107.0.10/tcp/40100/p2p/12D3KooWLweFn4GFfEa9X1St4d78HQqYYzXaH2oy5XahKrwar6w7`
  - `AEGIS_RELAY_PEERS=/ip4/51.107.0.10/tcp/40100/p2p/12D3KooWLweFn4GFfEa9X1St4d78HQqYYzXaH2oy5XahKrwar6w7`

---

## 6. 瀑布阶段（G 系列）

## G0：契约冻结（入口阶段）
### 目标
冻结接口命名、参数语义、兼容规则。

### DoD
- 本文 API 草案冻结；未经变更单不得改签名。

### 关口
- 输出兼容承诺：`订阅/推送/搜索/FeedStream` 回归不受影响。

---

## G1：Favorites 后端落地（P0）
### 范围
`post_favorites_state/post_favorite_ops` + 收藏增删查接口 + `favorites:updated` + 收藏索引同步协议。

### DoD
- 前端可完全替换 localStorage 收藏逻辑。
- 同助记词跨设备可同步收藏状态。

### 回归门禁
- 不得改动现有 Feed/Search 返回字段。

---

## G2：My Posts + 跳转辅助（P0/P1）
### 范围
`GetMyPosts`、`GetPostIndexByID`。

### DoD
- My Posts 不再依赖 localStorage。
- 搜索点击帖子可稳定打开详情。

### 回归门禁
- `GetFeedIndexBySubSorted` 行为不变。

---

## G2.5：P2P 配置产品化（P1）
### 范围
`GetP2PConfig`、`SaveP2PConfig`、启动阶段读取 SQLite 配置（保留 ENV 回退）。

### DoD
- 设置页可持久化 P2P 端口与 relay 列表。
- 启停 P2P 与配置写入解耦，且兼容现有启动参数。

### 回归门禁
- 既有 `StartP2P/StopP2P/GetP2PStatus` 行为不变。
- 订阅/推送/搜索/FeedStream 不回归。

---

## G3：Profile Details + Privacy（P1）
### 范围
bio 与隐私设置 API，且与现有 `UpdateProfile/GetProfile` 保持兼容。

### DoD
- Settings 的 Account/Privacy 可持久化。

### 回归门禁
- 旧 `UpdateProfile` 仍可正常工作（兼容路径）。

---

## G4：Governance 对账（P1）
### 范围
优先复用 `GetModerationState` 对接 Banned Users；评估是否新增 `GetUnbanRequests`。

### DoD
- 治理页“封禁列表”不再是静态占位。

### 回归门禁
- 不改变现有封禁广播协议。

---

## G5：Update API（P2）
### 范围
`CheckForUpdates/GetVersionHistory` 最小可用版本。

### DoD
- 更新页不再完全静态。

### 回归门禁
- 无更新源时需优雅降级，不阻断主流程。

---

## G6：兼容与防破坏门禁（持续）
### 固定门禁
1. 数据库迁移仅加法（禁止破坏性变更）。  
2. 导出 API 仅新增（禁止改名/删字段）。  
3. 每阶段必须跑回归：`订阅、推送、搜索、FeedStream`。  
4. App.d.ts 契约快照检查（防止无意改动前端契约）。  
5. 变更单机制：跨阶段插单必须记录原因、影响、回滚点。  
6. NAT/防火墙受限场景需覆盖 relay 回退联调（至少 1 公网 relay 节点 + 2 私网节点）。  

---

## 7. 当前状态（2026-02-16）
- G0：Done（本文件）
- G1：Done（收藏后端+分布式索引同步已实现）
- G2：Done（`GetMyPosts` + `GetPostIndexByID` 已实现并接入前端）
- G2.5：Done（P2P 配置持久化 API + Settings 接入已完成）
- G3：Done（Profile Details + Privacy API 已实现并接入 Settings）
- G4：Pending
- G5：Pending
- G6：In Progress（流程门禁持续执行）
