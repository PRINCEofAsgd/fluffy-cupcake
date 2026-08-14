# 开发指南

## 环境要求

- Go 1.26.3
- Docker 与 Docker Compose v2（MySQL 8.4 和 golang-migrate 工具容器）

## 安装依赖

```bash
go mod download
```

Go 模块缓存由 Go 工具链管理；若需清理全部模块缓存，可运行 `go clean -modcache`，下次构建会重新下载。

## 本地数据库与配置

不需要预先手工创建数据库。首次启动空的 MySQL 数据卷时，官方镜像会根据 `.env` 的 `MYSQL_DATABASE`、`MYSQL_USER` 和 `MYSQL_PASSWORD` 自动创建数据库与应用用户；随后 Migration 创建业务表。
### 使用现有 `.env` 重置测试数据库

以下操作会永久删除本项目 MySQL 数据卷中的用户、点击记录和 Migration 状态，只用于明确允许丢弃数据的开发环境；不会删除 Caddy 数据卷：

```bash
docker compose stop mysql
docker compose rm -f mysql
docker volume rm fluffy-cupcake_mysql_data
make db-up
make migrate-up
```

重置后 `users` 表为空，需要重新执行 `make create-user USERNAME=用户名` 创建登录用户。MySQL 初始化变量只对新建空数据卷生效，因此必须先删除旧的 `fluffy-cupcake_mysql_data`，再按当前 `.env` 启动。

复制模板并把示例密码、JWT Secret 换成随机值。分别运行以下命令，每次使用新输出，不要让三个值相同：

```bash
openssl rand -hex 24 # MYSQL_PASSWORD
openssl rand -hex 24 # MYSQL_ROOT_PASSWORD
openssl rand -hex 32 # JWT_SECRET
```

`MYSQL_PASSWORD` 是应用用户密码，必须原样复用在宿主机版 `DATABASE_DSN`；`MYSQL_ROOT_PASSWORD` 只属于 MySQL 管理员，不写入 DSN。以上命令只生成十六进制字符，可避免 DSN 转义问题。`.env` 不会提交到 Git 或进入镜像构建上下文。例如三个命令分别输出 `<APP_DB_PASSWORD>`、`<ROOT_DB_PASSWORD>`、`<JWT_RANDOM_SECRET>` 时，填写为：

```dotenv
MYSQL_DATABASE=fluffy_cupcake
MYSQL_USER=fluffy_cupcake
MYSQL_PASSWORD=<APP_DB_PASSWORD>
MYSQL_ROOT_PASSWORD=<ROOT_DB_PASSWORD>
MYSQL_PORT=3306
DATABASE_DSN='fluffy_cupcake:<APP_DB_PASSWORD>@tcp(127.0.0.1:3306)/fluffy_cupcake'
JWT_SECRET=<JWT_RANDOM_SECRET>
JWT_EXPIRE=24h
```

尖括号表示占位说明，实际填写时不要保留。DSN 外层单引号必须保留，否则本地脚本读取 `tcp(...)` 时会触发 shell 语法错误。若 MySQL 数据卷已经初始化，事后只修改 `.env` 不会自动修改库内旧密码；应继续使用初始化时的值，或先在 MySQL 内主动改密。

```bash
cp .env.example .env
chmod 600 .env
# 编辑 .env；确认全部 replace_with 占位值都已替换
make db-up
docker compose ps mysql
```

执行 Migration：

```bash
make migrate-up
```

该命令使用官方 `migrate/migrate:v4.19.1` 工具容器依次执行 `migrations/*.up.sql`。回滚最近一版可运行 `make migrate-down`；因为点击表引用用户表，完整回滚必须先回滚 `000002` 再回滚 `000001`。

查看实际结构：

```bash
docker compose exec mysql mysql -u root -p
```

进入 MySQL 后执行：

```sql
USE fluffy_cupcake;
SHOW TABLES;
DESCRIBE users\G
DESCRIBE button_click_minutes\G
SHOW INDEX FROM users\G
SHOW INDEX FROM button_click_minutes\G
```

## 创建固定用户

项目没有公开注册接口。CLI 默认在终端关闭密码回显，执行 bcrypt 后只把哈希交给 Repository：

```bash
make create-user USERNAME=yanlili
```

受控自动化可显式使用 `--password-stdin`；不要把密码直接放进命令参数。重复用户名会返回“用户名已存在”。验证只保存哈希：

```sql
SELECT id, username, password_hash, created_at FROM users\G
```

