// auth_test.go 验证认证凭证在 HTTP 响应中的浏览器策略。
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"github.com/gin-gonic/gin"
)

// TestAuthCookieAttributes 验证两个生产 Cookie 的属性以及退出清理。
func TestAuthCookieAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := service.NewAuthService(nil, nil, nil, "", 15*time.Minute, 360*time.Hour)
	handler := NewAuthHandler(auth, "auth", true)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	now := time.Now().UTC().Truncate(time.Second)
	credentials := service.AuthCredentials{
		AccessToken: "signed-token", AccessExpiresAt: now.Add(15 * time.Minute),
		RefreshToken: "refresh-token", SessionExpiresAt: now.Add(360 * time.Hour),
	}
	handler.setCredentials(context, credentials)

	cookies := recorder.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("Cookie 数量 = %d", len(cookies))
	}
	if cookies[0].Name != "auth" || cookies[0].Path != "/" || !cookies[0].Expires.Equal(credentials.AccessExpiresAt) ||
		cookies[1].Name != "auth_refresh" || cookies[1].Path != "/api/auth" || !cookies[1].Expires.Equal(credentials.SessionExpiresAt) {
		t.Fatalf("两个 Cookie 的名称、路径或截止时间不正确: %v", cookies)
	}
	for _, cookie := range cookies {
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("生产认证 Cookie 属性 = %#v", cookie)
		}
	}

	logoutRecorder := httptest.NewRecorder()
	logoutContext, _ := gin.CreateTestContext(logoutRecorder)
	logoutContext.Request = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	handler.Logout(logoutContext)
	cleared := logoutRecorder.Result().Cookies()
	if len(cleared) != 2 {
		t.Fatalf("退出 Cookie = %#v", cleared)
	}
	for i, cookie := range cleared {
		if cookie.Name != cookies[i].Name || cookie.Path != cookies[i].Path || cookie.MaxAge >= 0 || cookie.Value != "" {
			t.Fatalf("Cookie 没有在原路径清除: %#v", cookie)
		}
	}
}
