#!/bin/sh
# 本脚本构建指定架构的 Linux 容器镜像，不生成本机二进制文件。
set -eu

app_version="${APP_VERSION:-V0.0.2_20260814}"
target_platform="${TARGET_PLATFORM:-linux/amd64}"
image="${IMAGE:-fluffy-cupcake:${app_version}}"

# 仅接受项目当前支持的 Linux 服务器架构，避免误构建本机平台镜像。
case "${target_platform}" in
    linux/amd64|linux/arm64) ;;
    *)
        echo "不支持的目标平台：${target_platform}，可选 linux/amd64 或 linux/arm64" >&2
        exit 1
        ;;
esac

docker build \
    --platform "${target_platform}" \
    --build-arg "APP_VERSION=${app_version}" \
    --tag "${image}" \
    .
