# 项目工作上下文

## Step 3：固定用户鉴权与按钮点击分钟桶（2026-08-15）

### 目标

- 为私人页面增加无公开注册的固定用户登录、JWT Cookie 鉴权和人工创建用户 CLI。
- 将 `/yanlili` 连续点击批量、可靠地持久化到 MySQL，并提供共享总数、UTC 每日和分钟统计。
- 严格保持 Handler → Service → Repository → MySQL 分层和 golang-migrate 结构管理。

### 实现

- 新增统一 MySQL 连接池和 UTC session，增加 `users`、`button_click_minutes` 两组可回滚 Migration。
- 新增用户/点击模型和 Repository；点击写入使用联合唯一索引加 MySQL 行别名原子 Upsert，统计使用显式字段 `SUM/GROUP BY`。
- 新增 bcrypt 用户创建 Service/CLI、HS256 JWT AuthService、HttpOnly Cookie Handler 和统一 Gin Auth Middleware。
- 新增登录、退出、当前用户、点击写入和统计 API；业务 Handler 只使用 Middleware 写入 Context 的可信 `user_id`。
- 页面新增简单登录入口、共享统计区和 750ms 点击批处理；发送失败会恢复 pending 并轻量重试，动画和 Step 2 音效不等待网络。
- 根据实际交互反馈，将登录改为不遮挡按钮的独立入口：匿名点击只做页面反馈，登录点击才进入 pending 和数据库，统计仅在登录时展示。
- 将六实例完整音轨叠加改为单实例可中断播放；文字先进入 DOM，音频随后异步启动，避免快速点击时音频拖慢文字反馈。
- 根据连续点击实测，将单实例音轨改为 Web Audio 合成短音；每次点击创建独立声部并经过共享限幅器，允许并发反馈且不再加载 MP3。
- 补充首次数据库自动初始化、三类 Secret 生成和 DSN 密码复用说明，并修正 DSN 未加引号导致开发脚本无法读取 `.env` 的问题。
- 经用户确认测试数据可丢弃后，使用当前 `.env` 重建 MySQL 数据卷并重新执行全部 Migration；开发文档补充仅删除 MySQL 卷的可重复重置步骤。
- Compose 新增 MySQL 8.4 和按需 golang-migrate 工具容器，敏感配置只从未提交的 `.env`/环境变量读取。

### 验证

- `go test ./...`、`go test -race ./...` 与 `go vet ./...` 通过；测试覆盖密码登录、bcrypt 随机盐、JWT 生成/解析/过期、Cookie Middleware、HTTP API、点击范围、同分钟累加、跨分钟/UTC 日期、多用户聚合和原子 Upsert SQL。
- `node --check web/assets/app.js` 与包含 tools profile 的 `docker compose config` 通过。
- 真实 MySQL 8.4 成功执行 `000001/000002 up`；`SHOW/DESCRIBE/SHOW INDEX` 确认字段、外键、唯一约束和统计索引均正确。
- 创建用户 CLI 的终端无回显路径成功，重复用户名返回清晰错误；数据库哈希前缀为 bcrypt，且不等于明文。
- 真实接口验证：无 Cookie `/me` 为 401，错误密码为统一 401，正确登录返回 HttpOnly/Lax Cookie，携带 Cookie 的 `/me` 为 200 且不含哈希；退出返回过期 Cookie，随后 `/me` 恢复 401。
- 同一 UTC 分钟分别写入 `+3`、`+5` 后，MySQL 仅一行且 `click_count=8`、秒为 0；统计 API 返回 total/day/minute 均为 8。
- 总数与分钟统计 EXPLAIN 均选择 `idx_click_button_minute`、访问类型 `ref`；当前仅一行，估算 rows=1 不能外推大数据表现。
- 真实验证发现并修复 MySQL 将 `utc_date` 解析为内置函数名导致的每日统计 1064，改用 `click_date` 后回归通过。
- `docker build --platform linux/amd64` 成功生成 `fluffy-cupcake:V0.0.4_20260815` scratch 镜像。
- 交互优化后 `go test ./...`、`go vet ./...`、`node --check web/assets/app.js` 和页面结构回归通过。
- 本地浏览器未登录状态连续点击 5 次，同一时刻生成 5 条文字反馈、实时文案计数为 5，且没有控制台错误。
- 匿名点击前后数据库均为总数 53、分钟桶 3 行，服务日志没有点击 POST；独立登录表单展开时按钮仍可见，统计区保持隐藏。
- `docker build --platform linux/amd64 --build-arg APP_VERSION=V0.0.5_20260815` 成功生成 `fluffy-cupcake:V0.0.5_20260815` scratch 镜像。
- Web Audio 优化后 `go test ./...`、`go test -race ./...`、`go vet ./...`、JS/`.env.example` 语法、Compose 配置和差异格式检查通过；静态资源回归会阻止恢复单实例 `new Audio(...)`。
- `docker build --platform linux/amd64 --build-arg APP_VERSION=V0.0.6_20260815` 成功生成 `fluffy-cupcake:V0.0.6_20260815` scratch 镜像。
- 当前 `.env` 字段关系校验通过；旧 `fluffy-cupcake_mysql_data` 删除后由 Compose 创建新卷，Migration 成功执行至版本 2 且 `dirty=0`，用户与点击表均为 0 行。
- DSN 补齐外层单引号后，`make dev` 能直接读取当前 `.env`、连接新库并通过 `/healthz` 检查；验证完成后已正常停止临时 Gin 服务，MySQL 保持健康运行。
- 经用户明确确认外部目标后，`linux/amd64` 镜像已推送至 `docker.io/princeofasgd/fluffy-cupcake:V0.0.6_20260815`，远端摘要为 `sha256:fde971b112fe16f9dafef0a92644c4efb9b2ba1031ea7c4e3f50a5a3e0e6f786`；本轮未操作 Sealos。
- 为仅内网可达的 Sealos MySQL 新增一次性 Migration 镜像：运行时读取业务应用同款 `DATABASE_DSN`，镜像不携带凭据，并将可执行命令限制为 `up`/`version`。
- 本地 Migration 镜像以非 root 用户验证通过：缺少 DSN 和 `down` 均按预期拒绝，连接当前 MySQL 查询版本为 2，重复执行 `up` 返回 `no change`。
- 经用户明确确认后，Migration 镜像已推送至 `docker.io/princeofasgd/fluffy-cupcake-migrate:V0.0.6_20260815`，远端摘要为 `sha256:79f8645876eddebf47f1134f8e843540656d02a25808b2fc0160613ff26a5025`。
- Sealos 实测 Kubernetes Job 等待 180 秒仍未完成，且工作区报告 restricted PodSecurity 不满足；按用户决定放弃 Job 路径，改用目标 MySQL 的 Database Studio 直接执行 SQL。流程要求先删除旧 Job/Secret 并只读检查现有表，避免迟到执行或部分迁移引发重复建表。
- Database Studio 已在 `fluffy_cupcake` 库依次建立迁移记录表和两张业务表，最终核对为 `version=2`、`dirty=0`；生产镜像不含创建用户 CLI，部署文档补充 macOS 隐藏输入生成 bcrypt 哈希、Database Studio 仅写入哈希的固定用户创建方式。

