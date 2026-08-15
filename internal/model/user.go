// Package model 定义跨 Repository、Service 和 Handler 使用的数据结构。
package model

import "time"

// User 表示固定登录用户；PasswordHash 仅供鉴权服务比较密码使用。
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
