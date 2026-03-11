# 实现计划：通知中心（Notification Center）

## 概述

基于需求文档和设计文档，将通知中心功能拆分为后端（Go）和前端（React + TypeScript）两条主线，按数据模型 → 核心逻辑 → 集成 → 查询 → 状态管理 → 前端的顺序递进实现。每个任务构建在前一步之上，确保无孤立代码。

## 任务

- [x] 1. 后端数据模型与 Schema
  - [x] 1.1 在 `aegis-app/notifications.go` 中定义通知类型常量和数据结构
    - 定义 7 个通知类型常量：`NotifTypePostComment`、`NotifTypeCommentReply`、`NotifTypePostUpvote`、`NotifTypePostDownvote`、`NotifTypeCommentUpvote`、`NotifTypeCommentDownvote`、`NotifTypeGovernance`
    - 定义 `Notification` 结构体（含 JSON tag）和 `NotificationPage` 结构体
    - 实现 `buildNotificationID(notifType, sourcePubkey, targetEntityID string) string`，使用 `sha256(type|source|target)[:16]`
    - _Requirements: 1.1, 1.2, 1.4_

  - [x] 1.2 在 `aegis-app/db.go` 的 `ensureSchema` 中添加 notifications 表
    - 在 schema 切片中追加 `CREATE TABLE IF NOT EXISTS notifications` 语句，包含所有字段和 UNIQUE 约束
    - 追加 `idx_notifications_created_at` 和 `idx_notifications_is_read` 两个索引
    - _Requirements: 1.1, 1.2, 1.3, 8.3_

- [x] 2. 后端通知生成核心逻辑
  - [x] 2.1 在 `aegis-app/notifications.go` 中实现通知生成内部方法
    - 实现 `tryGenerateNotification(notifType, sourcePubkey, targetEntityID, targetType, postID string, createdAt int64)`
    - 内部调用 `getLocalIdentity()` 获取本地用户公钥，source == local 时跳过
    - 使用 `INSERT OR IGNORE` 写入 notifications 表，失败仅 `log.Printf`
    - 插入成功时通过 `runtime.EventsEmit` 发送 `"notifications:updated"` 事件
    - _Requirements: 2.8, 2.9, 5.1, 8.1, 8.2_

  - [x] 2.2 在 `aegis-app/notifications.go` 中实现辅助查询方法
    - 实现 `getPostAuthor(postID string) (string, error)` — 查询帖子作者公钥
    - 实现 `getCommentAuthor(commentID string) (string, error)` — 查询评论作者公钥
    - 实现 `getCommentPostID(commentID string) (string, error)` — 查询评论所属帖子 ID
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

- [x] 3. 后端 ProcessIncomingMessage 集成
  - [x] 3.1 在 `aegis-app/db.go` 的 COMMENT case 末尾追加通知生成调用
    - `insertComment` 成功后，查询帖子作者，若为本地用户则调用 `tryGenerateNotification(post_comment, ...)`
    - 若 `parentID` 非空，查询父评论作者，若为本地用户则调用 `tryGenerateNotification(comment_reply, ...)`
    - _Requirements: 2.1, 2.2_

  - [x] 3.2 在 `aegis-app/db.go` 的投票 case 末尾追加通知生成调用
    - POST_UPVOTE / POST_DOWNVOTE / POST_VOTE_SET：查询帖子作者，若为本地用户则生成 `post_upvote` 或 `post_downvote` 通知
    - COMMENT_UPVOTE / COMMENT_DOWNVOTE / COMMENT_VOTE_SET：查询评论作者和所属帖子 ID，若为本地用户则生成 `comment_upvote` 或 `comment_downvote` 通知
    - _Requirements: 2.3, 2.4, 2.5, 2.6_

  - [x] 3.3 在 `aegis-app/db.go` 的 SHADOW_BAN / UNBAN case 末尾追加通知生成调用
    - 若 `target_pubkey` 为本地用户则调用 `tryGenerateNotification(governance_action, ...)`
    - _Requirements: 2.7_

- [x] 4. 检查点 — 后端通知生成验证
  - 确保所有测试通过，如有疑问请询问用户。

- [x] 5. 后端查询 API
  - [x] 5.1 在 `aegis-app/notifications.go` 中实现游标编解码
    - 实现 `encodeNotificationCursor(createdAt int64, id string) string` — base64 编码
    - 实现 `decodeNotificationCursor(cursor string) (int64, string, error)` — base64 解码 + 解析
    - _Requirements: 3.1, 3.3, 3.4_

  - [x] 5.2 在 `aegis-app/notifications.go` 中实现 `GetNotifications(limit int, cursor string) (NotificationPage, error)`
    - cursor 为空时返回最新 limit 条通知，按 `created_at DESC, id DESC` 排序
    - cursor 非空时解码后查询该游标之前的通知
    - 返回 `NotificationPage`，包含 items 和 nextCursor
    - _Requirements: 3.1, 3.3, 3.4_

  - [x] 5.3 在 `aegis-app/notifications.go` 中实现 `GetUnreadNotificationCount() (int, error)`
    - 查询 `SELECT COUNT(*) FROM notifications WHERE is_read = 0`
    - _Requirements: 3.2_

