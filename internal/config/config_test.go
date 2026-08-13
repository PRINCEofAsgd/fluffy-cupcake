package config

import "testing"

// TestLoadDefaults 验证本地开发默认监听地址和运行模式。
func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ADDR", "")
	t.Setenv("APP_MODE", "")

	cfg := Load()
	if cfg.Address != defaultAddress {
		t.Fatalf("Address = %q，期望 %q", cfg.Address, defaultAddress)
	}
	if cfg.Mode != defaultMode {
		t.Fatalf("Mode = %q，期望 %q", cfg.Mode, defaultMode)
	}
}

// TestLoadFromEnvironment 验证环境变量能够覆盖默认配置。
func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("APP_ADDR", "127.0.0.1:9090")
	t.Setenv("APP_MODE", "release")

	cfg := Load()
	if cfg.Address != "127.0.0.1:9090" || cfg.Mode != "release" {
		t.Fatalf("Load() = %#v，环境变量未正确生效", cfg)
	}
}
