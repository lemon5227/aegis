# Aegis 增强实现任务清单（可执行版）

## 使用方式
- 新阶段默认主文档：`docs/NEXT_WATERFALL_PLAN_2026-02-15.md`（N 系列）。
- 按阶段顺序执行，不要跳阶段。
- 每阶段都包含：目标、任务、完成标准（DoD）、验证步骤。
- 每完成一阶段，建议做一次里程碑提交。
- 发布就绪（正式产品）请同时参考：`docs/RELEASE_WATERFALL_PLAN.md`

## 当前阶段状态
- A1 Sub：Done
- A2 帖子结构：Auto-Verified（待三节点手工清单勾选后置为 Done）
- A3 排序：In Progress（默认 Hot + New 切换，Hot 评分先占位）
- A4 Profile：Done
- B1 评论（楼中楼）：In Progress
- B2 统一投票（帖子+评论）：Done
- B3 Hot 评分稳定化：Done
- C1 治理产品化：Auto-Verified（自动化门禁通过，待三节点手工联调后置为 Done）
- C2 存储升级：In Progress（已完成 content_cid、索引列表 API、详情按 CID 取正文、前端正文缓存）
- D1 订阅与搜索（Sub Subscription + Search）：In Progress（后端 S1/S2/S3/S4 已完成，前端 S5 待开始）
- D2 FeedStream 聚合流：In Progress（后端 S6 已完成，策略扩展与前端接入待完成）

## 变更单（执行偏差记录）
### CR-2026-02-14-C1-EarlyStart
- 变更原因：业务侧要求先启动治理产品化，避免后续“做了什么”不可追溯。
- 偏差说明：在 B2/B3 未完全 Done 前，提前进入 C1 设计与骨架实现。
- 影响范围：`docs/ENHANCEMENT_TASKS.md`、`docs/PRODUCT_ENHANCEMENTS.md`、后续 C1 代码与联调计划。
- 回归要求：B2 验收（统一投票一致性）与 B3 验收（Hot 稳定化）仍需补齐并保留回归测试。
- 当前结论（2026-02-14）：已按“瀑布模型优先”要求暂停 C1，恢复至 B2 → B3 → C1 顺序执行。

## C1 启动清单（本轮）
- [x] C1 数据结构与 API 清单冻结（管理日志/策略配置/状态提示）。
- [x] 明确治理日志最小字段：`action/target/source/reason/timestamp/result`。
- [x] 明确封禁策略最小开关：`hide_future_only` / `hide_history_too`。
- [x] 明确前端最小视图：治理日志列表 + 操作结果提示。
- [x] 联调验证脚本补充到 C1 验收段。

## C2 启动清单（本轮）
- [x] 数据模型补充 `content_cid` 与 `content_blobs`。
- [x] 历史数据回填：旧帖自动补 `content_cid` 并写入正文 blob。
- [x] 后端新增索引读取 API：`GetFeedIndexBySubSorted`。
- [x] 后端新增正文按需读取 API：`GetPostBodyByCID` / `GetPostBodyByID`。
- [x] 前端列表改读索引、详情按需拉正文并增加本地缓存与加载状态。
- [x] 正文缓存引入 LRU 淘汰（仅淘汰 `content_blobs`，不删除 `messages` 索引，避免帖子从时间线消失）。

## 瀑布执行规则（强约束）
1. 严格阶段顺序
- 必须按 A1 → A2 → A3 → A4 → B1 → B2 → B3 → C1 → C2 执行。
- 未通过当前阶段 DoD，不允许进入下一阶段。

2. 阶段范围冻结（Scope Freeze）
- 当前阶段只允许实现本阶段任务，不允许混入后续阶段功能。
- 发现新需求时，只记录到后续阶段，不在当前阶段临时插入。

3. 设计先于实现
- 每阶段开始前先确认：数据模型、协议字段、API 清单、UI 清单。
- 阶段内不做接口来回改名，避免返工。
- 帖子图片按“索引与正文分离”设计：数据库只存元数据与 CID，不落库 base64 正文。

4. 出口门禁（Exit Gate）
- 必须满足：代码实现完成 + 三节点联调通过 + 文档状态更新。
- 通过后将该阶段标记为 Done，下一阶段才可开工。

5. 变更控制
- 若需要修改已完成阶段，必须创建“变更单”（原因、影响范围、回归点）。
- 变更单通过后再回改，避免流程失控。

6. 最终产品目标对齐
- A 阶段完成“内容结构”；B 阶段完成“互动讨论”；C 阶段完成“治理与规模化”。
- 最终成品形态对齐 Reddit/百度贴吧：分社区、帖子、评论楼中楼、排序、用户资料、治理可运营。

---

## 阶段门禁模板（每阶段都执行）
### 阶段入口（Entry Criteria）
- 上一阶段已 Done。
- 本阶段任务与 API 清单已确认。
- 本阶段验收用例已准备。

