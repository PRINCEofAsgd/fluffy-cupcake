# 仓库结构

## 文档索引

- `README.md`：项目简介、快速开始和完整文档入口。
- `docs/structures/repository-structure.md`：仓库目录、文件职责及版本记录位置。
- `docs/structures/call-chains.md`：运行入口与主要请求调用链。
- `docs/structures/database-structure.md`：数据库使用状态及后续结构记录约定。
- `docs/user/development.md`：开发、测试和构建方式。
- `docs/user/deployment.md`：容器和域名部署方法。
- `docs/user/user.md`：页面使用方法。
- `docs/user/release-notes.md`：当前发布版本和用户可见改动。
- `docs/record/project-context.md`：当前阶段工作记录。
- `docs/record/context-archives/`：仅在明确要求归档阶段时保存历史上下文。
- `scripts/README.md`：辅助脚本说明。

## 目录结构

```text
fluffy-cupcake/
├── cmd/server/                 # 程序入口与进程生命周期
├── internal/
│   ├── config/                 # 环境变量配置
│   ├── healthcheck/            # 无外部依赖的容器健康检查客户端
│   ├── handler/                # HTTP 请求处理器
│   ├── server/                 # Gin 路由和中间件组装
│   └── version/                # 当前应用版本常量
│
├── web/
│   ├── assets/                 # GIF、CSS 和 JavaScript 页面资源
│   ├── templates/              # Go HTML 模板
│   └── embed.go                # 资源嵌入入口
│
├── deploy/caddy/               # 域名 HTTPS 反向代理配置
├── docs/                       # 结构、用户及工作记录文档
├── scripts/                    # 开发与构建辅助脚本
├── Dockerfile                  # 多阶段应用镜像构建
├── compose.yaml                # Gin 与 Caddy 生产编排
├── Makefile                    # 常用开发命令入口
├── go.mod / go.sum             # Go 模块与依赖锁定
├── .dockerignore               # 镜像构建上下文排除规则
└── .gitignore                  # Git 排除规则
```

## 重要文件职责

- `cmd/server/main.go`：创建 HTTP Server，异步监听并处理 SIGINT/SIGTERM 优雅退出。
- `internal/config/config.go`：读取 `APP_ADDR`、`APP_MODE`，提供开发默认值。
- `internal/server/router.go`：初始化 Gin、中间件和全部公开路由。
- `internal/handler/page.go`：渲染页面、读取嵌入资源并返回健康状态。
- `web/templates/yanlili.html`：页面语义结构与可访问性标记。
- `web/assets/app.css`：响应式布局、按钮反馈、文字上浮渐隐动画。
- `web/assets/app.js`：点击、触摸和键盘激活后的动画元素管理。
- `deploy/caddy/Caddyfile`：`fluffy-cupcake.cn` HTTPS 和反向代理策略。

## 版本记录位置

需要同步更新版本时，检查以下位置：

- `internal/version/version.go`
- `README.md`
- `docs/user/release-notes.md`
- `Dockerfile` 的版本构建参数
- `compose.yaml` 的镜像标签和 `APP_VERSION` 构建参数
- `Makefile` 的镜像标签
- `scripts/build.sh` 的默认镜像版本
