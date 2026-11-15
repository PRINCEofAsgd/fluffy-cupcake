// auth_session.go 定义登录会话的持久化数据。
package model

import "time"

// AuthSession 记录一个浏览器会话的身份和固定 UTC 截止时间。
type AuthSession struct {
	ID        string
	UserID    int64
	Username  string
	ExpiresAt time.Time
}