- [x] 6. 后端已读状态管理
  - [x] 6.1 在 `aegis-app/notifications.go` 中实现 `MarkNotificationRead(notificationID string) error`
    - `UPDATE notifications SET is_read = 1 WHERE id = ?`，影响 0 行也返回 nil（幂等）
    - 成功后发送 `"notifications:updated"` 事件
    - _Requirements: 4.1, 4.3, 4.4_

  - [x] 6.2 在 `aegis-app/notifications.go` 中实现 `MarkAllNotificationsRead() error`
    - `UPDATE notifications SET is_read = 1 WHERE is_read = 0`
    - 成功后发送 `"notifications:updated"` 事件
    - _Requirements: 4.2, 4.3_

- [x] 7. 检查点 — 后端 API 完整性验证
  - 确保所有测试通过，如有疑问请询问用户。

- [ ] 8. 后端属性测试
  - [ ]* 8.1 编写 Property 1 属性测试：评论触发帖子通知
    - **Property 1: 评论触发帖子通知**
    - 在 `aegis-app/notifications_gen_test.go` 中，随机生成 COMMENT 消息，验证当帖子作者为本地用户且发送者非本地用户时，notifications 表中存在 `post_comment` 通知
    - **Validates: Requirements 2.1**

  - [ ]* 8.2 编写 Property 2 属性测试：回复触发评论通知
    - **Property 2: 回复触发评论通知**
    - 在 `aegis-app/notifications_gen_test.go` 中，随机生成带父评论的 COMMENT 消息，验证父评论作者为本地用户时生成 `comment_reply` 通知
    - **Validates: Requirements 2.2**

  - [ ]* 8.3 编写 Property 3 属性测试：帖子投票触发通知
    - **Property 3: 帖子投票触发通知**
    - 在 `aegis-app/notifications_gen_test.go` 中，随机生成 POST_UPVOTE/POST_DOWNVOTE 消息，验证帖子作者为本地用户时生成对应通知
    - **Validates: Requirements 2.3, 2.4**

  - [ ]* 8.4 编写 Property 4 属性测试：评论投票触发通知
    - **Property 4: 评论投票触发通知**
    - 在 `aegis-app/notifications_gen_test.go` 中，随机生成 COMMENT_UPVOTE/COMMENT_DOWNVOTE 消息，验证评论作者为本地用户时生成对应通知
    - **Validates: Requirements 2.5, 2.6**

  - [ ]* 8.5 编写 Property 5 属性测试：治理操作触发通知
    - **Property 5: 治理操作触发通知**
    - 在 `aegis-app/notifications_gen_test.go` 中，随机生成 SHADOW_BAN/UNBAN 消息，验证目标为本地用户时生成 `governance_action` 通知
    - **Validates: Requirements 2.7**

  - [ ]* 8.6 编写 Property 6 属性测试：自我通知过滤
    - **Property 6: 自我通知过滤**
    - 在 `aegis-app/notifications_gen_test.go` 中，验证 source_pubkey == local_pubkey 时不生成任何通知
    - **Validates: Requirements 2.8**

  - [ ]* 8.7 编写 Property 7 属性测试：通知生成不阻塞消息处理
    - **Property 7: 通知生成不阻塞消息处理**
    - 在 `aegis-app/notifications_gen_test.go` 中，模拟 DB 错误场景，验证 ProcessIncomingMessage 仍返回 nil error
    - **Validates: Requirements 2.9**

  - [ ]* 8.8 编写 Property 8 属性测试：分页完整性与排序
    - **Property 8: 分页完整性与排序**
    - 在 `aegis-app/notifications_query_test.go` 中，随机插入 N 条通知，逐页遍历验证总数 == N、严格降序、无重复无遗漏
    - **Validates: Requirements 3.1, 3.3, 3.4**

  - [ ]* 8.9 编写 Property 9 属性测试：未读计数一致性
    - **Property 9: 未读计数一致性**
    - 在 `aegis-app/notifications_query_test.go` 中，随机设置已读/未读状态，验证 GetUnreadNotificationCount 返回值与实际未读数一致
    - **Validates: Requirements 3.2**

  - [ ]* 8.10 编写 Property 10 属性测试：单条标记已读
    - **Property 10: 单条标记已读**
    - 在 `aegis-app/notifications_read_test.go` 中，随机选择通知标记已读，验证仅该通知状态变更
    - **Validates: Requirements 4.1**

  - [ ]* 8.11 编写 Property 11 属性测试：批量标记已读
    - **Property 11: 批量标记已读**
    - 在 `aegis-app/notifications_read_test.go` 中，随机生成通知集后调用 MarkAllNotificationsRead，验证未读计数归零
    - **Validates: Requirements 4.2**

  - [ ]* 8.12 编写 Property 12 属性测试：标记已读幂等性
    - **Property 12: 标记已读幂等性**
    - 在 `aegis-app/notifications_read_test.go` 中，对任意 ID（含不存在的）调用 MarkNotificationRead，验证返回 nil error
    - **Validates: Requirements 4.4**

  - [ ]* 8.13 编写 Property 15 属性测试：通知去重
    - **Property 15: 通知去重**
    - 在 `aegis-app/notifications_dedup_test.go` 中，相同参数连续调用 tryGenerateNotification 两次，验证记录数恰好为 1
    - **Validates: Requirements 8.1, 8.2, 8.3**

