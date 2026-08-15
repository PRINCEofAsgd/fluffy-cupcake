# 仓库结构

## 文档索引

- `README.md`：项目简介、快速开始和完整文档入口。
- `docs/structures/repository-structure.md`：仓库目录、文件职责及版本记录位置。
- `docs/structures/call-chains.md`：运行入口与主要请求调用链。
- `docs/foundation/db-dev-rules.md`：MySQL 分层、SQL、索引与 Migration 开发规范。
- `docs/structures/database-structure.md`：MySQL 表、字段、约束、索引和 UTC 时间语义。
- `docs/user/development.md`：开发、测试和构建方式。
- `docs/user/deployment.md`：容器和域名部署方法。
- `docs/user/user.md`：页面使用方法。
- `docs/user/release-notes.md`：当前发布版本和用户可见改动。
- `docs/record/project-context.md`：当前阶段工作记录。
- `docs/record/context-archives/`：仅在明确要求归档阶段时保存历史上下文。
- `scripts/README.md`：辅助脚本说明。

## 目录结构

```text
fluffy-cupcake/
├── cmd/
│   ├── server/                 # Web 程序入口与进程生命周期
│   └── create-user/            # 人工创建 bcrypt 固定用户的 CLI
├── internal/
│   ├── config/                 # 环境变量配置
│   ├── database/               # MySQL 连接池初始化与 UTC 连接语义
│   ├── healthcheck/            # 无外部依赖的容器健康检查客户端
│   ├── handler/                # HTTP 请求处理器
│   ├── middleware/             # JWT Cookie 鉴权与可信身份 Context
│   ├── model/                  # 用户、陪伴绑定与方向性点击数据模型
│   ├── repository/             # users/绑定信件/当前绑定/分钟桶显式 SQL
│   ├── service/                # bcrypt、JWT、绑定生命周期与统计业务规则
│   ├── server/                 # Gin 路由和中间件组装
│   └── version/                # 当前应用版本常量
│
├── migrations/                # golang-migrate 管理的 MySQL Up/Down 结构变更
├── web/
│   ├── assets/                 # GIF、MP3、CSS 和 JavaScript 页面资源
│   ├── templates/              # Go HTML 模板
│   └── embed.go                # 资源嵌入入口
│
├── deploy/caddy/               # 域名 HTTPS 反向代理配置
├── docs/                       # 结构、用户及工作记录文档
├── scripts/                    # 开发与构建辅助脚本
├── Dockerfile                  # 多阶段应用镜像构建
├── compose.yaml                # MySQL、Migration 工具、Gin 与 Caddy 编排
├── .env.example                # 不含真实 Secret 的本地配置模板
├── Makefile                    # 常用开发命令入口
├── go.mod / go.sum             # Go 模块与依赖锁定
├── .dockerignore               # 镜像构建上下文排除规则
└── .gitignore                  # Git 排除规则
```

## 重要文件职责

- `cmd/server/main.go`：校验配置、创建并关闭 MySQL 连接池、运行 HTTP Server 和优雅退出。
- `cmd/create-user/main.go`：终端无回显读取密码，经 UserService 生成 bcrypt 后写入固定用户。
- `internal/config/config.go`：读取应用、连接池、JWT 和 Cookie 环境变量，拒绝不完整敏感配置。
- `internal/database/mysql.go`：创建一次全局 `database/sql` 池，Ping 并强制连接使用 UTC。
- `internal/repository/user.go`：用户创建和按 username/id 的显式字段查询。
- `internal/repository/companion.go`：绑定信件、唯一当前绑定占位、备注、双向解绑和历史对象查询。
- `internal/repository/button_click.go`：绑定关系方向内分钟桶原子 Upsert、最新 8 条统计和 20 条详细分页查询。
- `internal/service/auth.go`：bcrypt 登录校验、HS256 JWT 签发解析和当前用户读取。
- `internal/service/companion.go`：邀请输入、备注、信件状态和 30 天未登录直接解绑规则。
- `internal/service/button_click.go`：限制 delta、解析当前绑定方向、生成 UTC 分钟桶和分页规则。
- `internal/middleware/auth.go`：从 HttpOnly Cookie 验证 JWT，把可信 `user_id` 写入 Gin Context。
- `internal/handler/auth.go`：登录、退出、当前用户接口和认证 Cookie 属性。
- `internal/handler/button_click.go`：点击参数解析、受保护写入与统计响应。
- `internal/handler/companion.go`：绑定、收件箱、备注、解绑与历史对象 HTTP 边界。
- `internal/server/router.go`：初始化 Gin，划分公开认证接口和受保护业务接口。
- `internal/handler/page.go`：渲染页面、读取嵌入资源并返回健康状态。
- `web/templates/yanlili.html`：页面语义结构与可访问性标记。
- `web/assets/app.css`：响应式布局、按钮反馈、文字上浮渐隐动画。
- `web/assets/app.js`：始终可用的点击动画、可并发 Web Audio 合成音效、独立登录状态和仅登录时的数据同步管理。
- `web/assets/miss-pop.mp3`：保留的历史提示音素材，当前页面不加载。
- `migrations/000001_create_users.*.sql`：创建/删除固定用户表。
- `migrations/000002_create_button_click_minutes.*.sql`：创建/删除用户维度分钟桶表。
- `migrations/000003_add_companion_bindings.*.sql`：升级用户/分钟桶并创建绑定信件与当前绑定占位表。
- `deploy/caddy/Caddyfile`：`fluffy-cupcake.cn` HTTPS 和反向代理策略。
- `deploy/sealos/Dockerfile.migrate`：为 Sealos 内网数据库构建不含凭据的一次性 Migration 镜像。
- `deploy/sealos/migrate-entrypoint.sh`：从运行时 `DATABASE_DSN` 执行受限的 `up` 或 `version` 命令。

## 版本记录位置

需要同步更新版本时，检查以下位置：

- `internal/version/version.go`
- `README.md`
- `docs/user/release-notes.md`
- `Dockerfile` 的版本构建参数
- `compose.yaml` 的镜像标签和 `APP_VERSION` 构建参数
- `Makefile` 的镜像标签
- `scripts/build.sh` 的默认镜像版本
