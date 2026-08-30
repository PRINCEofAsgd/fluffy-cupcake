# 项目工作上下文

## 当前阶段

- 初版上线阶段（2026-08-14—2026-08-22）已归档至 [`context-archives/initial-release-stage.md`](context-archives/initial-release-stage.md)。
- 当前进入生产基础设施维护阶段，版本为 `V1.0.1_20260830`。

## Step 1：Singapore 应用连接 Guangzhou Managed MySQL

### 目标

- 在不改变业务语义和 golang-migrate 体系的前提下，为小数据量生产库提供安全、简单、可回滚的跨区域搬迁路径。
- 保留本地 Compose MySQL 开发能力，同时明确生产只通过 `DATABASE_DSN` 使用广州数据库 External/Public Endpoint。

### 实现

- `internal/database/mysql.go` 在 DSN 未显式配置时补 `timeout/readTimeout/writeTimeout=5s`，继续强制 `parseTime=true`、`loc=UTC` 和 `time_zone='+00:00'`，不覆盖自定义超时和 TLS 选择。
- 新增 `scripts/db-migration/verify.sh`：密码隐藏输入、临时配置权限 0600、只读生成 OLD/NEW 快照并比较全部表精确行数、Migration、列、PK/UNIQUE/普通索引、外键与 UTC 时间边界。
- 新增 `docs/user/database-region-migration.md`，完整记录 preflight、平台备份、停写、dump/restore、验证、DSN 切换、真实 API 冒烟、观察、两类回滚和旧库下线条件。
- `.env.example`、Compose、部署/开发/结构/调用链/脚本文档明确 Local 与 Production 职责；示例 DSN 显式展示 5 秒连接、读写超时，dump 目录与常见 dump 后缀加入 Git 忽略规则。
- 未改写既有六组 migration，也未修改 repository/service/model 业务语义；Sealos 控制台、真实数据和生产 Secret 均未由 Codex 操作。

### 验证

- `bash -n scripts/db-migration/verify.sh` 通过。
- `go test ./internal/database` 通过，覆盖超时默认值、显式超时保留、TLS 参数保留及 UTC 标准化。
- `go test ./...` 全部通过；`docker compose config --quiet`、全部 Shell 语法检查和 `git diff --check` 通过。
- 使用一次性 MySQL 8.4.11 容器依次应用 `000001`—`000006`，预检正确识别 6 张表、Migration、精确行数与 UTC 时间；同库模拟 OLD/NEW 时全部元数据比较通过，测试容器随后已停止并自动移除。
- Secret 扫描仅命中示例占位值和脚本写入临时配置的预期实现；`.env`、`db-migration-dumps/`、`*.sql.gz` 与 `*.dump` 的 Git 忽略规则已验证。

### 使用方法

- 日常本地开发继续使用 `make db-up`、`make migrate-up` 和宿主机 `DATABASE_DSN`，不受生产 Managed MySQL 架构影响。
- 数据库搬迁或灾备恢复前，按 `docs/user/database-region-migration.md` 设置不含密码的 `OLD_DB_*`/`NEW_DB_*` 连接变量，分别运行 `make db-preflight SIDE=OLD|NEW`；密码由脚本交互式隐藏读取。
- 停写并把完整 dump 恢复到空目标库后运行 `make db-verify`；只有全部快照一致才允许切换生产 DSN。切换产生新写入后不再运行全库一致性比较，而应使用定向业务基线确认唯一写入源。
- 生产 DSN 可显式设置 `timeout/readTimeout/writeTimeout`；省略时连接层补 5 秒默认值。证书可验证时使用 `tls=true`；只有明确接受服务器身份不校验风险时才使用 `tls=skip-verify`，禁止使用可能回退明文的 `tls=preferred`。

### 运维边界

- 工具只执行只读检查，不自动 dump、restore、切换 DSN、修改业务数据或删除数据库。
- 完整库迁移必须把 `schema_migrations` 随 dump 一起恢复到空库，禁止先运行 `migrate up`。
- 目标库产生新业务写入后，旧库与新库已经分叉，不能直接回切；必须停写并评估或同步增量。
- Secret、连接端点、真实业务数据和单次生产操作流水不记录在仓库；平台备份、观察期和旧实例下线均由人工按 Runbook 执行。
