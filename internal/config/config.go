// Package config 负责从环境变量读取运行配置，并提供适合本地开发的默认值。
package config

import "os"

const (
	defaultAddress = "0.0.0.0:4819"
	defaultMode    = "debug"
)

// Config 表示 Web 服务运行所需的配置。
type Config struct {
	Address string
	Mode    string
}

// Load 从环境变量加载配置；缺省时监听全部本地开发网口的 4819 端口。
func Load() Config {
	return Config{
		Address: valueOrDefault("APP_ADDR", defaultAddress),
		Mode:    valueOrDefault("APP_MODE", defaultMode),
	}
}

// valueOrDefault 返回非空环境变量值，否则返回默认值。
func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
