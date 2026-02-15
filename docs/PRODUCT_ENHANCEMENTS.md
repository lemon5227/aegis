# Aegis 产品增强路线（Post-MVP）

## 1. 目标与范围
本文件用于定义 MVP 之后的产品增强方向，重点是做出类似 Reddit 的论坛体验，同时保持 Aegis 的去中心化原则（P2P、Shadow Ban、分区存储）。

执行方法采用瀑布模型：按 A（结构）→ B（互动）→ C（治理与规模化）分阶段推进，前一阶段验收通过后才进入下一阶段。

目标：
- 用户能按 Sub（社区）浏览内容。
- 帖子有主题（标题）+ 正文（内容）结构。
- 用户能对帖子评论，并对评论继续回复（楼中楼）。
- 时间线支持多种排序（最新、热门等）。
- 账户具备可识别的人类资料（昵称、头像）。
- 系统保留可配置能力，便于后续扩展与治理演进。

最终成品目标（产品形态）：
- 体验层面对齐 Reddit / 百度贴吧：分社区、帖子流、评论楼中楼、排序、个人资料与基础治理。

---

## 2. Reddit 化核心能力（优先级）

### P0（先做，形成可感知升级）
1) Sub 社区模型
- 每个帖子必须归属一个 Sub（例如：general、tech、go、web3）。
- 首页改为 Sub 列表 + 当前 Sub 时间线。
- 支持创建 Sub（先本地创建 + 广播元数据）。

2) 帖子结构升级
- 从纯文本帖升级为：
  - title（主题）
  - body（内容）
  - sub（归属社区）
- 列表展示标题，点开看正文。

3) 排序模式
- New：按发布时间倒序。
- Hot（简化版）：按互动分（见后文评分）与时间衰减混合排序。
- Top（可选周期）：按分值排序，支持 Today / Week / All。

4) 用户资料
- 增加 profile：displayName、avatar。
- 保留公钥作为真实身份主键，昵称头像只是展示层。

### P1（建议随后补）
1) 评论树（支持楼中楼回复）
- 用户可对帖子发表评论。
- 用户可对任意评论继续回复（reply_to），形成楼中楼讨论。
- 前端先做 2 层默认展开，深层回复可折叠/按需展开。

2) 互动机制
- Upvote / Downvote（可先只做 Upvote）。
- 收藏、关注 Sub。

3) 内容管理体验
- Sub 规则说明（描述、封面、发帖门槛占位）。
- 帖子编辑/删除（仅作者可操作，本地状态+广播）。

### P2（中期优化）
1) 用户发现
- 搜索帖子/作者/Sub。
- 推荐 Sub（按本地互动偏好）。

2) 资料增强
- 头像来源支持：本地上传、外链、默认生成头像。
- 个人主页（发帖历史、活跃 Sub）。

3) 治理 UX
- 管理日志页（封禁/解封历史）。
- 举报入口（先本地聚合，后续可上链或投票）。

---

## 3. 建议的数据模型扩展

### 3.1 Sub 模型
- sub_id: string
- slug: string（唯一）
- title: string
- description: string
- created_by: pubkey
- created_at: timestamp
- status: active/archived

### 3.2 Post 模型
- post_id: string
- sub_id: string
- author_pubkey: string
- title: string
- body: string
- image_cid: string（可选）
- thumb_cid: string（可选）
- image_mime: string（可选）
- image_size_bytes: number（可选）
- image_width: number（可选）
- image_height: number（可选）
- created_at: timestamp
- score: number
- comment_count: number
- visibility: normal/shadowed

### 3.3 Profile 模型
- pubkey: string
- display_name: string
- avatar_url or avatar_cid: string
- bio: string（可选）
- updated_at: timestamp

### 3.4 Vote 模型（可选）
- voter_pubkey: string
- target_type: post/comment
- target_id: string
- value: +1 / -1
- created_at: timestamp

### 3.5 Comment 模型
- comment_id: string
- post_id: string
- parent_id: string（顶层为空；回复评论时指向上级 comment_id）
- author_pubkey: string
- body: string
- created_at: timestamp
- visibility: normal/shadowed

---

## 4. 排序策略（先实用后精细）

### 4.1 New
- sort_key = created_at desc

### 4.2 Hot（简化）
可用近似公式：
- score = upvotes - downvotes
- age_hours = (now - created_at)/3600
- hot = score / (age_hours + 2)^1.2

说明：
- 该公式实现简单，能避免老帖长期霸榜。
- 后续可加“互动速率”与“Sub 权重”。

### 4.3 Top
- 按 score 排序，支持时间窗口（day/week/month/all）。

---

## 5. 可配置项（模仿 Reddit 且适配 Aegis）

