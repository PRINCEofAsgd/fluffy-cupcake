package model

import "time"

// CompanionBinding 是一封不可删除的陪伴绑定信件及其完整生命周期。
type CompanionBinding struct {
	ID                int64      `json:"id"`
	InviterID         int64      `json:"inviter_id"`
	InviterUsername   string     `json:"inviter_username"`
	InviteeID         int64      `json:"invitee_id"`
	InviteeUsername   string     `json:"invitee_username"`
	InviterNote       string     `json:"-"`
	InviteeNote       string     `json:"-"`
	Status            string     `json:"status"`
	AcceptedAt        *time.Time `json:"accepted_at,omitempty"`
	UnbindRequestedBy *int64     `json:"unbind_requested_by,omitempty"`
	UnbindRequestedAt *time.Time `json:"unbind_requested_at,omitempty"`
	UnbindStatus      string     `json:"unbind_status"`
	UnbindRespondedBy *int64     `json:"unbind_responded_by,omitempty"`
	UnbindRespondedAt *time.Time `json:"unbind_responded_at,omitempty"`
	EndedAt           *time.Time `json:"ended_at,omitempty"`
	EndedBy           *int64     `json:"ended_by,omitempty"`
	EndedReason       *string    `json:"ended_reason,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// 解绑子状态独立于主绑定状态，取消或拒绝申请不会结束当前陪伴关系。
const (
	UnbindStatusNone      = "none"
	UnbindStatusPending   = "pending"
	UnbindStatusCancelled = "cancelled"
	UnbindStatusRejected  = "rejected"
	UnbindStatusAccepted  = "accepted"
	UnbindStatusDirect    = "direct"
)

// CompanionState 是页面判断绑定入口、点击宾语和当前信件的轻量状态。
type CompanionState struct {
	Bound           bool   `json:"bound"`
	BindingID       int64  `json:"binding_id,omitempty"`
	PartnerID       int64  `json:"partner_id,omitempty"`
	PartnerUsername string `json:"partner_username,omitempty"`
	PartnerNote     string `json:"partner_note,omitempty"`
}

// CompanionPartner 表示历次已经接受绑定的可查询对象。
type CompanionPartner struct {
	ID             int64     `json:"id"`
	Username       string    `json:"username"`
	LastAcceptedAt time.Time `json:"last_accepted_at"`
}

// UnbindAction 表示统一解绑入口在服务端判定后的下一步动作。
type UnbindAction string

const (
	// UnbindActionRequestSent 表示对方最近登录过，已经写入双方确认的解绑申请。
	UnbindActionRequestSent UnbindAction = "request_sent"
	// UnbindActionInactiveConfirmationRequired 表示对方超过 30 天未登录，需要前端额外确认直接解绑。
	UnbindActionInactiveConfirmationRequired UnbindAction = "inactive_confirmation_required"
	// UnbindActionDirectlyEnded 表示额外确认后已经直接结束双向绑定。
	UnbindActionDirectlyEnded UnbindAction = "directly_ended"
)
