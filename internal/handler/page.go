// Package handler 实现 HTTP 请求处理逻辑。
package handler

import (
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PageHandler 保存页面模板和静态资源文件系统。
type PageHandler struct {
	template *template.Template
	assets   fs.FS
	version  string
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

	return &PageHandler{template: pageTemplate, assets: assets, version: version}, nil
}

// Yanlili 返回“想哥哥按钮”页面。
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

// Health 返回供容器和反向代理探测的服务状态。
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