### 阶段出口（Exit Criteria）
- 本阶段 DoD 全部满足。
- 三节点手工联调通过。
- `docs/PRODUCT_ENHANCEMENTS.md` 与本清单状态已更新。

---

## Phase A1：Sub 基础能力（先做）

### 目标
让论坛从“单一时间线”升级为“多 Sub 时间线”。

### 任务
1. 数据层
- 新建 `subs` 表：`sub_id/slug/title/description/created_by/created_at/status`
- `messages/posts` 表新增 `sub_id`
- 为 `sub_id + timestamp` 建索引

2. 协议层
- 新增消息类型：`SUB_CREATE`
- `POST` 消息增加 `sub_id`

3. 后端 API
- `CreateSub(slug,title,description)`
- `GetSubs()`
- `GetFeedBySub(subId, sortMode)`（先支持 `new`）

4. 前端 UI
- 左侧 Sub 列表
- 顶部当前 Sub 切换器
- 发帖时选择 Sub（默认 `general`）

### DoD
- 三节点联调时，创建的 Sub 在其他节点可见。
- 在 `subA` 发帖，不会出现在 `subB` 时间线。

### 验证
- 节点 A 创建 `sub=go`，节点 B 刷新后可见。
- A 在 `go` 发帖，B 切换到 `go` 能看到，切到 `general` 看不到。

---

## Phase A2：帖子结构升级（title + body）

### 目标
让帖子具备 Reddit 风格主题与正文结构。

### 任务
1. 数据模型
- `posts` 字段拆分：`title/body`（替代纯 content）
- 保留兼容读取：旧数据可映射为 `title=前20字`，`body=全文`

2. 协议层
- `POST` 消息携带 `title/body/sub_id`

3. 前端 UI
- 列表显示标题 + 摘要 + Sub 标签
- 点开帖子显示正文

### DoD
- 新发帖必须包含标题。
- 列表/详情展示一致。

### 验证
- 跨节点发帖后，标题和正文均正确同步。

---

## Phase A3：排序（New / Hot 预留）

### 目标
先完成可用排序框架：New 落地、Hot 先预留。

### 任务
1. 后端排序
- `new`: `created_at DESC`
- `hot`: 先预留接口（公式在 B1 启用）

2. 数据层
- `posts` 增加 `score` 字段（初始 0）

3. 前端
- Sub 页排序切换：`New/Hot`
- 排序切换后即时刷新列表

### DoD
- `new` 排序可用且稳定。
- `hot` 切换入口可用（B1 前允许与 `new` 接近）。

### 验证
- 人工构造 2 条不同时间帖子，验证 `new` 顺序正确。

---

## Phase A4：Profile（昵称 + 头像）

### 目标
让账户具备人类可识别信息。

### 任务
1. 数据层
- 新建 `profiles` 表：`pubkey/display_name/avatar_url/updated_at`

2. 后端 API
- `UpdateProfile(displayName, avatarURL)`
- `GetProfile(pubkey)`

3. 前端
- 个人资料编辑页
- 发帖列表显示昵称与头像

### DoD
- 资料跨重启保留。
- 其他节点可看到昵称头像（同步后）。

### 验证
- A 修改昵称头像，B 刷新后可见。

---

## Phase B1：评论（支持楼中楼回复）

### 目标
补齐论坛核心讨论能力，支持对帖子评论、对评论继续回复。

### 任务
- `comments` 表：`comment_id/post_id/parent_id/body/author/timestamp`
- 协议增加 `COMMENT` 消息（包含 `post_id/parent_id/body`）
- 支持评论回复评论（`parent_id` 指向 comment_id）
- 前端先做两层默认展开，深层回复可折叠展示
- UI：帖子详情页评论树
- 刷新机制采用“双层策略”：
  - 全局广播层：评论消息继续全网广播，保证所有节点最终可见（不依赖是否参与回复）
  - 当前帖订阅层：前端维护当前打开的 `postId`，仅在命中该帖时增量刷新评论
- 增加事件约定：`comments:updated`（携带 `postId`），用于评论区精准刷新，避免全量刷新 Dashboard

### DoD
- 用户可对帖子发评论，也可对评论继续回复。
- 评论可跨节点同步，父子关系与层级展示正确。
- 当前正在查看某帖时，收到该帖新评论可自动刷新（无需手动点击 Refresh Dashboard）。
- 未打开该帖的用户不强制拉全量评论，但在进入帖子详情时能看到最新评论（最终一致）。

### 验证
- A 对帖子发顶层评论，B/C 可见。
- B 对 A 的评论回复，A/C 可见且层级正确。
- C 再对该回复继续回复，A/B 可见。
- A 正在查看帖子详情，B 发表新回复后，A 端评论区自动出现新回复。
- C 未打开该帖时不触发评论全量拉取；C 打开该帖详情后可看到最新评论。

---

## Phase B2：统一投票（帖子 + 评论）

### 目标
在评论树可用后，一次性落地帖子与评论两类投票能力。

