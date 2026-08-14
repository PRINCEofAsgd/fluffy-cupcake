#!/bin/sh
# 本脚本以项目约定的开发配置启动 Gin 服务。
set -eu

# 本地存在 .env 时导入数据库和 JWT 配置；该文件已被 Git/Docker 忽略。
if [ -f ./.env ]; then
    set -a
    . ./.env
    set +a
fi

export APP_ADDR="${APP_ADDR:-0.0.0.0:4819}"
export APP_MODE="${APP_MODE:-debug}"

exec go run ./cmd/server
