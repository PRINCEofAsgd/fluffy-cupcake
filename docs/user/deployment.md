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

默认产物是 `linux/amd64` 镜像 `fluffy-cupcake:V1.1.0_20261115`。若服务器是 ARM64，运行 `make build TARGET_PLATFORM=linux/arm64`。镜像先以 Node 22 构建 Homepage，再进行 Go 多阶段交叉编译；无基础系统的 `scratch` 运行阶段最终仍仅包含静态服务二进制，以非 root 用户启动，并通过程序自身完成 `/healthz` 健康检查。

## 已有环境更新

局部更新部署手册按上一次已部署的版本维护。本次会话更新要求先迁移 MySQL，再一次性切换应用并更新环境变量，已有用户需重新登录。见 [从 V1.1.0_20260901 更新到 V1.1.0_20261115](incremental-deployments/from-V1.1.0_20260901.md)。

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

在 Singapore Application 设置 `ACCESS_TOKEN_TTL=15m` 和 `SESSION_TTL=360h`。从旧版升级时移除 `JWT_EXPIRE`，保留独立的 `JWT_SECRET`。认证 Cookie 沿用 release 模式的 `Secure` 和 `SameSite=Lax`。

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
    'qr_login_mappings', 'auth_sessions', 'auth_refresh_tokens'
  )
ORDER BY TABLE_NAME;
```

必须根据检查结果继续，不能在未知状态下重复执行建表 SQL：

- 空库从 `000001` 顺序执行至 `000007`，已部署库从当前干净版本的下一条 Up 开始。当前预期为 `version=7, dirty=0`；`version=6, dirty=0` 的已有库只需做 `000007`，见当前增量手册。
- 从 2 或更早版本跨过 `000003` 时，暂停应用写入。明确了旧点击的记录对象后，可创建真实用户并补旧行的 `target_user_id`，保留 `companion_binding_id = NULL`。
- 遇到部分建表或 dirty 状态时，先核对实际表结构及迁移记录。每步执行成功后再登记对应版本。

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
