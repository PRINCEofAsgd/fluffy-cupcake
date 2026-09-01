# 部署指南

## 架构

部署职责明确分离：本地/独立单机开发由 `compose.yaml` 管理 MySQL 8.4、Migration 工具、Gin 和 Caddy；Sealos 生产网站与 Gin 应用位于 Singapore，数据库位于 Guangzhou Managed MySQL，应用通过数据库自身提供的 External/Public Endpoint 和运行时 `DATABASE_DSN` 连接。生产不得依赖 Compose service name `mysql` 或跨 Region `.svc` 地址。公开域名为 `fluffy-cupcake.cn`。

```text
Local:      App → Docker Compose MySQL
Production: Singapore Sealos App → Guangzhou Sealos Managed MySQL
```

完整搬迁、验证、切换和回滚步骤见 [数据库跨区域迁移 Runbook](database-region-migration.md)。

## 部署前准备

1. 将 `fluffy-cupcake.cn` 指向 Singapore 应用入口并启用 HTTPS。
2. 在 Guangzhou 创建 Managed MySQL 8.x，开启 External/Public Endpoint 与自动备份；若实例支持来源 IP 白名单，只放行 Singapore 应用出口地址。
3. 为应用创建独立随机密码账号，禁止使用 root；管理/migration 权限与运行时账号分离。
4. 在 Sealos Secret/环境变量设置 `DATABASE_DSN` 和 JWT Secret，不提交或发送 Secret。DSN 必须使用 Guangzhou 公网端点，并配置 `timeout=5s&readTimeout=5s&writeTimeout=5s`；证书链可验证时使用 `tls=true`，无法取得 CA 且明确接受服务器身份不校验风险时可使用 `tls=skip-verify`，禁止使用会回退明文的 `tls=preferred`。
5. 部署前从 Singapore 环境验证目标 DNS、TCP 3306、认证与启动 Ping。

## 构建镜像

```bash
make build
```

默认产物是 `linux/amd64` 镜像 `fluffy-cupcake:V1.1.0_20260901`。若服务器是 ARM64，运行 `make build TARGET_PLATFORM=linux/arm64`。镜像先以 Node 22 构建 Homepage，再进行 Go 多阶段交叉编译；无基础系统的 `scratch` 运行阶段最终仍仅包含静态服务二进制，以非 root 用户启动，并通过程序自身完成 `/healthz` 健康检查。

### V1.1.0 纯应用更新

本版本细化 Homepage 的彩色章节标题、文安智能实习与 Sealos 开源经历文案，并优化主机辅助取景、圆角裁切和 10 枚技术栈 Logo 的统一尺寸与分布；没有数据库结构、Migration、环境变量、Secret、端口、健康检查或持久化数据变化。生产数据库保持当前 `schema_migrations.version=6`、`dirty=0`，不需要执行 `migrate up`。

部署时构建并推送新的不可变镜像标签，再把 Sealos Application 的镜像地址更新为该标签即可。保留现有 `DATABASE_DSN`、`JWT_SECRET`、`JWT_EXPIRE`、端口 `4819` 和健康检查配置；不需要额外部署资源或平台规则。

更新完成后至少检查 `/healthz` 返回 200、`/` 展示 Homepage、`/yanlili` 可访问，并完成一次登录或 `/api/auth/me` 冒烟。建议使用新标签触发滚动发布；不要复用旧标签依赖平台强制拉取。

## 本地/独立单机 Compose

```bash
docker compose up -d mysql
docker compose --profile tools run --rm migrate up
docker compose up -d --build
docker compose ps
docker compose logs -f mysql app caddy
```

首次 Migration 后使用 `make create-user USERNAME=用户名` 人工建立固定用户。DNS 生效且端口开放后，访问 `https://fluffy-cupcake.cn/` 查看 Homepage，想念页面位于 `/yanlili`。生产 `APP_MODE=release` 会给认证 Cookie 增加 `Secure`；Caddy 提供的 HTTPS 仍是保护 JWT 不在传输中泄露的必要条件。

这些命令中的 `mysql:3306` 只在本 Compose 网络内有效。MySQL 数据保存在 `mysql_data`，证书与 Caddy 状态分别保存在命名卷中，更新容器不会丢失。应用不会自动执行 Migration，部署新版本前应先审阅并显式运行 `migrate up`。

## Sealos 生产运行配置

Singapore Application 只部署应用镜像并从 Secret 注入配置，不在该 Region 随应用启动 Compose MySQL。示例 DSN 仅展示格式：

```text
fluffy_cupcake_app:URL_ENCODED_PASSWORD@tcp(GUANGZHOU_PUBLIC_ENDPOINT:3306)/fluffy_cupcake?timeout=5s&readTimeout=5s&writeTimeout=5s
```

连接层仍强制 `parseTime=true`、`loc=UTC` 和 `time_zone='+00:00'`。若 DSN 未显式设置三项网络超时，应用会分别补 5 秒；显式值不会被覆盖。公共可信证书或已注册自定义 CA 时使用 `tls=true`。平台只能提供无法验证的证书且已明确接受风险时可使用 `tls=skip-verify`：传输被加密，但不能验证服务器身份，仍存在主动中间人风险；取得官方 CA 后应升级为严格验证。禁止使用可能回退为明文的 `tls=preferred`。

