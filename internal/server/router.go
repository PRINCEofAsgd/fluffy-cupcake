// Package server 负责组装 Gin 路由、中间件和请求处理器。
package server

import (
	"database/sql"
	"net/http"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/handler"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/middleware"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/version"
	webcontent "github.com/PRINCEofAsgd/fluffy-cupcake/web"
	"github.com/gin-gonic/gin"
)

// NewRouter 组装页面、鉴权和按钮业务完整的 Handler → Service → Repository 链路。
func NewRouter(cfg config.Config, db *sql.DB) *gin.Engine {
	gin.SetMode(normalizeMode(cfg.Mode))
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), securityHeaders())
	_ = router.SetTrustedProxies(nil)

	pageHandler, err := handler.NewPageHandler(webcontent.Content, version.Current)
	if err != nil {
		panic("初始化页面资源失败: " + err.Error())
	}
	userRepository := repository.NewUserRepository(db)
	qrLoginRepository := repository.NewQrLoginRepository(db)
	companionRepository := repository.NewCompanionRepository(db)
	clickRepository := repository.NewButtonClickRepository(db)
	authService := service.NewAuthService(userRepository, qrLoginRepository, cfg.Auth.JWTSecret, cfg.Auth.JWTExpire)
	companionService := service.NewCompanionService(companionRepository, userRepository)
	clickService := service.NewButtonClickService(clickRepository, companionRepository)
	authHandler := handler.NewAuthHandler(authService, cfg.Auth.CookieName, cfg.Auth.CookieSecure, cfg.Auth.JWTExpire)
	clickHandler := handler.NewButtonClickHandler(clickService)
	companionHandler := handler.NewCompanionHandler(companionService)
	requireAuth := middleware.RequireAuth(cfg.Auth.CookieName, authService)

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/yanlili")
	})
	router.GET("/healthz", handler.Health)
	router.GET("/yanlili", pageHandler.Yanlili)
	router.GET("/assets/miss-button.gif", pageHandler.Asset("miss-button.gif", "image/gif", "public, max-age=86400"))
	router.GET("/assets/miss-pop.mp3", pageHandler.Asset("miss-pop.mp3", "audio/mpeg", "public, max-age=86400"))
	router.GET("/assets/app.css", pageHandler.Asset("app.css", "text/css; charset=utf-8", "public, max-age=3600"))
	router.GET("/assets/app.js", pageHandler.Asset("app.js", "text/javascript; charset=utf-8", "public, max-age=3600"))
	router.GET("/assets/qrdecoder.js", pageHandler.Asset("qrdecoder.js", "text/javascript; charset=utf-8", "public, max-age=3600"))

	authAPI := router.Group("/api/auth")
	authAPI.POST("/login", authHandler.Login)
	authAPI.POST("/qr-login", authHandler.QrLogin)
	authAPI.POST("/logout", authHandler.Logout)
	authAPI.GET("/me", requireAuth, authHandler.Me)

	yanliliAPI := router.Group("/api/yanlili", requireAuth)
	yanliliAPI.POST("/clicks", clickHandler.AddClicks)
	yanliliAPI.GET("/clicks/stats", clickHandler.Stats)
	yanliliAPI.GET("/clicks/details", clickHandler.Details)

	companionAPI := router.Group("/api/companion", requireAuth)
	companionAPI.GET("/state", companionHandler.State)
	companionAPI.GET("/inbox", companionHandler.Inbox)
	companionAPI.POST("/invitations", companionHandler.Invite)
	companionAPI.POST("/bindings/:id/accept", companionHandler.AcceptInvitation)
	companionAPI.PATCH("/bindings/:id/note", companionHandler.UpdateNote)
	companionAPI.POST("/bindings/:id/unbind-request", companionHandler.RequestUnbind)
	companionAPI.POST("/bindings/:id/unbind-accept", companionHandler.AcceptUnbind)
	companionAPI.POST("/bindings/:id/unbind-cancel", companionHandler.CancelUnbind)
	companionAPI.POST("/bindings/:id/unbind-reject", companionHandler.RejectUnbind)
	companionAPI.GET("/partners", companionHandler.Partners)
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
		c.Header("Content-Security-Policy", "default-src 'self'; img-src 'self' blob:; media-src 'self'; style-src 'self'; script-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Next()
	}
}
