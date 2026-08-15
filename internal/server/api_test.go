package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// TestAuthAndClickHTTPFlow 验证登录 Cookie、鉴权 Context、点击写入和统计读取的 HTTP 链路。
func TestAuthAndClickHTTPFlow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, username, password_hash, last_login_at, created_at, updated_at
		FROM users
		WHERE username = ?
	`)).WithArgs("tester").WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "last_login_at", "created_at", "updated_at"}).
		AddRow(int64(1), "tester", string(hash), nil, now, now))
	mock.ExpectExec("UPDATE users SET last_login_at").WithArgs(sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))

	cfg := config.Config{
		Mode: "test",
		Auth: config.AuthConfig{
			JWTSecret:  "test-secret-with-at-least-32-characters",
			JWTExpire:  time.Hour,
			CookieName: "auth",
		},
	}
	router := NewRouter(cfg, db)
	login := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"tester","password":"correct-password"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(login, loginRequest)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Set-Cookie 数量 = %d", len(cookies))
	}
	authCookie := cookies[0]
	if !authCookie.HttpOnly || authCookie.Secure || authCookie.SameSite != http.SameSiteLaxMode || authCookie.Path != "/" {
		t.Fatalf("认证 Cookie 属性 = %#v", authCookie)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, username, password_hash, last_login_at, created_at, updated_at
		FROM users
		WHERE id = ?
	`)).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "username", "password_hash", "last_login_at", "created_at", "updated_at"}).
		AddRow(int64(1), "tester", string(hash), now, now, now))
	me := httptest.NewRecorder()
	meRequest := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meRequest.AddCookie(authCookie)
	router.ServeHTTP(me, meRequest)
	if me.Code != http.StatusOK || !bytes.Contains(me.Body.Bytes(), []byte(`"username":"tester"`)) || bytes.Contains(me.Body.Bytes(), []byte("password_hash")) {
		t.Fatalf("me status = %d, body = %s", me.Code, me.Body.String())
	}

	expectActiveBinding(mock, 1, 9, 2, "partner", "亲爱的")
	mock.ExpectExec("INSERT INTO button_click_minutes").
		WithArgs("yanlili", sqlmock.AnyArg(), int64(6), int64(1), int64(9), int64(2), int64(6)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	click := httptest.NewRecorder()
	clickRequest := httptest.NewRequest(http.MethodPost, "/api/yanlili/clicks", bytes.NewBufferString(`{"count":6}`))
	clickRequest.Header.Set("Content-Type", "application/json")
	clickRequest.AddCookie(authCookie)
	router.ServeHTTP(click, clickRequest)
	if click.Code != http.StatusOK {
		t.Fatalf("click status = %d, body = %s", click.Code, click.Body.String())
	}

	expectActiveBinding(mock, 1, 9, 2, "partner", "亲爱的")
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(click_count\\), 0\\)").WithArgs(int64(9), int64(1), int64(2), "yanlili").
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(6)))
	mock.ExpectQuery("SELECT DATE_FORMAT\\(click_date").WithArgs(int64(9), int64(1), int64(2), "yanlili", 8).
		WillReturnRows(sqlmock.NewRows([]string{"click_date", "count"}).AddRow("2026-08-15", int64(6)))
	mock.ExpectQuery("SELECT minute_bucket, click_count").WithArgs(int64(9), int64(1), int64(2), "yanlili", 8).
		WillReturnRows(sqlmock.NewRows([]string{"minute_bucket", "count"}).AddRow(time.Date(2026, 8, 15, 3, 27, 0, 0, time.UTC), int64(6)))
	stats := httptest.NewRecorder()
	statsRequest := httptest.NewRequest(http.MethodGet, "/api/yanlili/clicks/stats", nil)
	statsRequest.AddCookie(authCookie)
	router.ServeHTTP(stats, statsRequest)
	if stats.Code != http.StatusOK || !bytes.Contains(stats.Body.Bytes(), []byte(`"total_count":6`)) {
		t.Fatalf("stats status = %d, body = %s", stats.Code, stats.Body.String())
	}

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/yanlili/clicks/stats", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("未认证 stats status = %d", unauthorized.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectActiveBinding(mock sqlmock.Sqlmock, userID, bindingID, partnerID int64, username, note string) {
	mock.ExpectQuery("SELECT membership.binding_id, membership.partner_user_id, partner.username").
		WithArgs(userID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"binding_id", "partner_user_id", "username", "note"}).AddRow(bindingID, partnerID, username, note))
}
