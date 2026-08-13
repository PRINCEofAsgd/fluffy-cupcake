// Package web 将页面模板与静态资源嵌入 Go 二进制，便于单文件部署。
package web

import "embed"

// Content 包含运行页面所需的全部模板、样式、脚本和图片。
//
//go:embed templates/*.html assets/*
var Content embed.FS
