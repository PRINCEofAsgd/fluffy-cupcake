package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/service"
	"github.com/gin-gonic/gin"
)

type fakeTokenParser struct{}

func (fakeTokenParser) ParseToken(raw string) (service.AuthClaims, error) {
	if raw != "valid-token" {
		return service.AuthClaims{}, errors.New("invalid")
	}
	return service.AuthClaims{UserID: 42, Username: "tester"}, nil
}

// TestRequireAuth 验证缺失、非法和有效 Cookie 的统一鉴权行为。
func TestRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", RequireAuth("auth", fakeTokenParser{}), func(c *gin.Context) {
		userID, ok := UserID(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	for _, testCase := range []struct {
		name       string
		cookie     *http.Cookie
		statusCode int
	}{
		{name: "无 Cookie", statusCode: http.StatusUnauthorized},
		{name: "非法 Cookie", cookie: &http.Cookie{Name: "auth", Value: "invalid"}, statusCode: http.StatusUnauthorized},
		{name: "有效 Cookie", cookie: &http.Cookie{Name: "auth", Value: "valid-token"}, statusCode: http.StatusOK},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if testCase.cookie != nil {
				request.AddCookie(testCase.cookie)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != testCase.statusCode {
				t.Fatalf("状态码 = %d，期望 %d", response.Code, testCase.statusCode)
			}
		})
	}
}
