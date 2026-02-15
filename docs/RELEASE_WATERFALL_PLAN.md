# Aegis 正式产品发布瀑布计划（Release Waterfall）

## 1. 现状结论（基于当前 C2 进度）
当前版本已达到“可运行的 C2 存储升级阶段”，但尚未达到正式产品发布标准。

已完成（可用但非最终）：
- 索引/正文分离（`messages` + `content_blobs`）
- 列表读索引、详情按需读取正文
- 正文缓存与 LRU（仅淘汰 blob，不删索引）
- C1 治理一致性修复与自动化门禁

关键缺口（必须补齐后才可发布）：
- 缺少真正“网络回源”（本地 miss 时向 peers 拉取 `content_cid`）
- 缺少反熵同步（离线节点回归后自动补齐索引与正文）
- 缺少图片/媒体链路（CID、缩略图、压缩、去重）
- 缺少互联网连通能力（NAT 穿透/中继/公网引导节点）
- 缺少发布级观测与告警（回源失败率、同步延迟、缓存命中率）
- 缺少发布级稳定性门禁（长稳压测、故障恢复、迁移回滚）

---

## 2. 瀑布执行规则（发布版）
1) 严格顺序：R1 → R2 → R3 → R4 → R5 → R6，不跳阶段。
2) 入口条件：上一阶段 Done + 本阶段 API/协议冻结。
3) 出口条件：自动化通过 + 三节点手工联调通过 + 文档同步。
4) 变更控制：阶段内发现新需求只记录到后续阶段，不插入当前阶段。

发布 UX 硬约束：
- 不要求用户手动点击“Start P2P”。
- 应用启动后自动进入“可连接/可同步”状态（失败可见、可重试，但默认自动启动）。
- 本地多实例调试时，端口冲突需自动回退到可用端口，避免要求用户手动改端口。

---

## 3. 发布阶段分解

## R1：网络回源（必须先做）
### 目标
本地正文缺失时，可从网络按 `content_cid` 自动拉取并回填本地。

### 任务
- 新增 P2P 内容请求/响应协议：
  - `CONTENT_FETCH_REQUEST { content_cid, request_id }`
  - `CONTENT_FETCH_RESPONSE { content_cid, body, size_bytes, found, request_id }`
- 本地 `GetPostBodyByCID` 逻辑改为：
  - 先本地查；miss 时触发网络回源；成功后落 `content_blobs` 再返回。
- 增加超时、重试、并发去重（同一 `content_cid` 只发一次网络请求）。

### DoD
- 任何节点只要有索引且网络中存在正文副本，详情可自动打开。

### 验证
- A 发布帖子，B 删除本地该 blob 后点击详情，能自动回源成功。

---

## R2：反熵同步（离线回归一致性）
### 目标
节点离线后重连，能自动补齐缺失索引与正文，避免“看得到标题看不到正文”。

### 任务
- 摘要交换（例如按时间窗/批次交换 `post_id/content_cid` 列表）。
- 缺口补齐流程：缺啥补啥（先索引后正文）。
- 后台周期任务：定时补齐 + 失败重试。

### DoD
- 离线节点回归 5 分钟内，关键内容可收敛。

### 验证
- B 断网期间 A 连续发帖，B 重连后自动补齐。

---

## R3：媒体链路（图片最小闭环）
### 目标
支持图片帖正式可用，不依赖 base64 落库。

### 任务
- 模型落地：`image_cid/thumb_cid/mime/size/width/height`。
- 上传前压缩与格式策略（优先 WebP/AVIF，失败回退）。
- 同图去重（哈希/CID 相同只存一份）。
- 列表优先缩略图，详情按需原图。

### DoD
- 图片在三节点可见；重复上传占用不线性增长。

### 验证
- 同图重复发帖，存储增长明显低于线性。

---

## R4：互联网连通与安全（LAN -> WAN）
### 目标
节点不仅能在局域网发现彼此，还能在公网稳定互联并具备基础抗滥用能力。

### 任务
- 零手动启动：
  - 应用启动自动拉起 P2P（端口与引导节点可配置）。
  - UI 仅保留诊断与重连能力，不把“手动启动网络”作为主路径。
- 连通能力：
  - 引入公网 bootstrap 节点（多地域，至少 3 个）
  - 启用 NAT 检测与打洞（AutoNAT / hole punching）
  - 打洞失败时自动回退 relay v2 中继
- 地址与发现：
  - 节点上报可达地址策略（避免仅上报内网地址）
  - 区分 LAN mDNS 与 WAN 引导发现通道
