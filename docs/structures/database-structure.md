# 数据库结构

当前使用 MySQL 8.x、InnoDB 和 `utf8mb4_0900_ai_ci`。结构变更只通过 `migrations/` 中的 golang-migrate Up/Down 文件管理。应用连接池强制 `parseTime=true`、`loc=UTC` 和 session `time_zone='+00:00'`；登录、邀请、解绑、分钟桶和 API 时间以 UTC 保存与传输，“每日想念”在查询时按浏览器 UTC 偏移划分本地日期。

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
| `unbind_requested_by` / `unbind_requested_at` | ID / 时间 | 是 | 最近一次解绑申请的发起人和 UTC 时间；取消、拒绝后仍保留以展示反馈 |
| `unbind_status` | `VARCHAR(16)` | 否 | 解绑子状态：`none`、`pending`、`cancelled`、`rejected`、`accepted` 或 `direct` |
| `unbind_responded_by` / `unbind_responded_at` | ID / 时间 | 是 | 最近一次取消、拒绝、接受或直接解绑的处理人和 UTC 时间 |
| `ended_at` / `ended_by` | 时间 / ID | 是 | 双向绑定结束信息 |
| `ended_reason` | `VARCHAR(24)` | 是 | `mutual` 双方确认或 `inactive` 30 天未登录直接解绑 |
| `created_at` / `updated_at` | `TIMESTAMP` | 否 | UTC 创建与更新时间 |

外键均指向 `users(id)` 且使用 `RESTRICT`。邀请方、收信方分别有 `(user_id, created_at DESC)` 索引；状态有 `(status, created_at DESC)` 索引。CHECK 约束限制双方不同、主状态、解绑子状态和结束原因。取消或拒绝只把解绑子状态从 `pending` 改为终态，主状态仍为 `active`，所以不会释放当前绑定占位；之后再次申请会覆盖最近一次申请/处理字段但不会新建第二封信。

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
| `click_count` | `BIGINT UNSIGNED` | 否 | 该方向在该分钟的真实点击次数 |
| `created_at` / `updated_at` | `TIMESTAMP` | 否 | UTC 创建与更新时间 |

索引与约束：

- `UNIQUE(companion_binding_id, user_id, target_user_id, button_key, minute_bucket)`：同一关系方向一分钟最多一行，也是原子 Upsert 冲突目标，并支持当前方向统计。
- `KEY idx_click_user_target_minute(user_id, target_user_id, button_key, minute_bucket DESC)`：支持跨历次绑定查询同一对象的双向详细记录。
- 三个关系 ID 均有外键；CHECK 禁止用户想念自己。实时新记录同时保存真实绑定 ID 和对象 ID；迁移旧记录允许只保存对象 ID，避免伪造收件箱信件。
- `click_count > 0`、`SECOND(minute_bucket)=0`。

“每日想念”和“最近想念”都按当前对象的用户对合并不同绑定实例及迁移旧数据，在 SQL 中按新到旧排序并限制 8 条；每日分组以 UTC `minute_bucket` 加浏览器传入的 UTC 偏移生成本地日期，避免本地零点后的记录落入前一天。详细记录同样按用户对查询，先按方向和分钟合并，再用 `UNION ALL` 从新到旧排序，固定每页 20 条，响应不返回绑定实例 ID。

## `qr_login_mappings`

二维码永久文本到真实注册用户的一一映射表，供扫码登录使用。二维码以物理媒介分发，文本本身是公开的永久内容；文本与用户关系由数据库唯一维护，删除映射即可停用对应卡片。

| 字段 | 类型 | NULL | 说明 |
| --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 主键 |
| `qr_text` | `VARCHAR(512)` | 否 | 二维码中的永久文本，唯一；145 字符以内可直接建立 utf8mb4 唯一索引 |
| `user_id` | `BIGINT UNSIGNED` | 否 | 映射到的真实注册用户，扫码登录即签发给该用户 |
| `created_at` / `updated_at` | `TIMESTAMP` | 否 | UTC 创建与更新时间 |

约束：`PRIMARY KEY(id)`、`UNIQUE KEY uk_qr_login_text(qr_text)`、外键 `fk_qr_login_user` 指向 `users(id)` 且使用 `RESTRICT`。扫码登录与密码登录完全等价：服务端在文本映射成功后同样更新该用户 `last_login_at`，30 天未登录判定不区分登录方式。

## Migration

- `000001_create_users.up/down.sql`：创建/删除固定用户表。
- `000002_create_button_click_minutes.up/down.sql`：创建/删除原始用户分钟桶表。
- `000003_add_companion_bindings.up/down.sql`：新增最后登录、绑定信件和当前占位，原地升级分钟桶的关系字段与索引；Down 会先把同一用户/按钮/分钟在不同关系中的行合并，再恢复旧唯一键。
- `000004_remove_utc_click_date.up/down.sql`：移除只表达 UTC 日期且会和浏览器本地日期冲突的生成列及索引；Down 可完整恢复。
- `000005_add_unbind_request_states.up/down.sql`：增加解绑子状态和处理人/时间，升级时回填已有待处理、双方接受或长期未登录结果；Down 会先清除已取消/已拒绝的旧申请人，避免旧程序误判为待处理。
- `000006_create_qr_login_mappings.up/down.sql`：创建/删除二维码文本到用户的一一映射表，Down 为直接 DROP TABLE。

`000003` 不弃用也不删除原点击表。升级后可按明确指定的现有用户补 `target_user_id`，同时保持 `companion_binding_id = NULL`；这不会创建绑定信件。双方以后正常接受唯一一封真实邀请后，详细记录会按用户对呈现这些旧数据。
