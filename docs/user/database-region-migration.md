# 数据库跨区域迁移 Runbook

本文用于把生产数据从 Singapore Sealos MySQL 迁移到 Guangzhou Sealos Managed MySQL。网站和 Gin 应用继续运行在 Singapore，迁移后只通过 Guangzhou 数据库提供的 External/Public Endpoint 访问 MySQL。

```text
Local:      App → Docker Compose MySQL（mysql:3306 仅在 Compose 网络内有效）
Production: Singapore Sealos App → Guangzhou Managed MySQL External/Public Endpoint
```

迁移采用短维护窗口：停写 → 平台备份 → 完整 dump → 空库 restore → 只读验证 → 切换 `DATABASE_DSN` → 冒烟 → 观察。它不使用跨集群 Service、VPN、代理、双写、主从或 CDC。

## 操作原则

- 全程不要把真实 Host、密码、Token、完整 DSN 或二维码文本写进 Git、聊天、截图和工单。
- 应用必须使用独立的 `fluffy_cupcake_app` 类账号，禁止使用 root；dump/restore/未来 Migration 可使用另一个受控管理账号。
- 目标库是空库时直接恢复完整 dump。dump 已包含表、数据、索引、约束和 `schema_migrations`，恢复前绝不能先执行 `make migrate-up`。
- TLS 和来源 IP 白名单只按 Sealos 控制台当前实例实际提供的能力配置。没有该能力时记录事实，不编造配置。
- 旧库在观察期内保持可用且禁止写入；任何删除动作都必须人工确认并先做最后备份。

## 命令准备

本地终端只导出非 Secret 连接信息，变量值请从 Sealos 控制台复制，不要把以下命令提交到仓库：

```bash
export OLD_DB_HOST='<SINGAPORE_EXTERNAL_ENDPOINT>'
export OLD_DB_PORT='3306'
export OLD_DB_USER='<OLD_BACKUP_OR_READONLY_USER>'
export OLD_DB_NAME='fluffy_cupcake'

export NEW_DB_HOST='<GUANGZHOU_EXTERNAL_ENDPOINT>'
export NEW_DB_PORT='3306'
export NEW_DB_USER='<NEW_RESTORE_OR_READONLY_USER>'
export NEW_DB_NAME='fluffy_cupcake'
```

验证工具会隐藏读取密码，并用 0600 临时 MySQL client 文件连接，退出即删除。若 Sealos 明确要求 MySQL 客户端 TLS，可按它给出的值增加 `OLD_DB_SSL_MODE`/`NEW_DB_SSL_MODE`（`REQUIRED`、`VERIFY_CA` 或 `VERIFY_IDENTITY`）及可选 CA 文件路径；未确认时不要设置。

## Step 0 Preflight

**目的**：确认源库版本、当前库、全部表、Migration、精确行数和 UTC 时间边界，并记录维护前基线。

**操作**：

```bash
make db-preflight SIDE=OLD
```

脚本等价覆盖 `SELECT VERSION()`、`SELECT DATABASE()`、`SHOW TABLES`、`SELECT version, dirty FROM schema_migrations`，并自动对库内所有 base table 执行精确 `COUNT(*)`。另在 Sealos 控制台确认旧库自动备份最近一次任务成功、磁盘余量充足，记录应用当前副本数和 `DATABASE_DSN` Secret 的旧版本（不要复制 Secret 到本地文档）。以当前生产镜像对应的 schema 版本为准，`schema_migrations` 应只有一行且为 `dirty=0`。版本 6 的业务表如下：

- `users`
- `button_click_minutes`
- `companion_bindings`
- `companion_active_memberships`
- `qr_login_mappings`

版本 7 还包含 `auth_sessions` 与 `auth_refresh_tokens`。跨区搬迁保持源库原有版本，应用升级按独立的局部更新部署手册执行。

**成功判据**：连接成功、数据库名正确、上述表与 `schema_migrations` 均存在、`dirty=0`，行数和时间统计可读取。

**失败处理**：停止迁移；先处理连接、权限、脏 Migration、缺表或备份异常，不在未知状态下修改版本号或目标库。

## Step 1 Create Guangzhou MySQL

**目的**：创建独立、空白、可恢复的 MySQL 8.x 目标实例和 `fluffy_cupcake` 数据库。

**操作**：[需要人工在 Sealos 操作] 在 Guangzhou Region 创建 Managed MySQL 8.x，开启自动备份，生成新的随机管理密码；通过 Database Studio 或平台管理能力创建 UTF-8 数据库。若使用 SQL，先确认管理账号权限：

```sql
CREATE DATABASE fluffy_cupcake
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
```

