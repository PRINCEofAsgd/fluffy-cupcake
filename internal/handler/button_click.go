package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/middleware"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"github.com/gin-gonic/gin"
)

// ButtonClickHandler 负责点击写入与统计查询的 HTTP 边界。
type ButtonClickHandler struct {
	clicks *service.ButtonClickService
}

// NewButtonClickHandler 创建按钮点击接口处理器。
func NewButtonClickHandler(clicks *service.ButtonClickService) *ButtonClickHandler {
	return &ButtonClickHandler{clicks: clicks}
}

type addClicksRequest struct {
	Count int64 `json:"count" binding:"required"`
}

// AddClicks 校验批量 count，并只使用 Middleware 提供的 user_id。
func (h *ButtonClickHandler) AddClicks(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "请先登录"})
		return
	}
	var request addClicksRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Count <= 0 || request.Count > service.MaxClickDelta {
		c.JSON(http.StatusBadRequest, gin.H{"message": "count 必须在 1 到 100 之间"})
		return
	}
	if err := h.clicks.AddClicks(c.Request.Context(), userID, request.Count); err != nil {
		if errors.Is(err, service.ErrInvalidClickCount) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		if errors.Is(err, service.ErrNotBound) {
			c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
			return
		}
		slog.Error("记录按钮点击失败", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "记录点击失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"accepted_count": request.Count})
}

// Stats 返回当前绑定所选方向的总数、最近 8 条每日和分钟统计。
func (h *ButtonClickHandler) Stats(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	direction := c.DefaultQuery("direction", "mine")
	stats, err := h.clicks.Stats(c.Request.Context(), userID, direction)
	if err != nil {
		if errors.Is(err, service.ErrInvalidDirection) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		if errors.Is(err, service.ErrNotBound) {
			c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
			return
		}
		slog.Error("查询按钮统计失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "查询统计失败"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// Details 返回与一个历次绑定对象之间按新到旧排列的 20 条分页明细。
func (h *ButtonClickHandler) Details(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	partnerID, err := strconv.ParseInt(c.Query("partner_id"), 10, 64)
	if err != nil || partnerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请选择绑定对象"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	details, err := h.clicks.Details(c.Request.Context(), userID, partnerID, page)
	if err != nil {
		if errors.Is(err, service.ErrCannotBindSelf) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "绑定对象无效"})
			return
		}
		slog.Error("查询详细想念记录失败", "error", err, "user_id", userID, "partner_id", partnerID)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "查询详细记录失败"})
		return
	}
	c.JSON(http.StatusOK, details)
}
