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
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/database"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/healthcheck"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/server"
)

// main 加载配置、启动 HTTP 服务，并在收到系统信号后完成优雅退出。
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("服务退出", "error", err)
		os.Exit(1)
	}
}

// run 管理全部可关闭资源，使异常返回路径也会执行数据库连接池清理。
func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		if err := healthcheck.Run(cfg.Address); err != nil {
			return err
		}
		return nil
	}
	if err := cfg.ValidateServer(); err != nil {
		return err
	}
	db, err := database.Open(context.Background(), cfg.Database)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("关闭数据库连接池失败", "error", err)
		}
	}()

	httpServer := &http.Server{
		Addr:              cfg.Address,
		Handler:           server.NewRouter(cfg, db),
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
			return err
		}
	case <-shutdownSignal.Done():
		logger.Info("正在关闭 HTTP 服务")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		return err
	}
	return nil
}
