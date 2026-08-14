package handler

import (
	"errors"
	"log/slog"
	"net/http"

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
		slog.Error("记录按钮点击失败", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "记录点击失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"accepted_count": request.Count})
}

// Stats 返回跨用户共享的总数、UTC 每日数和分钟聚合列表。
func (h *ButtonClickHandler) Stats(c *gin.Context) {
	stats, err := h.clicks.Stats(c.Request.Context())
	if err != nil {
		slog.Error("查询按钮统计失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "查询统计失败"})
		return
	}
	c.JSON(http.StatusOK, stats)
}
