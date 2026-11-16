# 调用链

## 服务启动链

1. `cmd/server/main.go:main` 调用 `config.Load/ValidateServer` 读取并校验应用、MySQL 和 JWT 配置。
2. `database.Open` 解析 `DATABASE_DSN`，保留显式参数、为缺省连接/读/写超时各补 5 秒，创建全局连接池、强制 UTC 会话并执行 `PingContext`。
3. `main` 调用 `server.NewRouter(cfg, db)`，组装 Repository、Service、Handler 和 Middleware。
4. `server.NewRouter` 注册日志、恢复、安全响应头中间件及公开/受保护路由。
5. `handler.NewPageHandler` 从 `web.Content` 解析想念模板，并取得 Homepage 入口、全站图标及两类静态资源子文件系统。
6. `http.Server.ListenAndServe` 默认监听 `0.0.0.0:4819`。
7. 进程收到 SIGINT 或 SIGTERM 后，先关闭 HTTP 连接，再关闭 MySQL 连接池。

## 登录链路

1. 浏览器 `POST /api/auth/login` 提交 username/password。
2. `AuthHandler.Login` 只解析 JSON，不记录密码，调用 `AuthService.Login`。
3. `AuthService` 调用 `UserRepository.GetByUsername`，Repository 以参数化 SQL 查询 `users`。
4. Service 用 `bcrypt.CompareHashAndPassword` 比较密码；用户不存在和密码错误统一返回 401。
5. `AuthService.startSession` 生成随机会话 ID 和刷新凭证，签发包含 `sid`、身份、`iat`、`exp` 的 HS256 访问 JWT。首次签发确定会话截止（默认 15 天），访问 JWT 默认 15 分钟且受会话截止限制。
6. `AuthSessionRepository.Create` 在一个事务内更新 `users.last_login_at`、创建会话并写入首枚刷新凭证的 SHA-256 哈希。
7. `AuthHandler` 下发两个 `HttpOnly`、`SameSite=Lax` Cookie。访问 Cookie 的路径为 `/`，刷新 Cookie 的路径为 `/api/auth`，release 模式两者均为 `Secure`。

## 二维码扫码登录链路

1. 登录面板点击“拍照扫码”后，`app.js` 调用 `getUserMedia` 打开后置摄像头，等待视频产生有效尺寸，再用 `requestVideoFrameCallback` 或 rAF 持续取帧；关闭面板会停止轨道并使迟到回调失效。
2. 每次识别先处理中心取景区域，再处理受控尺寸的完整画面，并同时尝试普通/反色二维码；相册文件优先由 `createImageBitmap` 读取并应用照片方向，旧浏览器回退到可释放的 `blob:` 图片。
3. 识别成功后浏览器 `POST /api/auth/qr-login` 提交文本；`AuthHandler.QrLogin` 只解析 JSON，不记录文本，调用 `AuthService.LoginByQrText`。
4. Service 先 `TrimSpace`，再调用 `QrLoginRepository.GetUserByQrText` 以参数化 SQL 联查 `qr_login_mappings` 与 `users`；未映射文本统一返回 401。
5. Service 复用密码登录的 `startSession` 和事务，Handler 下发相同的两个 Cookie。
6. 浏览器请求 `/api/auth/me` 初始化登录状态，再加载绑定与统计。同一流程也用于页面启动时恢复已有登录。

## 陪伴绑定与解绑链路

