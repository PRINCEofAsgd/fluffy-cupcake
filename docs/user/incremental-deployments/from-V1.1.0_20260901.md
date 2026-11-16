# 从 V1.1.0_20260901 更新到 V1.1.0_20261116

- 适用环境：Singapore Sealos Application 与 Guangzhou Managed MySQL。
- 数据库：`schema_migrations` 从 `version=6, dirty=0` 经版本 7 升至 `version=8, dirty=0`。
- 用户影响：切换后已有用户需重新登录一次。新会话默认从密码或扫码登录成功起固定有效 15 天。想念统计默认显示“ta想我”。
- 执行结果：2026-11-16 确认完成。生产库在确认备份后分步执行 Migration 7、8 的建表 SQL 并登记版本，验收为 `version=8, dirty=0`。Singapore 应用切换后，用户确认 `/healthz`、`/yanlili` 的资源版本及实际功能测试均通过。

## 切换前

1. 核对线上运行镜像和目标 MySQL schema，确认最近一次备份可用。`schema_migrations` 应只有一行且为 `version=6, dirty=0`。若基线不同或状态为 dirty，先根据实际状态核对迁移和部署记录。
2. 本地执行 `make test` 与 `make build`，将 `linux/amd64` 应用镜像推送为新不可变标签 `docker.io/princeofasgd/fluffy-cupcake:V1.1.0_20261116`。保留上一版镜像标签用于回退。
3. 为受信任的迁移执行环境准备 golang-migrate v4.19.1、本次完整 `migrations/` 目录及独立迁移账号。账号与 DSN 通过 Secret 注入。运行时账号若按表授权，补充 `auth_sessions`、`auth_refresh_tokens` 和 `companion_moods` 的 SELECT、INSERT、UPDATE 权限。

## 先迁移，再切换

1. 从可访问 Guangzhou Managed MySQL 的执行环境中，使用与应用 DSN 指向同一 schema 的迁移 DSN 检查版本、执行 `up`、再读版本。下例中的 `MIGRATION_DSN` 是该环境注入的 Go MySQL DSN，包含连接信息和凭据。

   ```bash
   migrate -path ./migrations -database "mysql://${MIGRATION_DSN}" version
   migrate -path ./migrations -database "mysql://${MIGRATION_DSN}" up
   migrate -path ./migrations -database "mysql://${MIGRATION_DSN}" version
   ```

2. 在目标库确认 `version=8, dirty=0`。三张新表均应存在并为空，已有用户、二维码映射、绑定和点击无需回填。

   ```sql
   SELECT version, dirty FROM schema_migrations;
   SHOW CREATE TABLE auth_sessions;
   SHOW CREATE TABLE auth_refresh_tokens;
   SHOW CREATE TABLE companion_moods;
   ```

3. 完成一次应用切换。安排维护窗口，先停止全部旧版副本接收流量，再以新镜像启动，避免版本并行接受认证请求。在 Singapore Application 设置 `ACCESS_TOKEN_TTL=15m`、`SESSION_TTL=360h` 并移除 `JWT_EXPIRE`。继续使用现有 `DATABASE_DSN`、`JWT_SECRET`、`APP_MODE=release`、端口 `4819` 和健康检查配置。
4. 等待新副本就绪，再恢复入口流量。首次打开旧标签页的用户需重新加载页面并登录，以取得新版请求逻辑和会话 Cookie。

## 上线验证

- 检查 `/healthz`、`/`、`/yanlili`，核对服务日志是否有会话表或数据库权限错误。
- 用升级前已登录的浏览器请求 `/api/auth/me`，预期得到 401，且仅凭旧 JWT 请求 `/api/auth/refresh` 同样返回 401。
- 分别用密码和二维码登录。检查响应含两个 `HttpOnly; Secure; SameSite=Lax` Cookie，访问 Cookie 到期时间约为 15 分钟后，刷新 Cookie 的截止时间约为 15 天后。
- 用已绑定账号打开 `/yanlili`，确认默认展示“ta想我”，点双箭头后心情和计数一起切换到“我想ta”。
- 用已绑定的测试账号请求 `GET /api/yanlili/mood` 和 `?direction=mine`，核对响应为当前关系的快照或空值，并核对未登录请求返回 401。
- 用两个互相绑定的测试账号打开 `/yanlili`。一方在对方视角点击铅笔，保存多组选项及自定义文字，核对同步提示位与保留的统计方向。另一方刷新后应默认看到更新。验证编辑面板的备注入口、保存清空后的默认文案及解绑后隐藏统计面板。
- 待访问 Cookie 到期后，从两个标签页请求受保护页面或统计。确认登录可恢复，刷新 Cookie 的截止时间保持原值；等待期间触发一次已绑定测试用户的点击同步，核对次数只增加一次。
- 退出后确认浏览器删除两个 Cookie，使用退出前的访问 JWT 或刷新凭证请求对应接口均得到 401。

## 失败与回退

- 迁移有错误或状态为 dirty 时保持旧应用，核查实际表结构和迁移状态后再处理。
- 回退应用时保留 `companion_moods`。Migration 8 的 Down 会删除整张表及其中的心情快照。
- 新应用上线后如需回退，停止新副本流量并保留新增表，再恢复上一不可变镜像及旧版本的 `JWT_EXPIRE` 配置。恢复流量前为回退镜像换用新的 `JWT_SECRET`。旧版仅校验签名和有效期，复用签名密钥会让它接受部分尚未到期的新版 JWT，无法检查会话撤销状态。
- 若回退前已向用户签发新会话，在停止新版后撤销仍有效的新会话，避免后续再次上线时恢复回退前的会话。用具备权限的账号在目标库执行下列 SQL。回退后用户需重新登录。

  ```sql
  UPDATE auth_sessions
  SET revoked_at = UTC_TIMESTAMP(6)
  WHERE revoked_at IS NULL AND expires_at > UTC_TIMESTAMP(6);
  ```
