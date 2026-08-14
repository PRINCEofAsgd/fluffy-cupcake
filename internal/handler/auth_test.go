package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestAuthCookieAttributes 验证生产 Cookie 的 HttpOnly、Secure、SameSite 和退出过期语义。
func TestAuthCookieAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &AuthHandler{cookieName: "auth", cookieSecure: true}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	handler.setCookie(context, "signed-token", time.Now().Add(time.Hour), 3600)

	response := recorder.Result()
	cookies := response.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Cookie 数量 = %d", len(cookies))
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.MaxAge != 3600 {
		t.Fatalf("生产认证 Cookie 属性 = %#v", cookie)
	}

	logoutRecorder := httptest.NewRecorder()
	logoutContext, _ := gin.CreateTestContext(logoutRecorder)
	handler.Logout(logoutContext)
	logoutCookies := logoutRecorder.Result().Cookies()
	if len(logoutCookies) != 1 || logoutCookies[0].MaxAge >= 0 || logoutCookies[0].Value != "" {
		t.Fatalf("退出 Cookie = %#v", logoutCookies)
	}
}
