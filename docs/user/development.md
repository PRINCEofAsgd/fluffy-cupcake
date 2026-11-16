# 开发指南

## 环境

- Go 1.26.3
- Node.js 22 LTS 与 npm（Homepage 构建）
- Docker 与 Docker Compose v2（MySQL 8.4 和 golang-migrate 工具容器）

安装依赖：

```bash
go mod download
npm --prefix web/homepage ci
```



## 数据库

### 启动本地数据库

`首次启动前需先执行 ### 初始化与配置`

启动数据库容器：

```bash
make db-up
docker compose ps mysql
```

停止数据库容器：

```bash
make db-stop
```

执行 Migration（当前应到版本 8）：

```bash
make migrate-up
```

查看数据库结构：

```bash
docker compose exec mysql mysql -u root -p
```

```sql
USE fluffy_cupcake;
SHOW TABLES;
DESCRIBE users\G
DESCRIBE button_click_minutes\G
DESCRIBE companion_bindings\G
DESCRIBE companion_active_memberships\G
DESCRIBE qr_login_mappings\G
DESCRIBE auth_sessions\G
DESCRIBE companion_moods\G
DESCRIBE auth_refresh_tokens\G
SHOW INDEX FROM users\G
SHOW INDEX FROM button_click_minutes\G
SHOW INDEX FROM companion_bindings\G
SHOW INDEX FROM companion_active_memberships\G
SHOW INDEX FROM qr_login_mappings\G
SHOW INDEX FROM auth_sessions\G
SHOW INDEX FROM companion_moods\G
SHOW INDEX FROM auth_refresh_tokens\G
```

### 初始化与配置

