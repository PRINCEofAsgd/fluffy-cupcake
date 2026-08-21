// Package service 实现鉴权、点击聚合等业务规则，不直接执行 SQL。
package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials 对不存在用户和错误密码使用同一错误，避免账号枚举。
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	// ErrInvalidToken 统一表示缺失业务声明、非法签名或过期 JWT。
	ErrInvalidToken = errors.New("认证信息无效或已过期")
)

// AuthUserRepository 描述鉴权服务需要的用户查询能力。
type AuthUserRepository interface {
	GetByUsername(ctx context.Context, username string) (model.User, error)
	GetByID(ctx context.Context, id int64) (model.User, error)
	UpdateLastLogin(ctx context.Context, id int64, loginAt time.Time) error
}

// QrLoginRepository 描述二维码登录需要的映射查询能力。
type QrLoginRepository interface {
	GetUserByQrText(ctx context.Context, qrText string) (model.User, error)
}

// AuthClaims 是 JWT 中经过签名保护的用户身份和标准时间声明。
type AuthClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthService 负责密码哈希比较、JWT 签发解析和当前用户读取。
type AuthService struct {
	users  AuthUserRepository
	qr     QrLoginRepository
	secret []byte
	expire time.Duration
	now    func() time.Time
}

// NewAuthService 创建使用 HS256 和指定短期有效期的鉴权服务。
func NewAuthService(users AuthUserRepository, qr QrLoginRepository, secret string, expire time.Duration) *AuthService {
	return &AuthService{users: users, qr: qr, secret: []byte(secret), expire: expire, now: time.Now}
}

// Login 使用 bcrypt 比较密码，成功时签发只包含必要身份字段的短期 JWT。
func (s *AuthService) Login(ctx context.Context, username, password string) (string, time.Time, error) {
	user, err := s.users.GetByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", time.Time{}, ErrInvalidCredentials
		}
		return "", time.Time{}, fmt.Errorf("读取登录用户: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}
	if err := s.users.UpdateLastLogin(ctx, user.ID, s.now().UTC()); err != nil {
		return "", time.Time{}, fmt.Errorf("记录最后登录时间: %w", err)
	}
	return s.issueToken(user)
}

// LoginByQrText 使用二维码永久文本查表映射用户，成功后更新最后登录时间并签发 JWT。
func (s *AuthService) LoginByQrText(ctx context.Context, qrText string) (string, time.Time, error) {
	user, err := s.qr.GetUserByQrText(ctx, strings.TrimSpace(qrText))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", time.Time{}, ErrInvalidCredentials
		}
		return "", time.Time{}, fmt.Errorf("读取二维码登录用户: %w", err)
	}
	if err := s.users.UpdateLastLogin(ctx, user.ID, s.now().UTC()); err != nil {
		return "", time.Time{}, fmt.Errorf("记录最后登录时间: %w", err)
	}
	return s.issueToken(user)
}

// issueToken 生成带 iat、exp、sub、user_id 和 username 的 HS256 JWT。
func (s *AuthService) issueToken(user model.User) (string, time.Time, error) {
	now := s.now().UTC()
	expiresAt := now.Add(s.expire)
	claims := AuthClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "fluffy-cupcake",
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("签发 JWT: %w", err)
	}
	return signed, expiresAt, nil
}

// ParseToken 严格限制 HS256、签发者和有效期，并返回可信身份声明。
func (s *AuthService) ParseToken(raw string) (AuthClaims, error) {
	claims := AuthClaims{}
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("fluffy-cupcake"), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.UserID <= 0 || claims.Username == "" || claims.IssuedAt == nil || claims.Subject != strconv.FormatInt(claims.UserID, 10) {
		return AuthClaims{}, ErrInvalidToken
	}
	return claims, nil
}

// CurrentUser 根据 Middleware 提供的可信 user_id 读取当前用户基本信息。
func (s *AuthService) CurrentUser(ctx context.Context, userID int64) (model.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return model.User{}, ErrInvalidToken
	}
	if err != nil {
		return model.User{}, fmt.Errorf("读取当前用户: %w", err)
	}
	return user, nil
}
