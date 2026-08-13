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
	router := NewRouter(config.Config{Mode: "test"})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/yanlili", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "按按钮，想哥哥+1") {
		t.Fatal("页面缺少点击后显示的目标文案")
	}
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("页面缺少 Content-Security-Policy 响应头")
	}
}

// TestHealthAndAsset 验证健康检查和 GIF 静态资源可以访问。
func TestHealthAndAsset(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"})

	for _, testCase := range []struct {
		path        string
		contentType string
	}{
		{path: "/healthz", contentType: "application/json"},
		{path: "/assets/miss-button.gif", contentType: "image/gif"},
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