1. 未登录用户点击“陪伴绑定”时，页面提示“该功能需登录后使用”并展开登录区；所有绑定 API 都由 Auth Middleware 保护。
2. 登录用户提交目标用户名和可空备注，`CompanionService.Invite` 禁止绑定自己并限制备注为 64 个字符。
3. `CompanionRepository.CreateInvitation` 锁定双方用户、确认双方都无活跃绑定，再创建 `pending` 信件；信件没有删除接口。
4. 收信人在同一信件上接受后，Repository 在事务中向 `companion_active_memberships` 为双方各写一行；`PRIMARY KEY(user_id)` 从数据库层保证每个用户最多一个活跃绑定，同时把冲突的其他待处理信件标记为 `superseded`。
5. 绑定任一方可修改自己的私有备注；页面点击宾语优先显示备注，否则显示对象用户名。
6. 页面只显示一个“解绑”入口，连续三次 `confirm` 后调用统一 `unbind-request`；Repository 锁定当前信件和对方用户，并读取 `last_login_at`（空值退回 `created_at`）。
7. 对方 30 天内登录过时，同一事务写入申请字段并把 `unbind_status` 设为 `pending`；发起方可调用 `unbind-cancel`，另一方可调用 `unbind-reject` 或 `unbind-accept`。取消/拒绝只原子结束本次申请，关系保持 `active`，双方收件箱显示同一处理结果，并可之后在原信件重新申请。
8. 对方已连续 30 天未登录时先返回 `inactive_confirmation_required`，页面第四次确认后重发同一接口，服务端重新判定并结束为 `ended/inactive`；此时子状态为 `direct`。接受普通申请结束为 `ended/mutual` 和 `accepted`。信件和点击历史始终保留。

## 访问鉴权、续期与退出

1. `middleware.RequireAuth` 读取访问 Cookie，`AuthService.Authenticate` 校验签名、签发者、有效期及 `sid`，再按主键检查 MySQL 会话的用户、截止及撤销状态。旧版 JWT 缺少 `sid`，会在签发格式检查时被拒绝。
2. 鉴权成功后中间件写入可信 `user_id/username` 并放行业务 Handler；认证拒绝返回 `401 AUTH_REQUIRED`，查询故障返回 `503 AUTH_UNAVAILABLE`。两个响应均在业务 Handler 执行前产生。
3. `app.js:requestJSON` 仅对 `401 AUTH_REQUIRED` 发起恢复。页面内合并并发请求，跨标签页优先用 Web Locks 串行化，并在锁内请求 `/api/auth/me` 确认是否已有新访问 Cookie，仍需续期时调用 `POST /api/auth/refresh`。
4. 刷新接口凭刷新 Cookie 找会话。`AuthSessionRepository.Rotate` 用会话行锁串行化刷新，原子标记旧哈希并写入新哈希。会话及刷新 Cookie 保留首次签发时的截止时间，访问 JWT 的 `exp` 也被限制在这个截止时间内；刷新不修改 `last_login_at`。
5. 旧凭证在轮换后 10 秒内重复出现返回 `409 REFRESH_CONFLICT`，响应该请求时不写 Cookie。此时浏览器短暂通过只读 `/api/auth/me` 等待其他标签页的新 Cookie。超过窗口再次使用旧凭证会撤销会话。刷新成功后，浏览器对最初被拦截的请求重放一次。普通业务错误及写入响应丢失不触发认证重放。
6. 刷新接口以 `401 SESSION_ENDED` 或 `401 NO_SESSION` 表示会话失效或缺少刷新 Cookie；页面提示需要登录。网络、数据库和短暂并发故障提供重试提示，保留当前界面登录状态。刷新只按请求触发。
7. `POST /api/auth/logout` 可使用当前或历史刷新凭证定位并撤销当前会话。刷新凭证缺失或无法匹配时，使用仍有效的访问 JWT 定位。服务端成功撤销后清除两个 Cookie。撤销提交后的新鉴权会拒绝该会话已签发的全部访问 JWT；已通过鉴权的在途业务请求可以完成。

## 按钮点击写入链路

1. `app.js:createFloatingMessage` 始终先创建文字动画；音频随后异步启动，新点击会中断旧音轨。
2. 未登录或未绑定时链路到此结束，反馈固定为“按按钮，想你+1”，不增加 `pendingClicks`。
3. 双向绑定后反馈宾语为绑定对象用户名或当前用户自己的备注，并增加 `pendingClicks`；750ms 后 `flushClicks` 把最多 100 次合并为一次 `POST /api/yanlili/clicks`。
4. 鉴权中间件校验访问 JWT 和会话；到期时先拒绝写入，浏览器恢复会话后可重试一次。`ButtonClickHandler` 放行后校验 `count` 范围。
5. `ButtonClickService` 读取当前绑定 ID 和对象 ID，用服务端 `time.Now().UTC().Truncate(time.Minute)` 生成分钟桶。
6. `ButtonClickRepository.AddClicks` 通过 `INSERT ... SELECT` 再次核对当前绑定占位并执行原子 Upsert，避免解绑竞态把点击写进已结束关系。

