package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type fakeAuthUsers struct {
	user model.User
}

func (f fakeAuthUsers) UpdateLastLogin(_ context.Context, _ int64, _ time.Time) error { return nil }

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

// TestAuthLoginAndToken 验证正确密码登录、JWT 签发解析和错误密码统一失败。
func TestAuthLoginAndToken(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	users := fakeAuthUsers{user: model.User{ID: 7, Username: "yanlili", PasswordHash: string(hash)}}
	auth := NewAuthService(users, fakeQrMappings{}, "test-secret-with-at-least-32-characters", time.Hour)

	token, expiresAt, err := auth.Login(context.Background(), "yanlili", "correct-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token == "" || expiresAt.Before(time.Now()) {
		t.Fatal("登录未返回有效 Token 或过期时间")
	}
	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
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
		if _, _, err := auth.Login(context.Background(), testCase.username, testCase.password); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Login(%q) error = %v，期望 ErrInvalidCredentials", testCase.username, err)
		}
	}
}

// TestAuthLoginByQrText 验证二维码文本成功登录、未映射文本统一失败。
func TestAuthLoginByQrText(t *testing.T) {
	users := fakeAuthUsers{user: model.User{ID: 11, Username: "coach"}}
	mappings := fakeQrMappings{qrText: "终身卡文本", user: users.user}
	auth := NewAuthService(users, mappings, "test-secret-with-at-least-32-characters", time.Hour)

	token, expiresAt, err := auth.LoginByQrText(context.Background(), " 终身卡文本 ")
	if err != nil {
		t.Fatalf("LoginByQrText() error = %v", err)
	}
	if token == "" || expiresAt.Before(time.Now()) {
		t.Fatal("登录未返回有效 Token 或过期时间")
	}
	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != 11 || claims.Username != "coach" {
		t.Fatalf("claims = %#v", claims)
	}

	if _, _, err := auth.LoginByQrText(context.Background(), "未映射的文本"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("LoginByQrText() error = %v，期望 ErrInvalidCredentials", err)
	}
}

// TestAuthRejectsExpiredAndInvalidToken 验证过期 Token 与篡改 Token 均被拒绝。
func TestAuthRejectsExpiredAndInvalidToken(t *testing.T) {
	users := fakeAuthUsers{user: model.User{ID: 9, Username: "user"}}
	auth := NewAuthService(users, fakeQrMappings{}, "test-secret-with-at-least-32-characters", time.Hour)
	auth.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	expired, _, err := auth.issueToken(users.user)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.ParseToken(expired); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("过期 Token error = %v", err)
	}
	if _, err := auth.ParseToken(expired + "tampered"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("篡改 Token error = %v", err)
	}
}
