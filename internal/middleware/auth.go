// auth.go 提供跨业务接口复用的，带身份请求的鉴权
package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	contextUserIDKey   = "auth_user_id"
	contextUsernameKey = "auth_username"
)

// AccessAuthenticator 描述受保护路由所需的访问 JWT 与会话校验能力。
type AccessAuthenticator interface {
	Authenticate(ctx context.Context, raw string) (service.AuthClaims, error)
}

// NoStore 阻止浏览器和中间缓存复用含会话状态的响应。
func NoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Next()
}

// RequireAuth 先校验访问 JWT 与会话状态，再把可信身份写入 Gin Context。
func RequireAuth(cookieName string, auth AccessAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		raw, err := c.Cookie(cookieName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "AUTH_REQUIRED", "message": "请先登录"})
			return
		}
		claims, err := auth.Authenticate(c.Request.Context(), raw)
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "AUTH_REQUIRED", "message": err.Error()})
			return
		case err != nil:
			slog.Error("访问鉴权暂时不可用", "error", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": "AUTH_UNAVAILABLE", "message": "登录校验暂时不可用，请稍后重试"})
			return
		}
		c.Set(contextUserIDKey, claims.UserID)
		c.Set(contextUsernameKey, claims.Username)
		c.Next()
	}
}

// UserID 只读取 Middleware 写入的可信用户 ID，不接受客户端参数替代。
func UserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get(contextUserIDKey)
	userID, ok := value.(int64)
	return userID, exists && ok && userID > 0
}

// Username 返回 JWT 中经过签名校验的用户名。
func Username(c *gin.Context) (string, bool) {
	value, exists := c.Get(contextUsernameKey)
	username, ok := value.(string)
	return username, exists && ok && username != ""
}
