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
	EndedAt           *time.Time `json:"ended_at,omitempty"`
	EndedBy           *int64     `json:"ended_by,omitempty"`
	EndedReason       *string    `json:"ended_reason,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

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
