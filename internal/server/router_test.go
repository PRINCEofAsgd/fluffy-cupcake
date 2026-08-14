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
	if !strings.Contains(response.Body.String(), "按按钮，想哥哥+1") {
		t.Fatal("页面缺少点击后显示的目标文案")
	}
	if strings.Contains(response.Body.String(), "/assets/miss-pop.mp3") {
		t.Fatal("页面不应再加载 MP3，点击音效应由 Web Audio 合成")
	}
	if !strings.Contains(response.Body.String(), `id="login-button"`) {
		t.Fatal("页面缺少独立登录入口")
	}
	if strings.Contains(response.Body.String(), `id="app-panel" class="app-panel" aria-label="想哥哥按钮与统计" hidden`) {
		t.Fatal("未登录状态不应隐藏主按钮面板")
	}
	if !strings.Contains(response.Body.String(), `id="stats-panel"`) || !strings.Contains(response.Body.String(), `id="stats-panel" class="stats-panel" aria-labelledby="stats-title" hidden`) {
		t.Fatal("数据库统计区域应默认隐藏并由登录状态控制")
	}
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("页面缺少 Content-Security-Policy 响应头")
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
	if strings.Contains(body, "new Audio(") {
		t.Fatal("app.js 不应再创建单实例 MP3 音频")
	}
}
