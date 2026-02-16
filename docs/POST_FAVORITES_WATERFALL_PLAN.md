# Aegis 帖子收藏功能瀑布计划（后端）

## 1. 目标与范围
新增“帖子收藏（Favorites）”后端能力，替代前端 localStorage 临时方案。
收藏能力按“分布式索引同步”实现：同一助记词（同 `pubkey`）可跨设备恢复收藏。

本计划仅覆盖后端与接口契约，不含前端实现。
若与 `docs/FRONTEND_BACKEND_API_GAP_WATERFALL_2026-02-16.md` 存在冲突，以后者冻结契约为准。

---

## 2. 前端 API 需求核对结论
已核对文档：`docs/FRONTEND_API_REQUIREMENTS.md`

前端文档当前标记的收藏相关待实现 API：
- `GetFavorites(pubkey)`
- `AddFavorite(pubkey, postId)`
- `RemoveFavorite(pubkey, postId)`
- `IsFavorited(pubkey, postId)`

后端建议冻结为“本机身份语义 + 跨设备索引同步”：
- `GetFavorites(limit int, cursor string) (PostIndexPage, error)`
- `AddFavorite(postID string) error`
- `RemoveFavorite(postID string) error`
- `IsFavorited(postID string) (bool, error)`

分布式语义冻结：
- 同步对象：仅收藏索引（`post_id` 级别操作），不同步正文与媒体。
- 同步身份：由本机助记词推导 `pubkey`，跨设备同 `pubkey` 自动收敛。
- 存储策略：收藏命中的正文/媒体仅缓存到本机私有区，不计入共享区配额。
- 中继拓扑：每个节点默认可启用 relay 能力；具备公网 IP 的节点可充当 relay，私网节点可通过 relay 保持同步连通。

---

## 3. 瀑布执行规则
1. 严格顺序：F1 -> F2 -> F3 -> F4 -> F5。  
2. 每阶段完成标准：代码完成 + 编译/测试通过 + 文档状态更新。  
3. 先冻结接口再编码。  
4. 新需求不得插入当前阶段，必须挂到后续阶段。  

---

## 4. 阶段拆解

## F1：数据模型与迁移
### 目标
为收藏索引提供“状态表 + 操作日志表”双层模型（与帖子索引同步范式一致）。

### 任务
1. 新增状态表 `post_favorites_state`：
- `post_id TEXT NOT NULL`
- `pubkey TEXT NOT NULL`
- `state TEXT NOT NULL`（`active`/`removed`）
- `updated_at INTEGER NOT NULL`
- `last_op_id TEXT NOT NULL`
- 主键：`(pubkey, post_id)`

2. 新增操作日志表 `post_favorite_ops`：
- `op_id TEXT PRIMARY KEY`
- `pubkey TEXT NOT NULL`
- `post_id TEXT NOT NULL`
- `op TEXT NOT NULL`（`ADD`/`REMOVE`）
- `created_at INTEGER NOT NULL`
- `signature TEXT NOT NULL`

3. 新增索引：
- `idx_post_favorites_state_pubkey_updated_at`
- `idx_post_favorites_state_post_id`
- `idx_post_favorite_ops_pubkey_created_at`

4. 迁移策略：
- `CREATE TABLE IF NOT EXISTS`，兼容历史数据库。
- 对旧本地收藏数据提供一次性回填脚本（可选，非阻断）。

### DoD
- 历史库可无损启动并自动补齐表结构。

### 验证
- 启动后可查询到新表与索引，无迁移错误。

---

## F2：收藏写接口
### 目标
提供幂等写接口，并落地可同步的收藏操作日志。

### 任务
1. 新增 API：
- `AddFavorite(postID string) error`
- `RemoveFavorite(postID string) error`

2. 规则：
- 需校验 `postID` 存在（不存在返回明确错误）。
- `AddFavorite` 幂等（重复收藏不报错）。
- `RemoveFavorite` 幂等（未收藏时返回成功）。
- 每次状态变化写入 `post_favorite_ops`（含 `op_id`、签名）。
- 冲突处理采用 LWW（`updated_at`，同时间戳按 `op_id` 字典序）。

3. 事件：
- 收藏状态变化时广播 `favorites:updated`（可选附带 `postId`）。

4. 分布式协议（内部）：
- `FAVORITE_OP`：广播单条收藏操作。
- `FAVORITE_SYNC_REQUEST`：按时间窗请求增量操作。
- `FAVORITE_SYNC_RESPONSE`：返回操作批次。
- 复用现有反熵 worker 的调度与批次参数管理。

### DoD
- 收藏状态可稳定变更，重复操作行为一致。
- 同助记词多设备间收藏操作可最终收敛。

### 验证
- 重复收藏/取消收藏场景结果正确。
- A/B 两设备同一助记词登录，A 收藏后 B 自动可见；A 取消后 B 自动消失。

---

## F3：收藏读接口
### 目标
支持读取收藏列表与单帖收藏状态。

