# fluffy-cupcake

fluffy-cupcake 是一个基于 Gin 与 MySQL 的私人轻量 Web 服务。任何访客都可在 `/yanlili` 使用 GIF 按钮和即时音效/动画；固定用户可通过收件箱完成唯一的双向“陪伴绑定”，把想念按方向和 UTC 分钟保存，并查询当前或历次绑定对象的记录。

## 当前版本

`V0.0.9_20260816`

## 快速开始

环境要求：Go 1.26.3、Docker 与 Docker Compose v2。

```bash
cp .env.example .env
# 编辑 .env，替换数据库密码和至少 32 字符的 JWT_SECRET
make db-up
make migrate-up
make create-user USERNAME=你的用户名
make dev
```

浏览器访问 `http://localhost:4819/yanlili`，健康检查地址为 `http://localhost:4819/healthz`。

## 构建与部署

```bash
make test
make build
```

`make build` 仅在 Docker 中生成 `linux/amd64` 镜像 `fluffy-cupcake:V0.0.9_20260816`，不会在仓库中生成本机二进制。ARM64 Linux 服务器可使用 `make build TARGET_PLATFORM=linux/arm64`。

生产环境使用 `compose.yaml` 启动 MySQL、Gin 与 Caddy，并通过独立工具容器执行 Migration。Caddy 通过 `fluffy-cupcake.cn` 对外提供自动 HTTPS，具体准备步骤见 [部署指南](docs/user/deployment.md)。

## 文档索引

- [项目结构](docs/structures/repository-structure.md)
- [调用链](docs/structures/call-chains.md)
- [数据库结构](docs/structures/database-structure.md)
- [开发指南](docs/user/development.md)
- [部署指南](docs/user/deployment.md)
- [操作手册](docs/user/user.md)
- [版本说明](docs/user/release-notes.md)
- [工作上下文](docs/record/project-context.md)
- [脚本说明](scripts/README.md)