## 点击统计读取链路

1. 页面载入当前绑定或换绑后，以 `direction=theirs` 和浏览器 `utc_offset_minutes` 请求受保护的 `GET /api/yanlili/clicks/stats`，默认显示“ta想我”。⇄ 在 `theirs`（“ta想我”）和 `mine`（“我想ta”）之间切换，只有最新统计请求可以更新标题和统计。API 调用方省略 `direction` 时仍按 `mine` 查询。
2. Service 根据当前绑定把方向解析为发起用户和目标用户；Repository 按用户对合并迁移旧行及历次绑定，查询方向总数，并分别倒序 `LIMIT 8` 返回“每日想念”和“最近想念”。每日分组在 UTC 分钟上叠加浏览器偏移，和明细本地时间使用同一日界线。
3. “详细记录”先通过 `GET /api/companion/partners` 取得历次已接受绑定的对象，再请求 `GET /api/yanlili/clicks/details?partner_id=&page=`。
4. 详细查询按用户对读取两个方向，把不同绑定实例和只有对象 ID 的迁移旧数据按分钟合并后 `UNION ALL`，固定 `LIMIT 20 OFFSET ...` 返回分页结果；响应不返回绑定实例 ID。
5. Handler 返回 RFC3339 时间 JSON；浏览器只在显示分钟时转换为本地时间。

## Homepage 调用链

1. 浏览器请求默认路径 `GET /`，Gin 调用 `PageHandler.Home` 直接返回嵌入式 Vite `index.html`；根路径不重定向，旧 `/homepage/` 页面入口不再提供。
2. 浏览器从 `/favicon.png` 加载全站统一图标，从 `/homepage/assets/*` 加载带哈希的 React/CSS 产物，并从 `/homepage/images/*`、`/homepage/fonts/*` 读取公开素材；Gin 对哈希产物使用 immutable 长缓存。
3. 静态入口立即显示加载页及“页面参考 github.com/LeoninCS/Homepage 设计。”说明。
4. `useResourceGate` 并发加载整页使用的全部图片并等待 `document.fonts.ready`，每完成一个任务即更新进度；全部完成前锁定滚动并隐藏正文。
5. 资源全部解码后加载门渐隐，React 展示 Loki Yin 的头像、简介、身份、技术方向、两段工作经历、四个项目经历、四项爱好及公开联系方式；主机用原作者的单一背景层加载重新取景的抹茶与蛋糕宽幅照片，左侧导航用半透明暗色覆盖形成副屏，10 枚自托管技术栈 SVG 以错落层次漂浮。
6. 经历组件直接复用原作者 MiniMax 品牌控制台与 Sealos 流程控制台，并在桌面端保持左右对称；项目标签按 Fluffy Cupcake、Reactor Web、My Dock、Card Game 排列并直接显示项目名，网易云音乐入口直达公开艺人主页。
7. Homepage 不调用新增 API 或数据库；但它与 `/yanlili` 仍由同一 Gin 进程提供，因此继承应用启动时的 MySQL 可用性要求。

## 陪伴心情接口调用链

