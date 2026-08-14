// Package config 负责从环境变量读取运行配置，并提供适合本地开发的默认值。
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAddress    = "0.0.0.0:4819"
	defaultMode       = "debug"
	defaultJWTExpire  = 24 * time.Hour
	defaultCookieName = "fluffy_cupcake_auth"
)

// Config 表示 Web 服务运行所需的配置。
type Config struct {
	Address  string
	Mode     string
	Database DatabaseConfig
	Auth     AuthConfig
}

// DatabaseConfig 保存 MySQL DSN 与全局连接池参数。
type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// AuthConfig 保存 JWT 与认证 Cookie 配置；Secret 只从环境变量读取。
type AuthConfig struct {
	JWTSecret    string
	JWTExpire    time.Duration
	CookieName   string
	CookieSecure bool
}

// Load 从环境变量加载配置，并校验需要解析的数值与时长格式。
func Load() (Config, error) {
	mode := valueOrDefault("APP_MODE", defaultMode)
	maxOpenConns, err := positiveInt("DB_MAX_OPEN_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	maxIdleConns, err := positiveInt("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return Config{}, err
	}
	connMaxLifetime, err := positiveDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}
	jwtExpire, err := positiveDuration("JWT_EXPIRE", defaultJWTExpire)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Address: valueOrDefault("APP_ADDR", defaultAddress),
		Mode:    mode,
		Database: DatabaseConfig{
			DSN:             os.Getenv("DATABASE_DSN"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: connMaxLifetime,
		},
		Auth: AuthConfig{
			JWTSecret:    os.Getenv("JWT_SECRET"),
			JWTExpire:    jwtExpire,
			CookieName:   defaultCookieName,
			CookieSecure: mode == "release",
		},
	}, nil
}

// ValidateServer 检查启动完整 Web 服务所需的敏感配置是否齐全。
func (c Config) ValidateServer() error {
	if c.Database.DSN == "" {
		return errors.New("缺少 DATABASE_DSN")
	}
	if len(c.Auth.JWTSecret) < 32 || strings.HasPrefix(c.Auth.JWTSecret, "replace_") {
		return errors.New("JWT_SECRET 必须替换为至少 32 个字符的真实随机值")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return errors.New("DB_MAX_IDLE_CONNS 不能大于 DB_MAX_OPEN_CONNS")
	}
	return nil
}

// ValidateDatabase 检查仅访问数据库的 CLI 所需配置。
func (c Config) ValidateDatabase() error {
	if c.Database.DSN == "" {
		return errors.New("缺少 DATABASE_DSN")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return errors.New("DB_MAX_IDLE_CONNS 不能大于 DB_MAX_OPEN_CONNS")
	}
	return nil
}

// valueOrDefault 返回非空环境变量值，否则返回默认值。
func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// positiveInt 读取正整数连接池配置。
func positiveInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s 必须是正整数", key)
	}
	return value, nil
}

// positiveDuration 读取 Go duration 格式的正时长配置。
func positiveDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s 必须是正数 Go duration，例如 24h", key)
	}
	return value, nil
}
