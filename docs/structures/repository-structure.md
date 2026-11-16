# 仓库结构

## 文档索引

- `README.md`：项目简介、快速开始和完整文档入口。
- `docs/structures/repository-structure.md`：仓库目录、文件职责及版本记录位置。
- `docs/structures/call-chains.md`：运行入口与主要请求调用链。
- `docs/foundation/db-dev-rules.md`：MySQL 分层、SQL、索引与 Migration 开发规范。
- `docs/structures/database-structure.md`：MySQL 表、字段、约束、索引和 UTC 时间语义。
- `docs/user/development.md`：开发、测试和构建方式。
- `docs/user/deployment.md`：容器和域名部署方法。
- `docs/user/incremental-deployments/README.md`：按上一部署版本维护增量手册的规则和入口。
- `docs/user/database-region-migration.md`：Singapore 应用连接 Guangzhou Managed MySQL 的迁移、验证、切换和回滚 Runbook。
- `docs/user/user.md`：页面使用方法。
- `docs/user/release-notes.md`：当前发布版本和用户可见改动。
- `docs/record/project-context.md`：当前阶段工作记录与上一阶段归档入口。
- `docs/record/context-archives/initial-release-stage.md`：2026-08-14 至 2026-08-22 的初版上线阶段完整工作记录。
- `docs/record/context-archives/homepage-development-and-standardized-deployment-stage.md`：2026-08-30 至 2026-09-01 的 Homepage 开发与规范化部署阶段完整工作记录。
- `docs/record/context-archives/`：仅在明确要求归档阶段时保存历史上下文。
- `scripts/README.md`：辅助脚本说明。



## 目录结构

```text
fluffy-cupcake/
├── cmd/
│   ├── server/                 # Web 程序入口与进程生命周期
│   ├── create-user/            # 人工创建 bcrypt 固定用户的 CLI
│   └── create-qr-login/        # 创建二维码文本到用户映射的 CLI
│
├── internal/
│   ├── config/                 # 环境变量配置
│   ├── database/               # MySQL 连接池初始化与 UTC 连接语义
│   ├── healthcheck/            # 无外部依赖的容器健康检查客户端
│   ├── handler/                # 负责处理 HTTP 请求与响应
│   ├── middleware/             # 中间件，即多个接口共用链路解耦
│   ├── model/                  # 业务数据模型
│   ├── repository/             # 显式 SQL 读写
│   ├── server/                 # Gin 路由和中间件组装
│   ├── service/                # 负责执行 handler 对应接口的业务规则
│   └── version/                # 当前应用版本常量
│
├── migrations/                # golang-migrate 管理的 MySQL Up/Down 结构变更
│
├── web/
│   ├── assets/                 # 想念页 GIF、MP3、CSS 和 JavaScript 页面资源
│   ├── templates/              # Go HTML 模板
│   ├── homepage/               # React/Vite Homepage 源码、公开个人资料、占位资源与构建产物
│   └── embed.go                # 资源嵌入入口
│
├── deploy/caddy/               # 域名 HTTPS 反向代理配置
│
├── docs/                       # 结构、用户及工作记录文档
│   ├── user/                   # 用户与部署文档
│   │   └── incremental-deployments/    # 按已部署版本命名的增量手册
│   └── record/
│       ├── project-context.md  # 当前阶段上下文与历史归档入口
│       └── context-archives/   # 按阶段保存的完整历史上下文
│
├── scripts/                    # 开发与构建辅助脚本
│   └── db-migration/           # OLD/NEW MySQL 只读预检与一致性验证
│
├── Dockerfile                  # 多阶段应用镜像构建
├── compose.yaml                # 本地/独立单机的 MySQL、Migration 工具、Gin 与 Caddy 编排
├── .env.example                # 不含真实 Secret 的本地配置模板
├── Makefile                    # 常用开发命令入口
├── go.mod / go.sum             # Go 模块与依赖锁定
├── .dockerignore               # 镜像构建上下文排除规则
└── .gitignore                  # Git 排除规则
```



## 重要文件职责

