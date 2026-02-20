# Lamport 顺序一致性 + Shadow Ban 存储治理落地计划

日期：2026-02-18

## 1. 背景与问题

当前系统的跨节点一致性主要依赖物理时间戳（`timestamp`）和增量反熵窗口。该方案在以下场景存在风险：

1. **治理顺序语义不稳**
   - “仅对未来内容生效”的策略在分布式乱序传播下可能出现判定偏差。
   - 根因：事件先后仅用 wall-clock 推断，不具备严格因果顺序。

2. **Shadow Ban 存储治理缺口**
   - 被 shadow ban 节点的内容不应由其他正常节点承担分布式存储成本。
   - 若不限制，恶意节点可通过垃圾评论/帖子消耗其他节点存储，并在治理策略切换后造成历史污染。

本计划将两件事合并治理：

- 引入 Lamport 逻辑时钟用于**因果顺序一致性**。
- 冻结 Shadow Ban 存储策略：**被 ban 节点内容仅本地私有保存，不参与他人分布式存储**。

---

## 2. 目标与非目标

### 2.1 目标

1. 在多节点乱序传播条件下，治理“未来/历史”语义可稳定复现。
2. 被 shadow ban 节点的新增内容默认不进入其他节点共享存储。
3. 不破坏现有帖子反熵主链路（稳定优先，渐进升级）。

### 2.2 非目标

1. 不在第一阶段重构全部同步协议。
2. 不在第一阶段引入复杂 CRDT（Lamport + deterministic tie-break 即可）。
3. 不改变现有前端主交互流程。

---

## 3. 核心设计决策

## 3.1 顺序模型：Lamport 优先，时间戳降级

每条治理/帖子/评论消息新增 `lamport` 字段（兼容可选）：

- 发送时：`localClock = localClock + 1`，写入 `lamport`。
- 接收时：`localClock = max(localClock, incomingLamport) + 1`。
- 排序比较：
  1. 先比 `lamport`
  2. 再用稳定 tie-break（建议 `messageID` 字典序）

兼容策略：

- 旧节点消息（无 `lamport`）按桥接规则赋值（见第 6 节）。

## 3.2 Shadow Ban 存储策略（冻结）

对“被 shadow ban 节点内容”的存储与传播规则统一为：

1. **本机作者视角**：可保留本地私有数据（用于自证与审计）。
2. **其他节点视角**：
   - 不写入共享索引（`messages/comments` 公共可见域）。
   - 不落地正文/媒体 blob 到共享缓存。
   - 不参与反熵摘要输出，不作为回源副本。

这条策略与 `hideHistoryOnShadowBan` 的关系：

- `hideHistoryOnShadowBan = true`：历史+未来都对他人不可见。
- `hideHistoryOnShadowBan = false`：仅“治理事件之后”的内容不可见。
- 无论 true/false，**新增被 ban 内容都不应成为其他节点的共享副本**（防存储攻击）。

---

## 4. 数据模型变更（加法兼容）

建议新增/扩展字段：

1. `messages.lamport INTEGER NOT NULL DEFAULT 0`
2. `comments.lamport INTEGER NOT NULL DEFAULT 0`
3. `moderation.lamport INTEGER NOT NULL DEFAULT 0`
4. `moderation_logs.lamport INTEGER NOT NULL DEFAULT 0`
5. 新增本地时钟表：
   - `logical_clock(scope TEXT PRIMARY KEY, value INTEGER NOT NULL, updated_at INTEGER NOT NULL)`
   - 初期 `scope='global'`

索引建议：

- `idx_messages_lamport`、`idx_comments_lamport`、`idx_moderation_lamport`

---

## 5. 治理判定规则（统一函数）

新增统一判定函数（概念）：

`shouldAcceptContent(pubkey, contentLamport, contentTs, viewerPubkey)`

返回：`accept/reject` + 原因

判定步骤：

1. 若 `viewerPubkey == pubkey`，允许（本人视图）。
2. 读取目标用户最新治理状态（含 `action`, `banLamport`, `banTimestamp`）。
3. 若未 ban，允许。
4. 若 ban：
   - 策略 true：拒绝。
   - 策略 false：
     - 优先比较 `contentLamport >= banLamport` 拒绝。
     - 无 lamport 时退化为 `contentTs >= banTimestamp`。

说明：

