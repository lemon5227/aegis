# 需求文档：通知中心

## 简介

Aegis 是一个去中心化 P2P 论坛桌面应用。通知中心为用户提供关键互动事件的本地通知能力，包括回复、投票、治理操作等。由于 Aegis 无中心服务器，所有通知均在本地节点接收到 P2P 消息时生成并存储在本地 SQLite 数据库中。通知中心是灰度上线的 P0 高频留存功能。

## 术语表

- **Notification_Center**：通知中心模块，负责通知的生成、存储、查询、状态管理和前端展示
- **Notification**：单条通知记录，包含类型、关联实体、来源用户、已读状态和时间戳
- **Notification_Store**：SQLite 中的通知存储层（notifications 表），负责通知的持久化和查询
- **Notification_Generator**：在 ProcessIncomingMessage 流程中，根据消息类型和本地用户身份判断是否生成通知的逻辑模块
- **Notification_Badge**：前端侧边栏中显示未读通知数量的视觉标记
- **Local_User**：当前节点的本地用户（通过 getLocalIdentity 获取的 ed25519 公钥身份）
- **Source_User**：触发通知事件的远程用户（消息发送者）
- **Target_Entity**：通知关联的目标内容（帖子或评论）

## 需求

### 需求 1：通知数据模型与存储

**用户故事：** 作为本地用户，我希望系统能持久化存储我收到的通知，以便我在任何时候查看历史通知。

#### 验收标准

1. THE Notification_Store SHALL 在 SQLite 数据库中维护 notifications 表，每条记录包含：通知 ID、通知类型、来源用户公钥、目标实体 ID、目标实体类型、关联帖子 ID、已读状态、创建时间戳
2. THE Notification_Store SHALL 使用通知 ID 作为主键，确保每条通知记录的唯一性
3. THE Notification_Store SHALL 为创建时间戳和已读状态建立索引，以支持高效的分页查询和未读计数
4. THE Notification_Store SHALL 支持以下通知类型：comment_reply（评论回复）、post_comment（帖子收到评论）、post_upvote（帖子被点赞）、post_downvote（帖子被踩）、comment_upvote（评论被点赞）、comment_downvote（评论被踩）、governance_action（治理操作）

### 需求 2：通知生成逻辑

**用户故事：** 作为本地用户，我希望当其他用户与我的内容互动时自动收到通知，以便我及时了解社区动态。

#### 验收标准

1. WHEN ProcessIncomingMessage 处理类型为 COMMENT 的消息且该评论的目标帖子作者为 Local_User 时，THE Notification_Generator SHALL 生成一条 post_comment 类型的通知
2. WHEN ProcessIncomingMessage 处理类型为 COMMENT 的消息且该评论的父评论作者为 Local_User 时，THE Notification_Generator SHALL 生成一条 comment_reply 类型的通知
3. WHEN ProcessIncomingMessage 处理类型为 POST_UPVOTE 的消息且目标帖子作者为 Local_User 时，THE Notification_Generator SHALL 生成一条 post_upvote 类型的通知
4. WHEN ProcessIncomingMessage 处理类型为 POST_DOWNVOTE 的消息且目标帖子作者为 Local_User 时，THE Notification_Generator SHALL 生成一条 post_downvote 类型的通知
5. WHEN ProcessIncomingMessage 处理类型为 COMMENT_UPVOTE 的消息且目标评论作者为 Local_User 时，THE Notification_Generator SHALL 生成一条 comment_upvote 类型的通知
6. WHEN ProcessIncomingMessage 处理类型为 COMMENT_DOWNVOTE 的消息且目标评论作者为 Local_User 时，THE Notification_Generator SHALL 生成一条 comment_downvote 类型的通知
7. WHEN ProcessIncomingMessage 处理类型为 SHADOW_BAN 或 UNBAN 的消息且目标公钥为 Local_User 时，THE Notification_Generator SHALL 生成一条 governance_action 类型的通知
8. WHILE Source_User 的公钥与 Local_User 的公钥相同时，THE Notification_Generator SHALL 跳过通知生成（不为自己的操作生成通知）
9. IF Notification_Generator 在生成通知过程中遇到数据库错误，THEN THE Notification_Generator SHALL 记录错误日志并继续处理原始消息（通知生成失败不阻塞消息处理）

### 需求 3：通知查询接口

**用户故事：** 作为本地用户，我希望能分页查看通知列表并获取未读数量，以便高效浏览通知。

#### 验收标准

