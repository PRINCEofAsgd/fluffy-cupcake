package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
)

// CompanionMoodRepository 查询当前绑定中的心情并按关系和填写者更新快照。
type CompanionMoodRepository struct {
	db *sql.DB
}

// NewCompanionMoodRepository 创建心情快照数据访问对象。
func NewCompanionMoodRepository(db *sql.DB) *CompanionMoodRepository {
	return &CompanionMoodRepository{db: db}
}

// GetCurrent 从当前绑定占位选择本人或对方，未填写时返回同一关系下的空快照。
func (r *CompanionMoodRepository) GetCurrent(ctx context.Context, userID int64, theirs bool) (model.CompanionMoodView, error) {
	view := model.CompanionMoodView{CompanionMood: model.CompanionMood{
		Statuses: []string{}, Reasons: []string{}, Comforts: []string{},
	}}
	var content []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT membership.binding_id, mood.content
		FROM companion_active_memberships AS membership
		JOIN companion_bindings AS binding ON binding.id = membership.binding_id AND binding.status = 'active'
		LEFT JOIN companion_moods AS mood ON mood.binding_id = membership.binding_id
			AND mood.author_user_id = CASE WHEN ? THEN membership.partner_user_id ELSE membership.user_id END
		WHERE membership.user_id = ?
	`, theirs, userID).Scan(&view.BindingID, &content)

	if errors.Is(err, sql.ErrNoRows) {
		return model.CompanionMoodView{}, ErrInvalidBindingState
	}
	if err != nil {
		return model.CompanionMoodView{}, fmt.Errorf("读取当前心情: %w", err)
	}
	if content != nil {
		if err := json.Unmarshal(content, &view.CompanionMood); err != nil {
			return model.CompanionMoodView{}, fmt.Errorf("解析已保存心情: %w", err)
		}
	}

	return view, nil
}

// SaveCurrent 锁定当前有效的绑定再更新登录用户自己的快照，与解绑操作串行。
func (r *CompanionMoodRepository) SaveCurrent(ctx context.Context, userID int64, view model.CompanionMoodView) error {
	content, err := json.Marshal(view.CompanionMood)
	if err != nil {
		return fmt.Errorf("编码心情快照: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始保存心情事务: %w", err)
	}
	defer tx.Rollback()

	// 客户端携带的绑定 ID 只用于校验编辑时看到的绑定是否仍然有效。
	var bindingID int64
	err = tx.QueryRowContext(ctx, `
		SELECT binding.id
		FROM companion_bindings AS binding
		WHERE binding.id = ? AND binding.status = 'active'
			AND EXISTS (
				SELECT 1 FROM companion_active_memberships AS membership
				WHERE membership.user_id = ? AND membership.binding_id = binding.id
			)
		FOR UPDATE
	`, view.BindingID, userID).Scan(&bindingID)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidBindingState
	}
	if err != nil {
		return fmt.Errorf("锁定心情所属绑定: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO companion_moods (binding_id, author_user_id, content)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE content = ?
	`, bindingID, userID, string(content), string(content)); err != nil {
		return fmt.Errorf("更新本人心情: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交心情快照: %w", err)
	}

	return nil
}