- 安全与防滥用：
  - 连接级限速、请求级限流（含 `CONTENT_FETCH`）
  - 黑名单/灰名单策略与短时封禁
  - 基础资源防护（单 peer 并发/带宽上限）

### DoD
- 新用户首次打开应用，不做手动网络操作即可看到节点进入联网状态（或明确失败原因）。
- 两个不同公网网络下的节点可互联、可同步、可回源。
- 在打洞失败场景可自动通过 relay 回退并保持可用。

### 验证
- 使用两台不同公网环境主机做端到端联调。
- 人工模拟 NAT 严格场景，验证 relay 回退成功。

---

## R5：发布级稳定性与观测
### 目标
具备可运维能力，出现问题能定位、能告警、能恢复。

### 任务
- 指标：
  - `content_fetch_success_rate`
  - `content_fetch_latency_p95`
  - `blob_cache_hit_rate`
  - `sync_lag_seconds`
- 结构化日志：回源请求、失败原因、重试次数。
- 关键告警阈值与手工排障手册。

### DoD
- 能在 10 分钟内定位“详情打不开”根因。

### 验证
- 人为制造回源失败，日志和指标可准确反映。

---

## R6：发布门禁与灰度
### 目标
建立可重复发布流程，降低线上事故概率。

### 任务
- 测试门禁：
  - 单测 + 三节点集成测试 + 回归脚本
  - 长稳测试（持续发帖/评论/回源）
- 数据迁移门禁：
  - 迁移演练
  - 回滚脚本
- 灰度发布：
  - 单节点灰度 -> 小规模 -> 全量

### DoD
- 连续两轮灰度无阻断问题，才允许全量。

### 验证
- 按灰度脚本执行一次完整演练并留档。

---

## 4. 建议里程碑（简版）
- M1（R1 完成）：正文“本地 miss 可网络回源”
- M2（R2 完成）：离线节点回归自动补齐
- M3（R3 完成）：图片链路可用
- M4（R4 完成）：具备公网互联与 relay 回退能力
- M5（R5 完成）：发布级可观测
- M6（R6 完成）：具备正式发布条件

---

## 5. 与现有任务清单关系
- `docs/ENHANCEMENT_TASKS.md` 继续记录 A/B/C 功能演进状态。
- 本文档用于“从可运行到可发布”的发布瀑布计划。
- 两份文档同时维护：功能进度（Enhancement）+ 发布就绪（Release）。

---

## 6. 当前执行状态（2026-02-15）

### R1（网络回源）
- 状态：Done
- 已完成：
  - `CONTENT_FETCH_REQUEST/RESPONSE` 协议落地
  - `GetPostBodyByCID` 本地 miss 自动触发网络请求
  - 回源成功后自动回填 `content_blobs`
  - 请求去重与并发控制（同一 CID 防重复风暴）
  - 更细粒度失败原因（超时/无 peers/无副本）
  - UI 侧默认自动请求与自动重试，不暴露技术型重试按钮
  - 两节点与失败场景测试通过：`TestR1ContentFetchFromPeerOnLocalMiss` / `TestR1ContentFetchNoPeers` / `TestR1ContentFetchTimeout`

### R2（反熵同步）
- 状态：Done
- 已完成：
  - `SYNC_SUMMARY_REQUEST/RESPONSE` 摘要交换协议落地
  - 摘要交换升级为“时间窗 + 批次增量”（`sync_since_timestamp/sync_window_seconds/sync_batch_size`）
  - 节点收到摘要后自动补齐缺失索引（`messages`）
  - 对缺失正文自动触发后台回源回填（`content_blobs`）
  - 后台周期反熵任务已启用（默认 12 秒，可通过 `AEGIS_ANTI_ENTROPY_INTERVAL_SEC` 调整）
  - 增量窗口与批次可配置（`AEGIS_ANTI_ENTROPY_WINDOW_SEC` / `AEGIS_ANTI_ENTROPY_BATCH_SIZE`）
  - 缺口补齐优先级与预算策略完成（按新到旧优先；`AEGIS_ANTI_ENTROPY_INDEX_BUDGET` / `AEGIS_ANTI_ENTROPY_BODY_BUDGET`）
  - R2 指标与结构化日志落地（请求/响应计数、摘要量、索引插入、正文抓取成功失败、同步滞后）
  - 新增手动触发入口：`TriggerAntiEntropySyncNow`（用于诊断与测试）
- 验证：
  - `TestR2AntiEntropyManualSyncRecoversMissedPost`
  - `TestR2AntiEntropyPeriodicSyncRecoversMissedPost`
  - `TestR2DigestWindowReturnsOnlySinceTimestamp`
  - `TestR2LatestPublicTimestamp`
  - `TestR2OfflineRejoinConvergesWithinWindow`

