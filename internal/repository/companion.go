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

// CompanionRepository 负责不可删除绑定信件、当前绑定占位和解绑状态的显式 SQL。
type CompanionRepository struct {
	db *sql.DB
}

// NewCompanionRepository 创建陪伴绑定数据访问对象。
func NewCompanionRepository(db *sql.DB) *CompanionRepository {
	return &CompanionRepository{db: db}
}

// GetActiveState 读取用户当前唯一活跃绑定及其对对方的私有备注。
func (r *CompanionRepository) GetActiveState(ctx context.Context, userID int64) (model.CompanionState, error) {
	var state model.CompanionState
	err := r.db.QueryRowContext(ctx, `
		SELECT membership.binding_id, membership.partner_user_id, partner.username,
			CASE WHEN binding.inviter_id = ? THEN binding.inviter_note ELSE binding.invitee_note END
		FROM companion_active_memberships AS membership
		JOIN companion_bindings AS binding ON binding.id = membership.binding_id AND binding.status = 'active'
		JOIN users AS partner ON partner.id = membership.partner_user_id
		WHERE membership.user_id = ?
	`, userID, userID).Scan(&state.BindingID, &state.PartnerID, &state.PartnerUsername, &state.PartnerNote)
	if errors.Is(err, sql.ErrNoRows) {
		return model.CompanionState{Bound: false}, nil
	}
	if err != nil {
		return model.CompanionState{}, fmt.Errorf("查询当前陪伴绑定: %w", err)
	}
	state.Bound = true
	return state, nil
}

// CreateInvitation 在锁定双方用户后写入待接受信件，避免与并发接受形成越界绑定。
func (r *CompanionRepository) CreateInvitation(ctx context.Context, inviterID, inviteeID int64, note string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("开始发起绑定事务: %w", err)
	}
	defer tx.Rollback()
	if err := lockUsers(ctx, tx, inviterID, inviteeID); err != nil {
		return 0, err
	}
	bound, err := activeMembershipCount(ctx, tx, inviterID, inviteeID)
	if err != nil {
		return 0, err
	}
	if bound > 0 {
		return 0, ErrAlreadyBound
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO companion_bindings (inviter_id, invitee_id, inviter_note, status)
		VALUES (?, ?, ?, 'pending')
	`, inviterID, inviteeID, note)
	if err != nil {
		return 0, fmt.Errorf("创建绑定邀请信件: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("读取绑定邀请 ID: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交绑定邀请: %w", err)
	}
	return id, nil
}

// AcceptInvitation 原子建立双方占位，并将其他受影响的旧待处理信件标记为已失效。
func (r *CompanionRepository) AcceptInvitation(ctx context.Context, bindingID, inviteeID int64, acceptedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始接受绑定事务: %w", err)
	}
	defer tx.Rollback()

	var inviterID, actualInviteeID int64
	var status string
	if err := tx.QueryRowContext(ctx, `
		SELECT inviter_id, invitee_id, status
		FROM companion_bindings
		WHERE id = ?
		FOR UPDATE
	`, bindingID).Scan(&inviterID, &actualInviteeID, &status); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("锁定绑定邀请: %w", err)
	}
	if actualInviteeID != inviteeID {
		return ErrForbidden
	}
	if status != "pending" {
		return ErrInvalidBindingState
	}
	if err := lockUsers(ctx, tx, inviterID, inviteeID); err != nil {
		return err
	}
	bound, err := activeMembershipCount(ctx, tx, inviterID, inviteeID)
	if err != nil {
		return err
	}
	if bound > 0 {
		return ErrAlreadyBound
	}
	for _, membership := range [][2]int64{{inviterID, inviteeID}, {inviteeID, inviterID}} {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO companion_active_memberships (user_id, binding_id, partner_user_id)
			VALUES (?, ?, ?)
		`, membership[0], bindingID, membership[1]); err != nil {
			var mysqlErr *mysqlDriver.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				return ErrAlreadyBound
			}
			return fmt.Errorf("创建当前绑定占位: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE companion_bindings
		SET status = 'active', accepted_at = ?
		WHERE id = ? AND status = 'pending'
	`, acceptedAt.UTC(), bindingID); err != nil {
		return fmt.Errorf("激活绑定信件: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE companion_bindings
		SET status = 'superseded'
		WHERE status = 'pending' AND id <> ?
		  AND (inviter_id IN (?, ?) OR invitee_id IN (?, ?))
	`, bindingID, inviterID, inviteeID, inviterID, inviteeID); err != nil {
		return fmt.Errorf("关闭冲突绑定邀请: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交接受绑定: %w", err)
	}
	return nil
}