### cmd/
- `cmd/server/main.go`：校验配置、创建并关闭 MySQL 连接池、运行 HTTP Server 和优雅退出。
- `cmd/create-user/main.go`：终端无回显读取密码，经 UserService 生成 bcrypt 后写入固定用户。
- `cmd/create-qr-login/main.go`：校验真实用户后写入二维码永久文本到用户的映射，可从命令行或标准输入读取文本。

### internal/
- `internal/model/companion_mood.go`：绑定内的三组心情选项及 API 快照模型。
- `internal/repository/companion_mood.go`：按活跃关系解析心情视角，并把本人保存与解绑按同一信件锁串行化。
- `internal/service/companion_mood.go`：校验固定选项、自定义文字和读写视角；`internal/service/companion_mood_test.go` 覆盖输入与方向校验。
- `internal/handler/companion_mood.go`：处理心情 API 的读写、请求体校验和响应。
- `internal/server/companion_mood_mysql_test.go`：通过真实 MySQL 和 HTTP 验证心情权限、保存、清空与解绑重绑。
- `internal/config/config.go`：读取应用、连接池、JWT 和 Cookie 环境变量，拒绝不完整敏感配置。
- `internal/database/mysql.go`：创建一次全局 `database/sql` 池，Ping、强制 UTC，并为未配置的连接/读/写网络超时补 5 秒默认值。
- `internal/server/router.go`：初始化 Gin，划分公开认证接口和受保护业务接口。

- `internal/repository/user.go`：用户创建和按 username/id 的显式字段查询。
- `internal/repository/qr_login.go`：二维码文本到用户的映射查询、创建与删除。
- `internal/repository/companion.go`：绑定信件、唯一当前绑定占位、备注、统一解绑判定、取消/拒绝申请和历史对象查询。
- `internal/repository/button_click.go`：关系方向内分钟桶原子 Upsert、跨历次用户对摘要和 20 条详细分页查询。
- `internal/repository/auth_session.go`：登录事务、凭证哈希及历史、刷新轮换的行锁和会话撤销。

- `internal/model/auth_session.go`：会话 ID、认证用户和固定截止时间。
- `internal/middleware/auth.go`：校验访问 Cookie 与 MySQL 会话；成功时写入可信身份，失败时标记 Handler 执行前的拒绝。

- `internal/service/auth.go`：密码和二维码共用的会话创建、HS256 访问 JWT 签发和校验、续期及退出。
- `internal/service/companion.go`：邀请输入、备注、信件状态和统一解绑的 30 天判定规则。
- `internal/service/button_click.go`：限制 delta、解析当前绑定方向、生成 UTC 分钟桶和分页规则。

- `internal/handler/auth.go`：登录、二维码登录、刷新、退出、当前用户接口和两个认证 Cookie 的属性。
- `internal/handler/button_click.go`：点击参数解析、受保护写入与统计响应。
- `internal/handler/companion.go`：绑定、收件箱、备注、解绑与历史对象 HTTP 边界。
- `internal/handler/page.go`：返回 Homepage、渲染想念页面、安全读取两类嵌入资源并返回健康状态。

### web/
- `web/homepage/src/main.jsx`：参考授权实现的 Homepage 内容、工作/项目/爱好数据、完整资源加载门和滚动动效。
- `web/homepage/src/styles.css`：Homepage 森林首屏、产品化预览、叙事卡片、社媒轨道和响应式视觉规则。
- `web/homepage/public/homepage/`：Homepage 的 `/homepage/*` 自托管字体、非个人装饰图、从用户照片重新取景的单层主机背景、资料占位图和社交分享图。
- `web/homepage/public/homepage/images/tech-stack/`：首屏使用的 10 枚自托管技术栈 SVG 及 Docker 图标来源说明。
- `web/homepage/public/homepage/images/experience/`：经历品牌面板使用的 Vion Tech Icon 与 VionVision WordMark SVG。
- `web/homepage/public/favicon.png`：从用户原图等比例缩放到 512×512，供所有页面和 `/favicon.ico` 回退共用的 PNG 站点图标。
- `web/homepage/dist/`：Vite 生成的根路径入口及 `homepage/` 资源目录，由 Go `embed.FS` 一并打包。
- `web/homepage/scripts/build-homepage.cjs`：清理并构建 Homepage，校验入口产物存在。
- `web/homepage/package.json / package-lock.json`：Homepage Node 依赖、构建命令和锁定版本。
- `web/templates/yanlili.html`：想念页面语义结构、心情三栏和编辑表单。
- `web/assets/app.css`：响应式布局、心情三栏与表单、色灯、按钮反馈及文字上浮动画。
- `web/assets/app.js`：管理点击动画、Web Audio 音效、扫码登录与会话恢复、绑定关系、想念记录以及心情视角和本人编辑草稿。
- `web/assets/qrdecoder.js`：内置的 jsQR 二维码解码库，页面扫码登录使用，不依赖外部 CDN。
- `web/assets/mood/reasons/`、`web/assets/mood/comforts/`：影响原因和想要安慰的 SVG 资源，文件名对应固定选项标识，每组的 `custom.svg` 为自定义文字的立方体图标。由 `web/embed.go` 嵌入并通过 `/assets/mood/:group/:name` 提供。
- `web/assets/stats-direction.svg`、`web/assets/mood-edit.svg`：统计区的方向和编辑图案，统一可见尺寸，由 `/assets/` 下的同名地址提供。
- `web/assets/miss-pop.mp3`：保留的历史提示音素材，当前页面不加载。

