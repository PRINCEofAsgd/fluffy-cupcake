// Package database 统一创建和管理项目使用的 MySQL 连接池。
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

const defaultNetworkTimeout = 5 * time.Second

// Open 创建可复用连接池，并在返回前通过 Ping 验证数据库可访问。
func Open(parent context.Context, cfg config.DatabaseConfig) (*sql.DB, error) {
	dsn, err := normalizedDSN(cfg.DSN)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("创建 MySQL 连接池: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingContext, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingContext); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接 MySQL: %w", err)
	}
	return db, nil
}

// normalizedDSN 强制所有池连接按 UTC 解释时间并启用 time.Time 扫描。
// 跨区域网络超时仅在 DSN 未显式配置时补默认值，调用方可按实际链路覆盖。
func normalizedDSN(raw string) (string, error) {
	parsed, err := mysqlDriver.ParseDSN(raw)
	if err != nil {
		return "", fmt.Errorf("解析 DATABASE_DSN: %w", err)
	}
	parsed.ParseTime = true
	parsed.Loc = time.UTC
	if parsed.Timeout == 0 {
		parsed.Timeout = defaultNetworkTimeout
	}
	if parsed.ReadTimeout == 0 {
		parsed.ReadTimeout = defaultNetworkTimeout
	}
	if parsed.WriteTimeout == 0 {
		parsed.WriteTimeout = defaultNetworkTimeout
	}
	if parsed.Params == nil {
		parsed.Params = make(map[string]string)
	}
	parsed.Params["time_zone"] = "'+00:00'"
	return parsed.FormatDSN(), nil
}