### 任务
- `post_votes` 与 `comment_votes` 表（先仅 `+1`）
- API：`UpvotePost(postId)`、`UpvoteComment(commentId)`
- 防重复：同一用户对同一对象只能投一次
- UI：帖子与评论都提供投票按钮和计数

### DoD
- 两类投票都可跨节点同步并最终一致。
- 同一用户重复投票被正确拦截。

### 验证
- A/B/C 分别对同一帖子和同一评论投票，计数在各节点一致。
- 重复投票不重复计数。

---

## Phase B3：Hot 评分稳定化

### 目标
在统一投票上线后，基于真实互动信号启用并稳定 Hot 排序。

### 任务
- 启用并校准 `hot` 排序公式：`hot = score / (age_hours + 2)^1.2`
- 验证不同时间与票数组合下排序可解释、可复现
- 增加最小回归测试覆盖 `new/hot` 差异

### DoD
- `hot` 与 `new` 在典型样本下呈现可观察差异。
- 多节点下 `hot` 排序最终收敛。

### 验证
- 构造不同时间和分数帖子，`hot` 顺序符合预期。

---

## Phase C1：治理产品化

### 目标
把治理从“能用”提升到“可运营”。

### 任务
- 管理日志页（封禁/解封记录）
- 封禁策略配置：
  - 仅隐藏未来帖
  - 同时隐藏历史帖
- 治理消息状态提示（成功/拒绝原因）

### DoD
- 管理员可追踪每次治理动作与结果。

### 验证
1. 自动化回归（本地）
- 运行：`bash scripts/test_phase_c1.sh`
- 预期：C1 治理策略/日志测试通过，同时 B2/B3 与 LAN 基线测试不回归。

2. 三节点手工联调
- A 节点切换治理策略 `Hide History On Shadow Ban` 为 OFF，对 B 执行 `SHADOW_BAN`。
- 预期：B 的历史帖仍可在其他节点时间线看到（仅未来帖受影响）。
- A 节点切换策略为 ON，再次对 B 执行 `SHADOW_BAN`。
- 预期：B 的历史帖在其他节点时间线被隐藏；`UNBAN` 后恢复可见。
- 在任意节点刷新后，`Moderation Logs` 面板可看到动作记录，字段包含 action/target/source/reason/result。

---

## Phase C2：存储升级 v2（索引 + CID）

### 目标
为大规模数据做准备。

### 任务
1. 模型调整
- `posts` 存索引字段 + `content_cid`
- 正文迁移到内容寻址层（先接口预留）
- 新增图片索引字段：`image_cid/mime/size_bytes/width/height/thumb_cid`
- 数据库仅存图片元数据，不存 base64 正文

2. 拉取策略
- 列表只读索引
- 打开详情时按 CID 拉正文
- 列表优先加载 `thumb_cid`，详情页按需加载原图 `image_cid`

3. 缓存策略
- 本地 100MB 作为热缓存
- 热图缓存优先保留缩略图，原图按 LRU 回收

4. 媒体去重与压缩
- 以内容哈希/CID 去重，相同图片只存一份
- 入库前统一压缩（优先 AVIF/WebP，失败回退原格式）

### DoD
- 即使正文不在本地，用户打开帖子能自动回源。
- 帖子图片在三节点可见且不使用 base64 落库。
- 同图重复发布时，存储占用不线性增长（命中去重）。

---

## 跨阶段通用任务（每阶段都做）

1. 自动化验证
- 保持 `scripts/test_phase3.sh` 可运行。
- 每阶段新增最小回归测试。

2. 观测与排障
- 统一错误提示：连接失败、权限拒绝、同步延迟。
- 关键事件日志：收帖、封禁、同步补齐。

3. 文档同步
- 阶段完成后更新：
  - `docs/PRODUCT_ENHANCEMENTS.md`
  - 当前阶段实现状态（Done/In Progress/Todo）

---

## 建议执行顺序（严格）
1. A1 Sub
2. A2 帖子结构
3. A3 排序
4. A4 Profile
5. B1 评论（楼中楼）
6. B2 统一投票（帖子+评论）
7. B3 Hot 评分稳定化
8. C1 治理产品化
9. C2 存储升级

这套顺序的原则是：先“信息结构”，再“互动反馈”，最后“规模化与治理产品化”。

---

## A2 验收清单（本轮执行）
- [ ] 三节点发帖时必须输入 `title + body`
- [ ] 节点 A 发布后，B/C 能同步看到相同 `title + body`
- [ ] 列表显示标题与摘要，点选后展示正文
- [ ] 同一帖子在不同节点内容一致（无旧 content 回退路径）

## A4 验收清单（本轮执行）
- [x] 节点 A 设置 `displayName/avatarURL` 后可成功保存并本地立即可见
- [x] 节点 B/C 刷新后，A 的帖子作者显示为新昵称
- [x] 选中帖子详情时，作者头像可正确加载（有 URL 时）
- [x] 重启任一节点后，已保存资料可从本地恢复
