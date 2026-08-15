package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

// UserRepository 负责 users 表的显式字段查询与写入。
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository 创建用户数据访问对象。
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 写入已由上层完成 bcrypt 处理的用户，并返回新用户 ID。
func (r *UserRepository) Create(ctx context.Context, username, passwordHash string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO users (username, password_hash)
		VALUES (?, ?)
	`, username, passwordHash)
	if err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return 0, ErrUsernameExists
		}
		return 0, fmt.Errorf("创建用户: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("读取新用户 ID: %w", err)
	}
	return id, nil
}

// GetByUsername 按唯一用户名读取鉴权所需的完整用户字段。
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (model.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, last_login_at, created_at, updated_at
		FROM users
		WHERE username = ?
	`, username))
}

// GetByID 按可信 Token 中的用户 ID 读取当前用户。
func (r *UserRepository) GetByID(ctx context.Context, id int64) (model.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, last_login_at, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id))
}

// UpdateLastLogin 在密码校验成功后记录服务端 UTC 登录时间，供 30 天未登录直接解绑判断。
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id int64, loginAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE users SET last_login_at = ? WHERE id = ?`, loginAt.UTC(), id)
	if err != nil {
		return fmt.Errorf("更新最后登录时间: %w", err)
	}
	if _, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("读取最后登录更新结果: %w", err)
	}
	return nil
}

// scanUser 统一处理用户扫描与未找到错误映射。
func (r *UserRepository) scanUser(row *sql.Row) (model.User, error) {
	var user model.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("查询用户: %w", err)
	}
	return user, nil
}