- 同步入站、查询过滤、反熵输出都必须复用该判定，不允许各自实现。

---

## 6. 兼容与迁移（零停机）

## 6.1 协议兼容

`lamport` 字段为可选；旧节点忽略新字段，新节点可处理旧消息。

## 6.2 历史数据桥接

对 `lamport=0` 的历史记录采用桥接：

- 读取时临时比较：
  - 优先 lamport（>0）
  - 否则降级 timestamp
- 可选后台任务按 `timestamp,id` 回填近似 lamport（非阻塞）。

## 6.3 发布策略

1. 先上线“读兼容”
2. 再上线“写 lamport”
3. 最后切换治理判定优先 lamport

---

## 7. 反熵与回源策略调整

## 7.1 帖子/评论反熵输出

- 在摘要构建阶段即应用治理判定。
- 被 ban 内容不进入对外摘要。

## 7.2 回源请求处理

- 接收到对被 ban 内容的回源请求时可拒绝（not found / policy blocked）。
- 防止把 ban 内容再次扩散为共享副本。

## 7.3 本地存储隔离

- ban 作者本地私有内容与共享缓存逻辑分离。
- 配额统计中将此类数据记入私有域，不计入共享可回源集合。

---

## 8. 分阶段实施（按 A 阶段推进）

## A1（P0）：治理判定统一化（不引入 lamport）

1. 抽取 `shouldAcceptContent` 统一函数（基于当前 timestamp + 策略）。
2. 应用于：
   - 入站 `POST/COMMENT`
   - `GetCommentsByPost` / feed 查询过滤
   - 反熵摘要输出
3. 目标：先消除策略分叉与查询/入站不一致。

DoD：

- `hideHistory=true/false` 在两节点测试下行为稳定。

## A2（P1）：引入 Lamport（写入 + 传输）

1. DB 加法字段与逻辑时钟表。
2. 出站消息写入 `lamport`。
3. 入站消息更新本地逻辑时钟。

DoD：

- 新旧节点混部不崩；新节点之间 lamport 连续增长。

## A3（P1）：治理判定切换为 Lamport 优先

1. `shouldAcceptContent` 先比 lamport，缺失时降级 timestamp。
2. 统一 tie-break 规则。

DoD：

- 乱序注入测试下，“未来生效”判定稳定。

## A4（P1）：存储治理封口

1. ban 内容不进入共享反熵摘要。
2. ban 内容不作为回源副本。
3. 配额统计区分私有与共享责任。

DoD：

- 恶意 ban 节点高频发文不会抬升其他节点共享存储。

---

## 9. 验证矩阵

1. **治理顺序一致性**
   - 同一用户：先评论后 ban、先 ban 后评论、乱序到达。
2. **策略开关一致性**
   - `hideHistory=true/false` 双模式回归。
3. **存储防滥用**
   - ban 节点高频发文，其他节点共享 blob/索引不显著增长。
4. **混部兼容**
   - 新旧协议节点混跑，帖子主同步不回归。

---

## 10. 风险与回滚

风险：

1. lamport 字段引入后兼容处理不完整导致排序异常。
2. 治理过滤过严导致误隐藏。

回滚策略：

1. 保留 timestamp 路径作为降级分支。
2. 通过配置开关回退到“仅 timestamp 判定”。
3. 任何阶段回滚不做破坏性迁移（仅加法字段）。

---

## 11. 当前状态

- A1：Done（治理判定入口统一，入站/查询/反熵链路复用统一函数）。
- A2：Done（`messages/comments/moderation/moderation_logs` 已写入 Lamport，逻辑时钟表与出入站推进已接入）。
- A3：Done（治理判定已切换 Lamport 优先，timestamp 为兼容回退，含稳定 tie-break）。
- A4：Done（ban 内容不进入共享摘要，不作为回源副本，配额统计区分共享/私有责任）。

### 11.1 已完成补充
1. 评论媒体已纳入 A4 存储治理判定（comment attachment -> media serve policy）。
2. `hideHistoryOnShadowBan=true/false` 两种策略均已接入 Lamport-first 过滤路径。
3. 回源侧对策略拒绝统一返回 not found/policy blocked 语义。

### 11.2 剩余收口
1. 三节点长时回归矩阵补齐并归档（乱序传播 + 首次回源延迟场景）。
2. 文档化保留：将验证结果同步到 release/runbook 文档。
