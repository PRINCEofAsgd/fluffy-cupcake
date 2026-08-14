# 数据库结构

当前使用 MySQL 8.x、InnoDB 和 `utf8mb4_0900_ai_ci`。结构变更只通过 `migrations/` 中的 golang-migrate Up/Down 文件管理。应用连接池强制 `parseTime=true`、`loc=UTC` 和 MySQL session `time_zone='+00:00'`；业务分钟桶、每日统计和 API 时间均以 UTC 为标准。

## `users`

固定登录用户表，不提供公开注册接口。

| 字段 | 类型 | NULL | 默认值/自动行为 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | `AUTO_INCREMENT` | 主键 |
| `username` | `VARCHAR(64)` | 否 | 无 | 登录用户名 |
| `password_hash` | `VARCHAR(255)` | 否 | 无 | bcrypt 哈希，禁止保存明文 |
| `created_at` | `TIMESTAMP` | 否 | `CURRENT_TIMESTAMP` | UTC 创建时间 |
| `updated_at` | `TIMESTAMP` | 否 | 自动创建/更新 | UTC 更新时间 |

索引与约束：

- `PRIMARY KEY (id)`。
- `UNIQUE KEY uk_users_username (username)`：同时保证唯一用户名并支持登录等值查询，不再重复建立普通 username 索引。

## `button_click_minutes`

按用户、稳定按钮业务标识和 UTC 分钟保存聚合点击数，不为每次物理点击插行。

| 字段 | 类型 | NULL | 默认值/自动行为 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | `AUTO_INCREMENT` | 主键 |
| `user_id` | `BIGINT UNSIGNED` | 否 | 无 | 点击用户，来自 JWT Context |
| `button_key` | `VARCHAR(64)` | 否 | 无 | 当前为 `yanlili`，不绑定具体 DOM |
| `minute_bucket` | `DATETIME` | 否 | 无 | 服务端 UTC 时间向下取整到分钟，秒为 0 |
| `click_count` | `BIGINT UNSIGNED` | 否 | 无 | 该用户在该按钮/分钟内的真实点击次数 |
| `created_at` | `TIMESTAMP` | 否 | `CURRENT_TIMESTAMP` | UTC 创建时间 |
| `updated_at` | `TIMESTAMP` | 否 | 自动创建/更新 | UTC 更新时间 |

索引、约束与关系：

- `PRIMARY KEY (id)`。
- `UNIQUE KEY uk_click_user_button_minute (user_id, button_key, minute_bucket)`：同一用户/按钮/分钟最多一行，也是并发 Upsert 的冲突目标。
- `KEY idx_click_button_minute (button_key, minute_bucket)`：支持共享按钮总数、按分钟范围和分组统计。
- `FOREIGN KEY fk_click_user (user_id) REFERENCES users(id) ON DELETE/UPDATE RESTRICT`：固定用户不可在仍有点击记录时删除，避免孤儿数据。
- `CHECK click_count > 0` 和 `CHECK SECOND(minute_bucket) = 0`：防止无意义数量或未对齐分钟进入表中。

内部保存用户维度；展示统计对相同 `button_key + minute_bucket` 的不同用户执行 `SUM`，因此时间列表每分钟只有一项、count 是所有用户合计。每日统计按 UTC 自然日聚合。

## Migration

- `000001_create_users.up/down.sql`：创建/删除 `users`。
- `000002_create_button_click_minutes.up/down.sql`：创建/删除点击分钟桶；Down 需先删除该子表再回滚用户表。
