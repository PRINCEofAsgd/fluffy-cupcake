// auth_session.go 管理登录会话和刷新凭证的 MySQL 持久化、事务与锁。
package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
)

// ErrRefreshConflict 表示刷新凭证刚被并发请求轮换。
var ErrRefreshConflict = errors.New("刷新凭证已被并发请求轮换")

// AuthSessionRepository 保存会话和刷新凭证哈希。
type AuthSessionRepository struct {
	db *sql.DB
}

// NewAuthSessionRepository 创建会话数据访问对象。
func NewAuthSessionRepository(db *sql.DB) *AuthSessionRepository {
	return &AuthSessionRepository{db: db}
}

// Create 原子写入会话、首个凭证哈希和成功登录时间。
func (r *AuthSessionRepository) Create(ctx context.Context, session model.AuthSession, hash [sha256.Size]byte, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始登录事务: %w", err)
	}
	defer tx.Rollback()
	// 同一用户并发登录时，较早签发的会话不能把登录时间倒退。
	if _, err := tx.ExecContext(ctx, `
		UPDATE users SET last_login_at = ?
		WHERE id = ? AND (last_login_at IS NULL OR last_login_at < ?)
	`, now, session.UserID, now); err != nil {
		return fmt.Errorf("记录最后登录时间: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO auth_sessions (id, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?)
	`, session.ID, session.UserID, now, session.ExpiresAt); err != nil {
		return fmt.Errorf("创建登录会话: %w", err)
	}
	if err := insertRefreshHash(ctx, tx, session.ID, hash); err != nil {
		return err
	}
	return tx.Commit()
}

// Active 检查已签访问 JWT 所绑定的会话是否仍可使用。
func (r *AuthSessionRepository) Active(ctx context.Context, id string, userID int64, now time.Time) (bool, error) {
	var active int
	err := r.db.QueryRowContext(ctx, `
		SELECT 1 FROM auth_sessions
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?
	`, id, userID, now).Scan(&active)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// Rotate 通过同一会话行串行化刷新，提交轮换或重用导致的撤销。
func (r *AuthSessionRepository) Rotate(ctx context.Context, oldHash, newHash [sha256.Size]byte, clock func() time.Time, grace time.Duration) (model.AuthSession, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.AuthSession{}, fmt.Errorf("开始刷新事务: %w", err)
	}
	defer tx.Rollback()

	id, err := findRefreshSession(ctx, tx, oldHash)
	if err != nil {
		return model.AuthSession{}, err
	}
	session := model.AuthSession{ID: id}
	var revokedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT user_id, expires_at, revoked_at FROM auth_sessions WHERE id = ? FOR UPDATE
	`, id).Scan(&session.UserID, &session.ExpiresAt, &revokedAt)
	if err != nil {
		return model.AuthSession{}, sessionQueryError(err)
	}
	now := clock().UTC().Truncate(time.Microsecond)
	if revokedAt.Valid || !now.Before(session.ExpiresAt) {
		return model.AuthSession{}, ErrNotFound
	}

	// 使用锁定读取得最新状态，避免 REPEATABLE READ 复用查找会话时的旧快照。
	var rotatedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT rotated_at FROM auth_refresh_tokens WHERE token_hash = ? FOR UPDATE
	`, oldHash[:]).Scan(&rotatedAt)
	if err != nil {
		return model.AuthSession{}, sessionQueryError(err)
	}
	if rotatedAt.Valid {
		if now.Before(rotatedAt.Time.Add(grace)) {
			return model.AuthSession{}, ErrRefreshConflict
		}
		if err := revokeSession(ctx, tx, id, now); err != nil {
			return model.AuthSession{}, err
		}
		if err := tx.Commit(); err != nil {
			return model.AuthSession{}, err
		}
		return model.AuthSession{}, ErrNotFound
	}

	if err := tx.QueryRowContext(ctx, `SELECT username FROM users WHERE id = ?`, session.UserID).Scan(&session.Username); err != nil {
		return model.AuthSession{}, sessionQueryError(err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE auth_refresh_tokens SET rotated_at = ? WHERE token_hash = ?
	`, now, oldHash[:]); err != nil {
		return model.AuthSession{}, fmt.Errorf("标记旧刷新凭证: %w", err)
	}
	if err := insertRefreshHash(ctx, tx, id, newHash); err != nil {
		return model.AuthSession{}, err
	}
	return session, tx.Commit()
}

// RevokeByRefresh 撤销当前或历史刷新哈希所属会话，支持刷新与退出同时发生。
func (r *AuthSessionRepository) RevokeByRefresh(ctx context.Context, hash [sha256.Size]byte, now time.Time) error {
	id, err := findRefreshSession(ctx, r.db, hash)
	if err != nil {
		return err
	}
	return r.Revoke(ctx, id, now)
}

// Revoke 按已验证的会话 ID 撤销登录会话，重复调用保持幂等。
func (r *AuthSessionRepository) Revoke(ctx context.Context, id string, now time.Time) error {
	return revokeSession(ctx, r.db, id, now)
}

// findRefreshSession 按哈希定位会话，数据库中没有刷新凭证明文。
func findRefreshSession(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, hash [sha256.Size]byte) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `SELECT session_id FROM auth_refresh_tokens WHERE token_hash = ?`, hash[:]).Scan(&id)
	return id, sessionQueryError(err)
}

// insertRefreshHash 保存新一代刷新凭证的哈希。
func insertRefreshHash(ctx context.Context, tx *sql.Tx, id string, hash [sha256.Size]byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO auth_refresh_tokens (token_hash, session_id) VALUES (?, ?)`, hash[:], id)
	if err != nil {
		return fmt.Errorf("保存刷新凭证哈希: %w", err)
	}
	return nil
}

// revokeSession 记录首次撤销时间，UPDATE 与刷新持有的会话行锁互斥。
func revokeSession(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, id string, now time.Time) error {
	_, err := db.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, now, id)
	if err != nil {
		return fmt.Errorf("撤销登录会话: %w", err)
	}
	return nil
}

// sessionQueryError 区分找不到会话与数据库暂时不可用。
func sessionQueryError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("查询登录会话: %w", err)
	}
	return nil
}
