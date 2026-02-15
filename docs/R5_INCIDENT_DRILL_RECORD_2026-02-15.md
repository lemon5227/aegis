# R5 手工演练记录（详情打不开）

日期：2026-02-15  
状态：Pending（待实机执行并回填）

## 1. 演练目标
- 在 10 分钟内定位“详情打不开”根因。
- 验证指标、日志、告警三者一致性。

## 2. 演练环境
- 节点数量：2（跨公网）
- relay：`/ip4/51.107.0.10/tcp/40100/p2p/12D3KooWLweFn4GFfEa9X1St4d78HQqYYzXaH2oy5XahKrwar6w7`
- 客户端：Windows + macOS

## 3. 注入故障步骤
1. 在接收端删除目标正文 blob（或人为阻断到 relay 的流量）。
2. 打开帖子详情，触发正文回源。
3. 观察 `content_fetch.result` 错误分布（如 timeout/no peers/not found）。

## 4. 观测记录（回填）
- 触发时间：
- 首次告警时间：
- 定位完成时间：
- 总耗时（分钟）：
- 期间关键指标：
  - `content_fetch_success_rate`：
  - `content_fetch_latency_p95`：
  - `blob_cache_hit_rate`：
  - `sync_lag_seconds`：
- 期间关键日志：
  - `content_fetch.result`：
  - `release_alert.raised`：
  - `release_alert.recovered`：

## 5. 根因与修复（回填）
- 根因：
- 修复动作：
- 恢复验证：

## 6. 结论
- 是否满足“10 分钟内定位”：
- 后续改进项：