### 5.1 用户级配置
- 默认首页 Sub（如 home / all / 某个自定义 Sub）
- 默认排序（New/Hot/Top）
- NSFW 显示开关（占位）
- 自动刷新频率
- 紧凑视图 / 卡片视图

### 5.2 Sub 级配置
- 是否允许新账号发帖（占位）
- 标题最小/最大长度
- 正文长度上限
- 是否允许链接帖/图片帖（占位）

### 5.3 节点级配置
- 私有区/公共区比例（总和固定 100MB）
- 帖子缓存上限
- 自动连接 peer 上限
- 风控阈值（消息大小、速率）

### 5.4 治理配置
- 创世管理员列表来源（env / 本地文件）
- 封禁生效策略（仅未来帖 / 历史帖也隐藏）
- 解封恢复策略（恢复历史可见性）

---

## 6. 与现有架构的兼容约束

1) 继续保持弱一致
- 不追求全网强一致排序。
- 排序以本地状态 + 最终收敛为主。

2) 保持 Shadow Ban 核心语义
- 被封用户“自己可见，别人不可见”。
- 管理消息必须经过受信管理员校验。

3) 控制复杂度
- 先支持文本帖子与头像 URL。
- 图片/视频与富文本后置。

---

## 6.1 存储扩展路线（节点多、数据多时）

当前 MVP 的 100MB 设计是“本地缓存上限”，不是全网总存储上限。

当节点与数据规模增长后，采用“控制面 + 数据面”分离：

1) 控制面（Gossip）
- 继续广播帖子元数据与治理消息。
- 重点传输：`post_id`、`sub_id`、`author`、`timestamp`、`content_cid`。
- 不再依赖 Gossip 传输大正文。

2) 数据面（分布式存储）
- 正文与附件写入内容寻址存储（CID）。
- 节点按需拉取正文（lazy fetch），本地只缓存热点内容。
- 图片遵循同一规则：列表拉缩略图 CID，详情拉原图 CID。
- 不使用 base64 作为长期存储格式（仅允许前端临时预览）。

3) 本地 100MB 定位
- 私有区 + 公共区依然保留。
- 主要用于：热内容缓存、时间线索引、治理状态。
- 历史冷数据通过 CID 回源，不要求常驻本地。

4) 冗余与可用性
- 为每条内容设置目标副本数（例如 3~5）。
- 副本不足时自动补副本（re-replication）。
- 节点离线不影响全网可用性，只降低局部命中率。

5) 反熵同步
- 节点定期交换索引摘要（我有哪些 post_id/cid）。
- 自动补齐缺失元数据与冷数据。
- 解决离线节点回归后的数据缺口。

6) Shadow Ban 兼容
- 即使正文已在分布式层存在，封禁作者内容仍不进入正常时间线。
- 被封作者“自己可见”语义保留（本机私有区可见）。

### 建议迁移阶段

Storage v1（当前）
- 全量正文可落本地，100MB 双分区淘汰。

Storage v2（下一步）
- 帖子拆分为：索引（本地）+ 正文（CID）。
- Sub 时间线默认只读索引，点开再拉正文。
- 图片拆分为：元数据索引（本地）+ 原图/缩略图（CID）。
- 入库前压缩转码（优先 AVIF/WebP）并做 CID 去重。

Storage v3（规模化）
- 引入副本策略、反熵同步、优先级缓存。
- 针对热门 Sub 做主动预热缓存。

---

## 7. 建议实施节奏（3 个迭代）

### Iteration A（Reddit 形态成型，约 1-2 周）
- Sub 模型 + Sub 切换 UI
- 帖子 title/body/sub 化
- New/Hot 排序
- Profile（昵称+头像）

### Iteration B（互动与留存，约 1 周）
- Upvote/Downvote
- 评论树（支持回复评论，楼中楼）
- 收藏与关注 Sub

### Iteration C（治理与配置产品化，约 1 周）
- 管理日志页
- 配置中心（用户/节点/治理）
- 封禁策略可配置（是否隐藏历史）

### 7.1 阶段门禁（瀑布）
- 只有当前迭代验收通过，才允许进入下一迭代。
- 不允许跨迭代提前实现功能（例如在 Iteration A 插入评论或投票）。
- 每迭代结束必须同步更新任务清单状态与验收记录。

---

## 8. 验收标准（增强版）

1) Sub 浏览可用
- 用户可创建/进入 Sub。
- 不同 Sub 时间线相互隔离。

2) 排序可感知
- 同一 Sub 下 New 与 Hot 顺序明显不同。

3) 资料可识别
- 昵称头像可跨重启保持。

4) 治理一致性可验证
- 创世节点封禁后，其他受信节点视图一致。
- 解封后恢复策略符合配置。

5) 性能与稳定性
- 三节点持续发帖 10 分钟不崩溃。
- 公共区配额与淘汰正常。

---

