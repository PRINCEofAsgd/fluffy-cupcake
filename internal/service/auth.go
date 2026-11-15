// auth.go 负责密码和扫码登录、固定截止的会话及访问 JWT。
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

const refreshConflictGrace = 10 * time.Second

var (
	// ErrInvalidCredentials 对不存在用户和错误密码使用同一错误，避免账号枚举。
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	// ErrInvalidToken 表示访问令牌无效或会话已失效。
	ErrInvalidToken = errors.New("认证信息无效或已过期")
	// ErrInvalidSession 表示刷新凭证无法恢复会话，前端应提示重新登录。
	ErrInvalidSession = errors.New("登录已失效，请重新登录")
	// ErrRefreshConflict 表示另一请求刚完成轮换，前端可以等待浏览器接收新 Cookie。
	ErrRefreshConflict = errors.New("正在其他请求中恢复登录")
)

// AuthUserRepository 描述鉴权服务需要的用户查询能力。
type AuthUserRepository interface {
	GetByUsername(ctx context.Context, username string) (model.User, error)
	GetByID(ctx context.Context, id int64) (model.User, error)
}

// QrLoginRepository 描述二维码登录需要的映射查询能力。
type QrLoginRepository interface {
	GetUserByQrText(ctx context.Context, qrText string) (model.User, error)
}

// AuthSessions 描述会话持久化、轮换和撤销能力。
type AuthSessions interface {
	Create(ctx context.Context, session model.AuthSession, hash [sha256.Size]byte, now time.Time) error
	Active(ctx context.Context, id string, userID int64, now time.Time) (bool, error)
	Rotate(ctx context.Context, oldHash, newHash [sha256.Size]byte, clock func() time.Time, grace time.Duration) (model.AuthSession, error)
	RevokeByRefresh(ctx context.Context, hash [sha256.Size]byte, now time.Time) error
	Revoke(ctx context.Context, id string, now time.Time) error
}