bcrypt 是单向、自带随机盐且成本可调的密码哈希，不是可解密的加密。相同密码每次生成的哈希通常不同，但哈希中包含算法、成本和盐，`CompareHashAndPassword` 仍可完成验证。

## 本地运行

```bash
make dev
# 或
./scripts/dev.sh
```

`scripts/dev.sh` 会读取本地 `.env`，启动时创建一次 MySQL 连接池并 Ping。默认 `APP_ADDR=0.0.0.0:4819`，因此本机可通过 `localhost:4819` 访问，同一局域网设备也可通过开发机 IP 的 4819 端口访问。

```bash
APP_ADDR=127.0.0.1:9090 make dev
```

## 鉴权与点击接口验证

```bash
curl -i -c /tmp/fluffy-cookie.txt \
  -H 'Content-Type: application/json' \
  -d '{"username":"yanlili","password":"你的密码"}' \
  http://localhost:4819/api/auth/login
curl -i -b /tmp/fluffy-cookie.txt http://localhost:4819/api/auth/me
curl -i -b /tmp/fluffy-cookie.txt \
  -H 'Content-Type: application/json' \
  -d '{"count":6}' \
  http://localhost:4819/api/yanlili/clicks
curl -s -b /tmp/fluffy-cookie.txt http://localhost:4819/api/yanlili/clicks/stats
```

不带 `-b` 请求受保护接口应返回 401。登录响应的 JWT 由 header（算法/类型）、payload（签名保护的身份和时间声明）、signature（防篡改）构成；payload 不是加密内容，所以仍必须用 HTTPS 防止 Token 在传输中被窃取。JWT 放 HttpOnly Cookie 而非 localStorage，可降低普通页面脚本读取 Token 的风险；`Secure` 限制 HTTPS 传输，`SameSite=Lax` 降低跨站请求携带 Cookie 的机会。

## 测试与 Linux 镜像构建

```bash
go test ./...
make build
```

`make build` 默认构建 `linux/amd64` 容器镜像 `fluffy-cupcake:V0.0.6_20260815`，不会生成适用于开发机的本地二进制文件。若部署服务器使用 ARM64：

```bash
make build TARGET_PLATFORM=linux/arm64
```

本地开发仍通过 `make dev` 直接运行源码。页面资源已通过 `embed.FS` 编入 Linux 镜像中的服务二进制，无需单独复制 `web` 目录。

## 配置项

| 环境变量 | 开发默认值 | 说明 |
| --- | --- | --- |
| `APP_ADDR` | `0.0.0.0:4819` | HTTP 监听地址 |
| `APP_MODE` | `debug` | Gin 模式，可用 `debug`、`test`、`release` |
| `DATABASE_DSN` | 无，必填 | MySQL DSN；连接层自动强制 `parseTime` 和 UTC session |
| `DB_MAX_OPEN_CONNS` | `10` | 最大打开连接数 |
| `DB_MAX_IDLE_CONNS` | `5` | 最大空闲连接数，不得大于打开数 |
| `DB_CONN_MAX_LIFETIME` | `30m` | 单连接最长复用时间 |
| `JWT_SECRET` | 无，必填 | HS256 签名密钥，至少 32 字符 |
| `JWT_EXPIRE` | `24h` | 短期 JWT 有效期，Go duration 格式 |

`APP_MODE=release` 时 Cookie 自动启用 `Secure`；debug/test 保持 HttpOnly 和 SameSite，但允许本地 HTTP 登录。

## 关键设计理由

- Middleware 集中验证 Cookie/JWT，避免每个 Handler 复制并逐渐分叉鉴权代码；业务 Handler 的 `user_id` 必须来自签名 Token 写入的 Context，不能信任请求 JSON。
- 每次物理点击插一行会快速产生大量重复时间；`minute_bucket` 是把服务端 UTC 时间向下截断到分钟，用一行的 `click_count` 保留全部次数。
- `UNIQUE(user_id, button_key, minute_bucket)` 既表达“一用户一按钮一分钟一行”，也让单条 Upsert 在并发下锁定同一唯一键并执行数据库侧 `click_count + delta`，避免 `SELECT → 判断 → UPDATE` 的竞争窗口。
- 总数直接 `SUM(click_count)`；每日和分钟统计分别 `GROUP BY` UTC 日期与分钟桶。不同用户同一分钟的行在查询时合并，所以时间只显示一次。
- 统一存 UTC 可避免服务器搬迁、夏令时或浏览器时区造成同一时刻归属不同桶；仅在页面显示分钟时转换为浏览器本地时间。