1. THE Notification_Center SHALL 提供 GetNotifications(limit int, cursor string) 接口，按创建时间倒序返回通知列表，支持基于游标的分页
2. THE Notification_Center SHALL 提供 GetUnreadNotificationCount() 接口，返回当前未读通知的总数
3. WHEN GetNotifications 被调用且 cursor 为空字符串时，THE Notification_Center SHALL 返回最新的 limit 条通知
4. WHEN GetNotifications 被调用且 cursor 为有效游标值时，THE Notification_Center SHALL 返回该游标之前的 limit 条通知

### 需求 4：通知已读状态管理

**用户故事：** 作为本地用户，我希望能标记通知为已读，包括单条已读和批量全部已读，以便管理通知状态。

#### 验收标准

1. WHEN 用户调用 MarkNotificationRead(notificationID string) 时，THE Notification_Store SHALL 将指定通知的已读状态更新为已读
2. WHEN 用户调用 MarkAllNotificationsRead() 时，THE Notification_Store SHALL 将所有未读通知的已读状态批量更新为已读
3. WHEN 通知已读状态发生变更时，THE Notification_Center SHALL 通过 runtime.EventsEmit 发送 "notifications:updated" 事件通知前端刷新
4. IF MarkNotificationRead 收到不存在的通知 ID，THEN THE Notification_Store SHALL 返回成功（幂等操作，不报错）

### 需求 5：实时通知推送

**用户故事：** 作为本地用户，我希望新通知到达时前端能实时更新，以便我无需手动刷新即可看到新通知。

#### 验收标准

1. WHEN Notification_Generator 成功生成一条新通知时，THE Notification_Center SHALL 通过 runtime.EventsEmit 发送 "notifications:updated" 事件
2. WHEN 前端收到 "notifications:updated" 事件时，THE Notification_Center SHALL 触发未读计数和通知列表的重新获取
3. THE Notification_Center SHALL 复用现有的 Wails runtime.EventsEmit 机制进行前端事件通知，与 feed:updated、comments:updated 等现有事件模式保持一致

### 需求 6：通知点击跳转定位

**用户故事：** 作为本地用户，我希望点击通知后能直接跳转到相关的帖子或评论，以便快速查看互动内容。

#### 验收标准

1. WHEN 用户点击类型为 post_comment 或 comment_reply 的通知时，THE Notification_Center SHALL 导航到对应帖子的详情页并定位到相关评论
2. WHEN 用户点击类型为 post_upvote 或 post_downvote 的通知时，THE Notification_Center SHALL 导航到对应帖子的详情页
3. WHEN 用户点击类型为 comment_upvote 或 comment_downvote 的通知时，THE Notification_Center SHALL 导航到对应帖子的详情页并定位到相关评论
4. WHEN 用户点击类型为 governance_action 的通知时，THE Notification_Center SHALL 导航到用户个人资料页面
5. WHEN 用户点击通知且该通知为未读状态时，THE Notification_Center SHALL 自动将该通知标记为已读
6. IF 通知关联的目标内容已被删除，THEN THE Notification_Center SHALL 显示"内容已删除"的提示信息，不执行跳转

### 需求 7：通知中心前端界面

**用户故事：** 作为本地用户，我希望在侧边栏中有一个通知入口，能方便地查看和管理所有通知。

#### 验收标准

1. THE Notification_Center SHALL 在 Sidebar 组件中添加通知入口图标
2. WHILE 存在未读通知时，THE Notification_Badge SHALL 在通知入口图标上显示未读数量角标
3. WHEN 未读通知数量超过 99 时，THE Notification_Badge SHALL 显示 "99+"
4. WHEN 用户点击通知入口时，THE Notification_Center SHALL 在主内容区域展示通知列表视图
5. THE Notification_Center SHALL 在通知列表视图中为每条通知显示：通知类型图标、来源用户名称、互动内容摘要、时间戳、已读/未读状态标记
6. THE Notification_Center SHALL 在通知列表视图顶部提供"全部标记为已读"按钮
7. WHEN 用户滚动到通知列表底部时，THE Notification_Center SHALL 自动加载下一页通知（无限滚动）
8. WHILE 通知列表为空时，THE Notification_Center SHALL 显示空状态提示信息

### 需求 8：通知去重

**用户故事：** 作为本地用户，我希望不会收到重复的通知，以保持通知列表的整洁。

#### 验收标准

1. THE Notification_Generator SHALL 基于通知类型、来源用户公钥和目标实体 ID 的组合进行去重判断
2. WHEN 一条语义相同的通知已存在于 Notification_Store 中时，THE Notification_Generator SHALL 跳过该通知的生成
3. THE Notification_Store SHALL 使用 UNIQUE 约束（通知类型 + 来源用户公钥 + 目标实体 ID）确保数据库层面的去重
