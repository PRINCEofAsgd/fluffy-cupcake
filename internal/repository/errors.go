// Package repository 封装所有 MySQL SQL 与数据访问错误语义。
package repository

import "errors"

var (
	// ErrNotFound 表示按业务唯一条件没有查到记录。
	ErrNotFound = errors.New("记录不存在")
	// ErrUsernameExists 表示创建用户时用户名违反唯一约束。
	ErrUsernameExists = errors.New("用户名已存在")
	// ErrAlreadyBound 表示当前用户或目标用户已经有活跃陪伴绑定。
	ErrAlreadyBound = errors.New("已有陪伴绑定")
	// ErrInvalidBindingState 表示信件状态不允许当前操作。
	ErrInvalidBindingState = errors.New("当前绑定状态不允许该操作")
	// ErrForbidden 表示当前用户不是该信件允许操作的一方。
	ErrForbidden = errors.New("无权操作该绑定信件")
	// ErrPartnerRecentlyActive 表示对方最近 30 天内登录过，不能直接解绑。
	ErrPartnerRecentlyActive = errors.New("对方最近 30 天内登录过")
)