// AuthClaims 是访问 JWT 中经签名保护的会话和用户身份。
type AuthClaims struct {
	SessionID string `json:"sid"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	jwt.RegisteredClaims
}

// AuthCredentials 是交给 HTTP 层写入 Cookie 的凭证对。
type AuthCredentials struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	SessionExpiresAt time.Time
}

// AuthService 负责身份校验、凭证签发与会话生命周期。
type AuthService struct {
	users      AuthUserRepository
	qr         QrLoginRepository
	sessions   AuthSessions
	secret     []byte
	accessTTL  time.Duration
	sessionTTL time.Duration
	now        func() time.Time
}

// NewAuthService 创建使用 HS256 和独立访问、会话时长的鉴权服务。
func NewAuthService(users AuthUserRepository, qr QrLoginRepository, sessions AuthSessions, secret string, accessTTL, sessionTTL time.Duration) *AuthService {
	return &AuthService{
		users: users, qr: qr, sessions: sessions, secret: []byte(secret),
		accessTTL: accessTTL, sessionTTL: sessionTTL, now: time.Now,
	}
}

// Login 使用 bcrypt 校验密码，成功时建立新会话。
func (s *AuthService) Login(ctx context.Context, username, password string) (AuthCredentials, error) {
	user, err := s.users.GetByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return AuthCredentials{}, ErrInvalidCredentials
		}
		return AuthCredentials{}, fmt.Errorf("读取登录用户: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return AuthCredentials{}, ErrInvalidCredentials
	}
	return s.startSession(ctx, user)
}

// LoginByQrText 按二维码文本查找用户并复用密码登录的会话创建流程。
func (s *AuthService) LoginByQrText(ctx context.Context, qrText string) (AuthCredentials, error) {
	user, err := s.qr.GetUserByQrText(ctx, strings.TrimSpace(qrText))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return AuthCredentials{}, ErrInvalidCredentials
		}
		return AuthCredentials{}, fmt.Errorf("读取二维码登录用户: %w", err)
	}
	return s.startSession(ctx, user)
}

// startSession 在首次签发时确定截止时间，随后原子保存会话和成功登录时间。
func (s *AuthService) startSession(ctx context.Context, user model.User) (AuthCredentials, error) {
	now := s.now().UTC().Truncate(time.Second)
	session := model.AuthSession{
		ID: randomCredential(16), UserID: user.ID, Username: user.Username, ExpiresAt: now.Add(s.sessionTTL),
	}
	refresh := randomCredential(32)
	credentials, err := s.issueCredentials(session, refresh, now)
	if err != nil {
		return AuthCredentials{}, err
	}
	if err := s.sessions.Create(ctx, session, sha256.Sum256([]byte(refresh)), now); err != nil {
		return AuthCredentials{}, err
	}
	return credentials, nil
}

// Refresh 仅使用刷新凭证，轮换成功后沿用原会话截止时间。
func (s *AuthService) Refresh(ctx context.Context, refresh string) (AuthCredentials, error) {
	if !validCredential(refresh, 32) {
		return AuthCredentials{}, ErrInvalidSession
	}
	next := randomCredential(32)
	session, err := s.sessions.Rotate(ctx, sha256.Sum256([]byte(refresh)), sha256.Sum256([]byte(next)), s.now, refreshConflictGrace)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return AuthCredentials{}, ErrInvalidSession
	case errors.Is(err, repository.ErrRefreshConflict):
		return AuthCredentials{}, ErrRefreshConflict
	case err != nil:
		return AuthCredentials{}, err
	}
	return s.issueCredentials(session, next, s.now().UTC())
}

// Logout 优先使用刷新凭证定位会话，缺失或未知时使用仍有效的访问 JWT。
func (s *AuthService) Logout(ctx context.Context, refresh, access string) error {
	now := s.now().UTC()
	if validCredential(refresh, 32) {
		err := s.sessions.RevokeByRefresh(ctx, sha256.Sum256([]byte(refresh)), now)
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
	}
	claims, err := s.ParseToken(access)
	if err != nil {
		return nil
	}
	return s.sessions.Revoke(ctx, claims.SessionID, now)
}

// issueCredentials 签发绑定会话的 JWT，限制其 exp 不超过会话截止时间。
func (s *AuthService) issueCredentials(session model.AuthSession, refresh string, now time.Time) (AuthCredentials, error) {
	expiresAt := jwt.NewNumericDate(minTime(now.Add(s.accessTTL), session.ExpiresAt))
	claims := AuthClaims{
		SessionID: session.ID,
		UserID:    session.UserID,
		Username:  session.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "fluffy-cupcake",
			Subject:   strconv.FormatInt(session.UserID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: expiresAt,
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return AuthCredentials{}, fmt.Errorf("签发 JWT: %w", err)
	}
	return AuthCredentials{signed, expiresAt.Time, refresh, session.ExpiresAt}, nil
}

// minTime 返回两个截止时间中较早的一个。
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// ParseToken 校验签名、签发者、有效期和会话声明，升级前缺少 sid 的 JWT 会被拒绝。
func (s *AuthService) ParseToken(raw string) (AuthClaims, error) {
	claims := AuthClaims{}
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("fluffy-cupcake"),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithTimeFunc(s.now))
	if err != nil || !token.Valid || !validCredential(claims.SessionID, 16) || claims.UserID <= 0 ||
		claims.Username == "" || claims.IssuedAt == nil || claims.Subject != strconv.FormatInt(claims.UserID, 10) {
		return AuthClaims{}, ErrInvalidToken
	}
	return claims, nil
}

// Authenticate 在验签后查会话状态，使退出或重用撤销立即影响后续请求。
func (s *AuthService) Authenticate(ctx context.Context, raw string) (AuthClaims, error) {
	claims, err := s.ParseToken(raw)
	if err != nil {
		return AuthClaims{}, err
	}
	active, err := s.sessions.Active(ctx, claims.SessionID, claims.UserID, s.now().UTC())
	if err != nil {
		return AuthClaims{}, fmt.Errorf("读取登录会话状态: %w", err)
	}
	if !active {
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

// randomCredential 生成可安全用于 Cookie 和 JWT 声明的随机值。
func randomCredential(size int) string {
	raw := make([]byte, size)
	rand.Read(raw)
	return base64.RawURLEncoding.EncodeToString(raw)
}

// validCredential 只接受预期长度的规范 Base64URL 编码。
func validCredential(value string, size int) bool {
	if len(value) != base64.RawURLEncoding.EncodedLen(size) {
		return false
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(value)
	return err == nil && len(raw) == size
}