## Sealos 内网 Migration

本节只适用于新建空 schema 或未来单独执行版本升级，不适用于数据库跨区域全库搬迁。跨区域搬迁会恢复包含 `schema_migrations` 的完整 dump，恢复前禁止执行本节操作或 `migrate up`。

当 MySQL 只有 Sealos 内网地址、工作区的 Kubernetes Job 又无法稳定启动时，使用目标 MySQL 实例自带的 Database Studio 直接执行 Migration SQL，不再创建迁移 Job。若此前创建过 Job 和临时 Secret，先在同一 namespace 的 Terminal 中删除，防止其稍后恢复并重复迁移：

```bash
kubectl delete job fluffy-cupcake-migrate-v006 --ignore-not-found
kubectl delete secret fluffy-cupcake-migrate-v006 --ignore-not-found
```

打开目标 MySQL 的 Database Studio，选中业务 DSN 指向的数据库，先只读检查当前状态：

```sql
SELECT DATABASE() AS current_database, VERSION() AS mysql_version;

SELECT TABLE_NAME
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN (
    'schema_migrations', 'users', 'button_click_minutes',
    'companion_bindings', 'companion_active_memberships',
    'qr_login_mappings'
  )
ORDER BY TABLE_NAME;
```

必须根据检查结果继续，不能在未知状态下重复执行建表 SQL：

- 全部业务表均不存在：先建立空的 `schema_migrations`；依次执行 `000001` 至 `000006` 的 Up SQL，每一步明确成功后再登记对应版本，最终为 `version=6, dirty=0`。
- 已有 `users`、`button_click_minutes` 且 `version=2, dirty=0`：暂停应用写入，依次执行 `000003` 至 `000006`；`000003` 原地升级点击表，不删除旧点击。若已明确旧记录对象，应先创建该用户，再只补旧行的 `target_user_id` 并保持 `companion_binding_id = NULL`；不要为了迁移伪造绑定信件。
- 五张业务表完整且 `version=3, dirty=0`：依次执行 `000004`、`000005` 和 `000006`，最终登记为 `version=6, dirty=0`。
- 五张业务表完整且 `version=4, dirty=0`：依次执行 `000005` 和 `000006`，成功后登记为 `version=6, dirty=0`。
- 五张业务表完整且 `version=5, dirty=0`：只执行 `migrations/000006_create_qr_login_mappings.up.sql`，成功后登记为 `version=6, dirty=0`。该步骤只新建二维码映射表，不修改用户、绑定或想念记录。
- 只存在其中部分表，或 `schema_migrations` 的 `dirty` 为 `1`：先核对现有表结构，再补齐缺失部分；不要先修改版本号。
- 上述六张表均存在：核对 `SELECT version, dirty FROM schema_migrations;`；当前期望仅一行 `version=6`、`dirty=0`。

Database Studio 操作不需要公开 MySQL 地址，也不需要把 DSN 或密码粘贴到 SQL、YAML、命令历史和聊天记录。建表和迁移版本登记完成后，生产镜像因采用 `scratch` 且只包含服务端二进制，不能在容器内运行本地 `create-user` CLI。可在 macOS Terminal 使用系统自带的 `htpasswd` 隐藏读取登录密码并生成与服务端兼容的 bcrypt cost 10 哈希：

```bash
htpasswd -nBC 10 fluffy_cupcake
```

密码必须为 8 到 72 个字节；该登录密码与 MySQL 密码、root 密码、JWT Secret 相互独立。命令输出格式为 `用户名:$2y$10$...`，只复制冒号后的完整哈希，在 Database Studio 中写入，不要把实际密码或哈希发到聊天：

```sql
INSERT INTO users (username, password_hash)
VALUES ('fluffy_cupcake', '粘贴冒号后的完整 bcrypt 哈希');

SELECT id, username, LEFT(password_hash, 7) AS hash_prefix, created_at
FROM users;
```

预期哈希前缀为 `$2y$10$`；服务端的 Go bcrypt 校验兼容该前缀。用户名具有唯一约束，重复执行插入会失败而不会覆盖现有用户。

## 二维码卡片映射

生产镜像同样不含创建映射 CLI，可在 Database Studio 中插入二维码文本与用户的映射：

```sql
INSERT INTO qr_login_mappings (qr_text, user_id)
SELECT '二维码中的完整永久文本', id FROM users WHERE username = '用户名';

SELECT m.id, m.qr_text, u.username
FROM qr_login_mappings m
JOIN users u ON u.id = m.user_id;
```

文本必须与卡片上的二维码内容逐字一致（扫描识别的结果），二维码文本具有唯一约束，重复插入会失败；删除该行即可停用对应卡片。已执行 `000006` 后 `qr_login_mappings` 才会存在。

## 更新与停止

```bash
docker compose up -d --build
docker compose down
```

`docker compose down` 不会删除命名卷；不要附加 `--volumes`，除非明确需要永久清除 MySQL 数据、证书和 Caddy 状态。
