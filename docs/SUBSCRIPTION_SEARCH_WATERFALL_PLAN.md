# Aegis 订阅与搜索功能瀑布实现文档

## 1. 目标与范围
本方案用于产品级增强以下能力：
- 节点可订阅指定 `sub`
- 提供“已订阅 sub 列表”
- 当订阅的 `sub` 有新帖子时，向订阅方推送更新事件
- 支持按关键词搜索 `sub` 与帖子
- 提供 `FeedStream` 聚合流：订阅内容 + 推荐内容（先用热度推荐，后续可替换算法）

本方案采用瀑布模式，严格按阶段推进，不跨阶段混做。

---

## 2. 瀑布执行规则（强约束）
1. 严格顺序：S1 -> S2 -> S3 -> S4 -> S5 -> S6，不跳阶段。  
2. 每阶段必须满足：`代码完成 + 验证通过 + 文档更新`。  
3. 阶段内范围冻结：只做本阶段任务，不插入后续需求。  
4. 如需跨阶段调整，必须记录变更单（原因、影响、回归点）。  
5. 流程纪律：先更新文档（需求/API/阶段）再写代码；未补文档视为阶段未完成。

---

## 3. 需求冻结（本轮）
### 3.1 功能需求
- FR-1：节点可订阅/取消订阅某个 `sub`
- FR-2：可查询当前节点订阅的 `sub` 列表
- FR-3：订阅的 `sub` 产生新帖子后，本节点收到推送事件
- FR-4：支持关键词搜索 `sub`
- FR-5：支持关键词搜索帖子（标题/正文）
- FR-6：支持 `FeedStream` 接口，混合返回订阅帖子与推荐帖子
- FR-7：推荐策略可插拔，默认 `hot-v1`，后续可替换

### 3.2 非功能需求
- NFR-1：接口响应稳定，搜索接口支持 `limit` 限流
- NFR-2：推送事件幂等可处理（前端可按 `postId` 去重）
- NFR-3：向后兼容现有数据，不破坏旧数据库启动

---

## 4. 阶段拆解

## S1：数据模型与迁移
### 目标
完成订阅关系的数据落库与索引准备。

### 任务
1. 新增表 `sub_subscriptions`（本地节点维度）：
- `sub_id TEXT PRIMARY KEY`
- `subscribed_at INTEGER NOT NULL`

2. 新增索引：
- `idx_sub_subscriptions_subscribed_at`

3. 启动迁移策略：
- `CREATE TABLE IF NOT EXISTS` 方式兼容历史版本
- 不做破坏性迁移

### DoD
- 老数据库可直接启动，无迁移报错
- 新库自动包含订阅表与索引

### 验证
- 冷启动新库检查表存在
- 用历史库启动后检查表自动补齐

---

## S2：订阅管理 API
### 目标
提供完整订阅管理能力。

### 任务
1. 新增后端 API：
- `SubscribeSub(subId) -> Sub`
- `UnsubscribeSub(subId) -> void`
- `GetSubscribedSubs() -> []Sub`

2. 行为定义：
- `SubscribeSub` 幂等（重复订阅不报错）
- `UnsubscribeSub` 幂等（不存在也可成功返回）
- `subId` 统一走 `normalizeSubID`

3. 事件定义：
- 订阅列表变化时广播：`subs:subscriptions_updated`

### DoD
- 三个 API 可调用且幂等
- 订阅后 `GetSubscribedSubs` 立即可见

### 验证
- 订阅/取消订阅循环执行，结果一致
- 重复调用无异常

---

## S3：订阅推送链路
### 目标
当订阅的 `sub` 出现新帖子时，向本节点推送更新事件。

### 任务
1. 在帖子入库成功后触发订阅判断（含本地发布与网络同步场景）。  
2. 仅当 `post.sub_id` 属于本地订阅集合时，触发事件：`sub:updated`。  
3. 事件载荷建议：
- `subId`
- `postId`
- `title`
- `timestamp`
- `pubkey`

4. 去重策略（消费侧）：
- 前端按 `postId` 去重，避免多路径刷新导致重复提示

### DoD
- 未订阅 sub 不触发推送
- 已订阅 sub 新帖可稳定触发推送

### 验证
- 订阅 `tech`，`tech` 新帖触发事件
- 未订阅 `news`，`news` 新帖不触发

---

## S4：搜索能力（Sub + 帖子）
### 目标
提供可用的关键词搜索接口。

### 任务
1. 新增 API：
- `SearchSubs(keyword, limit) -> []Sub`
- `SearchPosts(keyword, subId, limit) -> []ForumMessage`

2. 匹配规则（V1）：
- `SearchSubs`：匹配 `id/title/description`
- `SearchPosts`：匹配 `title/body`，可选 `subId` 过滤
- 大小写不敏感，空关键词直接返回空结果

