# R5 可观测与排障手册

更新时间：2026-02-15

## 1. 指标口径
- `content_fetch_success_rate`
  - 定义：`content_fetch_success / content_fetch_attempts`
  - 来源：`GetReleaseMetrics()`
- `content_fetch_latency_p95`
  - 定义：正文回源延迟采样窗口的 P95（单位：毫秒）
  - 来源：`GetReleaseMetrics()`
- `blob_cache_hit_rate`
  - 定义：`blob_cache_hits / (blob_cache_hits + blob_cache_misses)`
  - 来源：`GetReleaseMetrics()`
- `sync_lag_seconds`
  - 定义：当前节点观测到的反熵同步滞后秒数
  - 来源：`GetReleaseMetrics()`

## 2. 告警建议阈值（首版）
- `content_fetch_success_rate < 0.95` 持续 5 分钟：`warning`
- `content_fetch_success_rate < 0.85` 持续 3 分钟：`critical`
- `content_fetch_latency_p95 > 3000` 持续 5 分钟：`warning`
- `content_fetch_latency_p95 > 5000` 持续 3 分钟：`critical`
- `blob_cache_hit_rate < 0.60` 持续 10 分钟：`warning`
- `sync_lag_seconds > 180` 持续 5 分钟：`warning`
- `sync_lag_seconds > 600` 持续 3 分钟：`critical`

## 3. 关键日志
- 正文回源请求：
  - `content_fetch.request request_id=<id> cid=<cid> peer_count=<n> timeout_ms=<ms> retry_budget=<n>`
- 正文回源结果：
  - `content_fetch.result cid=<cid> success=<bool> elapsed_ms=<ms> dedup_shared=<bool> error=<err>`
- 媒体回源请求：
  - `media_fetch.request request_id=<id> cid=<cid> peer_count=<n> timeout_ms=<ms> retry_budget=<n>`
- 媒体回源结果：
  - `media_fetch.result cid=<cid> success=<bool> error=<err>`
- 告警触发：
  - `release_alert.raised key=<rule_key> metric=<metric> level=<warning|critical> value=<v> threshold=<t> window_sec=<s>`
- 告警恢复：
  - `release_alert.recovered key=<rule_key> metric=<metric> level=<warning|critical>`

## 3.1 告警评估入口
- 周期评估：
  - 由 `runReleaseAlertWorker` 自动执行
  - 评估周期：`AEGIS_RELEASE_ALERT_EVAL_INTERVAL_SEC`（默认 30 秒）
- 手动评估：
  - 调用 `TriggerReleaseAlertEvaluationNow`
- 查看当前活跃告警：
  - 调用 `GetReleaseAlerts`

## 4. 10 分钟排障流程（详情打不开）
1. 先看 `content_fetch_success_rate` 与 `content_fetch_latency_p95`：
   - 成功率低且延迟高：优先判定网络或 relay 问题。
2. 查 `content_fetch.result` 错误分布：
   - `content fetch no peers`：对端连接断开或引导节点不可达。
   - `content fetch timeout`：网络抖动、对端处理慢或 relay 拥塞。
   - `content fetch not found`：网络内无该 CID 副本，需检查发布端数据存在性。
3. 查 `blob_cache_hit_rate`：
   - 持续低位时，优先排查本地配额淘汰是否过于激进。
4. 查 `sync_lag_seconds`：
   - 滞后高时，先手动触发 `TriggerAntiEntropySyncNow` 观察恢复速度。
5. 验证修复：
   - 复测“列表可见 -> 详情可开 -> 图片可开”三步链路。

## 5. 运维参数（按需）
- `AEGIS_FETCH_RETRY_ATTEMPTS`：正文/媒体回源重试次数，默认 `1`，最大 `3`
- `AEGIS_FETCH_REQUEST_LIMIT`：窗口内单 peer 请求上限
- `AEGIS_FETCH_REQUEST_WINDOW_SEC`：请求限流窗口（秒）
- `AEGIS_MSG_MAX_BYTES`：单条入站消息大小上限（默认 `2097152`）
- `AEGIS_MSG_RATE_LIMIT`：入站消息每 peer 窗口限额（默认 `240`）
- `AEGIS_MSG_RATE_WINDOW_SEC`：入站消息限流窗口秒数（默认 `60`）
- `AEGIS_RELAY_SERVICE_ENABLED`：是否开启 relay service（默认 `true`）
- `AEGIS_ANTI_ENTROPY_INTERVAL_SEC`：反熵周期（秒）
- `AEGIS_RELEASE_ALERT_EVAL_INTERVAL_SEC`：告警评估周期（秒），默认 `30`
- `AEGIS_GOVERNANCE_SYNC_BATCH_SIZE`：治理同步单次返回上限，默认 `200`
- `AEGIS_GOVERNANCE_LOG_SYNC_LIMIT`：治理日志同步单次返回上限，默认 `200`

## 7. Relay 过载应急
1. 观察是否频繁出现：
   - `incoming message rate limited`
   - `incoming message too large`
2. 若 relay 压力持续过高：
   - 下调 `AEGIS_MSG_RATE_LIMIT`（例如 240 -> 120）
   - 临时收紧 `AEGIS_MAX_CONNECTED_PEERS`
3. 若疑似恶意流量：
   - 使用 `AEGIS_P2P_BLACKLIST_PEERS` 封禁明确恶意 peer
   - 重启后验证 join success 与 fetch success 是否恢复

## 6. 离线治理状态不一致排查
1. 观察重连后是否出现治理同步日志：
   - `governance_sync.request sent`
   - `governance_sync.response sent`
   - `governance_sync.response applied`
2. 若节点仍是旧治理状态：
   - 确认治理消息 `source_admin` 在受信任管理员集合中。
   - 确认该节点与 peers 已建立连接（`Connected Peers > 0`）。
3. 若日志出现 `governance_sync.skip_untrusted`：
   - 说明同步条目来自非受信任 admin，被安全策略忽略。
4. 若只看到状态同步、看不到日志补齐：
   - 确认 `AEGIS_GOVERNANCE_LOG_SYNC_LIMIT > 0`。
   - 当前日志同步口径仅包含 `result=applied` 的治理日志。