目标数据库必须为空，不运行 golang-migrate。执行：

```bash
make db-preflight SIDE=NEW
```

**成功判据**：`SELECT VERSION()` 为兼容的 MySQL 8.x，`SELECT DATABASE()` 为 `fluffy_cupcake`，`SHOW TABLES` 为空，自动备份已开启。

**失败处理**：不要恢复；修正 Region、版本、字符集、权限或数据库选择。若目标库已有业务表，确认无用后由人工重建空库，禁止脚本自动删除。

## Step 2 Configure security

**目的**：用最小权限账号和平台网络控制保护公网数据库。

**操作**：[需要人工在 Sealos 操作]

1. 使用 `openssl rand -hex 24` 为广州应用账号生成从未使用过的密码；不要复用旧库、root、登录用户或 JWT 密码。
2. 在 Managed MySQL 用户管理界面创建独立应用账号。若平台允许 SQL 管理账号，可采用等价权限：

   ```sql
   CREATE USER 'fluffy_cupcake_app'@'%' IDENTIFIED BY '<仅在 Database Studio 输入的新随机密码>';
   GRANT SELECT, INSERT, UPDATE, DELETE ON fluffy_cupcake.* TO 'fluffy_cupcake_app'@'%';
   ```

   `%` 只表示 MySQL 账号 Host 范围；若 Sealos 支持来源 IP 白名单，仍应只允许 Singapore 应用的固定出口 IP。应用账号不授予 `CREATE`、`ALTER`、`DROP`、`GRANT OPTION` 或全局权限。
3. 保留单独的 restore/migration 管理账号，不把它配置到应用。
4. 查看该实例是否明确支持白名单、TLS、CA 下载与证书主机名校验。支持白名单时只放行 Singapore 应用出口地址；不支持就记录这一限制。证书可验证时应用 DSN 使用 `tls=true`。若端点能加密但平台没有提供可验证 CA，只能在明确接受“传输加密但不验证服务器身份”的风险后选择 `tls=skip-verify`；禁止使用可能回退到明文的 `tls=preferred`，并应继续向平台索取 CA。

**成功判据**：应用账号能对目标业务库执行必要 DML、不能执行 DDL；restore 账号能完成恢复；网络和 TLS 设置与控制台能力一致。

**失败处理**：不要放宽到 root 或全局权限；修复账号、实例能力或出口地址后重测。

## Step 3 Test network

**目的**：迁移前确认本机可操作目标库，且 Singapore App 能到达 Guangzhou 公网端点。

**操作**：本机用提示式密码测试：

```bash
mysql --host="$NEW_DB_HOST" --port="$NEW_DB_PORT" \
  --user="$NEW_DB_USER" --password --connect-timeout=5 \
  --database="$NEW_DB_NAME" \
  --execute="SELECT VERSION(), DATABASE(), UTC_TIMESTAMP();"
```

[需要人工在 Sealos 操作] 在 Singapore 应用同网络的受控诊断环境测试 Guangzhou endpoint 的 DNS/TCP 3306；若无诊断终端，可临时以新 DSN 部署一个不接收公网流量的单副本进行启动 Ping，成功后立即删除，不能让它承担写流量。

**成功判据**：DNS、TCP、认证和查询在 5 秒内成功；记录实际 TLS cipher、证书验证结果以及最终采用严格验证或 `skip-verify` 的风险决定。

**失败处理**：保持生产连接旧库；检查 External/Public Endpoint、端口、白名单、出口 IP、账号与 TLS，不使用 `.svc` 地址跨 Region。

## Step 4 Backup old DB

**目的**：在停写前准备平台级可恢复点。

**操作**：[需要人工在 Sealos 操作] 在 Singapore Managed MySQL 控制台创建手动备份/快照，等待状态变为成功，记录备份 ID 和 UTC 时间。确认自动备份仍可用。

**成功判据**：手动备份显示完成，恢复入口可见。

**失败处理**：不进入停写和 dump；先解决备份任务、存储或实例健康问题。

## Step 5 Stop writes

**目的**：冻结源库，保证 dump 与验证期间没有业务漂移。

**操作**：[需要人工在 Sealos 操作] 将 Singapore Gin Application 副本缩容为 0，或使用平台提供的等价“停止应用”能力；不要只停 Caddy，因为应用仍可能被内部调用。等待旧连接关闭。随后连续两次运行：

```bash
make db-preflight SIDE=OLD
```

重点比较所有表行数及关键表更新时间，两次检查之间保持应用停止。版本 7 的登录、刷新与退出都会写入会话数据。

