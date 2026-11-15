// auth_session_mysql_test.go 用真实 MySQL 验证会话轮换、并发及业务写入。
package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/database"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// TestAuthSessionMySQL 在专用测试库覆盖登录、过期恢复、重用检测及退出撤销。
func TestAuthSessionMySQL(t *testing.T) {
	dsn := os.Getenv("FLUFFY_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("请在已执行 Migration 7 的独立测试库设置 FLUFFY_TEST_MYSQL_DSN")
	}
	db, err := database.Open(context.Background(), config.DatabaseConfig{
		DSN: dsn, MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var dbName string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&dbName); err != nil || dbName != "fluffy_session_test" {
		t.Fatalf("集成测试需要专用库 fluffy_session_test，得到 %q (%v)", dbName, err)
	}
	exec := func(query string, args ...any) int64 {
		t.Helper()
		result, err := db.Exec(query, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	username := "session-" + suffix
	userID := exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hash)
	partnerID := exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", "partner-"+suffix, hash)
	qrText := "test-qr-" + suffix
	exec("INSERT INTO qr_login_mappings (user_id, qr_text) VALUES (?, ?)", userID, qrText)
	bindingID := exec("INSERT INTO companion_bindings (inviter_id, invitee_id, status, accepted_at) VALUES (?, ?, 'active', UTC_TIMESTAMP())", userID, partnerID)
	exec("INSERT INTO companion_active_memberships (user_id, binding_id, partner_user_id) VALUES (?, ?, ?), (?, ?, ?)",
		userID, bindingID, partnerID, partnerID, bindingID, userID)
	defer func() {
		for _, query := range []string{
			"DELETE FROM button_click_minutes WHERE companion_binding_id = ?",
			"DELETE FROM companion_active_memberships WHERE binding_id = ?",
			"DELETE FROM companion_bindings WHERE id = ?",
		} {
			if _, err := db.Exec(query, bindingID); err != nil {
				t.Error(err)
			}
		}
		if _, err := db.Exec("DELETE FROM qr_login_mappings WHERE user_id = ?", userID); err != nil {
			t.Error(err)
		}
		if _, err := db.Exec("DELETE FROM users WHERE id IN (?, ?)", userID, partnerID); err != nil {
			t.Error(err)
		}
	}()

	const secret = "mysql-integration-secret-32-characters"
	cfg := config.Config{Mode: "test", Auth: config.AuthConfig{
		JWTSecret: secret, AccessTokenTTL: 2 * time.Second, SessionTTL: 360 * time.Hour, CookieName: "auth",
	}}
	router := NewRouter(cfg, db)
	login := authSessionRequest(router, "POST", "/api/auth/login", fmt.Sprintf(`{"username":%q,"password":"correct-password"}`, username))
	passwordCookies := expectSessionCookies(t, login, http.StatusOK)
	var loginAt, createdAt, expiresAt time.Time
	var sessionID string
	refreshHash := sha256.Sum256([]byte(passwordCookies[1].Value))
	err = db.QueryRow(`
		SELECT u.last_login_at, s.id, s.created_at, s.expires_at
		FROM auth_refresh_tokens r JOIN auth_sessions s ON s.id = r.session_id JOIN users u ON u.id = s.user_id
		WHERE r.token_hash = ?
	`, refreshHash[:]).Scan(&loginAt, &sessionID, &createdAt, &expiresAt)
	if err != nil || !createdAt.Equal(loginAt) || expiresAt.Sub(createdAt) != 360*time.Hour || !passwordCookies[1].Expires.Equal(expiresAt) {
		t.Fatalf("初始会话截止时间或登录时间错误: %v", err)
	}

	// 等待 JWT 实际到期，然后模拟前端在收到中间件拒绝后刷新并重试。
	time.Sleep(time.Until(passwordCookies[0].Expires) + 20*time.Millisecond)
	const clicksPath = "/api/yanlili/clicks"
	const clicks = `{"count":6}`
	rejected := authSessionRequest(router, "POST", clicksPath, clicks, passwordCookies[0])
	if rejected.Code != http.StatusUnauthorized || !strings.Contains(rejected.Body.String(), "AUTH_REQUIRED") {
		t.Fatalf("过期访问 JWT 未被中间件拦截: %d %s", rejected.Code, rejected.Body)
	}
	assertClickCount(t, db, bindingID, 0)

	responses := make(chan *httptest.ResponseRecorder, 2)
	ready := make(chan struct{})
	for range 2 {
		go func() {
			<-ready
			responses <- authSessionRequest(router, "POST", "/api/auth/refresh", "", passwordCookies[1])
		}()
	}
	close(ready)
	first, second := <-responses, <-responses
	if first.Code == http.StatusConflict {
		first, second = second, first
	}
	fresh := expectSessionCookies(t, first, http.StatusNoContent)
	if second.Code != http.StatusConflict || len(second.Result().Cookies()) != 0 ||
		!strings.Contains(second.Body.String(), "REFRESH_CONFLICT") {
		t.Fatalf("同一哈希并发轮换应只允许一个请求签发凭证: %d %s", second.Code, second.Body)
	}
	if fresh[1].Value == passwordCookies[1].Value || !fresh[1].Expires.Equal(expiresAt) {
		t.Fatal("刷新未轮换凭证或改变了原截止时间")
	}
	var seenLoginAt time.Time
	if err := db.QueryRow("SELECT last_login_at FROM users WHERE id = ?", userID).Scan(&seenLoginAt); err != nil || !seenLoginAt.Equal(loginAt) {
		t.Fatalf("刷新改变了成功登录时间: %v，%v", seenLoginAt, err)
	}
	if retry := authSessionRequest(router, "POST", clicksPath, clicks, fresh[0]); retry.Code != http.StatusOK {
		t.Fatalf("刷新后点击写入失败: %d %s", retry.Code, retry.Body)
	}
	assertClickCount(t, db, bindingID, 6)

	// 把旧凭证移出并发窗口，验证再次使用将撤销包括新凭证在内的会话。
	exec("UPDATE auth_refresh_tokens SET rotated_at = UTC_TIMESTAMP(6) - INTERVAL 11 SECOND WHERE token_hash = ?", refreshHash[:])
	if replay := authSessionRequest(router, "POST", "/api/auth/refresh", "", passwordCookies[1]); replay.Code != http.StatusUnauthorized {
		t.Fatalf("旧凭证重用未拒绝: %d %s", replay.Code, replay.Body)
	}
	var revoked bool
	if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM auth_sessions WHERE id = ?", sessionID).Scan(&revoked); err != nil || !revoked {
		t.Fatalf("重用后未撤销会话: %v", err)
	}
	if refresh := authSessionRequest(router, "POST", "/api/auth/refresh", "", fresh[1]); refresh.Code != http.StatusUnauthorized {
		t.Fatal("已撤销会话仍能刷新")
	}

	cfg.Auth.AccessTokenTTL = 15 * time.Minute
	router = NewRouter(cfg, db)
	qrBody := fmt.Sprintf(`{"text":%q}`, qrText)
	qrCookies := expectSessionCookies(t, authSessionRequest(router, "POST", "/api/auth/qr-login", qrBody), http.StatusOK)
	if me := authSessionRequest(router, "GET", "/api/auth/me", "", qrCookies[0]); me.Code != http.StatusOK {
		t.Fatalf("扫码登录后 /me 失败: %d %s", me.Code, me.Body)
	}
	if err := db.QueryRow("SELECT last_login_at FROM users WHERE id = ?", userID).Scan(&seenLoginAt); err != nil || !seenLoginAt.After(loginAt) {
		t.Fatalf("扫码登录未更新成功登录时间: %v，%v", seenLoginAt, err)
	}

	// 用近截止的数据库会话验证刷新后的访问和刷新 Cookie 均截断到相同时间。
	capAt := time.Now().UTC().Truncate(time.Second).Add(45 * time.Second)
	qrHash := sha256.Sum256([]byte(qrCookies[1].Value))
	exec("UPDATE auth_sessions SET expires_at = ? WHERE id = (SELECT session_id FROM auth_refresh_tokens WHERE token_hash = ?)", capAt, qrHash[:])
	capped := expectSessionCookies(t, authSessionRequest(router, "POST", "/api/auth/refresh", "", qrCookies[1]), http.StatusNoContent)
	if !capped[0].Expires.Equal(capAt) || !capped[1].Expires.Equal(capAt) {
		t.Fatal("刷新后的 JWT 或 Cookie 超出固定会话截止时间")
	}

	// 退出只携带已轮换的刷新凭证，模拟另一个标签页同时提交了刷新。
	cleared := expectSessionCookies(t, authSessionRequest(router, "POST", "/api/auth/logout", "", qrCookies[1]), http.StatusOK)
	if cleared[0].MaxAge >= 0 || cleared[1].MaxAge >= 0 ||
		authSessionRequest(router, "GET", "/api/auth/me", "", capped[0]).Code != http.StatusUnauthorized ||
		authSessionRequest(router, "POST", "/api/auth/refresh", "", capped[1]).Code != http.StatusUnauthorized {
		t.Fatal("退出未清除两个 Cookie 或未撤销尚未到期的 JWT/刷新凭证")
	}

	expiring := expectSessionCookies(t, authSessionRequest(router, "POST", "/api/auth/qr-login", qrBody), http.StatusOK)
	expiringHash := sha256.Sum256([]byte(expiring[1].Value))
	exec("UPDATE auth_sessions SET expires_at = UTC_TIMESTAMP(6) WHERE id = (SELECT session_id FROM auth_refresh_tokens WHERE token_hash = ?)", expiringHash[:])
	if authSessionRequest(router, "POST", "/api/auth/refresh", "", expiring[1]).Code != http.StatusUnauthorized {
		t.Fatal("达到会话截止时间仍可续期")
	}

	oldJWT, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "fluffy-cupcake", "sub": strconv.FormatInt(userID, 10), "user_id": userID, "username": username,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	legacy := &http.Cookie{Name: "auth", Value: oldJWT}
	if authSessionRequest(router, "GET", "/api/auth/me", "", legacy).Code != http.StatusUnauthorized ||
		authSessionRequest(router, "POST", "/api/auth/refresh", "", legacy).Code != http.StatusUnauthorized {
		t.Fatal("旧版 JWT 仍可访问或兑换新会话")
	}
}

// authSessionRequest 用指定 Cookie 执行一次实际 Gin 路由请求。
func authSessionRequest(router http.Handler, method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)
	return response
}

// expectSessionCookies 核对凭证签发状态并返回两个 Cookie。
func expectSessionCookies(t *testing.T, response *httptest.ResponseRecorder, status int) []*http.Cookie {
	t.Helper()
	cookies := response.Result().Cookies()
	if response.Code != status || len(cookies) != 2 {
		t.Fatalf("状态码 = %d，Cookie 数量 = %d，响应 = %s", response.Code, len(cookies), response.Body)
	}
	return cookies
}

// assertClickCount 从数据库验证受保护写入累计次数。
func assertClickCount(t *testing.T, db *sql.DB, bindingID, want int64) {
	t.Helper()
	var count int64
	err := db.QueryRow("SELECT COALESCE(SUM(click_count), 0) FROM button_click_minutes WHERE companion_binding_id = ?", bindingID).Scan(&count)
	if err != nil || count != want {
		t.Fatalf("点击累计 = %d，期望 %d (%v)", count, want, err)
	}
}
