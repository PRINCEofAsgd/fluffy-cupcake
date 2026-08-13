// Package server 负责组装 Gin 路由、中间件和请求处理器。
package server

import (
	"net/http"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/handler"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/version"
	webcontent "github.com/PRINCEofAsgd/fluffy-cupcake/web"
	"github.com/gin-gonic/gin"
)

// NewRouter 创建包含健康检查、页面及静态资源路由的 Gin 引擎。
func NewRouter(cfg config.Config) *gin.Engine {
	gin.SetMode(normalizeMode(cfg.Mode))
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), securityHeaders())
	_ = router.SetTrustedProxies(nil)

	pageHandler, err := handler.NewPageHandler(webcontent.Content, version.Current)
	if err != nil {
		panic("初始化页面资源失败: " + err.Error())
	}

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/yanlili")
	})
	router.GET("/healthz", handler.Health)
	router.GET("/yanlili", pageHandler.Yanlili)
	router.GET("/assets/miss-button.gif", pageHandler.Asset("miss-button.gif", "image/gif", "public, max-age=86400"))
	router.GET("/assets/app.css", pageHandler.Asset("app.css", "text/css; charset=utf-8", "public, max-age=3600"))
	router.GET("/assets/app.js", pageHandler.Asset("app.js", "text/javascript; charset=utf-8", "public, max-age=3600"))
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "页面不存在"})
	})

	return router
}

// normalizeMode 将应用模式限制为 Gin 支持的模式值。
func normalizeMode(mode string) string {
	switch mode {
	case gin.ReleaseMode, gin.TestMode:
		return mode
	default:
		return gin.DebugMode
	}
}

// securityHeaders 为所有响应添加基础浏览器安全策略。
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'self'; img-src 'self'; style-src 'self'; script-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Next()
	}
}
