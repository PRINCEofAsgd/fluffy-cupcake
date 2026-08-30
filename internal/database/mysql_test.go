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
	if parsed.Timeout != defaultNetworkTimeout || parsed.ReadTimeout != defaultNetworkTimeout || parsed.WriteTimeout != defaultNetworkTimeout {
		t.Fatalf("默认网络超时 = connect:%v read:%v write:%v", parsed.Timeout, parsed.ReadTimeout, parsed.WriteTimeout)
	}
}

// TestNormalizedDSNPreservesExplicitTimeoutsAndTLS 验证生产 DSN 的合理自定义值与 TLS 选择不会被默认值覆盖。
func TestNormalizedDSNPreservesExplicitTimeoutsAndTLS(t *testing.T) {
	for _, tlsConfig := range []string{"true", "skip-verify"} {
		t.Run(tlsConfig, func(t *testing.T) {
			dsn, err := normalizedDSN("user:pass@tcp(db.example.invalid:3306)/fluffy_cupcake?timeout=8s&readTimeout=12s&writeTimeout=15s&tls=" + tlsConfig)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := mysqlDriver.ParseDSN(dsn)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Timeout != 8*time.Second || parsed.ReadTimeout != 12*time.Second || parsed.WriteTimeout != 15*time.Second {
				t.Fatalf("显式网络超时被覆盖 = connect:%v read:%v write:%v", parsed.Timeout, parsed.ReadTimeout, parsed.WriteTimeout)
			}
			if parsed.TLSConfig != tlsConfig {
				t.Fatalf("TLS 配置 = %q，期望保留 %q", parsed.TLSConfig, tlsConfig)
			}
			if !parsed.ParseTime || parsed.Loc != time.UTC || parsed.Params["time_zone"] != "'+00:00'" {
				t.Fatalf("显式参数下 UTC 标准化失效 = %#v", parsed)
			}
		})
	}
}
