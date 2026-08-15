# 数据库结构

当前使用 MySQL 8.x、InnoDB 和 `utf8mb4_0900_ai_ci`。结构变更只通过 `migrations/` 中的 golang-migrate Up/Down 文件管理。应用连接池强制 `parseTime=true`、`loc=UTC` 和 session `time_zone='+00:00'`；登录、邀请、解绑、分钟桶、每日统计和 API 时间均以 UTC 为标准。

## `users`

固定登录用户表，不提供公开注册接口。

| 字段 | 类型 | NULL | 默认值/自动行为 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | `AUTO_INCREMENT` | 主键 |
| `username` | `VARCHAR(64)` | 否 | 无 | 唯一登录用户名 |
| `password_hash` | `VARCHAR(255)` | 否 | 无 | bcrypt 哈希，禁止保存明文 |
| `last_login_at` | `TIMESTAMP` | 是 | `NULL` | 最近一次密码登录 UTC 时间；用于 30 天未登录直接解绑 |
| `created_at` | `TIMESTAMP` | 否 | `CURRENT_TIMESTAMP` | UTC 创建时间 |
| `updated_at` | `TIMESTAMP` | 否 | 自动创建/更新 | UTC 更新时间 |

约束：`PRIMARY KEY(id)`、`UNIQUE KEY uk_users_username(username)`。

## `companion_bindings`

一行既是一封双方永久可见、没有删除接口的绑定信件，也是一段绑定关系的完整生命周期。重新与同一用户绑定会创建新行，不覆盖旧信件。

| 字段 | 类型 | NULL | 说明 |
| --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 信件与关系主键 |
| `inviter_id` / `invitee_id` | `BIGINT UNSIGNED` | 否 | 邀请双方，必须不同 |
| `inviter_note` / `invitee_note` | `VARCHAR(64)` | 否 | 双方各自私有的对象备注，空串表示显示用户名 |
| `status` | `VARCHAR(16)` | 否 | `pending`、`active`、`ended` 或 `superseded` |
| `accepted_at` | `TIMESTAMP` | 是 | 接受绑定时间；非空代表历史对象成立过 |
| `unbind_requested_by` / `unbind_requested_at` | ID / 时间 | 是 | 同一信件内未决解绑邀请 |
| `ended_at` / `ended_by` | 时间 / ID | 是 | 双向绑定结束信息 |
| `ended_reason` | `VARCHAR(24)` | 是 | `mutual` 双方确认或 `inactive` 30 天未登录直接解绑 |
| `created_at` / `updated_at` | `TIMESTAMP` | 否 | UTC 创建与更新时间 |

外键均指向 `users(id)` 且使用 `RESTRICT`。邀请方、收信方分别有 `(user_id, created_at DESC)` 索引；状态有 `(status, created_at DESC)` 索引。CHECK 约束限制双方不同、状态值和结束原因。

## `companion_active_memberships`

只保存当前活跃绑定的快速占位，每段绑定固定两行。解绑会在事务内删除这两行，但不会删除 `companion_bindings` 信件。

| 字段 | 类型 | NULL | 说明 |
| --- | --- | --- | --- |
| `user_id` | `BIGINT UNSIGNED` | 否 | 主键；数据库层保证一个用户最多一行当前绑定 |
| `binding_id` | `BIGINT UNSIGNED` | 否 | 活跃信件 ID |
| `partner_user_id` | `BIGINT UNSIGNED` | 否 | 对方用户 ID，必须与 `user_id` 不同 |
| `created_at` | `TIMESTAMP` | 否 | 当前占位建立时间 |

`UNIQUE(binding_id, user_id)` 同时服务按绑定释放双方占位的查询；三个 ID 均有外键保护。

## `button_click_minutes`

保留原表并升级为“用户 → 绑定对象”的方向性 UTC 分钟桶，不为每次物理点击插行。

| 字段 | 类型 | NULL | 说明 |
| --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 主键 |
| `user_id` | `BIGINT UNSIGNED` | 否 | 想念发起用户 |
| `companion_binding_id` | `BIGINT UNSIGNED` | 是 | 点击发生时的真实绑定关系；迁移旧数据不伪造信件，因此允许为 `NULL` |
| `target_user_id` | `BIGINT UNSIGNED` | 是 | 想念对象；旧数据可先补对象而保持绑定关系为 `NULL` |
| `button_key` | `VARCHAR(64)` | 否 | 当前为 `yanlili` |
| `minute_bucket` | `DATETIME` | 否 | 服务端 UTC 时间向下取整到分钟 |
| `click_date` | `DATE`（STORED） | 否 | 由 `DATE(minute_bucket)` 生成的 UTC 日期，只用于索引化每日聚合 |
| `click_count` | `BIGINT UNSIGNED` | 否 | 该方向在该分钟的真实点击次数 |
| `created_at` / `updated_at` | `TIMESTAMP` | 否 | UTC 创建与更新时间 |

索引与约束：

- `UNIQUE(companion_binding_id, user_id, target_user_id, button_key, minute_bucket)`：同一关系方向一分钟最多一行，也是原子 Upsert 冲突目标，并支持当前方向统计。
- `KEY idx_click_relation_date(companion_binding_id, user_id, target_user_id, button_key, click_date DESC)`：支持沿方向日期倒序聚合并只取最近 8 天。
- `KEY idx_click_user_target_minute(user_id, target_user_id, button_key, minute_bucket DESC)`：支持跨历次绑定查询同一对象的双向详细记录。
- 三个关系 ID 均有外键；CHECK 禁止用户想念自己。实时新记录同时保存真实绑定 ID 和对象 ID；迁移旧记录允许只保存对象 ID，避免伪造收件箱信件。
- `click_count > 0`、`SECOND(minute_bucket)=0`。

“每日想念”和“最近想念”分别在 SQL 中按新到旧排序并限制 8 条；详细记录按用户对查询，先把不同绑定实例及迁移旧数据按方向和分钟合并，再用 `UNION ALL` 从新到旧排序，固定每页 20 条，响应不返回绑定实例 ID。

## Migration

- `000001_create_users.up/down.sql`：创建/删除固定用户表。
- `000002_create_button_click_minutes.up/down.sql`：创建/删除原始用户分钟桶表。
- `000003_add_companion_bindings.up/down.sql`：新增最后登录、绑定信件和当前占位，原地升级分钟桶的关系字段与索引；Down 会先把同一用户/按钮/分钟在不同关系中的行合并，再恢复旧唯一键。

`000003` 不弃用也不删除原点击表。升级后可按明确指定的现有用户补 `target_user_id`，同时保持 `companion_binding_id = NULL`；这不会创建绑定信件。双方以后正常接受唯一一封真实邀请后，详细记录会按用户对呈现这些旧数据。
