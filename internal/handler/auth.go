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
	auth         *service.AuthService
	cookieName   string
	cookieSecure bool
	expire       time.Duration
}

// NewAuthHandler 创建认证接口处理器。
func NewAuthHandler(auth *service.AuthService, cookieName string, cookieSecure bool, expire time.Duration) *AuthHandler {
	return &AuthHandler{auth: auth, cookieName: cookieName, cookieSecure: cookieSecure, expire: expire}
}

// LoginRequest 是用户名密码登录请求；密码不会写入任何日志。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 校验请求并把短期 JWT 写入 HttpOnly Cookie。
func (h *AuthHandler) Login(c *gin.Context) {
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请输入用户名和密码"})
		return
	}
	token, expiresAt, err := h.auth.Login(c.Request.Context(), request.Username, request.Password)
	if errors.Is(err, service.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "用户名或密码错误"})
		return
	}
	if err != nil {
		slog.Error("登录失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "登录暂时不可用"})
		return
	}
	h.setCookie(c, token, expiresAt, int(h.expire.Seconds()))
	c.JSON(http.StatusOK, gin.H{"message": "登录成功"})
}

// QrLoginRequest 是二维码文本登录请求；文本内容不会写入任何日志。
type QrLoginRequest struct {
	Text string `json:"text" binding:"required,max=512"`
}

// QrLogin 按二维码永久文本查表登录并把短期 JWT 写入 HttpOnly Cookie。
func (h *AuthHandler) QrLogin(c *gin.Context) {
	var request QrLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "二维码内容无效"})
		return
	}
	token, expiresAt, err := h.auth.LoginByQrText(c.Request.Context(), request.Text)
	if errors.Is(err, service.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "二维码无法识别"})
		return
	}
	if err != nil {
		slog.Error("二维码登录失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "登录暂时不可用"})
		return
	}
	h.setCookie(c, token, expiresAt, int(h.expire.Seconds()))
	c.JSON(http.StatusOK, gin.H{"message": "登录成功"})
}

// Logout 通过过期同名 Cookie 清除浏览器认证信息。
func (h *AuthHandler) Logout(c *gin.Context) {
	h.setCookie(c, "", time.Unix(1, 0), -1)
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

// setCookie 统一设置 HttpOnly、SameSite=Lax、Path=/ 和按环境区分的 Secure。
func (h *AuthHandler) setCookie(c *gin.Context, value string, expires time.Time, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
