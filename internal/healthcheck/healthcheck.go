// Package healthcheck 提供无需 shell 或外部工具的容器健康检查客户端。
package healthcheck

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// Run 请求本机服务的健康检查端点，非 200 响应会作为错误返回。
func Run(address string) error {
	return runWithClient(address, &http.Client{Timeout: 3 * time.Second})
}

// runWithClient 执行健康检查，并允许测试注入内存 HTTP 客户端。
func runWithClient(address string, client *http.Client) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("解析监听地址: %w", err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	response, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return fmt.Errorf("请求健康检查: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("健康检查状态码: %d", response.StatusCode)
	}
	return nil
}
