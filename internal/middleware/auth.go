// Package middleware 提供跨业务接口复用的 Gin 鉴权中间件。
package middleware

import (
	"net/http"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	contextUserIDKey   = "auth_user_id"
	contextUsernameKey = "auth_username"
)

// TokenParser 描述 Middleware 所需的 JWT 解析能力，便于独立测试。
type TokenParser interface {
	ParseToken(raw string) (service.AuthClaims, error)
}

// RequireAuth 从 HttpOnly Cookie 校验 JWT，并把可信身份写入 Gin Context。
func RequireAuth(cookieName string, parser TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(cookieName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "请先登录"})
			return
		}
		claims, err := parser.ParseToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "认证信息无效或已过期"})
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
