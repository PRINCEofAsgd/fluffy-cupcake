# 项目工作上下文

## Step 1：项目基线、Gin 服务与互动页面（2026-08-14）

### 目标

- 首次建立符合 `AGENTS.md` 职责划分的项目文档框架。
- 按 Gin 项目常用分层方式建立可维护的 Web 服务。
- 在 `/yanlili` 提供 GIF 按钮和点击文字上浮渐隐动效。
- 保持本地开发监听全部网口，并为 `fluffy-cupcake.cn` 准备容器化 HTTPS 部署。

### 实现

- 建立 `cmd/server`、`internal/config`、`internal/handler`、`internal/server`、`web` 分层。
- 将模板、样式、脚本和 `miss-button.gif` 嵌入二进制。
- 增加优雅关闭、请求日志、异常恢复、安全响应头、404 和 `/healthz`。
- 页面支持鼠标、触屏、键盘连续激活，以及系统减少动画偏好。
- 增加多阶段 `Dockerfile`、生产 `compose.yaml` 和 Caddy 自动 HTTPS 配置。
- 根据后续要求，将本地开发、容器运行、健康检查和反向代理目标端口统一调整为 4819。
- 移除本机二进制构建流程，将 `make build` 和 `scripts/build.sh` 统一调整为构建 Linux 镜像；默认目标为 `linux/amd64`，并支持显式切换 `linux/arm64`。
- 将运行镜像调整为 `scratch`，由静态 Go 二进制提供容器健康检查子命令，避免最终镜像依赖目标架构的基础系统、shell 或 wget。
- 建立说明、结构、调用链、数据库、开发、部署、操作、版本、脚本等文档。

### 验证

- `go test ./...` 通过。
- `go vet ./...` 通过。
- `go build ./cmd/server` 通过。
- `docker compose config` 通过。
- 启动本地发布模式二进制后，`/yanlili`、`/healthz`、页面文案、GIF 引用和安全响应头端到端检查通过，并验证 SIGINT 优雅退出。
- 端口调整后在 `127.0.0.1:4819` 再次完成页面、健康检查、版本资源参数和优雅退出验证。
- 初次 `docker build` 受 Docker Hub 鉴权连接超时影响；改用缓存构建器与 `scratch` 运行阶段后已完成构建。
- 调整为 `scratch` 运行镜像后，`make build` 已成功生成 `linux/amd64` 镜像 `fluffy-cupcake:V0.0.2_20260814`，镜像大小约 8.1 MB。
- 镜像元数据验证为 `linux/amd64`、用户 `10001:10001`、暴露 4819、内置二进制健康检查；在 ARM64 开发机通过模拟运行后，`/yanlili` 和 `/healthz` 正常且容器状态为 `healthy`。

### 后续

- 服务器上线前需配置域名 DNS、开放 80/443 端口并启动 Compose。
