// auth.go 处理登录、刷新、退出和当前用户的 HTTP 接口。
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/middleware"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler 负责鉴权 HTTP 参数、Cookie 写入和安全错误响应。
type AuthHandler struct {
	auth              *service.AuthService
	cookieName        string
	refreshCookieName string
	cookieSecure      bool
}

// NewAuthHandler 创建认证接口处理器。
func NewAuthHandler(auth *service.AuthService, cookieName string, cookieSecure bool) *AuthHandler {
	return &AuthHandler{auth: auth, cookieName: cookieName, refreshCookieName: cookieName + "_refresh", cookieSecure: cookieSecure}
}

// LoginRequest 是用户名密码登录请求；密码不会写入任何日志。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 校验请求并把短期 JWT 和刷新凭证写入 HttpOnly Cookie。
func (h *AuthHandler) Login(c *gin.Context) {
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请输入用户名和密码"})
		return
	}
	credentials, err := h.auth.Login(c.Request.Context(), request.Username, request.Password)
	if errors.Is(err, service.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "用户名或密码错误"})
		return
	}
	if err != nil {
		slog.Error("登录失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "登录暂时不可用"})
		return
	}
	h.setCredentials(c, credentials)
	c.JSON(http.StatusOK, gin.H{"message": "登录成功"})
}

// QrLoginRequest 是二维码文本登录请求；文本内容不会写入任何日志。
type QrLoginRequest struct {
	Text string `json:"text" binding:"required,max=512"`
}

// QrLogin 按二维码永久文本查表登录并把短期 JWT 和刷新凭证写入 HttpOnly Cookie。
func (h *AuthHandler) QrLogin(c *gin.Context) {
	var request QrLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "二维码内容无效"})
		return
	}
	credentials, err := h.auth.LoginByQrText(c.Request.Context(), request.Text)
	if errors.Is(err, service.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "二维码无法识别"})
		return
	}
	if err != nil {
		slog.Error("二维码登录失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "登录暂时不可用"})
		return
	}
	h.setCredentials(c, credentials)
	c.JSON(http.StatusOK, gin.H{"message": "登录成功"})
}

// Refresh 凭 HttpOnly 刷新 Cookie 续期，冲突和异常响应不会覆盖其他请求的新 Cookie。
func (h *AuthHandler) Refresh(c *gin.Context) {
	refresh, _ := c.Cookie(h.refreshCookieName)
	if refresh == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "NO_SESSION", "message": service.ErrInvalidSession.Error()})
		return
	}
	credentials, err := h.auth.Refresh(c.Request.Context(), refresh)
	switch {
	case errors.Is(err, service.ErrInvalidSession):
		c.JSON(http.StatusUnauthorized, gin.H{"code": "SESSION_ENDED", "message": err.Error()})
	case errors.Is(err, service.ErrRefreshConflict):
		c.JSON(http.StatusConflict, gin.H{"code": "REFRESH_CONFLICT", "message": err.Error()})
	case err != nil:
		slog.Error("刷新登录会话失败", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "暂时无法恢复登录，请稍后重试"})
	default:
		h.setCredentials(c, credentials)
		c.Status(http.StatusNoContent)
	}
}

// Logout 撤销当前会话，成功后清除访问和刷新 Cookie。
func (h *AuthHandler) Logout(c *gin.Context) {
	refresh, _ := c.Cookie(h.refreshCookieName)
	access, _ := c.Cookie(h.cookieName)
	if err := h.auth.Logout(c.Request.Context(), refresh, access); err != nil {
		slog.Error("撤销登录会话失败", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "暂时无法退出登录，请稍后重试"})
		return
	}
	past := time.Unix(1, 0).UTC()
	h.setCookie(c, h.cookieName, "/", "", past, -1)
	h.setCookie(c, h.refreshCookieName, "/api/auth", "", past, -1)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

// Me 返回当前用户的非敏感基本信息，绝不序列化 password_hash。
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "请先登录"})
		return
	}
	user, err := h.auth.CurrentUser(c.Request.Context(), userID)
	if errors.Is(err, service.ErrInvalidToken) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "认证信息无效或已过期"})
		return
	}
	if err != nil {
		slog.Error("读取当前用户失败", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "读取用户信息失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "username": user.Username})
}

// setCredentials 用绝对截止时间写入认证 Cookie，刷新 Cookie 只发往认证路由。
func (h *AuthHandler) setCredentials(c *gin.Context, credentials service.AuthCredentials) {
	h.setCookie(c, h.cookieName, "/", credentials.AccessToken, credentials.AccessExpiresAt, 0)
	h.setCookie(c, h.refreshCookieName, "/api/auth", credentials.RefreshToken, credentials.SessionExpiresAt, 0)
}

// setCookie 统一应用现有的 HttpOnly、SameSite=Lax 和按环境区分的 Secure 策略。
func (h *AuthHandler) setCookie(c *gin.Context, name, path, value string, expires time.Time, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
