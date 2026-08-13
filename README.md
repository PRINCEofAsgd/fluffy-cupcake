# fluffy-cupcake

fluffy-cupcake 是一个基于 Gin 的轻量 Web 服务。访问 `/yanlili` 可以看到可爱的 GIF 按钮；每点击一次，页面会向上漂浮并渐隐一行“按按钮，想哥哥+1”。

## 当前版本

`V0.0.2_20260814`

## 快速开始

环境要求：Go 1.26.3。

```bash
make dev
```

浏览器访问 `http://localhost:4819/yanlili`，健康检查地址为 `http://localhost:4819/healthz`。

## 构建与部署

```bash
make test
make build
```

`make build` 仅在 Docker 中生成 `linux/amd64` 镜像 `fluffy-cupcake:V0.0.2_20260814`，不会在仓库中生成本机二进制。ARM64 Linux 服务器可使用 `make build TARGET_PLATFORM=linux/arm64`。

生产环境使用 `compose.yaml` 启动 Gin 与 Caddy。Caddy 通过 `fluffy-cupcake.cn` 对外提供自动 HTTPS，具体准备步骤见 [部署指南](docs/user/deployment.md)。

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
