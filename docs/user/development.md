# 开发指南

## 环境要求

- Go 1.26.3
- 可选：Docker 与 Docker Compose v2

## 安装依赖

```bash
go mod download
```

Go 模块缓存由 Go 工具链管理；若需清理全部模块缓存，可运行 `go clean -modcache`，下次构建会重新下载。

## 本地运行

```bash
make dev
# 或
./scripts/dev.sh
```

默认 `APP_ADDR=0.0.0.0:4819`，因此本机可通过 `localhost:4819` 访问，同一局域网设备也可通过开发机 IP 的 4819 端口访问。可按需覆盖配置：

```bash
APP_ADDR=127.0.0.1:9090 APP_MODE=debug go run ./cmd/server
```

## 测试与 Linux 镜像构建

```bash
go test ./...
make build
```

`make build` 默认构建 `linux/amd64` 容器镜像 `fluffy-cupcake:V0.0.2_20260814`，不会生成适用于开发机的本地二进制文件。若部署服务器使用 ARM64：

```bash
make build TARGET_PLATFORM=linux/arm64
```

本地开发仍通过 `make dev` 直接运行源码。页面资源已通过 `embed.FS` 编入 Linux 镜像中的服务二进制，无需单独复制 `web` 目录。

## 配置项

| 环境变量 | 开发默认值 | 说明 |
| --- | --- | --- |
| `APP_ADDR` | `0.0.0.0:4819` | HTTP 监听地址 |
| `APP_MODE` | `debug` | Gin 模式，可用 `debug`、`test`、`release` |
