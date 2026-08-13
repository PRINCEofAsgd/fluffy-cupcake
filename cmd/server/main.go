// Package main 提供 fluffy-cupcake Web 服务的程序入口。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/healthcheck"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/server"
)

// main 加载配置、启动 HTTP 服务，并在收到系统信号后完成优雅退出。
func main() {
	cfg := config.Load()
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		if err := healthcheck.Run(cfg.Address); err != nil {
			os.Exit(1)
		}
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	httpServer := &http.Server{
		Addr:              cfg.Address,
		Handler:           server.NewRouter(cfg),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP 服务已启动", "address", cfg.Address, "mode", cfg.Mode)
		serverErrors <- httpServer.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP 服务异常退出", "error", err)
			os.Exit(1)
		}
	case <-shutdownSignal.Done():
		logger.Info("正在关闭 HTTP 服务")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		logger.Error("HTTP 服务未能正常关闭", "error", err)
		os.Exit(1)
	}
}
