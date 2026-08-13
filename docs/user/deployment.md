# 部署指南

## 架构

生产环境由 Docker Compose 管理两个容器：Gin 应用在内部网络监听统一的应用端口 4819；Caddy 通过标准公网 80/443 端口申请 HTTPS 证书并把请求转发到 4819。公开域名为 `fluffy-cupcake.cn`。

## 部署前准备

1. 将 `fluffy-cupcake.cn` 的 A 记录指向服务器公网 IPv4；若服务器可用 IPv6，再配置 AAAA 记录。
2. 放通服务器 TCP 80、TCP 443；若需要 HTTP/3，同时放通 UDP 443。
3. 安装 Docker Engine 和 Docker Compose v2，将本仓库放到服务器。
4. 确认 80/443 端口未被其他服务占用。

## 构建镜像

```bash
make build
```

默认产物是 `linux/amd64` 镜像 `fluffy-cupcake:V0.0.2_20260814`。若服务器是 ARM64，运行 `make build TARGET_PLATFORM=linux/arm64`。镜像采用多阶段交叉编译和无基础系统的 `scratch` 运行阶段，最终仅包含静态服务二进制，以非 root 用户启动，并通过程序自身完成 `/healthz` 健康检查。

## 启动服务

```bash
docker compose up -d --build
docker compose ps
docker compose logs -f caddy app
```

DNS 生效且端口开放后，访问 `https://fluffy-cupcake.cn/yanlili`。证书和 Caddy 状态保存在命名卷中，更新容器不会丢失。

## 更新与停止

```bash
docker compose up -d --build
docker compose down
```

`docker compose down` 不会删除命名卷；不要附加 `--volumes`，除非明确需要清除证书和 Caddy 状态。