3. 限流策略：
- `limit` 默认值（建议 20）
- `limit` 上限（建议 100）

### DoD
- 可通过关键词命中 sub 与帖子
- 查询性能在常见数据量下可接受（本地 SQLite）

### 验证
- 构造多组关键词，验证命中准确性
- 空关键词、超大 limit、无命中场景处理正确

---

## S5：前端接入与联调验收
### 目标
完成产品可见能力闭环。

### 任务
1. 前端接入订阅管理：
- Sub 列表增加“订阅/取消订阅”
- 增加“我的订阅”视图

2. 前端接入推送：
- 监听 `sub:updated`
- 命中当前订阅时展示提醒或自动刷新列表

3. 前端接入搜索：
- 顶部搜索输入框支持选择 `sub`/`帖子`
- 支持 `subId` 范围搜索帖子

4. Wails 绑定同步：
- 更新 `frontend/wailsjs/go/main/App.d.ts`
- 更新 `frontend/wailsjs/go/models.ts`

### DoD
- 用户可完成：订阅 -> 收到更新 -> 搜索定位内容 的全流程
- 端到端三节点联调通过

### 验证
- A/B/C 三节点中任一节点发帖，订阅节点收到推送并可点击进入
- 搜索可定位目标 sub 与帖子

---

## S6：FeedStream 聚合流（后端）
### 目标
提供一个统一 feeds 接口，混合“订阅内容”和“推荐内容”。

### 任务
1. 新增 API：
- `GetFeedStream(limit) -> FeedStream`
- `GetFeedStreamWithStrategy(limit, algorithm) -> FeedStream`

2. 返回模型：
- `FeedStream { items, algorithm, generatedAt }`
- `FeedStreamItem { post, reason, isSubscribed, recommendationScore }`

3. V1 推荐策略：
- 订阅内容优先（默认约 70% 配额）
- 非订阅内容按热度推荐（`score` + 时间衰减）
- 支持策略名 `hot-v1`，并保留分支扩展点

4. 混排策略（V1）：
- 按“2 条订阅 + 1 条推荐”插入
- 按 `postId` 去重

### DoD
- 单接口可返回混合流
- 结果可标记来源（订阅/推荐）
- 策略参数可传入并回显

### 验证
- 存在订阅 sub 时，结果含 `reason=subscribed`
- 存在未订阅高热帖子时，结果含 `reason=recommended_hot`
- `algorithm` 传入后返回值正确

---

## 5. API 草案（冻结）
### 5.1 后端
- `SubscribeSub(subId string) (Sub, error)`
- `UnsubscribeSub(subId string) error`
- `GetSubscribedSubs() ([]Sub, error)`
- `SearchSubs(keyword string, limit int) ([]Sub, error)`
- `SearchPosts(keyword string, subId string, limit int) ([]ForumMessage, error)`
- `GetFeedStream(limit int) (FeedStream, error)`
- `GetFeedStreamWithStrategy(limit int, algorithm string) (FeedStream, error)`

### 5.2 事件
- `subs:subscriptions_updated`
- `sub:updated`

---

## 6. 风险与对策
1. 搜索性能随数据增长下降  
对策：V1 先 `LIKE`；V2 引入 SQLite FTS5。  

2. 推送重复触发  
对策：事件载荷固定 `postId`，消费侧去重。  

3. 订阅语义歧义（本地订阅 vs 全网广播）  
对策：文档明确“订阅仅影响本节点展示与推送，不影响全网数据同步”。  

---

## 7. 里程碑与建议工期
- M1（S1 完成）：0.5 天
- M2（S2 完成）：0.5 天
- M3（S3 完成）：0.5 天
- M4（S4 完成）：0.5 天
- M5（S5 完成）：1 天
- M6（S6 完成）：0.5 天

总计建议：3.5 天（含联调与回归）

---

## 8. 验收清单（最终）
- [ ] 可订阅/取消订阅 sub
- [ ] 可查看订阅 sub 列表
- [ ] 订阅 sub 新帖可触发推送
- [ ] 可按关键词搜索 sub
- [ ] 可按关键词搜索帖子（支持 sub 过滤）
- [ ] `FeedStream` 可混合返回订阅与推荐内容
- [ ] 推荐策略参数可扩展（至少支持 `hot-v1`）
- [ ] 三节点联调通过
- [ ] 文档与接口定义同步更新

---

## 9. 当前执行状态（2026-02-15）
- S1：Done（后端已落地 `sub_subscriptions`）
- S2：Done（后端已落地订阅管理 API）
- S3：Done（后端已落地 `sub:updated` 推送）
- S4：Done（后端已落地 `SearchSubs` / `SearchPosts`）
- S5：Pending（前端未开始）
- S6：Done（后端已落地 `GetFeedStream` / `GetFeedStreamWithStrategy`，默认 `hot-v1`）