**成功判据**：应用无运行副本/无写入口，两次源库基线不再变化。

**失败处理**：找出仍在写入的副本、Job 或 CLI；在写入完全停止前不 dump。

## Step 6 Dump

**目的**：生成包含 schema、数据、索引、约束和 `schema_migrations` 的第二份本地备份。

**操作**：确保本机 `mysqldump` 与源 MySQL 8.x 兼容，然后执行：

```bash
mkdir -p db-migration-dumps
chmod 700 db-migration-dumps
umask 077
mysqldump \
  --host="$OLD_DB_HOST" \
  --port="$OLD_DB_PORT" \
  --user="$OLD_DB_USER" \
  --password \
  --single-transaction \
  --quick \
  --no-tablespaces \
  --set-gtid-purged=OFF \
  --default-character-set=utf8mb4 \
  --hex-blob \
  "$OLD_DB_NAME" \
  > db-migration-dumps/fluffy_cupcake_backup.sql
chmod 600 db-migration-dumps/fluffy_cupcake_backup.sql
test -s db-migration-dumps/fluffy_cupcake_backup.sql
shasum -a 256 db-migration-dumps/fluffy_cupcake_backup.sql
```

使用 `--password` 让客户端提示输入，绝不使用 `--password=plaintext`。项目表全为 InnoDB，停写加 `--single-transaction --quick` 可获得一致快照；`--no-tablespaces` 避免托管实例 tablespace 权限；`--set-gtid-purged=OFF` 避免目标库需要 GTID/SUPER 类权限；`--hex-blob` 安全保存二进制值。检查常见托管库阻碍：

```bash
rg -n 'DEFINER=|TABLESPACE|GTID_PURGED' db-migration-dumps/fluffy_cupcake_backup.sql
git check-ignore -v db-migration-dumps/fluffy_cupcake_backup.sql
```

仓库 migration 不创建 view、routine、event 或 trigger；若检查意外发现 `DEFINER` 等对象，先确认来源及目标权限，不直接删掉关键 SQL 绕过错误。

**成功判据**：dump 命令为 0、文件非空且权限 600、哈希已记录、Secret 未进入命令行，文件被 `.gitignore` 命中。

**失败处理**：保留停写；删除或隔离不完整文件，修复客户端版本/网络/只读备份权限后重新完整 dump，不使用失败的部分文件恢复。

## Step 7 Restore

**目的**：把完整源库原样恢复到广州空库，包括 `schema_migrations`。

**操作**：再次确认 `make db-preflight SIDE=NEW` 的 `SHOW TABLES` 为空，然后用 restore 管理账号执行：

```bash
mysql \
  --host="$NEW_DB_HOST" \
  --port="$NEW_DB_PORT" \
  --user="$NEW_DB_USER" \
  --password \
  --default-character-set=utf8mb4 \
  "$NEW_DB_NAME" \
  < db-migration-dumps/fluffy_cupcake_backup.sql
```

这里明确禁止先运行 `make migrate-up`。恢复成功后，未来新版本仍照常用 golang-migrate 对已恢复的 `schema_migrations` 继续 `up`。

**成功判据**：mysql 退出码为 0，无权限、DEFINER、GTID、tablespace 或外键错误。

**失败处理**：保持生产停写且不切 DSN；保存完整错误。不要在半恢复库上继续补 SQL，人工重建新的空目标数据库后，修复根因并从完整 dump 重试。

## Step 8 Verify

**目的**：证明 OLD/NEW 的所有表、数据量、Migration、关键约束和 UTC 时间边界一致。

**操作**：

```bash
make db-preflight SIDE=NEW
make db-verify
```

`db-verify` 自动比较：全部 base table/引擎/collation、每表精确行数、`schema_migrations`、全部列定义、PK、UNIQUE/普通索引、所有外键和规则，以及每个 `DATE/DATETIME/TIMESTAMP` 列在 UTC session 下的非空数、最小值、最大值。它也检查核心表，并在源库版本至少为 7 时检查会话表与刷新凭证表。

可在两边 Database Studio 补充人工抽查（不要导出密码哈希或完整二维码文本）：

```sql
SET SESSION time_zone = '+00:00';
SELECT id, username, last_login_at, created_at FROM users ORDER BY id LIMIT 10;
SELECT id, user_id, target_user_id, minute_bucket, click_count
FROM button_click_minutes ORDER BY id DESC LIMIT 10;
SELECT id, inviter_id, invitee_id, status, accepted_at, ended_at
FROM companion_bindings ORDER BY id DESC LIMIT 10;
SELECT user_id, binding_id, partner_user_id FROM companion_active_memberships ORDER BY user_id;
SELECT m.id, m.user_id, u.username, CHAR_LENGTH(m.qr_text) AS qr_text_length
FROM qr_login_mappings m JOIN users u ON u.id = m.user_id ORDER BY m.id;
```

