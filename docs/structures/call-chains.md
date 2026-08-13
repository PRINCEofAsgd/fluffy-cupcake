# 调用链

## 服务启动链

1. `cmd/server/main.go:main` 调用 `config.Load` 读取监听地址和 Gin 模式。
2. `main` 调用 `server.NewRouter` 创建 Gin 引擎。
3. `server.NewRouter` 注册日志、恢复、安全响应头中间件及业务路由。
4. `handler.NewPageHandler` 从 `web.Content` 解析模板并取得静态资源子文件系统。
5. `http.Server.ListenAndServe` 默认监听 `0.0.0.0:4819`。
6. 进程收到 SIGINT 或 SIGTERM 后，`http.Server.Shutdown` 在 10 秒超时内优雅关闭连接。

## `/yanlili` 页面调用链

1. 浏览器请求 `GET /yanlili`。
2. Gin 依次执行请求日志、异常恢复和安全响应头中间件。
3. `PageHandler.Yanlili` 用当前版本渲染 `web/templates/yanlili.html`。
4. 浏览器继续请求 `/assets/miss-button.gif`、`/assets/app.css` 和 `/assets/app.js`。
5. `PageHandler.Asset` 从嵌入式文件系统读取相应资源并附带缓存策略返回。
6. 用户点击按钮后，`app.js:createFloatingMessage` 创建提示元素；`app.css:float-away` 让文字向上漂浮并渐隐，动画结束后元素被移除。

## 健康检查调用链

1. Docker 在容器内调用 `/app/fluffy-cupcake healthcheck`，无需 shell 或额外命令。
2. 程序入口识别 `healthcheck` 子命令，调用 `healthcheck.Run` 请求本机 `GET /healthz`。
3. `handler.Health` 返回 HTTP 200 和 `{"status":"ok"}`，健康检查进程以状态码 0 退出。
4. 生产编排在应用健康后启动 Caddy，Caddy 将域名流量反向代理至 `app:4819`。
