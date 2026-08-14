# 部署指南

## 架构

生产环境由 Docker Compose 管理 MySQL 8.4、Gin 和 Caddy：MySQL 使用命名卷持久化；Gin 在内部网络监听 4819；Caddy 通过公网 80/443 申请 HTTPS 证书并反向代理。`migrate/migrate:v4.19.1` 仅作为按需工具容器执行结构变更，不常驻运行。公开域名为 `fluffy-cupcake.cn`。

## 部署前准备

1. 将 `fluffy-cupcake.cn` 的 A 记录指向服务器公网 IPv4；若服务器可用 IPv6，再配置 AAAA 记录。
2. 放通服务器 TCP 80、TCP 443；若需要 HTTP/3，同时放通 UDP 443。
3. 安装 Docker Engine 和 Docker Compose v2，将本仓库放到服务器。
4. 复制 `.env.example` 为 `.env`；分别使用 `openssl rand -hex 24` 生成应用数据库密码和 root 密码，再用 `openssl rand -hex 32` 生成独立 JWT Secret。`DATABASE_DSN` 复用应用数据库密码，不使用 root 密码；不要提交或发送 `.env`。
5. 确认 80/443 端口未被其他服务占用；MySQL 3306 只绑定服务器回环地址。

## 构建镜像

```bash
make build
```

默认产物是 `linux/amd64` 镜像 `fluffy-cupcake:V0.0.6_20260815`。若服务器是 ARM64，运行 `make build TARGET_PLATFORM=linux/arm64`。镜像采用多阶段交叉编译和无基础系统的 `scratch` 运行阶段，最终仅包含静态服务二进制，以非 root 用户启动，并通过程序自身完成 `/healthz` 健康检查。

## 启动服务

```bash
docker compose up -d mysql
docker compose --profile tools run --rm migrate up
docker compose up -d --build
docker compose ps
docker compose logs -f mysql app caddy
```

首次 Migration 后使用 `make create-user USERNAME=用户名` 人工建立固定用户。DNS 生效且端口开放后，访问 `https://fluffy-cupcake.cn/yanlili`。生产 `APP_MODE=release` 会给认证 Cookie 增加 `Secure`；Caddy 提供的 HTTPS 仍是保护 JWT 不在传输中泄露的必要条件。

MySQL 数据保存在 `mysql_data`，证书与 Caddy 状态分别保存在命名卷中，更新容器不会丢失。应用不会自动执行 Migration，部署新版本前应先审阅并显式运行 `migrate up`。

## Sealos 内网 Migration

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
  AND TABLE_NAME IN ('schema_migrations', 'users', 'button_click_minutes')
ORDER BY TABLE_NAME;
```

必须根据检查结果继续，不能在未知状态下重复执行建表 SQL：

- 三张表均不存在：先建立空的 `schema_migrations`；执行 `migrations/000001_create_users.up.sql` 成功后登记 `version=1, dirty=0`；再执行 `migrations/000002_create_button_click_minutes.up.sql`，成功后将版本更新为 `version=2, dirty=0`。每次只有在对应 DDL 明确成功后才能登记版本。
- 只存在其中部分表，或 `schema_migrations` 的 `dirty` 为 `1`：先核对现有表结构，再补齐缺失部分；不要先修改版本号。
- 三张表均存在：核对 `SELECT version, dirty FROM schema_migrations;`；期望仅一行 `version=2`、`dirty=0`。

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

## 更新与停止

```bash
docker compose up -d --build
docker compose down
```

`docker compose down` 不会删除命名卷；不要附加 `--volumes`，除非明确需要永久清除 MySQL 数据、证书和 Caddy 状态。