### 后续

- 首次部署需生成真实 MySQL 密码和 JWT Secret，并按开发/部署文档执行 Migration 和创建固定用户。

## Step 2：按钮点击音效（2026-08-15）

### 目标

- 每次点击 `/yanlili` 页面按钮时播放用户提供的 `miss-pop.mp3` 音效。
- 保留现有登录、点击统计、文字动画和 Linux 镜像部署流程。

### 实现

- 将 MP3 作为嵌入式页面资源提供，并补充 `audio/mpeg` 路由与同源媒体安全策略。
- 页面通过版本化资源地址加载音效，使用六个音频实例轮换播放，支持快速连续点击。
- 音效播放失败不会影响文字动画、点击计数或服务端同步。

### 验证

- `node --check web/assets/app.js` 通过。
- `go test ./...` 与 `go vet ./...` 通过，包含 MP3 路由、媒体类型和页面资源引用验证。
- `docker compose config` 在校验用环境变量下通过。
- `make build` 成功生成 `linux/amd64` 镜像 `fluffy-cupcake:V0.0.3_20260815`。

## Step 1：项目基线、Gin 服务与互动页面（2026-08-14）

### 目标

- 首次建立符合 `AGENTS.md` 职责划分的项目文档框架。
- 按 Gin 项目常用分层方式建立可维护的 Web 服务。
- 在 `/yanlili` 提供 GIF 按钮和点击文字上浮渐隐动效。
- 保持本地开发监听全部网口，并为 `fluffy-cupcake.cn` 准备容器化 HTTPS 部署。

### 实现

- 建立 `cmd/server`、`internal/config`、`internal/handler`、`internal/server`、`web` 分层。
- 将模板、样式、脚本和 `miss-button.gif` 嵌入二进制。
- 增加优雅关闭、请求日志、异常恢复、安全响应头、404 和 `/healthz`。
- 页面支持鼠标、触屏、键盘连续激活，以及系统减少动画偏好。
- 增加多阶段 `Dockerfile`、生产 `compose.yaml` 和 Caddy 自动 HTTPS 配置。
- 根据后续要求，将本地开发、容器运行、健康检查和反向代理目标端口统一调整为 4819。
- 移除本机二进制构建流程，将 `make build` 和 `scripts/build.sh` 统一调整为构建 Linux 镜像；默认目标为 `linux/amd64`，并支持显式切换 `linux/arm64`。
- 将运行镜像调整为 `scratch`，由静态 Go 二进制提供容器健康检查子命令，避免最终镜像依赖目标架构的基础系统、shell 或 wget。
- 建立说明、结构、调用链、数据库、开发、部署、操作、版本、脚本等文档。

### 验证

- `go test ./...` 通过。
- `go vet ./...` 通过。
- `go build ./cmd/server` 通过。
- `docker compose config` 通过。
- 启动本地发布模式二进制后，`/yanlili`、`/healthz`、页面文案、GIF 引用和安全响应头端到端检查通过，并验证 SIGINT 优雅退出。
- 端口调整后在 `127.0.0.1:4819` 再次完成页面、健康检查、版本资源参数和优雅退出验证。
- 初次 `docker build` 受 Docker Hub 鉴权连接超时影响；改用缓存构建器与 `scratch` 运行阶段后已完成构建。
- 调整为 `scratch` 运行镜像后，`make build` 已成功生成 `linux/amd64` 镜像 `fluffy-cupcake:V0.0.2_20260814`，镜像大小约 8.1 MB。
- 镜像元数据验证为 `linux/amd64`、用户 `10001:10001`、暴露 4819、内置二进制健康检查；在 ARM64 开发机通过模拟运行后，`/yanlili` 和 `/healthz` 正常且容器状态为 `healthy`。

### 后续

- 服务器上线前需配置域名 DNS、开放 80/443 端口并启动 Compose。