### R3（媒体链路）
- 状态：Done
- 已完成：
  - 帖子模型增加图片元数据：`image_cid/thumb_cid/image_mime/image_size/image_width/image_height`
  - 新增 `media_blobs`（按 CID 去重存储媒体数据）
  - 新增本地 API：`AddLocalPostWithImageToSub` / `GetPostMediaByID` / `GetMediaByCID`
  - 新增网络协议：`MEDIA_FETCH_REQUEST/RESPONSE`，支持图片按 CID 回源
  - R2 反熵摘要已携带图片元数据，离线回归可补齐图片索引并触发媒体回源
  - 上传链路补齐：图片尺寸压缩（主图最大边 1920）、真实缩略图生成（最大边 320）、JPEG/PNG 编码策略
  - 前端链路补齐：发帖图片选择与预览、列表缩略图展示、详情原图按需加载
- 验证：
  - `TestR3ImageBlobDeduplicatedByCID`
  - `TestR3MediaFetchFromPeerOnLocalMiss`
  - `go test ./... -count=1` 全量通过（含 R3 媒体链路回归）

### R4（互联网连通与安全）
- 状态：Done
- 已完成：
  - 自动启动网络主路径已启用：应用启动默认自动拉起 P2P（可通过 `AEGIS_AUTOSTART_P2P` 关闭）
  - 端口冲突自动回退已启用：优先端口不可用时，在 `[AEGIS_P2P_PORT, AEGIS_P2P_PORT+20]` 内自动选择可用端口
  - 引导节点配置已启用：支持 `AEGIS_BOOTSTRAP_PEERS` 注入 bootstrap 列表
  - NAT 连通性能力已启用：`NATPortMap + NATService + AutoNATv2`
  - 打洞能力已启用：`EnableHolePunching`
  - relay 回退能力已启用：`EnableRelay`，并在配置静态 relay（`AEGIS_RELAY_PEERS` 或 bootstrap 含 relay）时自动启用 `AutoRelayWithStaticRelays`
  - 防滥用基线已落地：`CONTENT_FETCH` / `MEDIA_FETCH` 按来源 peer 做窗口限流（`AEGIS_FETCH_REQUEST_LIMIT` + `AEGIS_FETCH_REQUEST_WINDOW_SEC`）
  - 请求来源绑定修正：媒体/正文回源响应目标改为真实来源 peer，降低伪造 requester 字段风险
  - 连接级防护已落地：最大连接数限制（`AEGIS_MAX_CONNECTED_PEERS`）+ 黑名单（`AEGIS_P2P_BLACKLIST_PEERS`）+ 灰名单短时封禁（`AEGIS_P2P_GREYLIST_PEERS` / `AEGIS_P2P_GREYLIST_TTL_SEC`）
- 验证：
  - `go test ./... -count=1` 全量通过

### R5（发布级稳定性与观测）
- 状态：Done
- 已完成（第一批）：
  - 新增发布指标快照接口：`GetReleaseMetrics`
  - 指标口径落地：
    - `content_fetch_success_rate`
    - `content_fetch_latency_p95`（毫秒）
    - `blob_cache_hit_rate`
    - `sync_lag_seconds`
  - 指标采集路径落地：
    - 正文/媒体读取路径缓存命中与 miss 计数（`GetPostBodyByCID` / `GetMediaByCID`）
    - 正文回源请求与结果计数、延迟采样（用于 success rate 与 p95）
  - 结构化日志增强：
    - `content_fetch.request/result`（含 cid、peer_count、timeout、耗时、结果）
    - `media_fetch.request/result`（含 cid、peer_count、timeout、结果）
    - 新增 `retry_budget` 字段，结合 `AEGIS_FETCH_RETRY_ATTEMPTS` 观测重试预算
  - 新增 R5 回归测试：
    - `TestR5ReleaseMetricsDerivedValues`
    - `TestR5BlobCacheMetricsRecordedFromReadPath`
  - 新增排障文档：`docs/R5_OBSERVABILITY_RUNBOOK.md`
- 已完成（第二批）：
  - 告警规则引擎接入应用运行时：
    - 周期评估 worker：`runReleaseAlertWorker`
    - 手动触发入口：`TriggerReleaseAlertEvaluationNow`
    - 活跃告警查询：`GetReleaseAlerts`
  - 告警规则按阈值+持续窗口落地（warning/critical）：
    - `content_fetch_success_rate`
    - `content_fetch_latency_p95`
    - `blob_cache_hit_rate`
    - `sync_lag_seconds`
  - 告警结构化日志落地：
    - `release_alert.raised`
    - `release_alert.recovered`
  - 新增 R5 告警回归测试：
    - `TestR5ReleaseAlertRaisedAfterSustainWindow`
    - `TestR5ReleaseAlertRecoveryClearsActiveState`
