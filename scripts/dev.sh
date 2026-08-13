#!/bin/sh
# 本脚本以项目约定的开发配置启动 Gin 服务。
set -eu

export APP_ADDR="${APP_ADDR:-0.0.0.0:4819}"
export APP_MODE="${APP_MODE:-debug}"

exec go run ./cmd/server
