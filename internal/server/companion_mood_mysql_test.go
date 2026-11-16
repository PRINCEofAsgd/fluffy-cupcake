// companion_mood_mysql_test.go 在独立 MySQL 测试库验证心情接口和真实的绑定生命周期。
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/database"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// TestCompanionMoodMySQL 验证默认视角、独立写入、清空、越权与解绑重绑后的隔离。
func TestCompanionMoodMySQL(t *testing.T) {
	dsn := os.Getenv("FLUFFY_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("请在已执行 Migration 8 的独立测试库设置 FLUFFY_TEST_MYSQL_DSN")
	}
	db, err := database.Open(t.Context(), config.DatabaseConfig{
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

	// 只清理由本测试生成的关系与用户，允许在已有专用库重复验收。
	var usernames []string
	var bindingIDs []int64
	defer func() {
		for _, id := range bindingIDs {
			for _, query := range []string{
				"DELETE FROM companion_moods WHERE binding_id = ?",
				"DELETE FROM companion_active_memberships WHERE binding_id = ?",
				"DELETE FROM companion_bindings WHERE id = ?",
			} {
				if _, err := db.Exec(query, id); err != nil {
					t.Error(err)
				}
			}
		}
		for _, name := range usernames {
			if _, err := db.Exec("DELETE FROM users WHERE username = ?", name); err != nil {
				t.Error(err)
			}
		}
	}()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	router := NewRouter(config.Config{Mode: "test", Auth: config.AuthConfig{
		JWTSecret: "mysql-mood-integration-secret-32-characters", AccessTokenTTL: 15 * time.Minute,
		SessionTTL: 360 * time.Hour, CookieName: "auth",
	}}, db)
	const moodPath = "/api/yanlili/mood"
	// request 核对每条 HTTP 响应的状态码，并返回响应供具体断言。
	request := func(method, path, body string, status int, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		response := authSessionRequest(router, method, path, body, cookies...)
		if response.Code != status {
			t.Fatalf("%s %s 状态码 = %d，期望 %d，响应 = %s", method, path, response.Code, status, response.Body)
		}
		return response
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	// login 创建独立账号，并通过真实密码登录取得访问 Cookie。
	login := func(prefix string) (string, *http.Cookie) {
		t.Helper()
		name := prefix + "-" + suffix
		if _, err := db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", name, hash); err != nil {
			t.Fatal(err)
		}
		usernames = append(usernames, name)
		body := fmt.Sprintf(`{"username":%q,"password":"correct-password"}`, name)
		return name, expectSessionCookies(t, authSessionRequest(router, "POST", "/api/auth/login", body), http.StatusOK)[0]
	}
	_, alice := login("mood-a")
	bobName, bob := login("mood-b")
	_, outsider := login("mood-c")
	// bind 使用现有邀请和接受接口建立新关系。
	bind := func() int64 {
		t.Helper()
		var invite struct{ ID int64 }
		response := request("POST", "/api/companion/invitations", fmt.Sprintf(`{"username":%q}`, bobName), http.StatusCreated, alice)
		if err := json.Unmarshal(response.Body.Bytes(), &invite); err != nil || invite.ID == 0 {
			t.Fatalf("绑定邀请响应不含 ID: %s (%v)", response.Body, err)
		}
		bindingIDs = append(bindingIDs, invite.ID)
		request("POST", fmt.Sprintf("/api/companion/bindings/%d/accept", invite.ID), "", http.StatusOK, bob)
		return invite.ID
	}
	// assertView 比较完整快照并确认鉴权中间件禁止缓存。
	assertView := func(response *httptest.ResponseRecorder, id int64, mood model.CompanionMood) {
		t.Helper()
		var view model.CompanionMoodView
		if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil || view.BindingID != id || !reflect.DeepEqual(view.CompanionMood, mood) {
			t.Fatalf("快照 = %+v，期望绑定 %d 与 %+v (%v)", view, id, mood, err)
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("心情数据必须禁止缓存")
		}
	}
	// assertConflict 区分无绑定读取与过期编辑。
	assertConflict := func(response *httptest.ResponseRecorder, code string) {
		t.Helper()
		if !strings.Contains(response.Body.String(), fmt.Sprintf(`"code":%q`, code)) {
			t.Fatalf("响应缺少 %s: %s", code, response.Body)
		}
	}

	empty := model.CompanionMood{Statuses: []string{}, Reasons: []string{}, Comforts: []string{}}
	request("GET", moodPath, "", http.StatusUnauthorized)
	request("PUT", moodPath, `{}`, http.StatusUnauthorized)
	assertConflict(request("GET", moodPath, "", http.StatusConflict, alice), "MOOD_NOT_BOUND")
	id := bind()
	assertView(request("GET", moodPath, "", http.StatusOK, alice), id, empty)
	request("GET", moodPath+"?direction=invalid", "", http.StatusBadRequest, alice)
	payload := fmt.Sprintf(`{"binding_id":%d,"statuses":["tired","hungry"],"reasons":["work"],"reason_custom":" 加班有点累 ","comforts":["company","milk_tea"],"comfort_custom":"抱一下"}`, id)
	aliceMood := model.CompanionMood{
		Statuses: []string{"tired", "hungry"}, Reasons: []string{"work"}, ReasonCustom: "加班有点累",
		Comforts: []string{"company", "milk_tea"}, ComfortCustom: "抱一下",
	}
	for _, invalid := range []string{
		strings.Replace(payload, `"statuses"`, `"user_id":123,"statuses"`, 1),
		strings.Replace(payload, `"work"`, `"other"`, 1),
		strings.Replace(payload, `"抱一下"`, `"下班想喝奶茶"`, 1),
		payload + `{}`,
		payload + strings.Repeat(" ", 4096),
	} {
		request("PUT", moodPath, invalid, http.StatusBadRequest, alice)
	}
	assertView(request("GET", moodPath+"?direction=mine", "", http.StatusOK, alice), id, empty)
	assertConflict(request("PUT", moodPath, payload, http.StatusConflict, outsider), "MOOD_BINDING_CHANGED")
	assertConflict(request("GET", moodPath, "", http.StatusConflict, outsider), "MOOD_NOT_BOUND")
	assertView(request("PUT", moodPath, payload, http.StatusOK, alice), id, aliceMood)
	assertView(request("PUT", moodPath, payload, http.StatusOK, alice), id, aliceMood)
	assertView(request("GET", moodPath, "", http.StatusOK, bob), id, aliceMood)
	assertView(request("GET", moodPath, "", http.StatusOK, alice), id, empty)
	assertView(request("GET", moodPath+"?direction=mine", "", http.StatusOK, alice), id, aliceMood)

	bobMood := model.CompanionMood{Statuses: []string{"relaxed"}, Reasons: []string{}, Comforts: []string{}}
	assertView(request("PUT", moodPath, fmt.Sprintf(`{"binding_id":%d,"statuses":["relaxed"],"reasons":[],"comforts":[]}`, id), http.StatusOK, bob), id, bobMood)
	assertView(request("GET", moodPath, "", http.StatusOK, alice), id, bobMood)
	assertView(request("PUT", moodPath, fmt.Sprintf(`{"binding_id":%d,"statuses":[],"reasons":[],"comforts":[]}`, id), http.StatusOK, bob), id, empty)
	assertView(request("GET", moodPath, "", http.StatusOK, alice), id, empty)
	assertView(request("GET", moodPath, "", http.StatusOK, bob), id, aliceMood)

	bindingPath := fmt.Sprintf("/api/companion/bindings/%d", id)
	request("POST", bindingPath+"/unbind-request", `{}`, http.StatusOK, alice)
	assertView(request("GET", moodPath, "", http.StatusOK, bob), id, aliceMood)
	request("POST", bindingPath+"/unbind-accept", "", http.StatusOK, bob)
	assertConflict(request("GET", moodPath, "", http.StatusConflict, bob), "MOOD_NOT_BOUND")
	assertConflict(request("PUT", moodPath, payload, http.StatusConflict, alice), "MOOD_BINDING_CHANGED")
	newID := bind()
	assertConflict(request("PUT", moodPath, payload, http.StatusConflict, alice), "MOOD_BINDING_CHANGED")
	assertView(request("GET", moodPath, "", http.StatusOK, bob), newID, empty)
	assertView(request("GET", moodPath+"?direction=mine", "", http.StatusOK, alice), newID, empty)
}