- 验证：
  - `go test ./... -run 'TestR5ReleaseMetricsDerivedValues|TestR5BlobCacheMetricsRecordedFromReadPath|TestR5ReleaseAlertRaisedAfterSustainWindow|TestR5ReleaseAlertRecoveryClearsActiveState' -count=1` 通过
  - 手工演练留档完成：`docs/R5_INCIDENT_DRILL_RECORD_2026-02-15.md`
  - 治理离线回归补测通过：
    - `TestR5GovernanceSyncResponseAppliesTrustedModeration`
    - `TestR5GovernanceSyncResponseSkipsUntrustedModeration`

R5 收口补充（2026-02-15）：
- 治理状态离线补齐能力落地（复用反熵思路）：
  - 新增协议：`GOVERNANCE_SYNC_REQUEST/RESPONSE`
  - 周期任务在反熵 worker 中并行触发治理同步请求
  - 重连后可自动补齐 `moderation` 状态（含 `UNBAN/SHADOW_BAN`）
  - 同步应用时增加 trusted admin 校验，忽略非受信任来源治理状态
  - 同步最近窗口治理日志（`moderation_logs`，仅 `result=applied`），用于离线回归后的审计追踪
  - 治理日志同步去重（同一条日志不会因周期同步重复写入）

产品 UI 约束（正式版）：
- 回源失败提示应用户友好（例如“内容暂时不可用，请稍后重试”）。
- “立即重试/抓包/调试”类按钮仅出现在诊断模式，不作为默认产品 UI。

---

## 7. 执行纪律（自动文档同步）
- 每次完成任一阶段任务或收口项后，立即同步更新本文档状态（不等待人工提醒）。
- 更新内容至少包含：状态变化（In Progress/Done）、完成项、验证结果（测试名或联调结论）。
- 若代码已完成但文档未更新，视为该项未完成，不可进入下一阶段。

---

## 8. 数据库迁移纪律（必须执行）
- 任何 schema 变更必须同时提供“新库建表路径 + 旧库升级路径”，两条路径都要可启动。
- 迁移顺序必须遵循：先 `ALTER TABLE ADD COLUMN`，再创建依赖该列的索引/查询/回填逻辑。
- 每次新增列后必须补三项验证：
  - 旧库启动验证（保留旧 DB 文件直接启动）
  - 空库启动验证（新建 DB 首次启动）
  - 身份回读验证（`local_identity` 可正常加载）
- 出现迁移事故后，必须在本文档记录“根因 + 修复 + 回归用例”，避免重复踩坑。

### 近期迁移事故记录（2026-02-15）
- 现象：应用启动报错 `database initialization failed: no such column: content_cid`，导致身份加载失败表现为“identity 丢失”。
- 根因：`messages.content_cid` 在旧库尚未 `ALTER` 完成前，代码先执行了 `idx_messages_content_cid` 索引创建。
- 修复：调整迁移顺序为“先补列再建索引”，并补 `content_blobs.last_accessed_at` 默认值兼容。
- 状态：已修复并通过全量测试。

### 近期一致性观察记录（2026-02-15）
- 现象：三节点手测时 `Public Feed` 计数出现差异（A=13，B/Genesis=8）。
- 根因：R2 反熵默认时间窗为 24h，导致超过 24h 的历史 `normal` 帖子未进入摘要交换；同时 `shadowed` 帖子按产品语义仅作者本地可见。
- 修复：反熵默认时间窗调整为 30 天（`AEGIS_ANTI_ENTROPY_WINDOW_SEC` 仍可覆盖配置）。
- 手测口径：比较“全局一致性”时应对齐 `visibility=normal` 集合；`shadowed` 的本地可见差异属于预期。

### 近期治理权限修正记录（2026-02-15）
- 现象：普通节点 UI 可点击治理策略开关，造成“谁都能改策略”的误解与风险。
- 修复：治理策略更新改为管理员强校验（消息必须携带受信任 `admin_pubkey`）；普通节点前端禁用 `Toggle Policy`。
- 验证：
  - `TestC1GovernancePolicyUpdateMessageApplied`（受信任管理员可生效）
  - `TestC1GovernancePolicyUpdateMessageRejectsUntrustedAdmin`（非管理员被拒绝）
