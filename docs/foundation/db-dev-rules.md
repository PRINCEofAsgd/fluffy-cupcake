# 数据库开发规范

> 本规范适用于基于 **Gin + MySQL** 的后端项目。目标是保持数据库设计、访问和版本管理清晰可靠，不过度设计。

## 1. 技术选型

* 数据库：MySQL 8.x
* 数据库访问：`database/sql` 或 `sqlx`
* MySQL Driver：`go-sql-driver/mysql`
* Migration：`golang-migrate/migrate`
* 业务 SQL 优先显式编写，避免过度依赖 ORM 自动生成 SQL。

## 2. 项目结构

```text
internal/
├── database/
│   └── mysql.go           # MySQL 连接与连接池初始化
├── model/                 # 数据模型
├── repository/            # 数据访问层
├── service/               # 业务逻辑
└── handler/               # HTTP 接口

migrations/
├── 000001_init.up.sql
├── 000001_init.down.sql
├── 000002_xxx.up.sql
└── 000002_xxx.down.sql
```

调用关系保持：

```text
Handler → Service → Repository → MySQL
```

Handler 和 Service 不直接编写 SQL。

## 3. MySQL 连接管理

统一在 `internal/database` 中初始化数据库连接。

* 全局复用 `*sql.DB` / `*sqlx.DB` 连接池，不为每次请求创建连接。
* DSN、用户名、密码等信息通过配置文件或环境变量提供，不硬编码。
* 根据部署环境合理配置：

  * `MaxOpenConns`
  * `MaxIdleConns`
  * `ConnMaxLifetime`
* 服务启动时执行 `Ping` 验证数据库可用性。

## 4. Repository 规范

所有数据库访问统一封装在 Repository 层，例如：

```text
UserRepository
├── Create()
├── GetByID()
├── GetByUsername()
├── Update()
└── Delete()
```

要求：

* Repository 负责 SQL 和数据库访问。
* Service 负责业务逻辑和事务流程。
* Handler 负责 HTTP 参数解析和响应。
* 禁止在 Handler 中直接执行 SQL。

## 5. SQL 规范

### 查询

禁止：

```sql
SELECT * FROM users;
```

推荐：

```sql
SELECT id, username, created_at
FROM users
WHERE id = ?;
```

明确指定需要的字段。

### 参数

禁止字符串拼接用户输入：

```go
"SELECT * FROM users WHERE id = " + id
```

必须使用参数化查询：

```sql
SELECT id, username
FROM users
WHERE id = ?;
```

避免 SQL 注入。

### 更新和删除

`UPDATE`、`DELETE` 必须明确指定 `WHERE` 条件。

执行后根据业务需要检查 `RowsAffected()`，避免把“没有更新任何数据”误认为操作成功。

## 6. 表设计规范

推荐：

* 表名、字段名统一使用 `snake_case`。
* 主键统一使用 `id`。
* 时间字段统一使用：

  * `created_at`
  * `updated_at`
* 根据业务语义合理设置 `NOT NULL`。
* 不依赖默认值表达重要业务含义。
* 外键是否使用数据库约束根据项目规模决定，但数据关系必须在代码和文档中明确。

示例：

```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP
);
```

## 7. 索引规范

索引根据实际查询场景设计，而不是为所有字段建立索引。

重点关注：

* 高频 `WHERE` 条件
* `JOIN` 字段
* 排序、分页字段
* 唯一性约束
* 联合索引的最左前缀

新增重要查询后，应使用：

```sql
EXPLAIN
```

检查是否正确使用索引。

避免：

* 重复索引
* 无意义的单列索引
* 在低区分度字段上盲目建立索引

## 8. Transaction 规范

涉及多个必须同时成功或失败的数据库操作时使用事务。

基本流程：

```text
Begin
  ↓
执行操作 A
  ↓
执行操作 B
  ↓
成功 → Commit
失败 → Rollback
```

事务边界由 Service 层的业务逻辑决定。

原则：

* 事务尽可能短。
* 事务中避免执行耗时的网络请求。
* 所有异常路径必须能够正确 Rollback。
* 注意并发更新、锁竞争和死锁问题。

## 9. Migration 规范

数据库 Schema 变更必须通过 Migration 管理，不手工修改生产数据库后不留记录。

目录：

```text
migrations/
├── 000001_init.up.sql
├── 000001_init.down.sql
├── 000002_add_users.up.sql
├── 000002_add_users.down.sql
└── ...
```

每次 Schema 修改创建新的 Migration：

```text
V1 → 初始化数据库
V2 → 新增 users 表
V3 → 新增字段
V4 → 新增索引
```

禁止修改已经在其他环境执行过的历史 Migration。

### Up

描述数据库如何升级：

```sql
ALTER TABLE users
ADD COLUMN email VARCHAR(255);
```

### Down

描述如何回滚：

```sql
ALTER TABLE users
DROP COLUMN email;
```

Migration 应独立于 Gin 服务运行：

```text
migrate up
        ↓
更新数据库 Schema
        ↓
启动 Gin Server
```

不建议在 Gin 启动时自动执行 `CREATE TABLE IF NOT EXISTS` 代替 Migration。

## 10. 配置与安全

数据库密码等敏感配置禁止：

* 硬编码到 Go 源码。
* 提交到 Git 仓库。
* 写入公开配置文件。

本地开发推荐：

```text
.env
```

生产环境使用环境变量或部署平台 Secret。

`.env` 加入：

```text
.gitignore
```

## 11. 开发流程

数据库相关功能按照以下流程开发：

```text
设计数据模型
    ↓
创建 Migration
    ↓
执行 migrate up
    ↓
检查数据库 Schema
    ↓
编写 Repository
    ↓
编写 Service
    ↓
接入 Handler
    ↓
接口测试
    ↓
检查实际数据库数据
    ↓
必要时使用 EXPLAIN 分析 SQL
```

核心原则：

> **Migration 管结构，Repository 管数据访问，Service 管业务与事务，Gin Handler 管 HTTP。**
