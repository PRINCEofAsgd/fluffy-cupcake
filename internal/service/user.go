package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidUsername = errors.New("用户名长度必须为 1 到 64 个字符")
	ErrInvalidPassword = errors.New("密码长度必须为 8 到 72 个字节")
)

// UserWriter 描述人工创建用户所需的写入能力。
type UserWriter interface {
	Create(ctx context.Context, username, passwordHash string) (int64, error)
}

// UserService 负责人工用户创建时的输入规则和 bcrypt 哈希。
type UserService struct {
	users UserWriter
}

// NewUserService 创建固定用户管理服务；当前仅供本地 CLI 使用。
func NewUserService(users UserWriter) *UserService {
	return &UserService{users: users}
}

// Create 对明文密码执行 bcrypt 后立即丢弃，仅把哈希交给 Repository。
func (s *UserService) Create(ctx context.Context, username, password string) (int64, error) {
	username = strings.TrimSpace(username)
	if username == "" || utf8.RuneCountInString(username) > 64 {
		return 0, ErrInvalidUsername
	}
	if len(password) < 8 || len(password) > 72 {
		return 0, ErrInvalidPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("生成密码哈希: %w", err)
	}
	id, err := s.users.Create(ctx, username, string(hash))
	if err != nil {
		if errors.Is(err, repository.ErrUsernameExists) {
			return 0, repository.ErrUsernameExists
		}
		return 0, fmt.Errorf("保存用户: %w", err)
	}
	return id, nil
}
