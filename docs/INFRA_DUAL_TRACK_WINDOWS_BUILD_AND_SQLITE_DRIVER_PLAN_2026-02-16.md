# 基础设施双轨推进计划（Windows 构建 + SQLite 驱动）

日期：2026-02-16

## 1. 背景

业务瀑布中的 G2 已完成。本计划不重开业务阶段，而是单独处理基础设施稳定性：

1. Windows 测试分发稳定性（可执行包可直接给同学验证）。
2. SQLite 驱动跨平台构建风险（CGO 依赖导致跨平台构建不确定性）。

## 2. 决策：双轨制

### 轨道 A（短期 P0）
GitHub Actions 在 Windows 原生环境构建可执行包，先保证交付稳定。

### 轨道 B（中期 P1）
迁移到纯 Go SQLite 驱动，目标选型：`modernc.org/sqlite`（当前纯 Go 方案中主流且稳定）。

## 3. 非目标

1. 不修改现有业务 API 契约。
2. 不改变表结构语义与现有数据格式。
3. 不阻塞现有功能联调与测试节奏。

## 4. 轨道 A 计划（立即执行）

### A1. Windows 原生构建流水线

新增 GitHub Actions Workflow：

1. 触发方式：
   - `workflow_dispatch`（手动打包）
   - `push tags: v*`（可选用于 release）
2. 构建环境：`windows-latest`。
3. 依赖：Go 1.24、Node 20、Wails CLI v2.11.0。
4. 构建命令：`wails build -platform windows/amd64`。
5. 产物：
   - `aegis-app.exe`
   - `aegis-app-windows-amd64.zip`
   - `run-aegis.bat`（默认设置 `AEGIS_DB_PATH=%USERPROFILE%\\aegis_node.db`）

### A2. 验收标准（DoD）

1. 在新设备下载 artifact 后可直接启动。
2. 登录弹窗中“创建身份/导入助记词”可正常写入本地数据库。
3. 不再出现“按钮点击无响应（实际后端异常未显式提示）”阻断测试。

## 5. 轨道 B 计划（已启动）

### B1. 最小替换

1. 驱动由 `github.com/mattn/go-sqlite3` 切换到 `modernc.org/sqlite`。
2. 仅调整数据库初始化入口（`sql.Open` driver name + DSN/PRAGMA），业务 SQL 不改。
3. 已完成：数据库初始化改为 `sqlite` 驱动，并在启动时显式设置 `busy_timeout`、`journal_mode=WAL`、`foreign_keys=ON`。

### B2. 双实现可切换（建议）

1. 保留 CGO 构建路径作为回滚兜底。
2. 通过构建标签或独立构建目标管理 CGO/纯 Go 两套产物。

### B3. 灰度切换

1. 先由测试同学使用纯 Go 驱动构建包回归。
2. 回归通过后再设为默认构建路径。

## 6. 风险与回滚

1. 轨道 A 风险：CI 依赖环境波动（toolchain 下载失败、网络波动）。
2. 轨道 B 风险：锁行为/性能边界与 CGO 版本存在差异。
3. 回滚策略：轨道 A 始终保留，轨道 B 任一步异常均可回退到 CGO 产物。

## 7. 当前执行状态

1. A1：Done（Windows workflow 已落地）。
2. A2：待测试同学验证。
3. B1：Done（纯 Go 驱动最小替换已完成）。
4. B2：Pending。
5. B3：Pending。
