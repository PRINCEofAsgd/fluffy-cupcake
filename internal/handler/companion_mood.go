package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/middleware"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"github.com/gin-gonic/gin"
)

// CompanionMoodHandler 提供当前陪伴关系的心情 HTTP 读写接口。
type CompanionMoodHandler struct {
	moods *service.CompanionMoodService
}

// NewCompanionMoodHandler 创建心情接口处理器。
func NewCompanionMoodHandler(moods *service.CompanionMoodService) *CompanionMoodHandler {
	return &CompanionMoodHandler{moods: moods}
}

// Get 默认读取当前绑定对象的心情，direction=mine 时读取本人。
func (h *CompanionMoodHandler) Get(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	view, err := h.moods.Get(c.Request.Context(), userID, c.DefaultQuery("direction", "theirs"))
	writeCompanionMoodResponse(c, view, err)
}

// Put 全量替换本人快照，拒绝额外字段和拼接的 JSON，限制请求体大小。
func (h *CompanionMoodHandler) Put(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	var request model.CompanionMoodView
	var extra any
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&extra) != io.EOF {
		writeCompanionMoodResponse(c, model.CompanionMoodView{}, service.ErrInvalidMood)
		return
	}
	view, err := h.moods.Save(c.Request.Context(), userID, request)
	writeCompanionMoodResponse(c, view, err)
}

// writeCompanionMoodResponse 将校验和绑定错误转成稳定的 HTTP 响应。
func writeCompanionMoodResponse(c *gin.Context, view model.CompanionMoodView, err error) {
	switch {
	case err == nil:
		c.JSON(http.StatusOK, view)
	case errors.Is(err, service.ErrInvalidMood), errors.Is(err, service.ErrInvalidDirection):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrNotBound):
		c.JSON(http.StatusConflict, gin.H{"code": "MOOD_NOT_BOUND", "message": err.Error()})
	case errors.Is(err, service.ErrMoodBindingChanged):
		c.JSON(http.StatusConflict, gin.H{"code": "MOOD_BINDING_CHANGED", "message": err.Error()})
	default:
		slog.Error("陪伴心情请求失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "心情请求失败"})
	}
}