### migrations/
- `migrations/000001_create_users.*.sql`：创建/删除固定用户表。
- `migrations/000002_create_button_click_minutes.*.sql`：创建/删除用户维度分钟桶表。
- `migrations/000003_add_companion_bindings.*.sql`：升级用户/分钟桶并创建绑定信件与当前绑定占位表。
- `migrations/000004_remove_utc_click_date.*.sql`：移除与浏览器本地日界线冲突的 UTC 生成日期列及索引。
- `migrations/000005_add_unbind_request_states.*.sql`：增加可反馈、可重复申请的解绑子状态和处理字段。
- `migrations/000006_create_qr_login_mappings.*.sql`：创建/删除二维码永久文本到真实用户的一一映射表。
- `migrations/000007_create_auth_sessions.*.sql`：创建/删除固定截止的会话和刷新凭证哈希表。
- `migrations/000008_create_companion_moods.*.sql`：创建/删除按关系和填写者保存的心情快照表。

### deploy/
- `deploy/caddy/Caddyfile`：`fluffy-cupcake.cn` HTTPS 和反向代理策略。
- `deploy/sealos/Dockerfile.migrate`：为 Sealos 内网数据库构建不含凭据的一次性 Migration 镜像。
- `deploy/sealos/migrate-entrypoint.sh`：从运行时 `DATABASE_DSN` 执行受限的 `up` 或 `version` 命令。
- `scripts/db-migration/verify.sh`：隐藏读取 OLD/NEW 密码，以只读查询比较表、行数、Migration、列、索引、外键和 UTC 时间边界。

### docs/
- `docs/user/incremental-deployments/from-V1.1.0_20260901.md`：会话和心情功能已完成上线的迁移、切换、验收与回退记录。
- `docs/user/incremental-deployments/from-V1.1.0_20261116.md`：当前生产部署基线及下一次局部更新手册。
- `docs/user/database-region-migration.md`：跨区域全库 dump/restore、生产切换、冒烟、观察、回滚和旧库下线操作手册。
- `docs/record/project-context.md`：记录当前阶段状态，并指向已经收口的阶段归档。
- `docs/record/context-archives/initial-release-stage.md`：保存初版上线阶段 Step 1–5 的目标、实现、验证和后续事项。
- `docs/record/context-archives/homepage-development-and-standardized-deployment-stage.md`：保存 Homepage 开发与规范化部署阶段 Step 6–8 的最终目标、实现、验证和后续事项。



## 版本记录位置

需要同步更新版本时，检查以下位置：

- `internal/version/version.go`
- `README.md`
- `docs/user/release-notes.md`
- `Dockerfile` 的版本构建参数及 Homepage Node 构建阶段
- `compose.yaml` 的镜像标签和 `APP_VERSION` 构建参数
- `Makefile` 的镜像标签
- `scripts/build.sh` 的默认镜像版本及 `scripts/README.md` 的镜像示例
- `web/homepage/index.html` 和重建后的 `web/homepage/dist/index.html` 中的图标缓存版本
- `docs/user/development.md`、`docs/user/deployment.md` 中的镜像示例及当前增量部署手册的目标版本