// ListInbox 返回用户作为任一方参与的全部信件；业务没有删除接口。
func (r *CompanionRepository) ListInbox(ctx context.Context, userID int64) ([]model.CompanionBinding, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT binding.id, binding.inviter_id, inviter.username, binding.invitee_id, invitee.username,
			binding.inviter_note, binding.invitee_note, binding.status, binding.accepted_at,
			binding.unbind_requested_by, binding.unbind_requested_at, binding.ended_at,
			binding.ended_by, binding.ended_reason, binding.created_at, binding.updated_at
		FROM companion_bindings AS binding
		JOIN users AS inviter ON inviter.id = binding.inviter_id
		JOIN users AS invitee ON invitee.id = binding.invitee_id
		WHERE binding.inviter_id = ? OR binding.invitee_id = ?
		ORDER BY binding.created_at DESC, binding.id DESC
	`, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("查询绑定收件箱: %w", err)
	}
	defer rows.Close()
	bindings := make([]model.CompanionBinding, 0)
	for rows.Next() {
		binding, err := scanBinding(rows)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历绑定收件箱: %w", err)
	}
	return bindings, nil
}

// UpdateNote 仅允许活跃绑定的一方修改自己看向对方的备注。
func (r *CompanionRepository) UpdateNote(ctx context.Context, bindingID, userID int64, note string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE companion_bindings AS binding
		JOIN companion_active_memberships AS membership
		  ON membership.binding_id = binding.id AND membership.user_id = ?
		SET binding.inviter_note = CASE WHEN binding.inviter_id = ? THEN ? ELSE binding.inviter_note END,
			binding.invitee_note = CASE WHEN binding.invitee_id = ? THEN ? ELSE binding.invitee_note END
		WHERE binding.id = ? AND binding.status = 'active'
	`, userID, userID, note, userID, note, bindingID)
	if err != nil {
		return fmt.Errorf("修改绑定备注: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取备注修改结果: %w", err)
	}
	if changed == 0 {
		var exists int
		if err := r.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM companion_active_memberships
			WHERE user_id = ? AND binding_id = ?
		`, userID, bindingID).Scan(&exists); err != nil {
			return fmt.Errorf("核对备注绑定状态: %w", err)
		}
		if exists == 0 {
			return ErrInvalidBindingState
		}
	}
	return nil
}

// RequestUnbind 在原绑定信件中记录发起人；同一时刻只存在一个未决解绑邀请。
func (r *CompanionRepository) RequestUnbind(ctx context.Context, bindingID, userID int64, requestedAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE companion_bindings AS binding
		JOIN companion_active_memberships AS membership
		  ON membership.binding_id = binding.id AND membership.user_id = ?
		SET binding.unbind_requested_by = ?, binding.unbind_requested_at = ?
		WHERE binding.id = ? AND binding.status = 'active' AND binding.unbind_requested_by IS NULL
	`, userID, userID, requestedAt.UTC(), bindingID)
	if err != nil {
		return fmt.Errorf("发起解绑邀请: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取解绑邀请结果: %w", err)
	}
	if changed == 0 {
		return ErrInvalidBindingState
	}
	return nil
}

// AcceptUnbind 只允许另一方接受未决邀请，并原子结束绑定与删除当前占位。
func (r *CompanionRepository) AcceptUnbind(ctx context.Context, bindingID, userID int64, endedAt time.Time) error {
	return r.endBinding(ctx, bindingID, userID, endedAt)
}

// DirectUnbind 在对方最后登录早于截止时间时允许单方直接结束绑定。
func (r *CompanionRepository) DirectUnbind(ctx context.Context, bindingID, userID int64, endedAt, inactiveCutoff time.Time) error {
	return r.endBindingWithInactivity(ctx, bindingID, userID, endedAt, inactiveCutoff)
}

// ListPartners 返回历次已接受绑定的用户，供详细记录选择器使用。
func (r *CompanionRepository) ListPartners(ctx context.Context, userID int64) ([]model.CompanionPartner, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT partner.id, partner.username, MAX(binding.accepted_at) AS last_accepted_at
		FROM companion_bindings AS binding
		JOIN users AS partner
		  ON partner.id = CASE WHEN binding.inviter_id = ? THEN binding.invitee_id ELSE binding.inviter_id END
		WHERE (binding.inviter_id = ? OR binding.invitee_id = ?) AND binding.accepted_at IS NOT NULL
		GROUP BY partner.id, partner.username
		ORDER BY last_accepted_at DESC, partner.id DESC
	`, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("查询历史绑定对象: %w", err)
	}
	defer rows.Close()
	partners := make([]model.CompanionPartner, 0)
	for rows.Next() {
		var partner model.CompanionPartner
		if err := rows.Scan(&partner.ID, &partner.Username, &partner.LastAcceptedAt); err != nil {
			return nil, fmt.Errorf("读取历史绑定对象: %w", err)
		}
		partner.LastAcceptedAt = partner.LastAcceptedAt.UTC()
		partners = append(partners, partner)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历历史绑定对象: %w", err)
	}
	return partners, nil
}

// endBinding 校验当前用户是解绑邀请的另一方，再以双方确认原因结束关系。
func (r *CompanionRepository) endBinding(ctx context.Context, bindingID, userID int64, endedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始接受解绑事务: %w", err)
	}
	defer tx.Rollback()
	var inviterID, inviteeID int64
	var status string
	var requester sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT inviter_id, invitee_id, status, unbind_requested_by FROM companion_bindings WHERE id = ? FOR UPDATE`, bindingID).
		Scan(&inviterID, &inviteeID, &status, &requester); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("锁定解绑信件: %w", err)
	}
	if userID != inviterID && userID != inviteeID {
		return ErrForbidden
	}
	if status != "active" || !requester.Valid || requester.Int64 == userID {
		return ErrInvalidBindingState
	}
	if err := finishBinding(ctx, tx, bindingID, userID, endedAt, "mutual"); err != nil {
		return err
	}
	return tx.Commit()
}

