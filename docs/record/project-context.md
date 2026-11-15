# 项目工作上下文

## 已归档阶段

- 初版上线阶段（2026-08-14—2026-08-22，Step 1—5）已归档至 [`context-archives/initial-release-stage.md`](context-archives/initial-release-stage.md)。
- Homepage 开发与规范化部署阶段（2026-08-30—2026-09-01，Step 6—8）已归档至 [`context-archives/homepage-development-and-standardized-deployment-stage.md`](context-archives/homepage-development-and-standardized-deployment-stage.md)，阶段收口版本为 `V1.1.0_20260901`。

## 当前阶段

### Step 9：15 天固定登录会话（2026-11-15）

- 密码和二维码登录共用会话创建事务。两种登录均更新 `last_login_at`，刷新和登出沿用原登录时间，30 天未登录解绑仍以最后一次成功登录为准。
- 默认会话有效期 15 天、访问 JWT 有效期 15 分钟。MySQL 保存会话及刷新哈希历史，事务轮换沿用固定截止。10 秒并发窗口内返回冲突，窗口外重用撤销会话；退出撤销后，尚未到期的访问 JWT 也无法通过后续鉴权。
- 页面启动和业务请求按需恢复会话。只有鉴权中间件提前拒绝的业务请求可在续期后重试一次。点击写入响应丢失时提示查看记录，避免不确定的重复写入。
- 通过 golang-migrate 在本地 MySQL 8.4 验证 `6 → 7 → 6 → 7`，通过 Go 全量竞态及真实 MySQL HTTP 集成测试。使用缩短的访问时长验证 Chrome 密码登录和相册扫码、页面启动恢复、同页及跨页并发、无 Web Locks 回退、网络故障、点击仅写一次及退出；用近截止和已截止的数据库样本验证固定期限。Compose 校验与本地 Linux/amd64 镜像冒烟通过，两种会话查询均使用 MySQL 主键。
- Web 版本更新为 `V1.1.0_20261115`。规范化 [局部更新部署目录](../user/incremental-deployments/README.md)，以 `V1.1.0_20260901` 为基线准备先迁移、后切换的手册。Sealos 生产迁移与发布待单独安排。

待办事项由 [todo-list.md](todo-list.md) 独立维护。
