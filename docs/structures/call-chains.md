# 调用链

## 服务启动链

1. `cmd/server/main.go:main` 调用 `config.Load/ValidateServer` 读取并校验应用、MySQL 和 JWT 配置。
2. `database.Open` 创建全局连接池、强制 UTC 会话并执行 `PingContext`。
3. `main` 调用 `server.NewRouter(cfg, db)`，组装 Repository、Service、Handler 和 Middleware。
4. `server.NewRouter` 注册日志、恢复、安全响应头中间件及公开/受保护路由。
5. `handler.NewPageHandler` 从 `web.Content` 解析模板并取得静态资源子文件系统。
6. `http.Server.ListenAndServe` 默认监听 `0.0.0.0:4819`。
7. 进程收到 SIGINT 或 SIGTERM 后，先关闭 HTTP 连接，再关闭 MySQL 连接池。

## 登录链路

1. 浏览器 `POST /api/auth/login` 提交 username/password。
2. `AuthHandler.Login` 只解析 JSON，不记录密码，调用 `AuthService.Login`。
3. `AuthService` 调用 `UserRepository.GetByUsername`，Repository 以参数化 SQL 查询 `users`。
4. Service 用 `bcrypt.CompareHashAndPassword` 比较密码；用户不存在和密码错误统一返回 401。
5. 成功后 Service 签发含 `user_id`、`username`、`iat`、`exp` 的 HS256 JWT。
6. `UserRepository.UpdateLastLogin` 记录服务端 UTC 登录时间，作为 30 天未登录直接解绑的可信依据。
7. Handler 把 JWT 写入 `HttpOnly`、`SameSite=Lax`、`Path=/` Cookie；release 模式同时设置 `Secure`。

## 陪伴绑定与解绑链路

1. 未登录用户点击“陪伴绑定”时，页面提示“该功能需登录后使用”并展开登录区；所有绑定 API 都由 Auth Middleware 保护。
2. 登录用户提交目标用户名和可空备注，`CompanionService.Invite` 禁止绑定自己并限制备注为 64 个字符。
3. `CompanionRepository.CreateInvitation` 锁定双方用户、确认双方都无活跃绑定，再创建 `pending` 信件；信件没有删除接口。
4. 收信人在同一信件上接受后，Repository 在事务中向 `companion_active_memberships` 为双方各写一行；`PRIMARY KEY(user_id)` 从数据库层保证每个用户最多一个活跃绑定，同时把冲突的其他待处理信件标记为 `superseded`。
5. 绑定任一方可修改自己的私有备注；页面点击宾语优先显示备注，否则显示对象用户名。
6. 页面只显示一个“解绑”入口，连续三次 `confirm` 后调用统一 `unbind-request`；Repository 锁定当前信件和对方用户，并读取 `last_login_at`（空值退回 `created_at`）。
7. 对方 30 天内登录过时，同一事务写入申请字段并把 `unbind_status` 设为 `pending`；发起方可调用 `unbind-cancel`，另一方可调用 `unbind-reject` 或 `unbind-accept`。取消/拒绝只原子结束本次申请，关系保持 `active`，双方收件箱显示同一处理结果，并可之后在原信件重新申请。
8. 对方已连续 30 天未登录时先返回 `inactive_confirmation_required`，页面第四次确认后重发同一接口，服务端重新判定并结束为 `ended/inactive`；此时子状态为 `direct`。接受普通申请结束为 `ended/mutual` 和 `accepted`。信件和点击历史始终保留。

## JWT 鉴权链路

1. 浏览器后续同源请求自动携带 Cookie，JavaScript 无需也不能读取 HttpOnly Token。
2. `middleware.RequireAuth` 校验 HS256 签名、issuer、exp 和必要身份声明。
3. Middleware 将可信 `user_id/username` 写入 Gin Context；缺失、非法或过期 Token 统一返回 401。
4. `AuthHandler.Me` 和按钮 Handler 只从 Context 取 `user_id`，不接受客户端伪造的用户 ID。

## 按钮点击写入链路

1. `app.js:createFloatingMessage` 始终先创建文字动画；音频随后异步启动，新点击会中断旧音轨。
2. 未登录或未绑定时链路到此结束，反馈固定为“按按钮，想你+1”，不增加 `pendingClicks`。
3. 双向绑定后反馈宾语为绑定对象用户名或当前用户自己的备注，并增加 `pendingClicks`；750ms 后 `flushClicks` 把最多 100 次合并为一次 `POST /api/yanlili/clicks`。
4. Auth Middleware 校验 Cookie，`ButtonClickHandler` 校验 `count` 范围。
5. `ButtonClickService` 读取当前绑定 ID 和对象 ID，用服务端 `time.Now().UTC().Truncate(time.Minute)` 生成分钟桶。
6. `ButtonClickRepository.AddClicks` 通过 `INSERT ... SELECT` 再次核对当前绑定占位并执行原子 Upsert，避免解绑竞态把点击写进已结束关系。

## 点击统计读取链路

1. 页面以 `direction=mine|theirs` 和浏览器 `utc_offset_minutes` 请求受保护的 `GET /api/yanlili/clicks/stats`，⇄ 图标在“我想ta”与“ta想我”之间切换。
2. Service 根据当前绑定把方向解析为发起用户和目标用户；Repository 按用户对合并迁移旧行及历次绑定，查询方向总数，并分别倒序 `LIMIT 8` 返回“每日想念”和“最近想念”。每日分组在 UTC 分钟上叠加浏览器偏移，和明细本地时间使用同一日界线。
3. “详细记录”先通过 `GET /api/companion/partners` 取得历次已接受绑定的对象，再请求 `GET /api/yanlili/clicks/details?partner_id=&page=`。
4. 详细查询按用户对读取两个方向，把不同绑定实例和只有对象 ID 的迁移旧数据按分钟合并后 `UNION ALL`，固定 `LIMIT 20 OFFSET ...` 返回分页结果；响应不返回绑定实例 ID。
5. Handler 返回 RFC3339 时间 JSON；浏览器只在显示分钟时转换为本地时间。

## `/yanlili` 页面调用链

1. 浏览器请求 `GET /yanlili`。
2. Gin 依次执行请求日志、异常恢复和安全响应头中间件。
3. `PageHandler.Yanlili` 用当前版本渲染 `web/templates/yanlili.html`。
4. 浏览器继续请求 `/assets/miss-button.gif`、`/assets/app.css` 和 `/assets/app.js`，页面不再下载 MP3。
5. `PageHandler.Asset` 从嵌入式文件系统读取相应资源并附带缓存策略返回。
6. `app.js:createFloatingMessage` 创建提示元素；`app.css:float-away` 立即让文字上浮渐隐。
7. 文字进入 DOM 后，`app.js:playClickSound` 通过共享 `AudioContext` 为每次点击创建约 90ms 的独立合成声部；连续声部可重叠，共享限幅器避免叠加爆音。
8. 主按钮始终可用；登录和当前绑定共同决定是否持久化点击，收件箱、绑定入口、备注、当前方向统计与详细历史各自独立显示。

## 健康检查调用链

1. Docker 在容器内调用 `/app/fluffy-cupcake healthcheck`，无需 shell 或额外命令。
2. 程序入口识别 `healthcheck` 子命令，调用 `healthcheck.Run` 请求本机 `GET /healthz`。
3. `handler.Health` 返回 HTTP 200 和 `{"status":"ok"}`，健康检查进程以状态码 0 退出。
4. 生产编排在应用健康后启动 Caddy，Caddy 将域名流量反向代理至 `app:4819`。
