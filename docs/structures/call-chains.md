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
6. Handler 把 JWT 写入 `HttpOnly`、`SameSite=Lax`、`Path=/` Cookie；release 模式同时设置 `Secure`。

## JWT 鉴权链路

1. 浏览器后续同源请求自动携带 Cookie，JavaScript 无需也不能读取 HttpOnly Token。
2. `middleware.RequireAuth` 校验 HS256 签名、issuer、exp 和必要身份声明。
3. Middleware 将可信 `user_id/username` 写入 Gin Context；缺失、非法或过期 Token 统一返回 401。
4. `AuthHandler.Me` 和按钮 Handler 只从 Context 取 `user_id`，不接受客户端伪造的用户 ID。

## 按钮点击写入链路

1. `app.js:createFloatingMessage` 始终先创建文字动画；音频随后异步启动，新点击会中断旧音轨。
2. 未登录时链路到此结束，不增加 `pendingClicks`，因此之后登录也不会补传匿名点击。
3. 已登录时才增加 `pendingClicks`；750ms 后 `flushClicks` 把最多 100 次合并为一次 `POST /api/yanlili/clicks`，失败会恢复 pending 并重试。
4. Auth Middleware 校验 Cookie，`ButtonClickHandler` 校验 `count` 范围。
5. `ButtonClickService` 用服务端 `time.Now().UTC().Truncate(time.Minute)` 生成分钟桶。
6. `ButtonClickRepository.AddClicks` 执行单条 MySQL Upsert；联合唯一索引命中时原子执行 `click_count = click_count + incoming.click_count`。

## 点击统计读取链路

1. 页面请求受保护的 `GET /api/yanlili/clicks/stats`。
2. `ButtonClickService.Stats` 调用 Repository 的总数、每日和分钟查询。
3. Repository 按 `button_key` 对所有用户执行 `SUM`，并分别按 UTC 日期、`minute_bucket` 做 `GROUP BY`。
4. Handler 返回 RFC3339 时间 JSON；浏览器将分钟时间转换为本地显示，UTC 每日语义保持明确。

## `/yanlili` 页面调用链

1. 浏览器请求 `GET /yanlili`。
2. Gin 依次执行请求日志、异常恢复和安全响应头中间件。
3. `PageHandler.Yanlili` 用当前版本渲染 `web/templates/yanlili.html`。
4. 浏览器继续请求 `/assets/miss-button.gif`、`/assets/app.css` 和 `/assets/app.js`，页面不再下载 MP3。
5. `PageHandler.Asset` 从嵌入式文件系统读取相应资源并附带缓存策略返回。
6. `app.js:createFloatingMessage` 创建提示元素；`app.css:float-away` 立即让文字上浮渐隐。
7. 文字进入 DOM 后，`app.js:playClickSound` 通过共享 `AudioContext` 为每次点击创建约 90ms 的独立合成声部；连续声部可重叠，共享限幅器避免叠加爆音。
8. 主按钮始终可用；独立登录入口只控制是否持久化点击和是否展示统计区域。

## 健康检查调用链

1. Docker 在容器内调用 `/app/fluffy-cupcake healthcheck`，无需 shell 或额外命令。
2. 程序入口识别 `healthcheck` 子命令，调用 `healthcheck.Run` 请求本机 `GET /healthz`。
3. `handler.Health` 返回 HTTP 200 和 `{"status":"ok"}`，健康检查进程以状态码 0 退出。
4. 生产编排在应用健康后启动 Caddy，Caddy 将域名流量反向代理至 `app:4819`。