不需要预先手工创建数据库。首次启动空的 MySQL 数据卷时，官方镜像会根据 `.env` 的 `MYSQL_DATABASE`、`MYSQL_USER` 和 `MYSQL_PASSWORD` 自动创建数据库与应用用户；随后 Migration 创建业务表。

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
DATABASE_DSN='fluffy_cupcake:<APP_DB_PASSWORD>@tcp(127.0.0.1:3306)/fluffy_cupcake?timeout=5s&readTimeout=5s&writeTimeout=5s'
JWT_SECRET=<JWT_RANDOM_SECRET>
ACCESS_TOKEN_TTL=15m
SESSION_TTL=360h
```

尖括号表示占位说明，实际填写时不要保留。DSN 外层单引号必须保留，否则本地脚本读取 `tcp(...)` 时会触发 shell 语法错误。若 MySQL 数据卷已经初始化，事后只修改 `.env` 不会自动修改库内旧密码；应继续使用初始化时的值，或先在 MySQL 内主动改密。

```bash
# 编辑 .env；确认全部 replace_with 占位值都已替换
cp .env.example .env
chmod 600 .env
```



## 业务测试数据

### 创建固定用户

项目没有公开注册接口。CLI 默认在终端关闭密码回显，执行 bcrypt 后只把哈希交给 Repository：

```bash
make create-user USERNAME=yanlili
```

受控自动化可显式使用 `--password-stdin`；不要把密码直接放进命令参数。重复用户名会返回“用户名已存在”。验证只保存哈希：

```sql
SELECT id, username, password_hash, created_at FROM users\G
```

bcrypt 是单向、自带随机盐且成本可调的密码哈希，不是可解的加密。相同密码每次生成的哈希通常不同，但哈希中包含算法、成本和盐，`CompareHashAndPassword` 仍可完成验证。

### 创建二维码登录映射

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

### 启动完整开发服务

```bash
make dev
# 或
./scripts/dev.sh
```

`make dev` 会先构建 Homepage，再由 `scripts/dev.sh` 读取本地 `.env`，启动时创建一次 MySQL 连接池并 Ping。默认 `APP_ADDR=0.0.0.0:4819`，因此本机可通过 `localhost:4819` 访问，同一局域网设备也可通过开发机 IP 的 4819 端口访问。根路径 `/` 是 Homepage，`/yanlili` 保留原想念页面。

### 只启动 Homepage Vite 预览

```bash
make homepage-build
npm --prefix web/homepage run dev
```

Vite 开发服务器同样以默认路径 `/` 打开 Homepage；`/homepage/*` 只保留为构建后的脚本、样式、字体和图片资源命名空间，不是页面入口。

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
# 续期时同时读写 Cookie jar，保留轮换后的刷新凭证。
curl -i -b /tmp/fluffy-cookie.txt -c /tmp/fluffy-cookie.txt \
  -X POST http://localhost:4819/api/auth/refresh
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

不带 Cookie 请求受保护接口会在 Handler 执行前收到 `401 AUTH_REQUIRED`。密码和扫码登录下发访问与刷新 Cookie，默认有效期分别是 15 分钟和 15 天。访问 JWT 绑定 MySQL 会话，浏览器无法通过 JavaScript 读取任一凭证。服务端撤销会话后，新到达的请求立即鉴权失败，已通过鉴权的在途请求可完成。

本地观察续期可临时设置 `ACCESS_TOKEN_TTL=5s`，登录后等访问 Cookie 到期再加载页面，或用现有 Cookie jar 调用刷新接口。成功刷新返回 `204`，刷新 Cookie 的值改变且截止不变，`users.last_login_at` 不改变。已轮换凭证在 10 秒并发窗口内返回 `409 REFRESH_CONFLICT`；窗口外再次提交会撤销会话。若一次刷新已提交但响应丢失，浏览器先报告临时故障，之后仍无法取得胜出请求的 Cookie 时可能需要重新登录。

### 陪伴心情 API

`GET /api/yanlili/mood` 默认读取当前绑定对象，同一路径加 `?direction=mine` 读取本人。响应包含 `binding_id` 与下面示例中的五个心情字段。未填写时三组均返回 `[]`、自定义文字均返回空串。

`PUT /api/yanlili/mood` 使用 GET 返回的 `binding_id`，整份替换当前登录用户在该关系内的心情。三个数组都必须显式提供。清空时传入 `[]` 和空串。两条自定义文字各自可省略，服务端去掉首尾空白，按 Unicode 码点计数，最多 5 个且限单行。成功后返回归一化的快照。

```bash
curl -s -b /tmp/fluffy-cookie.txt 'http://localhost:4819/api/yanlili/mood?direction=mine'
# 把示例中的 123 换为读取时返回的 binding_id。
curl -s -b /tmp/fluffy-cookie.txt -X PUT -H 'Content-Type: application/json' \
  -d '{"binding_id":123,"statuses":["tired","hungry"],"reasons":["work"],"reason_custom":"加班有点累","comforts":["milk_tea"],"comfort_custom":""}' \
  http://localhost:4819/api/yanlili/mood
```

| 字段 | 固定标识与界面文案 |
| --- | --- |
| `statuses` | `happy` 🟢 好开心 ^-^、`relaxed` 🔵 轻松!、`tired` 🟡 累累嘟 >.<、`exhausted` 🟤 要鼠掉 ...、`angry` 🔴 滚、`disappointed` ⚫️ 失望、`normal` ⚪️ 一切正常 ~、`hungry` 🟠 好饿好饿 @-@ |
| `reasons` | `you` 你、`family` 家人、`friends` 朋友、`work` 上班、`health` 身体、`weather` 天气、`food` 食物、`bad_luck` 倒霉、`good_luck` 好运、`games` 游戏 |
| `comforts` | `company` 陪陪我、`talk` 想跟你说、`see_you` 想看看你、`milk_tea` 想喝奶茶、`tasty_food` 想好吃的、`go_out` 想出去玩 |

输入格式错误、未知或重复选项返回 `400`。读取时无活跃绑定返回 `409 MOOD_NOT_BOUND`，保存时绑定结束或 ID 已过期返回 `409 MOOD_BINDING_CHANGED`。收到后者时应重新读取当前绑定及快照。

### 心情图标资源

固定选项的文字和 SVG 共用上表的固定标识。展示时按组选文件，例如「上班」的 `work` 对应 `web/assets/mood/reasons/work.svg`，由 `/assets/mood/reasons/work.svg` 提供。「想喝奶茶」的 `milk_tea` 对应 `web/assets/mood/comforts/milk_tea.svg`。各组的 `custom.svg` 使用同一立方体图案。更换图标时保留文件名并重新构建，浏览器会重新验证图标缓存。数据库与 API 只保存选项标识和自定义文字。



## 测试与 Linux 镜像构建

```bash
go test ./...
make build
```

会话和心情集成测试使用本机独立的 `fluffy_session_test` 库。对该库运行完整 Migration 至版本 8，设置 `FLUFFY_TEST_MYSQL_DSN` 后运行以下命令：

```bash
go test -race -count=1 ./internal/server -run '^Test(AuthSession|CompanionMood)MySQL$' -v
```

`make build` 默认先用 Node 22 构建 Homepage，再构建 `linux/amd64` 容器镜像 `fluffy-cupcake:V1.1.0_20261116`，不会生成适用于开发机的本地二进制文件。若部署服务器使用 ARM64：

```bash
make build TARGET_PLATFORM=linux/arm64
```

本地开发仍通过 `make dev` 直接运行源码。页面资源已通过 `embed.FS` 编入 Linux 镜像中的服务二进制，无需单独复制 `web` 目录。



## 配置项

| 环境变量 | 开发默认值 | 说明 |
| --- | --- | --- |
| `APP_ADDR` | `0.0.0.0:4819` | HTTP 监听地址 |
| `APP_MODE` | `debug` | Gin 模式，可用 `debug`、`test`、`release` |
| `DATABASE_DSN` | 无，必填 | MySQL DSN；连接层强制 `parseTime`/UTC session，并在未显式设置时补 5s 的连接/读/写超时 |
| `DB_MAX_OPEN_CONNS` | `10` | 最大打开连接数 |
| `DB_MAX_IDLE_CONNS` | `5` | 最大空闲连接数，不得大于打开数 |
| `DB_CONN_MAX_LIFETIME` | `30m` | 单连接最长复用时间 |
| `JWT_SECRET` | 无，必填 | HS256 签名密钥，至少 32 字符 |
| `ACCESS_TOKEN_TTL` | `15m` | 访问 JWT 时长，上限为所属会话剩余时间。Go duration，至少 1 秒且为整秒 |
| `SESSION_TTL` | `360h` | 从首次签发计算的固定会话时长。Go duration，至少 1 秒且为整秒 |

`APP_MODE=release` 时 Cookie 自动启用 `Secure`；debug/test 保持 HttpOnly 和 SameSite，但允许本地 HTTP 登录。

本地宿主机 DSN 使用 `127.0.0.1:${MYSQL_PORT}`；Compose 内的应用使用 `mysql:3306`。Sealos 生产 DSN 必须来自平台 Secret，并使用 Guangzhou Managed MySQL External/Public Endpoint；完整切换流程见 [数据库跨区域迁移 Runbook](database-region-migration.md)。



## 关键设计理由

- Middleware 集中验证 Cookie/JWT，避免每个 Handler 复制并逐渐分叉鉴权代码；业务 Handler 的 `user_id` 必须来自签名 Token 写入的 Context，不能信任请求 JSON。
- `companion_bindings` 是不可删除的生命周期信件；`companion_active_memberships` 是可释放的当前占位。后者以 `user_id` 为主键，使两个并发邀请也不能让同一用户获得两个活跃绑定。
- 每次物理点击插一行会快速产生大量重复时间；关系 ID、方向双方和 `minute_bucket` 组成唯一键，用一行 `click_count` 保留同方向一分钟的全部次数。
- 点击写入使用 `INSERT ... SELECT` 同时核对当前绑定占位和原子累加，避免先查绑定再解绑所产生的竞态。
- 当前摘要先用当前绑定确定对象，再按用户对跨历次绑定聚合并分别 `ORDER BY ... DESC LIMIT 8`；详细历史按对象双向索引查询并固定每页 20 条，浏览器不再接收完整历史后自行截断。
- 统一存 UTC 可避免服务器搬迁、夏令时或浏览器时区造成同一时刻归属不同桶；仅在页面显示分钟时转换为浏览器本地时间。
