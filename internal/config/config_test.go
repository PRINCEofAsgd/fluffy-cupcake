package config

import (
	"testing"
	"time"
)

// TestLoadDefaults 验证本地开发默认监听地址和运行模式。
func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ADDR", "")
	t.Setenv("APP_MODE", "")
	t.Setenv("ACCESS_TOKEN_TTL", "")
	t.Setenv("SESSION_TTL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Address != defaultAddress {
		t.Fatalf("Address = %q，期望 %q", cfg.Address, defaultAddress)
	}
	if cfg.Mode != defaultMode {
		t.Fatalf("Mode = %q，期望 %q", cfg.Mode, defaultMode)
	}
	if cfg.Auth.AccessTokenTTL != 15*time.Minute || cfg.Auth.SessionTTL != 360*time.Hour {
		t.Fatalf("默认会话配置 = %#v", cfg.Auth)
	}
}

// TestLoadFromEnvironment 验证环境变量能够覆盖默认配置。
func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("APP_ADDR", "127.0.0.1:9090")
	t.Setenv("APP_MODE", "release")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Address != "127.0.0.1:9090" || cfg.Mode != "release" {
		t.Fatalf("Load() = %#v，环境变量未正确生效", cfg)
	}
}

// TestLoadDatabaseAndAuth 验证数据库、JWT 和 Cookie 环境语义。
func TestLoadDatabaseAndAuth(t *testing.T) {
	t.Setenv("APP_MODE", "release")
	t.Setenv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/db")
	t.Setenv("JWT_SECRET", "12345678901234567890123456789012")
	t.Setenv("ACCESS_TOKEN_TTL", "8m")
	t.Setenv("SESSION_TTL", "24h")
	t.Setenv("DB_MAX_OPEN_CONNS", "12")
	t.Setenv("DB_MAX_IDLE_CONNS", "4")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := cfg.ValidateServer(); err != nil {
		t.Fatalf("ValidateServer() error = %v", err)
	}
	if !cfg.Auth.CookieSecure || cfg.Auth.AccessTokenTTL != 8*time.Minute || cfg.Auth.SessionTTL != 24*time.Hour {
		t.Fatalf("Auth = %#v，生产 Cookie 或 JWT 有效期不正确", cfg.Auth)
	}
	if cfg.Database.MaxOpenConns != 12 || cfg.Database.MaxIdleConns != 4 {
		t.Fatalf("Database = %#v，连接池配置不正确", cfg.Database)
	}
}

// TestLoadRejectsInvalidDuration 验证错误配置在服务启动前被拒绝。
func TestLoadRejectsInvalidDuration(t *testing.T) {
	for _, key := range []string{"ACCESS_TOKEN_TTL", "SESSION_TTL"} {
		t.Run(key, func(t *testing.T) {
			for _, value := range []string{"tomorrow", "0s", "900ms", "1.5s"} {
				t.Setenv(key, value)
				if _, err := Load(); err == nil {
					t.Fatalf("Load() 未拒绝 %s=%q", key, value)
				}
			}
		})
	}
}
