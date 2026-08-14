// Package repository 封装所有 MySQL SQL 与数据访问错误语义。
package repository

import "errors"

var (
	// ErrNotFound 表示按业务唯一条件没有查到记录。
	ErrNotFound = errors.New("记录不存在")
	// ErrUsernameExists 表示创建用户时用户名违反唯一约束。
	ErrUsernameExists = errors.New("用户名已存在")
)
