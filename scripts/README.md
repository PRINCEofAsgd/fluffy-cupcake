# 脚本说明

## `dev.sh`

若存在 `.env`，先导入 MySQL 与 JWT 配置，再以本地默认值启动服务：`APP_ADDR=0.0.0.0:4819`、`APP_MODE=debug`。调用方设置的同名环境变量优先。

```bash
./scripts/dev.sh
```

## `build.sh`

通过 Dockerfile 构建 Linux 容器镜像，不生成本机二进制。默认平台为 `linux/amd64`，默认镜像为 `fluffy-cupcake:V0.0.6_20260815`。

```bash
./scripts/build.sh
TARGET_PLATFORM=linux/arm64 ./scripts/build.sh
```

脚本仅接受 `linux/amd64` 和 `linux/arm64`。可以通过 `APP_VERSION`、`TARGET_PLATFORM` 和 `IMAGE` 覆盖版本、目标平台和镜像名称。

生产镜像与域名服务通过根目录 `Dockerfile` 和 `compose.yaml` 管理，具体见 `docs/user/deployment.md`。
