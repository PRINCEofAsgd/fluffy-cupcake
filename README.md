# fluffy-cupcake

fluffy-cupcake 是一个基于 Gin 与 MySQL 的私人轻量 Web 服务。根路径提供个人 Homepage，任何访客都可从中进入 `/yanlili` 使用 GIF 按钮和即时音效/动画；固定用户可通过收件箱完成唯一的双向“陪伴绑定”，把想念按方向和 UTC 分钟保存，并查询当前或历次绑定对象的记录。除用户名密码登录外，还支持扫描“终身卡”二维码文本免密登录。

## 当前版本

`V1.0.7_20260830`

## 快速开始

环境要求：Go 1.26.3、Node.js 22 LTS、npm、Docker 与 Docker Compose v2。

```bash
cp .env.example .env
# 编辑 .env，替换数据库密码和至少 32 字符的 JWT_SECRET
make db-up
make migrate-up
make create-user USERNAME=你的用户名
make dev
```

浏览器访问 `http://localhost:4819/` 查看 Homepage，想念页面保留在 `http://localhost:4819/yanlili`，健康检查地址为 `http://localhost:4819/healthz`。二维码映射通过 `make create-qr-login` 或 `go run ./cmd/create-qr-login --username 用户名 --qr-text 文本` 建立。

## 构建与部署

```bash
make test
make build
```

`make build` 先在 Docker 中构建 Homepage，再生成 `linux/amd64` 镜像 `fluffy-cupcake:V1.0.7_20260830`，不会在仓库中生成本机二进制。ARM64 Linux 服务器可使用 `make build TARGET_PLATFORM=linux/arm64`。

`compose.yaml` 保留给本地/独立单机开发；Sealos 生产环境由 Singapore Gin 应用通过 `DATABASE_DSN` 连接 Guangzhou Managed MySQL 的 External/Public Endpoint，不依赖 Compose 的 `mysql` service name。部署准备见 [部署指南](docs/user/deployment.md)，跨区域搬迁按 [数据库迁移 Runbook](docs/user/database-region-migration.md) 执行。

## 文档索引

- [项目结构](docs/structures/repository-structure.md)
- [调用链](docs/structures/call-chains.md)
- [数据库结构](docs/structures/database-structure.md)
- [开发指南](docs/user/development.md)
- [部署指南](docs/user/deployment.md)
- [数据库跨区域迁移 Runbook](docs/user/database-region-migration.md)
- [操作手册](docs/user/user.md)
- [版本说明](docs/user/release-notes.md)
- [工作上下文](docs/record/project-context.md)
- [脚本说明](scripts/README.md)