## 9. 下一步建议
从产品收益与实现成本看，优先做：
1) Sub + title/body（立刻提升论坛结构）
2) New/Hot 排序（立刻提升可读性）
3) displayName/avatar（立刻提升社区感）

完成这三项后，Aegis 就会从“技术验证论坛”进入“可长期使用社区产品”的阶段。

---

## 10. 执行清单入口
为便于逐步落地实现，请按可执行任务清单推进：

- `docs/ENHANCEMENT_TASKS.md`

该清单已按阶段拆分为：目标、任务、DoD、验证步骤，并给出推荐执行顺序。

---

## 11. 执行状态同步（2026-02-15）

为避免“后续不知道做了啥”，本节记录当前真实执行轨迹（以 `docs/ENHANCEMENT_TASKS.md` 为准）：

- 已完成：A1、A4
- 进行中：A3、B1
- 已完成（本轮）：B2、B3
- Auto-Verified（待手工联调）：C1（治理日志 + 策略配置 + 状态提示 + 一致性修复）
- 进行中（本轮新增）：C2（Storage v2 首批：content_cid + 索引/正文分离读取 API）

### 11.1 顺序调整记录（已发生）
- B 阶段按产品决策调整为：B1 评论（楼中楼）→ B2 统一投票（帖子+评论）→ B3 Hot 稳定化。
- 该顺序用于先保证讨论能力，再统一互动信号，最后校准热门排序。

### 11.2 C1 提前启动记录
- 变更单：`CR-2026-02-14-C1-EarlyStart`
- 原因：治理产品化需要尽早建立可追溯界面与规则，减少后续认知断层。
- 约束：提前启动不豁免 B2/B3 的 DoD；投票一致性与 Hot 稳定化仍必须补齐验收。
- 最新决议（2026-02-14）：已恢复严格瀑布顺序；B2/B3 已完成，下一步进入 C1。

### 11.4 C1 本轮落地范围（第一版）
- 后端：新增治理日志表与查询 API（`GetModerationLogs`）。
- 后端：新增治理策略配置 API（`GetGovernancePolicy` / `SetGovernancePolicy`）。
- 策略生效：`SHADOW_BAN` 支持“是否隐藏历史帖”开关，`UNBAN` 恢复历史可见性。
- 前端：新增治理策略开关、治理日志列表、治理消息成功/失败状态提示。
- 验证脚本：新增 `scripts/test_phase_c1.sh` 用于 C1 回归与基线守护。

### 11.5 C1 一致性修复与门禁结果（2026-02-15）
- 治理乱序保护：`moderation` 更新改为按时间戳单调生效，旧消息不会覆盖新状态。
- 离线治理语义收敛：未启动 P2P 时，治理广播请求直接拒绝，不落 `moderation`/`moderation_logs(applied)`。
- 自动化门禁结果：`bash scripts/test_phase_c1.sh` 通过；重点用例 `TestC1PublishGovernanceWithoutP2PRejectedAndNoLog` 与 `TestC1ModerationIgnoresOlderOutOfOrderUpdates` 通过。
- 当前出口结论：C1 达到 Auto-Verified，待三节点手工联调通过后标记为 Done。

### 11.6 C2 首批实现进展（2026-02-15）
- 数据层：`messages` 新增 `content_cid`，新增 `content_blobs` 表（`content_cid/body/size_bytes/created_at`），并提供历史数据回填。
- 写入链路：本地发帖/网络收帖统一生成或接收 `content_cid`，正文写入 `content_blobs`，索引写入 `messages`。
- 读取链路：新增 `GetFeedIndexBySubSorted`（索引列表）与 `GetPostBodyByCID` / `GetPostBodyByID`（按 CID/帖子按需取正文）。
- 前端链路：公共 Feed 改为索引列表渲染；选中帖子后按 `post_id` 拉取正文，符合“列表读索引、详情读正文”。
- 体验改进：新增正文本地缓存（避免重复拉取）与正文加载状态提示（降低误判为空内容）。
- LRU 策略：缓存淘汰仅作用于 `content_blobs`，不删除 `messages` 索引，因此帖子不会从时间线消失。
- 存储统计：`GetStorageUsage` 改为按 blob 实际占用统计（而非消息索引行大小）。
- 自动化验证：新增 `phase_c2_storage_test.go`，用例 `TestC2PostStoresContentCIDAndBlob`、`TestC2IndexFeedAndBodyFetchByPostID` 已通过。
- 额外验证：`TestC2BlobLRUEvictKeepsMessageIndexes` 通过，确认 LRU 仅影响正文命中不影响帖子可见性。

### 11.3 UI 适配层策略（同步）
- 当前 UI 作为测试/联调面板，后续可替换。
- `UI 适配层` 定级为后置重构项（建议 C 阶段末实施），避免骨架期反复返工。