**成功判据**：`make db-verify` 输出全部 `[一致]` 并以 0 退出，抽查 UTC 值与源库一致。

**失败处理**：保持停写、不切 DSN；依据 diff 定位 dump、恢复、版本、字符集或时区问题，重建空目标库并重新恢复。不能通过手改 `schema_migrations` 伪造通过。

## Step 9 Change DATABASE_DSN

**目的**：让新加坡应用使用广州公网数据库和独立应用账号。

**操作**：[需要人工在 Sealos 操作] 在 Singapore Application 的 Secret/环境变量中更新，不修改 Git：

```text
fluffy_cupcake_app:<NEW_RANDOM_PASSWORD>@tcp(<GUANGZHOU_PUBLIC_ENDPOINT>:3306)/fluffy_cupcake?timeout=5s&readTimeout=5s&writeTimeout=5s
```

密码若不是生成的纯十六进制字符，需按 go-sql-driver/mysql DSN 规则编码。应用会继续强制 `parseTime=true`、`loc=UTC`、`time_zone='+00:00'`，并只在 DSN 未显式设置时补三项 5 秒超时。证书验证成功时追加 `&tls=true`；若已确认端点支持 TLS、无法取得 CA 且明确接受身份不校验风险，追加 `&tls=skip-verify` 并记录待升级事项。

**成功判据**：Secret 只存在 Sealos，DSN 使用 External/Public Endpoint、应用账号和目标数据库，不含 `mysql:3306`、`.svc` 或 root。

**失败处理**：不要启动生产副本；修正 DSN 编码、Host、端口、账号或 TLS。

## Step 10 Redeploy

**目的**：以新 DSN 恢复应用服务，并利用启动 Ping 阻止数据库不可达的实例接流量。

**操作**：[需要人工在 Sealos 操作] 重新部署 Singapore Application，先启动 1 个副本，观察启动与健康探针，再恢复原副本数。

**成功判据**：实例保持 Running/Ready，启动未出现“连接 MySQL”错误，`GET /healthz` 为 200；连接池 Ping 在 5 秒上下文内完成。

**失败处理**：保持应用停止或不接流量；若尚无任何广州业务写入，按 Rollback A 切回旧 DSN。先检查网络、账号权限、DSN 编码与 TLS，不反复重启制造噪声。

## Step 11 Smoke test

**目的**：验证真实 API 调用链，并证明新写入只发生在广州。

**操作**：先选用专门的、已绑定且有二维码映射的测试用户；没有安全测试账号时只执行只读项，不创建/解绑真实关系。切换前已记录 OLD/NEW 基线，切换后按顺序：

1. `curl -fsS https://fluffy-cupcake.cn/healthz` 应返回 `{"status":"ok"}`。
2. 不带 Cookie 请求 `curl -i https://fluffy-cupcake.cn/api/auth/me` 应为 401，验证 JWT 边界。
3. 从 Web 页面用测试用户密码登录，再刷新页面；`/api/auth/me` 应返回该用户，验证登录、JWT、用户查询和 `users.last_login_at` 写入。
4. 页面打开绑定区、收件箱、历次对象和统计；对应 `/api/companion/state`、`/inbox`、`/partners`、`/api/yanlili/clicks/stats` 均应成功。不要新建邀请、改备注或解绑真实关系。
5. 使用测试二维码卡登录；验证 `/api/auth/qr-login`、映射联查和 `/api/auth/me`。二维码文本不要写入命令历史或日志。
6. 用已绑定测试账号点按钮恰好 1 次，等待同步；统计总数应增加 1。这是计划内最小写入，用于验证 `POST /api/yanlili/clicks` 和统计读取。
7. 分别在 OLD/NEW 执行以下定向查询，把 `<SMOKE_USERNAME>` 只替换为非敏感测试用户名：

   ```sql
   SET SESSION time_zone = '+00:00';
   SELECT id, username, last_login_at, updated_at
   FROM users WHERE username = '<SMOKE_USERNAME>';

   SELECT COUNT(*) AS minute_rows, COALESCE(SUM(b.click_count), 0) AS total_clicks,
          MAX(b.minute_bucket) AS latest_minute
   FROM button_click_minutes b
   JOIN users u ON u.id = b.user_id
   WHERE u.username = '<SMOKE_USERNAME>' AND b.button_key = 'yanlili';
   ```

