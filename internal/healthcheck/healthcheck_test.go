package healthcheck

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// roundTripFunc 将函数适配为 http.RoundTripper，测试时无需监听真实端口。
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip 返回内存构造的 HTTP 响应。
func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

// TestRun 验证健康检查客户端接受 200 响应并拒绝异常响应。
func TestRun(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		wantError  bool
	}{
		{name: "healthy", statusCode: http.StatusOK},
		{name: "unhealthy", statusCode: http.StatusServiceUnavailable, wantError: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.String() != "http://127.0.0.1:4819/healthz" {
					t.Fatalf("健康检查 URL = %q", request.URL.String())
				}
				return &http.Response{
					StatusCode: testCase.statusCode,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			})}

			err := runWithClient("0.0.0.0:4819", client)
			if (err != nil) != testCase.wantError {
				t.Fatalf("runWithClient error = %v，wantError = %v", err, testCase.wantError)
			}
		})
	}
}
