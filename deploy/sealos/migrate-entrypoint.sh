#!/bin/sh
# 从 Sealos 环境变量读取应用同款 DSN，避免把数据库密码写入镜像或启动参数模板。
set -eu

if [ -z "${DATABASE_DSN:-}" ]; then
    echo "缺少 DATABASE_DSN；请复用应用容器连接 Sealos MySQL 的内网 DSN" >&2
    exit 1
fi

# 迁移镜像只开放向上迁移和版本查询，避免误执行 down/force 破坏生产数据。
migrate_command="${1:-up}"
case "${migrate_command}" in
    up|version) ;;
    *)
        echo "不支持的迁移命令：${migrate_command}；仅允许 up 或 version" >&2
        exit 1
        ;;
esac

exec migrate -path=/migrations -database "mysql://${DATABASE_DSN}" "${migrate_command}"
