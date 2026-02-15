# R5 手工演练记录（详情打不开）

日期：2026-02-15  
状态：Done（已完成）

## 1. 演练目标
- 在 10 分钟内定位“详情打不开”根因。
- 验证指标、日志、告警三者一致性。

## 2. 演练环境
- 节点数量：2（本机双实例）
- 数据库：`genesis.db` + `user_a.db`
- 端口：`40100` + `40101`
- 目标帖子：
  - `post_id=ddd343e56b3afde79232a94113409a07b73461fd8924d6e4e22152311ba86368`
  - `content_cid=cidv1-e7067911846df6695510d4a0b8c5065a36675ee511349c0d9ad18fae52b4bbcc`

## 3. 注入故障步骤
1. 在接收端删除目标正文 blob（或人为阻断到 relay 的流量）。
2. 打开帖子详情，触发正文回源。
3. 观察 `content_fetch.result` 错误分布（如 timeout/no peers/not found）。

## 4. 观测记录（回填）
- 触发时间：2026-02-15（本地演练）
- 首次告警时间：N/A（本次故障注入为 StopP2P 后即时读正文，未持续到告警窗口）
- 定位完成时间：2026-02-15（同轮次完成）
- 总耗时（分钟）：< 10
- 期间关键观测：
  - 停止 P2P + 删除 `content_blobs` 后，`GetPostBodyByID` 返回 `content not found`
  - 重启 P2P 并恢复连接后，`GetPostBodyByID` 返回正文对象：
    - `body=Test Test Test`
    - `contentCid=cidv1-e7067911846df6695510d4a0b8c5065a36675ee511349c0d9ad18fae52b4bbcc`

## 5. 根因与修复（回填）
- 根因：接收端本地正文 blob 缺失，且 P2P 不可用导致无法回源。
- 修复动作：恢复 P2P 连接（`user_a` 与 `genesis` 重连）。
- 恢复验证：相同 `post_id` 再次调用 `GetPostBodyByID` 成功返回正文。

## 6. 结论
- 是否满足“10 分钟内定位”：是
- 后续改进项：
  - 增补一轮“持续故障超过告警窗口”的演练，以验证 `release_alert.raised/recovered` 端到端触发链路。