// endBindingWithInactivity 锁定信件和对方用户，以最后登录截止时间决定是否允许直接解绑。
func (r *CompanionRepository) endBindingWithInactivity(ctx context.Context, bindingID, userID int64, endedAt, inactiveCutoff time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始直接解绑事务: %w", err)
	}
	defer tx.Rollback()
	var inviterID, inviteeID int64
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT inviter_id, invitee_id, status FROM companion_bindings WHERE id = ? FOR UPDATE`, bindingID).
		Scan(&inviterID, &inviteeID, &status); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("锁定直接解绑信件: %w", err)
	}
	if userID != inviterID && userID != inviteeID {
		return ErrForbidden
	}
	if status != "active" {
		return ErrInvalidBindingState
	}
	partnerID := inviterID
	if userID == inviterID {
		partnerID = inviteeID
	}
	var lastSeen time.Time
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(last_login_at, created_at) FROM users WHERE id = ? FOR UPDATE`, partnerID).Scan(&lastSeen); err != nil {
		return fmt.Errorf("查询对方最后登录时间: %w", err)
	}
	if lastSeen.After(inactiveCutoff.UTC()) {
		return ErrPartnerRecentlyActive
	}
	if err := finishBinding(ctx, tx, bindingID, userID, endedAt, "inactive"); err != nil {
		return err
	}
	return tx.Commit()
}

// finishBinding 在同一事务中释放双方当前占位并完成信件结束字段。
func finishBinding(ctx context.Context, tx *sql.Tx, bindingID, endedBy int64, endedAt time.Time, reason string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM companion_active_memberships WHERE binding_id = ?`, bindingID); err != nil {
		return fmt.Errorf("释放当前绑定占位: %w", err)
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE companion_bindings
		SET status = 'ended', ended_at = ?, ended_by = ?, ended_reason = ?
		WHERE id = ? AND status = 'active'
	`, endedAt.UTC(), endedBy, reason, bindingID)
	if err != nil {
		return fmt.Errorf("结束绑定信件: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return ErrInvalidBindingState
	}
	return nil
}

// lockUsers 按主键顺序锁定双方用户，统一并发操作的加锁顺序。
func lockUsers(ctx context.Context, tx *sql.Tx, firstID, secondID int64) error {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM users WHERE id IN (?, ?) ORDER BY id FOR UPDATE`, firstID, secondID)
	if err != nil {
		return fmt.Errorf("锁定绑定用户: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历绑定用户锁: %w", err)
	}
	if count != 2 {
		return ErrNotFound
	}
	return nil
}

// activeMembershipCount 检查双方是否已有任意当前绑定占位。
func activeMembershipCount(ctx context.Context, tx *sql.Tx, firstID, secondID int64) (int, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM companion_active_memberships WHERE user_id IN (?, ?)`, firstID, secondID).Scan(&count); err != nil {
		return 0, fmt.Errorf("检查当前绑定占位: %w", err)
	}
	return count, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

// scanBinding 统一扫描收件箱信件的全部生命周期字段。
func scanBinding(row rowScanner) (model.CompanionBinding, error) {
	var binding model.CompanionBinding
	if err := row.Scan(
		&binding.ID, &binding.InviterID, &binding.InviterUsername, &binding.InviteeID, &binding.InviteeUsername,
		&binding.InviterNote, &binding.InviteeNote, &binding.Status, &binding.AcceptedAt,
		&binding.UnbindRequestedBy, &binding.UnbindRequestedAt, &binding.EndedAt,
		&binding.EndedBy, &binding.EndedReason, &binding.CreatedAt, &binding.UpdatedAt,
	); err != nil {
		return model.CompanionBinding{}, fmt.Errorf("读取绑定信件: %w", err)
	}
	return binding, nil
}
