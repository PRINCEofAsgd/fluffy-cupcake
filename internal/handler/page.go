// Package handler 实现 HTTP 请求处理逻辑。
package handler

import (
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// PageHandler 保存页面模板和静态资源文件系统。
type PageHandler struct {
	template       *template.Template
	assets         fs.FS
	homepage       []byte
	siteIcon       []byte
	homepageAssets fs.FS
	version        string
}

// NewPageHandler 解析嵌入式模板并创建页面处理器。
func NewPageHandler(content fs.FS, version string) (*PageHandler, error) {
	pageTemplate, err := template.ParseFS(content, "templates/yanlili.html")
	if err != nil {
		return nil, err
	}

	assets, err := fs.Sub(content, "assets")
	if err != nil {
		return nil, err
	}
	homepageRoot, err := fs.Sub(content, "homepage/dist")
	if err != nil {
		return nil, err
	}
	homepage, err := fs.ReadFile(homepageRoot, "index.html")
	if err != nil {
		return nil, err
	}
	siteIcon, err := fs.ReadFile(homepageRoot, "favicon.png")
	if err != nil {
		return nil, err
	}
	homepageAssets, err := fs.Sub(homepageRoot, "homepage")
	if err != nil {
		return nil, err
	}

	return &PageHandler{
		template:       pageTemplate,
		assets:         assets,
		homepage:       homepage,
		siteIcon:       siteIcon,
		homepageAssets: homepageAssets,
		version:        version,
	}, nil
}

// Home 返回公开 Homepage；该页面允许自身生成的内联样式以支持滚动动效和加载进度。
func (h *PageHandler) Home(c *gin.Context) {
	c.Header("Cache-Control", "no-cache")
	c.Header("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; media-src 'self'; font-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	c.Data(http.StatusOK, "text/html; charset=utf-8", h.homepage)
}

// Yanlili 返回“想念按钮”与陪伴绑定页面。
func (h *PageHandler) Yanlili(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	if err := h.template.ExecuteTemplate(c.Writer, "yanlili.html", map[string]string{"Version": h.version}); err != nil {
		_ = c.Error(err)
	}
}

// Asset 返回一个仅服务指定嵌入式资源的 Gin 处理函数。
func (h *PageHandler) Asset(name, contentType, cacheControl string) gin.HandlerFunc {
	return func(c *gin.Context) {
		content, err := fs.ReadFile(h.assets, name)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		c.Header("Cache-Control", cacheControl)
		c.Data(http.StatusOK, contentType, content)
	}
}

// SiteIcon 返回全站统一的 PNG 图标；同时供显式 favicon 链接和 /favicon.ico 回退使用。
func (h *PageHandler) SiteIcon(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/png", h.siteIcon)
}

// HomepageAsset 安全返回 Vite 构建产物，并为带哈希资源设置长期缓存。
func (h *PageHandler) HomepageAsset(c *gin.Context) {
	name := strings.TrimPrefix(c.Param("filepath"), "/")
	cleanName := path.Clean(name)
	if cleanName == "." || cleanName != name || strings.HasPrefix(cleanName, "../") {
		c.Status(http.StatusNotFound)
		return
	}

	content, err := fs.ReadFile(h.homepageAssets, cleanName)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	cacheControl := "public, max-age=86400"
	if strings.HasPrefix(cleanName, "assets/") {
		cacheControl = "public, max-age=31536000, immutable"
	}
	contentType := mime.TypeByExtension(path.Ext(cleanName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Cache-Control", cacheControl)
	c.Data(http.StatusOK, contentType, content)
}

// Health 返回供容器和反向代理探测的服务状态。
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