### 任务
1. 新增 API：
- `GetFavorites(limit int, cursor string) ([]PostIndex, string, error)`
- `IsFavorited(postID string) (bool, error)`

2. 查询规则：
- `GetFavorites` 按 `created_at DESC` 返回。
- 仅返回“当前身份 `pubkey` 且 `state=active`”的数据。
- 默认 `limit=50`，上限 `200`。
- 支持 cursor 分页，返回 `nextCursor`。

### DoD
- 可正确返回收藏列表与单帖收藏状态。

### 验证
- 多帖收藏后顺序正确。
- 已收藏/未收藏判定正确。

---

## F4：Feed 与搜索联动（后端）
### 目标
让 feed/search 结果可直接携带收藏状态，减少前端额外请求。

### 任务
1. 扩展返回模型（最小改动优先）：
- 方案 A：在 `FeedStreamItem` 增加 `isFavorited`
- 方案 B：新增 `GetFavoritePostIDs()` 由前端自行映射

2. 与现有接口联动：
- `GetFeedStream` / `SearchPosts` 输出可获得收藏状态。
- 收藏列表进入详情时，正文按 CID 回源并写入本机私有缓存。

### DoD
- 前端可在单次列表渲染中显示收藏态，不必逐条调用状态接口。
- 收藏内容缓存不进入共享区统计，不影响公共区配额。

### 验证
- feed/search 返回与 `IsFavorited` 一致。
- 收藏帖详情拉取后，缓存写入私有区且不影响共享区用量。

---

## F5：接口契约与联调收口
### 目标
完成后端契约冻结与联调基线。

### 任务
1. 更新 Wails 绑定（仅后端导出，前端接入可后置）。  
2. 更新文档：
- `docs/FRONTEND_API_REQUIREMENTS.md`
- `docs/NEXT_WATERFALL_PLAN_2026-02-15.md`

3. 补充回归用例：
- 收藏增删查
- 收藏与 feed/search 联动一致性
- 收藏操作跨设备同步一致性
- 收藏详情私有缓存与配额隔离

### DoD
- API 签名冻结，后端回归通过，文档状态同步。

### 验证
- `go test ./...` 通过
- 三节点最小联调通过（同助记词跨设备收藏可同步）

---

## 5. 冻结 API（V1）
- `AddFavorite(postID string) error`
- `RemoveFavorite(postID string) error`
- `GetFavorites(limit int, cursor string) (PostIndexPage, error)`
- `IsFavorited(postID string) (bool, error)`
- `GetFavoritePostIDs() ([]string, error)`（批量态可选）

事件：
- `favorites:updated`（payload: `{ postId: string }` 可选）

内部同步协议（非前端 API）：
- `FAVORITE_OP`
- `FAVORITE_SYNC_REQUEST`
- `FAVORITE_SYNC_RESPONSE`

---

## 6. 非功能与风险
1. 收藏语义是否跨设备同步  
当前定义：跨设备同步（同 `pubkey`）；通过操作日志 + 反熵增量收敛。

2. 收藏列表读取性能  
对策：索引 + limit；后续可做分页 cursor。

3. 与帖子删除/不可见状态冲突  
对策：`GetFavorites` 只返回当前可见帖子，失效收藏可异步清理。

4. 同时多设备操作冲突  
对策：LWW + `op_id` 稳定排序；所有节点按同规则重放。

---

## 7. 简洁优雅优化逻辑（必须遵守）
1. 机制复用
- 收藏同步直接复用帖子索引同步框架（时间窗、批次、反熵调度），不新建并行系统。

2. 数据最小化
- 网络只传 `favorite op` 索引，不传正文/图片，减少复杂度和流量。

3. 读写分离
- 写路径只改状态表 + 追加 op log；读路径只查状态表，避免复杂 join 链。

4. 兼容优先
- 仅新增表/字段/API，不改旧签名；保持向后兼容。

5. 一致性可解释
- 冲突规则固定且可重放（LWW + `op_id`），排查问题不靠“猜”。

6. 存储边界明确
- 收藏命中内容缓存到本机私有区；共享区仅承载公共索引与公共内容。

7. 网络职责单一
- relay 仅负责转发连接/消息，收藏一致性仍由 op log + 反熵收敛负责。

---

## 8. 当前状态（2026-02-16，已推进）
- F1：Done（已新增`post_favorites_state/post_favorite_ops`与索引迁移）
- F2：Done（已实现`AddFavorite/RemoveFavorite`、签名、`FAVORITE_OP`广播、LWW）
- F3：Done（已实现`GetFavorites/IsFavorited/GetFavoritePostIDs`）
- F4：Partial（已提供`favorites:updated`事件与`feed:updated`联动；Feed/Search直接带收藏态后续补）
- F5：In Progress（后端`go test ./...`已通过；前端联调与契约快照更新后收口）

本轮已完成后端收藏主链路与分布式同步闭环，后续按瀑布关口推进前端联调与回归用例补齐。
