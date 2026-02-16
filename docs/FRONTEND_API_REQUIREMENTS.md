# 前端 API 需求文档

本文档列出前端需要但后端尚未实现的 API。
执行主计划请同时参考：`docs/FRONTEND_BACKEND_API_GAP_WATERFALL_2026-02-16.md`。

---

## 一、已有 API（已接入）

### 1.1 订阅与推荐
- [x] `GetSubs()` - 获取所有 Sub
- [x] `GetSubscribedSubs()` - 获取已订阅 Sub
- [x] `SubscribeSub(subId)` - 订阅 Sub
- [x] `UnsubscribeSub(subId)` - 取消订阅
- [x] `GetFeedStream(limit)` - 获取推荐流（订阅 70% + 热门 30%）

### 1.2 帖子
- [x] `GetFeedIndexBySubSorted(subId, sortMode)` - 获取帖子流
- [x] `PublishPostUpvote(pubkey, postId)` - 帖子点赞
- [x] `PublishPostStructuredToSub(pubkey, title, body, subId)` - 发布帖子

### 1.3 帖子详情
- [x] `GetPostBodyByID(postId)` - 获取帖子正文
- [x] `GetCommentsByPost(postId)` - 获取评论列表
- [x] `PublishComment(pubkey, postId, parentId, body)` - 发布评论
- [x] `PublishCommentUpvote(pubkey, postId, commentId)` - 评论点赞

### 1.4 用户与资料
- [x] `LoadSavedIdentity()` - 加载已保存身份
- [x] `GenerateIdentity()` - 创建新身份
- [x] `ImportIdentityFromMnemonic(mnemonic)` - 导入身份
- [x] `GetProfile(pubkey)` - 获取用户资料
- [x] `UpdateProfile(displayName, avatarURL)` - 更新本地资料
- [x] `PublishProfileUpdate(pubkey, displayName, avatarURL)` - 广播资料更新
- [x] `GetTrustedAdmins()` - 获取管理员列表

### 1.5 搜索
- [x] `SearchSubs(keyword, limit)` - 搜索 Sub
- [x] `SearchPosts(keyword, subId, limit)` - 搜索帖子

### 1.6 治理
- [x] `PublishShadowBan(targetPubkey, adminPubkey, reason)` - 广播封禁
- [x] `PublishUnban(targetPubkey, adminPubkey, reason)` - 广播解封
- [x] `GetModerationLogs(limit)` - 获取治理日志

---

## 二、待实现 API（需要后端实现）

### 2.1 帖子功能

| API | 说明 | 优先级 |
|-----|------|--------|
| `DownvotePost(pubkey, postId)` | 帖子踩 | 中 |

### 2.2 用户内容

| API | 说明 | 优先级 |
|-----|------|--------|
| `GetMyPosts(limit, cursor)` | 获取当前身份发布的帖子列表（分页，返回 `PostIndexPage`） | **高（已实现）** |
| `GetPostIndexByID(postId)` | 搜索结果点击帖子时按 ID 获取索引详情 | **高（已实现）** |
| `GetFavorites(limit, cursor)` | 获取用户收藏的帖子列表（分页，返回 `PostIndexPage`） | **高（已实现）** |
| `AddFavorite(postId)` | 添加收藏（本机身份语义） | **高（已实现）** |
| `RemoveFavorite(postId)` | 取消收藏（本机身份语义） | **高（已实现）** |
| `IsFavorited(postId)` | 检查帖子是否已收藏（本机身份语义） | 中（已实现） |
| `GetFavoritePostIDs()` | 批量获取收藏帖子ID（用于列表态映射） | 中（已实现） |

### 2.3 隐私设置

| API | 说明 | 优先级 |
|-----|------|--------|
| `SetPrivacySettings(pubkey, showOnlineStatus, allowSearch)` | 设置隐私选项 | 低 |
| `GetPrivacySettings(pubkey)` | 获取隐私设置 | 低 |

### 2.4 更新检查

| API | 说明 | 优先级 |
|-----|------|--------|
| `CheckForUpdates()` | 检查是否有新版本 | 低 |
| `GetVersionHistory()` | 获取版本历史 | 低 |

### 2.5 治理增强

