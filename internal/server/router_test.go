package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
)

// TestYanliliPage 验证页面、按钮文案和安全响应头均正确返回。
func TestYanliliPage(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/yanlili", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "按按钮，想你+1") {
		t.Fatal("页面缺少点击后显示的目标文案")
	}
	if strings.Contains(response.Body.String(), "/assets/miss-pop.mp3") {
		t.Fatal("页面不应再加载 MP3，点击音效应由 Web Audio 合成")
	}
	if !strings.Contains(response.Body.String(), `id="login-button"`) {
		t.Fatal("页面缺少独立登录入口")
	}
	if !strings.Contains(response.Body.String(), `id="companion-button"`) {
		t.Fatal("页面缺少未登录可见的陪伴绑定入口")
	}
	for _, expected := range []string{"我想ta", "每日想念", "最近想念", `id="direction-switch"`, `id="details-button"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("页面缺少新想念记录交互 %q", expected)
		}
	}
	if strings.Contains(response.Body.String(), `id="app-panel" class="app-panel" aria-label="想念按钮与陪伴绑定" hidden`) {
		t.Fatal("未登录状态不应隐藏主按钮面板")
	}
	if !strings.Contains(response.Body.String(), `id="stats-panel"`) || !strings.Contains(response.Body.String(), `id="stats-panel" class="stats-panel" aria-labelledby="stats-title" hidden`) {
		t.Fatal("数据库统计区域应默认隐藏并由登录状态控制")
	}
	contentSecurityPolicy := response.Header().Get("Content-Security-Policy")
	if contentSecurityPolicy == "" {
		t.Fatal("页面缺少 Content-Security-Policy 响应头")
	}
	if !strings.Contains(contentSecurityPolicy, "img-src 'self' blob:") {
		t.Fatal("页面 CSP 必须允许旧浏览器通过 blob URL 读取用户选择的二维码图片")
	}
}

// TestHealthAndAsset 验证健康检查与嵌入式静态资源可以访问。
func TestHealthAndAsset(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)

	for _, testCase := range []struct {
		path        string
		contentType string
	}{
		{path: "/healthz", contentType: "application/json"},
		{path: "/assets/miss-button.gif", contentType: "image/gif"},
		{path: "/assets/miss-pop.mp3", contentType: "audio/mpeg"},
		{path: "/assets/app.js", contentType: "text/javascript"},
		{path: "/assets/qrdecoder.js", contentType: "text/javascript"},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("GET %s 状态码 = %d，期望 %d", testCase.path, response.Code, http.StatusOK)
		}
		if !strings.Contains(response.Header().Get("Content-Type"), testCase.contentType) {
			t.Fatalf("GET %s Content-Type = %q", testCase.path, response.Header().Get("Content-Type"))
		}
	}
}

// TestQrScannerImplementation 防止摄像头首帧退出、相册 CSP 冲突和反色识别能力回退。
func TestQrScannerImplementation(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)

	router.ServeHTTP(response, request)
	body := response.Body.String()
	for _, expected := range []string{
		"requestVideoFrameCallback",
		"loadeddata",
		"scheduleQrScan(generation)",
		"createImageBitmap",
		`inversionAttempts: "attemptBoth"`,
		"decodeQrSource",
		"URL.revokeObjectURL",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("app.js 缺少稳健二维码识别实现 %q", expected)
		}
	}
	if strings.Contains(body, `readyState < 2) return`) {
		t.Fatal("摄像头首帧未就绪时不得直接退出整个扫码循环")
	}
	if strings.Contains(body, `inversionAttempts: "dontInvert"`) {
		t.Fatal("二维码识别不得禁用反色尝试")
	}
}

// TestClickSoundImplementation 防止页面退回会互相打断的单实例 MP3 播放方式。
func TestClickSoundImplementation(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)

	router.ServeHTTP(response, request)

	body := response.Body.String()
	for _, expected := range []string{"AudioContext", "createOscillator", "createDynamicsCompressor"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("app.js 缺少可并发 Web Audio 实现 %q", expected)
		}
	}
	if !strings.Contains(body, "该功能需登录后使用") {
		t.Fatal("app.js 缺少未登录绑定的明确提示")
	}
	if !strings.Contains(body, "ta想我") || !strings.Contains(body, "direction=mine") && !strings.Contains(body, "statsDirection") {
		t.Fatal("app.js 缺少双向记录切换逻辑")
	}
	for _, expected := range []string{"confirmUnbindRequest", "prompts.every", "confirm_inactive", "inactive_confirmation_required"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("app.js 缺少统一解绑确认逻辑 %q", expected)
		}
	}
	for _, expected := range []string{"取消解绑申请", "拒绝解绑", "unbind-cancel", "unbind-reject", "unbind_status"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("app.js 缺少解绑申请撤销/拒绝交互 %q", expected)
		}
	}
	if strings.Contains(body, "direct-unbind") || strings.Contains(body, "unbind-direct") {
		t.Fatal("app.js 不应暴露第二条直接解绑路线")
	}
	if strings.Contains(body, "new Audio(") {
		t.Fatal("app.js 不应再创建单实例 MP3 音频")
	}
}
