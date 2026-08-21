package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

// QrLoginRepository 负责 qr_login_mappings 表的显式 SQL 查询与写入。
type QrLoginRepository struct {
	db *sql.DB
}

// NewQrLoginRepository 创建二维码登录映射数据访问对象。
func NewQrLoginRepository(db *sql.DB) *QrLoginRepository {
	return &QrLoginRepository{db: db}
}

// ErrQrTextExists 表示二维码文本已映射到某个用户，不允许重复发卡。
var ErrQrTextExists = errors.New("二维码文本已存在")

// GetUserByQrText 按二维码永久文本联查映射与用户，返回可签 Token 的用户信息。
func (r *QrLoginRepository) GetUserByQrText(ctx context.Context, qrText string) (model.User, error) {
	var user model.User
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.password_hash, u.last_login_at, u.created_at, u.updated_at
		FROM qr_login_mappings m
		JOIN users u ON u.id = m.user_id
		WHERE m.qr_text = ?
	`, qrText).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("查询二维码登录用户: %w", err)
	}
	return user, nil
}

// Create 写入一条文本到真实注册用户的映射，重复文本返回 ErrQrTextExists。
func (r *QrLoginRepository) Create(ctx context.Context, qrText string, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO qr_login_mappings (qr_text, user_id)
		VALUES (?, ?)
	`, qrText, userID)
	if err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return ErrQrTextExists
		}
		return fmt.Errorf("创建二维码登录映射: %w", err)
	}
	return nil
}

// Delete 删除一条映射；文本不存在时返回 ErrNotFound。
func (r *QrLoginRepository) Delete(ctx context.Context, qrText string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM qr_login_mappings WHERE qr_text = ?`, qrText)
	if err != nil {
		return fmt.Errorf("删除二维码登录映射: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取删除结果: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