| API | 说明 | 优先级 |
|-----|------|--------|
| `GetUnbanRequests()` | 获取解封请求列表 | 中 |
| `GetBannedUsers()` | 获取已封禁用户列表 | 中 |
| `GetModerationState()` | 获取当前封禁状态 | 中 |

### 2.6 推送事件（后端 → 前端）

| 事件 | 说明 | 优先级 |
|------|------|--------|
| `sub:updated` | 订阅的 Sub 有新帖子 | 高 |
| `comment:new` | 新评论通知 | 中 |
| `moderation:action` | 治理操作通知（被封禁等） | 中 |

### 2.7 网络与 P2P 配置

| API | 说明 | 优先级 |
|-----|------|--------|
| `GetP2PConfig()` | 获取本地节点 P2P 配置（端口/relay/autoStart） | 高 |
| `SaveP2PConfig(listenPort, relayPeers, autoStart)` | 保存本地节点 P2P 配置到 SQLite | 高 |

说明：
- 运行控制继续复用：`StartP2P`、`StopP2P`、`GetP2PStatus`、`ConnectPeer`。
- 设置页通过 `p2p:updated` 事件刷新状态。

---

## 三、前端已实现功能

### 3.1 Dashboard
- [x] 左侧 Sidebar：推荐 Feed、已订阅 Sub、Discover More
- [x] 中间：帖子列表（Hot/New 排序）
- [x] 右侧：社区信息 + 订阅按钮
- [x] 顶部：搜索框

### 3.2 帖子详情
- [x] 帖子正文展示
- [x] 帖子投票（upvote）
- [x] 评论列表（楼中楼样式）
- [x] 发布评论
- [x] 评论点赞
- [x] 面包屑导航返回

### 3.3 用户菜单
- [x] 点击头像显示菜单：
  - Profile（打开设置面板）
  - My Posts（中间显示我的帖子）
  - Favorites（中间显示收藏列表）
  - Log Out

### 3.4 My Posts 页面
- [x] 显示当前用户发布的所有帖子
- [x] 使用后端 `GetMyPosts(limit, cursor)` 读取权威数据

### 3.5 Favorites 页面
- [x] 显示用户收藏的帖子
- [x] 暂时用 localStorage 存储收藏的帖子 ID

### 3.6 设置面板（SettingsPanel）
- [x] **Account** 标签页：
  - 头像上传
  - 昵称修改
  - Bio 设置
  - Public Status 开关
  - Save Changes 按钮
- [x] **Privacy & Keys** 标签页：
  - 公钥显示
  - Mnemonic Phrase（备份）
  - 隐私设置（Online Status, Allow Search）
- [x] **Updates** 标签页：
  - 当前版本显示
  - What's New（功能列表）
  - Check for Updates 按钮
- [x] **Governance** 标签页（仅管理员）：
  - Ban User 表单
  - 封禁用户列表
  - Unban Requests 标签页
  - Operation Log 标签页

### 3.7 其他
- [x] 深色/浅色主题切换
- [x] 搜索功能（搜 Sub 和帖子）
- [x] 创建 Sub
- [x] 发帖功能

---

## 四、注意事项

1. **Favorites 持久化**：后端收藏API已实现（`GetFavorites`, `AddFavorite`, `RemoveFavorite`, `IsFavorited`, `GetFavoritePostIDs`），前端需要从 localStorage 迁移到后端调用。
   - 同步语义：跨设备同步收藏索引（同助记词同 `pubkey`），正文/媒体仅本机按需缓存，不占共享区。

2. **My Posts**：已接入后端 `GetMyPosts(limit, cursor)`，不再依赖 localStorage。

3. **隐私设置**：Privacy 页面有 UI，但设置暂时不保存到后端，需要实现 `SetPrivacySettings` 和 `GetPrivacySettings` API。

4. **更新检查**：Updates 页面有 UI，但检查更新功能未实现，需要 `CheckForUpdates` 和 `GetVersionHistory` API。

5. **推送事件接入**：后端实现 `sub:updated` 事件后，前端需要添加事件监听来显示通知或自动刷新。

6. **P2P 配置持久化**：Network 设置应保存到 SQLite（后端 API），不使用 localStorage 作为权威配置源。

7. **窗口大小**：已修改 main.go，初始窗口大小改为 1200x800，背景色改为浅色。
