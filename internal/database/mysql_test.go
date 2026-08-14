package database

import (
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

// TestNormalizedDSN 验证连接池始终以 UTC 读写时间，不依赖服务器所在地时区。
func TestNormalizedDSN(t *testing.T) {
	dsn, err := normalizedDSN("user:pass@tcp(localhost:3306)/fluffy_cupcake")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.ParseTime || parsed.Loc != time.UTC || parsed.Params["time_zone"] != "'+00:00'" {
		t.Fatalf("标准化 DSN = %#v", parsed)
	}
}