- [x] 9. 检查点 — 后端测试全部通过
  - 确保所有测试通过，如有疑问请询问用户。

- [x] 10. 前端类型定义
  - [x] 10.1 在 `aegis-app/frontend/src/types/index.ts` 中添加通知相关类型
    - 添加 `Notification` 接口（id, type, sourcePubkey, targetEntityId, targetType, postId, isRead, createdAt）
    - 添加 `NotificationPage` 接口（items, nextCursor）
    - _Requirements: 1.1, 1.4_

- [x] 11. 前端 NotificationsView 组件
  - [x] 11.1 创建 `aegis-app/frontend/src/components/NotificationsView.tsx`
    - 实现通知列表视图，接收 `profiles`、`onNavigateToPost`、`onNavigateToProfile` props
    - 调用 `GetNotifications(limit, cursor)` 分页加载，实现无限滚动
    - 每条通知渲染：类型图标 + 来源用户名 + 内容摘要 + 时间戳 + 已读/未读状态
    - 顶部"全部标记为已读"按钮，调用 `MarkAllNotificationsRead()`
    - 点击通知 → 调用 `MarkNotificationRead(id)` + 根据类型跳转（post_comment/comment_reply/comment_upvote/comment_downvote → 帖子详情+评论定位；post_upvote/post_downvote → 帖子详情；governance_action → 个人资料页）
    - 空状态提示
    - 已删除内容处理：显示"内容已删除"提示
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 7.4, 7.5, 7.6, 7.7, 7.8_

- [x] 12. 前端 Sidebar 集成
  - [x] 12.1 修改 `aegis-app/frontend/src/components/Sidebar.tsx` 添加通知入口
    - 新增 `unreadNotificationCount` 和 `onNotificationsClick` props
    - 在 Recommended Feed 和 My Subscriptions 之间添加通知图标按钮
    - 使用 `notifications` material icon
    - 未读数 > 0 时显示角标数字，> 99 时显示 "99+"
    - _Requirements: 7.1, 7.2, 7.3_

- [x] 13. 前端 App.tsx 集成
  - [x] 13.1 修改 `aegis-app/frontend/src/App.tsx` 集成通知中心
    - `ViewMode` 类型新增 `'notifications'`
    - 新增 `unreadNotificationCount` 状态
    - 使用 `EventsOn` 监听 `"notifications:updated"` 事件，触发时调用 `GetUnreadNotificationCount()` 刷新计数
    - 应用启动时（`useEffect`）调用 `GetUnreadNotificationCount()` 获取初始未读计数
    - 在主内容区域路由中添加 `notifications` → `NotificationsView` 的渲染分支
    - 将 `unreadNotificationCount` 和 `onNotificationsClick` 传递给 Sidebar
    - _Requirements: 5.2, 5.3, 7.1, 7.4_

- [x] 14. 检查点 — 前端集成验证
  - 确保所有测试通过，如有疑问请询问用户。

- [ ] 15. 前端属性测试
  - [ ]* 15.1 编写 Property 13 属性测试：通知点击路由正确性
    - **Property 13: 通知点击路由正确性**
    - 在 `aegis-app/frontend/src/components/NotificationsView.test.ts` 中，使用 fast-check 随机生成通知类型，验证路由函数返回正确的跳转目标
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4**

  - [ ]* 15.2 编写 Property 14 属性测试：未读角标显示格式
    - **Property 14: 未读角标显示格式**
    - 在 `aegis-app/frontend/src/components/NotificationBadge.test.ts` 中，使用 fast-check 随机生成非负整数，验证 count==0 不显示、1-99 显示数字、>99 显示 "99+"
    - **Validates: Requirements 7.2, 7.3**

- [x] 16. 最终检查点 — 全部测试通过
  - 确保所有测试通过，如有疑问请询问用户。

## 备注

- 标记 `*` 的任务为可选任务，可跳过以加速 MVP 交付
- 每个任务引用了具体的需求编号，确保可追溯性
- 检查点任务确保增量验证，及时发现问题
- 属性测试验证通用正确性属性，单元测试验证具体示例和边界条件
- 后端使用 Go 标准库 `testing/quick` 进行属性测试，前端使用 `fast-check`