**成功判据**：health、登录、JWT、用户、绑定只读接口、二维码登录、点击写入和统计全部符合预期；广州的测试用户登录时间更新且点击合计增加 1，旧新加坡库保持 Step 5 基线不变。此时 OLD/NEW 全库验证因计划内新增广州写入而不再完全相同，这是正常现象。

**失败处理**：立即停止写入并记录广州是否已有新数据。若没有新写入走 Rollback A；若登录或点击已落广州，走 Rollback B，不能无脑切回。

## Step 12 Observe

**目的**：确认跨区域链路的可用性、延迟和备份满足小型生产服务要求。

**操作**：[需要人工在 Sealos 操作] 保留旧库但禁止应用访问，观察至少一个约定周期；监控应用重启、数据库连接错误、接口 5xx/延迟、广州连接数/CPU/存储、自动备份状态。执行一次广州手动备份，并在不影响生产的条件下验证平台恢复流程说明。

**成功判据**：应用稳定、延迟可接受、无持续 timeout/权限错误、自动和手动备份成功。

**失败处理**：按写入状态选择修复广州或 Rollback B；不要删除旧库。

## Step 13 Rollback if necessary

**目的**：在异常时恢复服务且不丢失切换后的真实业务数据。

**操作**：见下方“Rollback Runbook”。任何回滚都先停止 Singapore 应用写入。

**成功判据**：选定数据库成为唯一写入源，应用冒烟通过，另一数据库保持冻结并有备份。

**失败处理**：保持双库停写，保全两边 dump/备份和日志，人工制定增量合并方案。

## Step 14 Decommission old DB

**目的**：观察期结束后安全下线高成本 Singapore MySQL。

**操作**：[需要人工在 Sealos 操作] 仅当广州稳定、应用正常、Step 8 验证通过、Step 11/12 通过、广州备份成功且观察期结束后：

1. 再对 Singapore 旧库创建最后一次手动备份。
2. 记录备份 ID、保留策略和最终 UTC 时间，确认广州仍是唯一 DSN。
3. 人工删除 Singapore MySQL；本工具绝不自动删除生产数据库。
4. 本地 dump 到期后再按组织策略安全删除。删除前确认其哈希和平台备份仍按策略保留。

**成功判据**：最后备份成功、生产仍连接广州、旧实例已按审批删除且费用停止。

**失败处理**：取消删除，继续保留旧实例和 dump；备份、观察期或审批任一条件不满足都不能下线。

## Rollback Runbook

### A. 刚切换，广州尚无新业务写入

1. [需要人工在 Sealos 操作] 将 Singapore App 缩容为 0，确认广州无登录、刷新、退出、点击或其他业务写入。
2. 把 `DATABASE_DSN` Secret 改回原 Singapore 地址和旧应用账号，重新部署 1 个副本。
3. 检查启动 Ping、`/healthz`、登录、JWT、绑定只读接口、统计和二维码映射，再恢复副本数。
4. 广州实例保持不写并保留，调查根因；不要删除 dump 或任一数据库。

成功判据：旧库重新成为唯一写入源，应用正常，两库仍可审计。失败时保持停写并修复连接/权限，不能让两个实例同时写。

### B. 广州已经产生新业务数据

1. [需要人工在 Sealos 操作] 立即将 Singapore App 缩容为 0，冻结广州；分别创建 OLD/NEW 平台备份和完整 dump。
2. 记录切换 UTC 时间、Smoke Test 写入及业务写入范围；对比 `users.last_login_at/updated_at`、`button_click_minutes`、绑定表、二维码映射、会话和刷新凭证哈希的增量。
3. 优先修复广州并继续以广州为唯一源。若确实必须回旧库，先设计、审阅并演练针对实际差异的增量同步；由于分钟桶使用 Upsert、绑定有事务状态机，不能用整表覆盖或盲目重放 SQL。
4. 增量同步完成并重新验证后，才允许把 DSN 切回旧库并做完整冒烟。

成功判据：所有广州新增真实数据已被明确保留/同步，且任何时刻只有一个写入源。无法证明时继续停写并人工处理，绝不直接切回旧库。

## Secret 与文件收尾检查

```bash
git status --short
git ls-files '.env*' '*.sql' '*.sql.gz' '*.dump'
git check-ignore -v .env db-migration-dumps/fluffy_cupcake_backup.sql
```

`git ls-files '*.sql'` 会正常列出仓库受版本管理的 `migrations/*.sql`；不得出现 `db-migration-dumps/` 下的 dump。若 Secret 曾被误加到 Git，仅删除工作树文件不够，应立即轮换凭据并按仓库流程清理历史。
