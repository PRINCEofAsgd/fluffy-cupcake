package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
)

const (
	// MaxCompanionNoteLength 限制备注长度，数据库使用相同上限。
	MaxCompanionNoteLength = 64
	// DirectUnbindInactiveDays 是允许单方直接解绑的连续未登录天数。
	DirectUnbindInactiveDays = 30
)

var (
	// ErrCannotBindSelf 表示邀请目标或详细对象不能是当前用户自己。
	ErrCannotBindSelf = errors.New("不能绑定自己")
	// ErrCompanionNoteTooLong 表示备注超过数据库与页面共同限制。
	ErrCompanionNoteTooLong = errors.New("备注最多 64 个字符")
	// ErrNotBound 表示需要活跃双向关系的操作无法继续。
	ErrNotBound = errors.New("当前没有陪伴绑定")
	// ErrInvalidDirection 表示统计方向不属于公开枚举。
	ErrInvalidDirection = errors.New("direction 必须是 mine 或 theirs")
)

// CompanionStore 描述陪伴绑定业务需要的信件、当前占位和历史对象能力。
type CompanionStore interface {
	GetActiveState(ctx context.Context, userID int64) (model.CompanionState, error)
	CreateInvitation(ctx context.Context, inviterID, inviteeID int64, note string) (int64, error)
	AcceptInvitation(ctx context.Context, bindingID, inviteeID int64, acceptedAt time.Time) error
	ListInbox(ctx context.Context, userID int64) ([]model.CompanionBinding, error)
	UpdateNote(ctx context.Context, bindingID, userID int64, note string) error
	RequestUnbind(ctx context.Context, bindingID, userID int64, requestedAt, inactiveCutoff time.Time, confirmInactive bool) (model.UnbindAction, error)
	AcceptUnbind(ctx context.Context, bindingID, userID int64, endedAt time.Time) error
	CancelUnbind(ctx context.Context, bindingID, userID int64, respondedAt time.Time) error
	RejectUnbind(ctx context.Context, bindingID, userID int64, respondedAt time.Time) error
	ListPartners(ctx context.Context, userID int64) ([]model.CompanionPartner, error)
}

// CompanionUserStore 描述邀请用户名解析所需的用户查询能力。
type CompanionUserStore interface {
	GetByUsername(ctx context.Context, username string) (model.User, error)
}

// CompanionService 负责输入规则、可信当前用户和绑定生命周期业务语义。
type CompanionService struct {
	bindings CompanionStore
	users    CompanionUserStore
	now      func() time.Time
}

// NewCompanionService 创建陪伴绑定业务服务。
func NewCompanionService(bindings CompanionStore, users CompanionUserStore) *CompanionService {
	return &CompanionService{bindings: bindings, users: users, now: time.Now}
}

// State 返回当前绑定，不存在时返回 bound=false 而不是错误。
func (s *CompanionService) State(ctx context.Context, userID int64) (model.CompanionState, error) {
	if userID <= 0 {
		return model.CompanionState{}, ErrInvalidToken
	}
	return s.bindings.GetActiveState(ctx, userID)
}

// Invite 按用户名解析目标并发起一封不可删除的绑定邀请。
func (s *CompanionService) Invite(ctx context.Context, userID int64, username, note string) (int64, error) {
	username = strings.TrimSpace(username)
	note = strings.TrimSpace(note)
	if username == "" {
		return 0, repository.ErrNotFound
	}
	if utf8.RuneCountInString(note) > MaxCompanionNoteLength {
		return 0, ErrCompanionNoteTooLong
	}
	target, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	if target.ID == userID {
		return 0, ErrCannotBindSelf
	}
	return s.bindings.CreateInvitation(ctx, userID, target.ID, note)
}

// AcceptInvitation 只把可信当前用户作为收信人传给 Repository 原子接受。
func (s *CompanionService) AcceptInvitation(ctx context.Context, userID, bindingID int64) error {
	return s.bindings.AcceptInvitation(ctx, bindingID, userID, s.now().UTC())
}

// Inbox 返回当前用户参与的全部绑定信件。
func (s *CompanionService) Inbox(ctx context.Context, userID int64) ([]model.CompanionBinding, error) {
	return s.bindings.ListInbox(ctx, userID)
}

// UpdateNote 修改当前用户对绑定对象的备注；空字符串表示恢复显示用户名。
func (s *CompanionService) UpdateNote(ctx context.Context, userID, bindingID int64, note string) error {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > MaxCompanionNoteLength {
		return ErrCompanionNoteTooLong
	}
	return s.bindings.UpdateNote(ctx, bindingID, userID, note)
}

// RequestUnbind 统一判定普通申请与长期未登录直接解绑，后者必须收到前端额外确认标记。
func (s *CompanionService) RequestUnbind(ctx context.Context, userID, bindingID int64, confirmInactive bool) (model.UnbindAction, error) {
	now := s.now().UTC()
	return s.bindings.RequestUnbind(ctx, bindingID, userID, now, now.AddDate(0, 0, -DirectUnbindInactiveDays), confirmInactive)
}

// AcceptUnbind 接受另一方在同一信件中发起的解绑邀请。
func (s *CompanionService) AcceptUnbind(ctx context.Context, userID, bindingID int64) error {
	return s.bindings.AcceptUnbind(ctx, bindingID, userID, s.now().UTC())
}

// CancelUnbind 允许申请发起方取消仍处于待处理状态的解绑申请。
func (s *CompanionService) CancelUnbind(ctx context.Context, userID, bindingID int64) error {
	return s.bindings.CancelUnbind(ctx, bindingID, userID, s.now().UTC())
}

// RejectUnbind 允许申请接收方拒绝仍处于待处理状态的解绑申请。
func (s *CompanionService) RejectUnbind(ctx context.Context, userID, bindingID int64) error {
	return s.bindings.RejectUnbind(ctx, bindingID, userID, s.now().UTC())
}

// Partners 返回历次已接受绑定的对象列表。
func (s *CompanionService) Partners(ctx context.Context, userID int64) ([]model.CompanionPartner, error) {
	partners, err := s.bindings.ListPartners(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("读取历次绑定对象: %w", err)
	}
	return partners, nil
}
