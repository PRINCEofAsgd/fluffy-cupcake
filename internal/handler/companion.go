package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/middleware"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"github.com/gin-gonic/gin"
)

// CompanionHandler 负责陪伴绑定信件的 HTTP 参数和状态码边界。
type CompanionHandler struct{ companions *service.CompanionService }

// NewCompanionHandler 创建陪伴绑定接口处理器。
func NewCompanionHandler(companions *service.CompanionService) *CompanionHandler {
	return &CompanionHandler{companions: companions}
}

type inviteCompanionRequest struct {
	Username string `json:"username" binding:"required"`
	Note     string `json:"note"`
}

type updateCompanionNoteRequest struct {
	Note string `json:"note"`
}

type requestUnbindRequest struct {
	ConfirmInactive bool `json:"confirm_inactive"`
}

type inboxBindingResponse struct {
	model.CompanionBinding
	MyNote string `json:"my_note"`
}

// State 返回当前唯一绑定和用户自己的备注。
func (h *CompanionHandler) State(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	state, err := h.companions.State(c.Request.Context(), userID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, state)
}

// Invite 通过用户名发起绑定邀请；备注允许为空。
func (h *CompanionHandler) Invite(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	var request inviteCompanionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请输入用户名"})
		return
	}
	id, err := h.companions.Invite(c.Request.Context(), userID, request.Username, request.Note)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "绑定邀请已发送"})
}

// Inbox 返回双方都永久可见的信件，但只返回当前用户自己的私有备注。
func (h *CompanionHandler) Inbox(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	bindings, err := h.companions.Inbox(c.Request.Context(), userID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	items := make([]inboxBindingResponse, 0, len(bindings))
	for _, binding := range bindings {
		note := binding.InviteeNote
		if binding.InviterID == userID {
			note = binding.InviterNote
		}
		items = append(items, inboxBindingResponse{CompanionBinding: binding, MyNote: note})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// AcceptInvitation 接受当前用户收到的待处理绑定邀请。
func (h *CompanionHandler) AcceptInvitation(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	bindingID, ok := bindingIDParam(c)
	if !ok {
		return
	}
	if err := h.companions.AcceptInvitation(c.Request.Context(), userID, bindingID); err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已完成双向绑定"})
}

// UpdateNote 修改当前用户对当前绑定对象的备注。
func (h *CompanionHandler) UpdateNote(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	bindingID, ok := bindingIDParam(c)
	if !ok {
		return
	}
	var request updateCompanionNoteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "备注格式错误"})
		return
	}
	if err := h.companions.UpdateNote(c.Request.Context(), userID, bindingID, request.Note); err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "备注已修改"})
}

// RequestUnbind 通过单一入口完成活跃度判定、普通申请或额外确认后的直接解绑。
func (h *CompanionHandler) RequestUnbind(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	bindingID, ok := bindingIDParam(c)
	if !ok {
		return
	}
	var request requestUnbindRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "解绑确认参数无效"})
		return
	}
	action, err := h.companions.RequestUnbind(c.Request.Context(), userID, bindingID, request.ConfirmInactive)
	if err != nil {
		h.writeError(c, err)
		return
	}
	messages := map[model.UnbindAction]string{
		model.UnbindActionRequestSent:                  "解绑申请已发送",
		model.UnbindActionInactiveConfirmationRequired: "对方已连续30天未登录，将直接解绑",
		model.UnbindActionDirectlyEnded:                "已直接双向解绑",
	}
	c.JSON(http.StatusOK, gin.H{"action": action, "message": messages[action]})
}

// AcceptUnbind 接受另一方在同一封信中发起的解绑邀请。
func (h *CompanionHandler) AcceptUnbind(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	bindingID, ok := bindingIDParam(c)
	if !ok {
		return
	}
	if err := h.companions.AcceptUnbind(c.Request.Context(), userID, bindingID); err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已完成双向解绑"})
}

// CancelUnbind 取消当前用户自己发起且尚未处理的解绑申请。
func (h *CompanionHandler) CancelUnbind(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	bindingID, ok := bindingIDParam(c)
	if !ok {
		return
	}
	if err := h.companions.CancelUnbind(c.Request.Context(), userID, bindingID); err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已取消解绑申请"})
}

// RejectUnbind 拒绝对方在同一封信中发起且尚未处理的解绑申请。
func (h *CompanionHandler) RejectUnbind(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	bindingID, ok := bindingIDParam(c)
	if !ok {
		return
	}
	if err := h.companions.RejectUnbind(c.Request.Context(), userID, bindingID); err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已拒绝解绑"})
}

// Partners 返回详细记录选择器所需的历次绑定对象。
func (h *CompanionHandler) Partners(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	partners, err := h.companions.Partners(c.Request.Context(), userID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": partners})
}

// bindingIDParam 解析并校验路由中的正整数信件 ID。
func bindingIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "绑定信件 ID 无效"})
		return 0, false
	}
	return id, true
}

// writeError 把稳定业务错误映射为不泄露内部 SQL 的 HTTP 响应。
func (h *CompanionHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "用户或绑定信件不存在"})
	case errors.Is(err, service.ErrCannotBindSelf), errors.Is(err, service.ErrCompanionNoteTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, repository.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error()})
	case errors.Is(err, repository.ErrAlreadyBound), errors.Is(err, repository.ErrInvalidBindingState):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	default:
		slog.Error("陪伴绑定操作失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "陪伴绑定操作失败"})
	}
}