1. `GET /api/yanlili/mood` 和 `PUT /api/yanlili/mood` 共用 `RequireAuth`，请求身份来自已校验的访问 Cookie，响应禁止缓存。
2. `CompanionMoodHandler.Get` 将 `direction` 交给 `CompanionMoodService.Get`，默认为 `theirs`。`CompanionMoodRepository.GetCurrent` 用一次查询从活跃占位解析本人或对方，以绑定 ID 和填写者 ID 匹配快照。没有快照返回空数组和空串，无活跃绑定返回 `409 MOOD_NOT_BOUND`。
3. `CompanionMoodHandler.Put` 读取完整快照，Service 校验三组枚举及两条自定义文字。Repository 在事务中锁定客户端所见的绑定信件并检查登录用户的当前占位，再按绑定与登录用户覆盖存储。绑定结束或变更时返回 `409 MOOD_BINDING_CHANGED`，保存成功返回归一化后快照。
4. 页面确认活跃绑定后，`loadStats` 并行读取同一方向的心情和想念统计。默认使用 `direction=theirs`，方向按钮同时切换两组数据，只有最新一轮请求可以更新面板。遇到 `409` 或心情响应的绑定 ID 不符时重新读取一次绑定；解绑后隐藏统计面板，换绑后从对方视角读取。
5. `renderMood` 按固定选项标识渲染色灯与文字。原因及安慰选项用同名标识请求 `/assets/mood/:group/:name`，自定义文字使用本组的 `custom.svg`。`PageHandler.MoodIcon` 从 Go 嵌入资源返回 SVG。
6. 统计区的铅笔按钮打开编辑面板。`openMoodEditor` 独立读取本人快照并生成草稿，保留统计方向。`saveMood` 校验自定义文字后一次性 `PUT` 三组数据，当前是本人视角时同步更新展示。保存使用统计区的 `sync-status`，提交期间锁住表单，失败保留草稿，绑定冲突时关闭编辑并重查绑定。编辑面板底部可打开备注面板，关闭备注后回到原草稿。

## `/yanlili` 页面调用链

1. 浏览器请求 `GET /yanlili`。
2. Gin 依次执行请求日志、异常恢复和安全响应头中间件。
3. `PageHandler.Yanlili` 用当前版本渲染 `web/templates/yanlili.html`。
4. 浏览器继续请求全站图标 `/favicon.png`、主按钮 `/assets/miss-button.gif`、`/assets/app.css`、`/assets/qrdecoder.js` 和 `/assets/app.js`，页面不再下载 MP3；未读取页面声明的客户端可回退到相同内容的 `/favicon.ico`。
5. `PageHandler.SiteIcon` 返回统一 PNG，`PageHandler.Asset` 从嵌入式文件系统读取其余资源并附带缓存策略返回。
6. `app.js:createFloatingMessage` 创建提示元素；`app.css:float-away` 立即让文字上浮渐隐。
7. 文字进入 DOM 后，`app.js:playClickSound` 通过共享 `AudioContext` 为每次点击创建约 90ms 的独立合成声部；连续声部可重叠，共享限幅器避免叠加爆音。
8. 主按钮始终可用；登录和当前绑定共同决定是否持久化点击，收件箱、绑定入口、备注、当前方向统计与详细历史各自独立显示。

## 健康检查调用链

1. Docker 在容器内调用 `/app/fluffy-cupcake healthcheck`，无需 shell 或额外命令。
2. 程序入口识别 `healthcheck` 子命令，调用 `healthcheck.Run` 请求本机 `GET /healthz`。
3. `handler.Health` 返回 HTTP 200 和 `{"status":"ok"}`，健康检查进程以状态码 0 退出。
4. 生产编排在应用健康后启动 Caddy，Caddy 将域名流量反向代理至 `app:4819`。

## 跨区域数据库迁移验证链

1. 操作者以 `OLD_DB_*`/`NEW_DB_*` 提供非 Secret 连接信息；`scripts/db-migration/verify.sh` 在终端隐藏读取密码并创建权限为 0600 的临时 client 文件。
2. `preflight OLD|NEW` 以 UTC session 读取版本、当前库、全部表、Migration、精确行数和时间列边界，不执行 DDL/DML。
3. `verify` 分别生成确定排序的 OLD/NEW 快照，比较所有表/列、行数、PK/UNIQUE/普通索引、外键规则、`schema_migrations` 和 UTC 时间统计。
4. 任一预期表缺失、查询失败或快照不一致即非零退出；工具不修改 DSN、不恢复数据，也不删除数据库。
