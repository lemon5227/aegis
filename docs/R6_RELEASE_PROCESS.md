# Aegis 灰度发布流程

## 发布前门禁

执行 R6 门禁脚本，全部通过后才可进入灰度：

```bash
./scripts/r6_release_gate.sh
```

6 项检查：
1. Go 编译 + 单元测试
2. 前端构建
3. Relay 二进制构建
4. 三节点集成测试（A2/A3/B1/C1）
5. 迁移安全扫描（禁止 DROP/RENAME）
6. API 契约完整性

## 灰度策略

### Round 1: 单节点验证
- 在一台测试机上安装新版本
- 执行基本操作：发帖、评论、投票、收藏、搜索
- 验证 P2P 连接 relay 正常
- 验证离线/重连流程正常
- 通过标准：无崩溃、无数据丢失

### Round 2: 小规模验证（3-5 节点）
- 部署到 3-5 个测试节点
- 验证跨节点同步：发帖、评论、投票、收藏
- 验证反熵同步收敛
- 验证治理操作跨节点生效
- 验证媒体链路（图片发帖、缩略图、回源）
- 通过标准：所有三节点集成测试场景通过

### Round 3: 全量发布
- 前两轮无阻断问题后执行
- 打 tag 触发 CI 构建正式 release
- 更新 relay 二进制（如需）
- 发布桌面安装包

## 发布命令

```bash
# 1. 运行门禁
./scripts/r6_release_gate.sh

# 2. 打 tag
git tag -a v0.3.0 -m "release v0.3.0"
git push origin v0.3.0

# 3. CI 自动构建并发布到 GitHub Release
# 4. 下载产物进行灰度验证
```

## 回滚方案

1. 数据库迁移仅做加法（ADD COLUMN / CREATE TABLE），不做破坏性变更
2. 新增列均有 DEFAULT 值，旧版本可正常启动
3. 如需回滚：安装上一版本二进制，数据库自动兼容
4. 回滚验证：旧版本启动 + 身份回读 + 基本读写

## 发布检查清单

- [ ] R6 门禁脚本全部通过
- [ ] CHANGELOG 已更新
- [ ] 版本号已更新（wails.json + go ldflags）
- [ ] GitHub Secrets 已配置（AEGIS_RELAY_PEERS, AEGIS_BOOTSTRAP_PEERS）
- [ ] Relay 服务器运行正常（curl http://localhost:40101/health）
- [ ] 单节点灰度通过
- [ ] 小规模灰度通过
- [ ] 灰度结果留档
