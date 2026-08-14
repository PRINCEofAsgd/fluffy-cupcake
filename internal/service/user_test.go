package service

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type capturingUserWriter struct {
	hashes []string
}

func (w *capturingUserWriter) Create(_ context.Context, _ string, passwordHash string) (int64, error) {
	w.hashes = append(w.hashes, passwordHash)
	return int64(len(w.hashes)), nil
}

// TestUserServiceHashesPasswords 验证明文不会交给 Repository，且随机盐使同密码哈希不同但都可验证。
func TestUserServiceHashesPasswords(t *testing.T) {
	writer := &capturingUserWriter{}
	users := NewUserService(writer)
	const password = "same-secure-password"
	for _, username := range []string{"user-one", "user-two"} {
		if _, err := users.Create(context.Background(), username, password); err != nil {
			t.Fatal(err)
		}
	}
	if len(writer.hashes) != 2 || writer.hashes[0] == password || writer.hashes[1] == password {
		t.Fatalf("Repository 未收到两个非明文哈希")
	}
	if writer.hashes[0] == writer.hashes[1] {
		t.Fatal("相同密码的两次 bcrypt 哈希不应相同")
	}
	for _, hash := range writer.hashes {
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
			t.Fatalf("bcrypt 哈希无法验证原密码: %v", err)
		}
	}
}
