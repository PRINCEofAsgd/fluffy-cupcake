# 项目工作上下文

## 当前阶段

- 初版上线阶段（2026-08-14—2026-08-22）已归档至 [`context-archives/initial-release-stage.md`](context-archives/initial-release-stage.md)。
- 当前进入 Homepage 与生产基础设施维护阶段，版本为 `V1.0.7_20260830`。

## Step 2：根路径个人 Homepage

### 目标

- 将根路径从跳转 `/yanlili` 改为个人 Homepage，同时保留现有想念页面、API、鉴权与数据库语义。
- 经授权参考 `github.com/LeoninCS/Homepage` 的视觉与交互实现，保留“等待全页资源全部加载”策略并在加载页明确标注来源。
- 删除参考作者个人身份、履历、联系方式、照片与外链，先以结构化占位内容交付，等待用户资料后再填写。

### 实现

- 新增 `web/homepage/` React/Vite 工程，复用森林首屏、滚动叙事、项目标签、经历/兴趣卡片及社媒轨道；公开图片和字体统一由 `/homepage/*` 返回。
- `useResourceGate` 并发预加载整页图片并等待字体就绪，全部完成前锁定滚动；静态首屏和 React 加载器均在进度条下显示“页面参考 github.com/LeoninCS/Homepage 设计。”。
- 页面已使用用户提供的原始头像和 `Loki Yin's Profile` 标题，首屏提供“给伴侣的点滴纪念”“可向我申请内测账号”入口；前者进入 `/yanlili`，后者进入联系区。
- 信息区按三组多行内容展示魔方、HOK、田径、教育与技术方向，并更新姓名、技术 ID、学校、当前工作地、职业和技术方向；首屏资料卡使用 GitHub、Sealos 与用户提供的 Vion 标识。
- 联系区已展示 GitHub、公开邮箱 `lokiyin4819@foxmail.com`、微信号 `Yyg1061` 与抖音主页；右上角原想念入口改为 GitHub，奖项区块与顶部“竞赛”入口已删除。
- 部署后将根路径浏览器标签标题改为“Loki Yin's Homepage”；进一步将 Homepage、`/yanlili`、Apple Touch Icon 和浏览器 `/favicon.ico` 回退统一为用户提供的 fluffy-cupcake PNG，并使用新 URL 与版本参数刷新旧图标缓存。
- `PageHandler` 在启动时读取 Vite 入口和资源文件系统，根路径直接返回 Homepage；带哈希资源使用 immutable 长缓存，想念页面继续使用原模板和资源路由。
- Vite 开发与预览入口由 `/homepage/` 调整为默认路径 `/`；构建脚本、样式、字体和图片仍隔离在 `/homepage/*`，`PageHandler` 从嵌套资源目录读取，避免与 `/yanlili` 的 `/assets/*` 冲突。旧 `/homepage/` 页面路径返回 404，根路径不发生重定向。
- Docker 增加 Node 22 Homepage 构建阶段，Go `embed.FS` 打包 `dist`；Makefile 在本地开发和测试前构建 Homepage。
- 新增脱敏社交分享图，占位元数据不包含真实个人信息；未修改数据库、Migration、Repository、Service、认证和业务 API。

### 待填写资料

- 代表经历、项目和兴趣的文字、时间、链接及图片。
- 博客或其他希望展示的社媒地址。
- SEO 标题、简介、关键词与社交分享图主副标题；当前继续使用明确占位，不根据已有资料推测填写。

### 验证

- `npm --prefix web/homepage run build` 通过，Vite 7.3.6 生成带哈希的 CSS/JavaScript 和完整公开资源；`npm audit --audit-level=high` 为 0 个已知漏洞。
- 本轮优化后再次运行 `make test` 通过；测试覆盖新首屏文案、分行信息、GitHub/邮箱/微信/抖音链接、三枚技术标识、奖项区删除与 PNG 头像公开资源，同时继续验证根路径、来源说明、参考作者信息脱敏、页面 CSP、带哈希资源和 immutable 缓存。
- 本地最终预览及 Loki Yin 头像均返回 HTTP 200；森林前景、占位图、字体和社交分享图仍由同一构建产物提供。`docker compose config --quiet`、Shell 语法检查和 `git diff --check` 通过。
- V1.0.5 根路径 Vite 预览返回 HTTP 200；本版本无 Migration、配置项、Secret、端口或数据库变化，Sealos 只需切换到新镜像标签并清理可能存在的外部根路径重定向。
- Docker Desktop 代理恢复后，显式拉取 `linux/amd64` 的 `node:22-alpine` 与 `golang:1.26.3-alpine`，`make build IMAGE=princeofasgd/fluffy-cupcake:V1.0.5_20260830` 完整通过；最终镜像为 `linux/amd64`、约 12.3 MB，并包含 `/app/fluffy-cupcake healthcheck` 健康检查。
- V1.0.5 已由用户成功部署；本轮标签标题与独立 favicon 修复提升为 V1.0.6，仍是无 Migration、配置或平台规则变更的纯应用更新。
- V1.0.6 再次通过 Homepage 生产构建和 `make test`；测试覆盖新 `<title>`、想念页面独立图标引用、PNG 资源类型以及两页图标隔离。`princeofasgd/fluffy-cupcake:V1.0.6_20260830` 已在本地构建为 `linux/amd64` 镜像，大小约 12.96 MB，继续以 `10001:10001` 运行并保留内置健康检查。
- V1.0.7 将用户提供的 fluffy-cupcake 原图等比例缩放为 512×512、保留透明通道，并统一用于 Homepage、`/yanlili`、Apple Touch Icon、`/favicon.png` 和 `/favicon.ico`；生产构建和 `make test` 通过，测试确认两条根级图标路由返回相同 PNG 且页面不再引用旧图标。`princeofasgd/fluffy-cupcake:V1.0.7_20260830` 已在本地构建为 `linux/amd64` 镜像，大小约 12.67 MB，运行用户和健康检查保持不变。

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
