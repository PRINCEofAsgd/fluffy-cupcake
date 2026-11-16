# 脚本说明

## `dev.sh`

若存在 `.env`，先导入 MySQL 与 JWT 配置，再以本地默认值启动服务：`APP_ADDR=0.0.0.0:4819`、`APP_MODE=debug`。调用方设置的同名环境变量优先。

```bash
./scripts/dev.sh
```

## `build.sh`

通过 Dockerfile 先构建 Homepage，再构建 Linux 容器镜像，不生成本机二进制。默认平台为 `linux/amd64`，默认镜像为 `fluffy-cupcake:V1.1.0_20261116`。

```bash
./scripts/build.sh
TARGET_PLATFORM=linux/arm64 ./scripts/build.sh
```

脚本仅接受 `linux/amd64` 和 `linux/arm64`。可以通过 `APP_VERSION`、`TARGET_PLATFORM` 和 `IMAGE` 覆盖版本、目标平台和镜像名称。

生产镜像由根目录 `Dockerfile` 构建；`compose.yaml` 只负责本地/独立单机编排，Sealos 部署见 `docs/user/deployment.md`。

## `db-migration/verify.sh`

只读检查 OLD/NEW MySQL，自动获取所有表的精确行数，并比较 Migration、列、PK、UNIQUE/普通索引、外键和 UTC 时间边界。连接信息从 `OLD_DB_*`/`NEW_DB_*` 环境变量读取，密码只通过终端隐藏输入并短暂写入权限为 0600 的临时 client 文件；脚本不打印 Secret，任何查询或比较失败都会非零退出。

```bash
export OLD_DB_HOST='<OLD_EXTERNAL_ENDPOINT>'
export OLD_DB_PORT='3306'
export OLD_DB_USER='<OLD_READONLY_USER>'
export OLD_DB_NAME='fluffy_cupcake'

export NEW_DB_HOST='<NEW_EXTERNAL_ENDPOINT>'
export NEW_DB_PORT='3306'
export NEW_DB_USER='<NEW_READONLY_USER>'
export NEW_DB_NAME='fluffy_cupcake'

make db-preflight SIDE=OLD
make db-preflight SIDE=NEW
make db-verify
```

上述变量不包含密码；每次连接由终端隐藏询问密码。先分别预检，停写并把完整 dump 恢复到空目标库后再运行 OLD/NEW 比较。目标库开始接受新写入后，两库差异属于预期状态，不应再次用全库比较判断健康。正式操作顺序见 `docs/user/database-region-migration.md`。脚本不执行 DDL/DML、不切换生产 DSN，也不删除数据库。
