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

以下操作会永久删除本项目 MySQL 数据卷中的用户、绑定信件、点击记录和 Migration 状态，只用于明确允许丢弃数据的开发环境；不会删除 Caddy 数据卷：

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

执行 Migration（当前应到版本 3）：

```bash
make migrate-up
```

该命令使用官方 `migrate/migrate:v4.19.1` 工具容器依次执行 `migrations/*.up.sql`。回滚最近一版可运行 `make migrate-down`；完整回滚必须按 `000003`、`000002`、`000001` 的逆序执行。

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
DESCRIBE companion_bindings\G
DESCRIBE companion_active_memberships\G
SHOW INDEX FROM users\G
SHOW INDEX FROM button_click_minutes\G
SHOW INDEX FROM companion_bindings\G
SHOW INDEX FROM companion_active_memberships\G
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

bcrypt 是单向、自带随机盐且成本可调的密码哈希，不是可解的加密。相同密码每次生成的哈希通常不同，但哈希中包含算法、成本和盐，`CompareHashAndPassword` 仍可完成验证。

## 创建二维码登录映射

为“终身卡”二维码建立文本到用户的映射，文本可直接放在命令行参数或从标准输入读取：

```bash
make create-qr-login USERNAME=yanlili QR_TEXT_FLAGS="--qr-text 二维码中的永久文本"
# 长文本建议从标准输入读取，避免命令历史残留
make create-qr-login USERNAME=yanlili QR_TEXT_FLAGS="--qr-text-stdin" <<< '二维码中的完整永久文本'
```

文本必须与卡片二维码内容逐字一致，限制为 1 到 512 个字符；重复文本会返回“二维码文本已存在”。查询当前映射：

```sql
SELECT m.id, m.qr_text, u.username FROM qr_login_mappings m
JOIN users u ON u.id = m.user_id\G
```

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

前台运行的 `make dev` 或 `./scripts/dev.sh` 使用 `Ctrl+C` 停止。若此前通过 Docker Compose 启动了本项目容器，使用以下命令停止全部项目容器；该操作保留容器和数据库卷，之后可用 `docker compose start` 恢复：

```bash
docker compose stop
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

绑定相关接口均需要同一认证 Cookie：

```bash
curl -s -b /tmp/fluffy-cookie.txt http://localhost:4819/api/companion/state
curl -s -b /tmp/fluffy-cookie.txt http://localhost:4819/api/companion/inbox
curl -s -b /tmp/fluffy-cookie.txt \
  -H 'Content-Type: application/json' \
  -d '{"username":"对方用户名","note":"可空备注"}' \
  http://localhost:4819/api/companion/invitations
curl -s -b /tmp/fluffy-cookie.txt 'http://localhost:4819/api/yanlili/clicks/stats?direction=mine&utc_offset_minutes=480'
curl -s -b /tmp/fluffy-cookie.txt 'http://localhost:4819/api/yanlili/clicks/details?partner_id=2&page=1'
```

接受绑定、修改备注、统一解绑判定、接受解绑、取消申请和拒绝申请分别使用 `/api/companion/bindings/:id/accept`、`:id/note`、`:id/unbind-request`、`:id/unbind-accept`、`:id/unbind-cancel`、`:id/unbind-reject`。除备注使用 `PATCH` 外均为 `POST`；项目没有独立的直接解绑或删除信件接口。统一解绑请求体为 `{"confirm_inactive":false}`：对方近期登录时写入 `pending` 申请；超过 30 天未登录时返回 `inactive_confirmation_required`，前端额外确认后以 `{"confirm_inactive":true}` 重发，服务端重新判定并决定直接解绑或改发普通申请。只有发起方可取消、只有接收方可拒绝或接受；取消和拒绝保留反馈字段但不结束活跃关系。

不带 `-b` 请求受保护接口应返回 401。登录响应的 JWT 由 header（算法/类型）、payload（签名保护的身份和时间声明）、signature（防篡改）构成；payload 不是加密内容，所以仍必须用 HTTPS 防止 Token 在传输中被窃取。JWT 放 HttpOnly Cookie 而非 localStorage，可降低普通页面脚本读取 Token 的风险；`Secure` 限制 HTTPS 传输，`SameSite=Lax` 降低跨站请求携带 Cookie 的机会。

## 测试与 Linux 镜像构建

```bash
go test ./...
make build
```

`make build` 默认构建 `linux/amd64` 容器镜像 `fluffy-cupcake:V1.0.0_20260822`，不会生成适用于开发机的本地二进制文件。若部署服务器使用 ARM64：

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
- `companion_bindings` 是不可删除的生命周期信件；`companion_active_memberships` 是可释放的当前占位。后者以 `user_id` 为主键，使两个并发邀请也不能让同一用户获得两个活跃绑定。
- 每次物理点击插一行会快速产生大量重复时间；关系 ID、方向双方和 `minute_bucket` 组成唯一键，用一行 `click_count` 保留同方向一分钟的全部次数。
- 点击写入使用 `INSERT ... SELECT` 同时核对当前绑定占位和原子累加，避免先查绑定再解绑所产生的竞态。
- 当前摘要先用当前绑定确定对象，再按用户对跨历次绑定聚合并分别 `ORDER BY ... DESC LIMIT 8`；详细历史按对象双向索引查询并固定每页 20 条，浏览器不再接收完整历史后自行截断。
- 统一存 UTC 可避免服务器搬迁、夏令时或浏览器时区造成同一时刻归属不同桶；仅在页面显示分钟时转换为浏览器本地时间。
