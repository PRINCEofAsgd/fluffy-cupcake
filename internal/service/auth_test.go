package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type fakeAuthUsers struct {
	user model.User
}

func (f fakeAuthUsers) GetByUsername(_ context.Context, username string) (model.User, error) {
	if username != f.user.Username {
		return model.User{}, repository.ErrNotFound
	}
	return f.user, nil
}

func (f fakeAuthUsers) GetByID(_ context.Context, id int64) (model.User, error) {
	if id != f.user.ID {
		return model.User{}, repository.ErrNotFound
	}
	return f.user, nil
}

type fakeQrMappings struct {
	qrText string
	user   model.User
}

func (f fakeQrMappings) GetUserByQrText(_ context.Context, qrText string) (model.User, error) {
	if qrText != f.qrText {
		return model.User{}, repository.ErrNotFound
	}
	return f.user, nil
}

// loginSessionStore 捕获登录写入，验证两个入口都创建关联的会话与哈希。
type loginSessionStore struct {
	AuthSessions
	session model.AuthSession
	hash    [sha256.Size]byte
	loginAt time.Time
}

// Create 记录会话写入参数。
func (f *loginSessionStore) Create(_ context.Context, session model.AuthSession, hash [sha256.Size]byte, now time.Time) error {
	f.session, f.hash, f.loginAt = session, hash, now
	return nil
}

// TestAuthLoginAndToken 验证正确密码登录、JWT 签发解析和错误密码统一失败。
func TestAuthLoginAndToken(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	users := fakeAuthUsers{user: model.User{ID: 7, Username: "yanlili", PasswordHash: string(hash)}}
	store := &loginSessionStore{}
	auth := NewAuthService(users, fakeQrMappings{}, store, "test-secret-with-at-least-32-characters", 15*time.Minute, 360*time.Hour)

	credentials, err := auth.Login(context.Background(), "yanlili", "correct-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if credentials.AccessToken == "" || credentials.AccessExpiresAt.Before(time.Now()) {
		t.Fatal("登录未返回有效 Token 或过期时间")
	}
	claims, err := auth.ParseToken(credentials.AccessToken)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.SessionID != store.session.ID || store.hash != sha256.Sum256([]byte(credentials.RefreshToken)) ||
		!credentials.SessionExpiresAt.Equal(store.loginAt.Add(360*time.Hour)) ||
		!credentials.AccessExpiresAt.Equal(store.loginAt.Add(15*time.Minute)) {
		t.Fatal("密码登录未保存会话或有效期不正确")
	}
	if claims.UserID != 7 || claims.Username != "yanlili" {
		t.Fatalf("claims = %#v", claims)
	}

	for _, testCase := range []struct {
		username string
		password string
	}{
		{username: "yanlili", password: "wrong-password"},
		{username: "missing", password: "correct-password"},
	} {
		if _, err := auth.Login(context.Background(), testCase.username, testCase.password); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Login(%q) error = %v，期望 ErrInvalidCredentials", testCase.username, err)
		}
	}
}

// TestAuthLoginByQrText 验证二维码文本成功登录、未映射文本统一失败。
func TestAuthLoginByQrText(t *testing.T) {
	users := fakeAuthUsers{user: model.User{ID: 11, Username: "coach"}}
	mappings := fakeQrMappings{qrText: "终身卡文本", user: users.user}
	store := &loginSessionStore{}
	auth := NewAuthService(users, mappings, store, "test-secret-with-at-least-32-characters", 15*time.Minute, 360*time.Hour)

	credentials, err := auth.LoginByQrText(context.Background(), " 终身卡文本 ")
	if err != nil {
		t.Fatalf("LoginByQrText() error = %v", err)
	}
	if credentials.AccessToken == "" || credentials.AccessExpiresAt.Before(time.Now()) {
		t.Fatal("登录未返回有效 Token 或过期时间")
	}
	claims, err := auth.ParseToken(credentials.AccessToken)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if store.session.UserID != 11 || claims.SessionID != store.session.ID || store.hash != sha256.Sum256([]byte(credentials.RefreshToken)) {
		t.Fatal("扫码登录未保存会话")
	}
	if claims.UserID != 11 || claims.Username != "coach" {
		t.Fatalf("claims = %#v", claims)
	}

	if _, err := auth.LoginByQrText(context.Background(), "未映射的文本"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("LoginByQrText() error = %v，期望 ErrInvalidCredentials", err)
	}
}

// TestAuthRejectsExpiredInvalidAndLegacyToken 验证到期、篡改及上线前的 JWT 一律被拒绝。
func TestAuthRejectsExpiredInvalidAndLegacyToken(t *testing.T) {
	auth := NewAuthService(nil, nil, nil, "test-secret-with-at-least-32-characters", 15*time.Minute, 360*time.Hour)
	now := time.Now().UTC().Truncate(time.Second)
	auth.now = func() time.Time { return now }
	user := model.AuthSession{ID: randomCredential(16), UserID: 9, Username: "user", ExpiresAt: now.Add(time.Hour)}
	expired, err := auth.issueCredentials(user, randomCredential(32), now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.ParseToken(expired.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("过期 Token error = %v", err)
	}
	user.ExpiresAt = now.Add(20 * time.Second)
	valid, err := auth.issueCredentials(user, randomCredential(32), now)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ParseToken(valid.AccessToken)
	if err != nil || !claims.ExpiresAt.Time.Equal(user.ExpiresAt) {
		t.Fatalf("JWT 未限制到会话截止时间: %v, %v", claims.ExpiresAt, err)
	}
	if _, err := auth.ParseToken(valid.AccessToken + "tampered"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("篡改 Token error = %v", err)
	}
	claims.SessionID = ""
	claims.Subject = strconv.FormatInt(user.UserID, 10)
	legacy, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(auth.secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.ParseToken(legacy); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("旧版 Token error = %v", err)
	}
}
