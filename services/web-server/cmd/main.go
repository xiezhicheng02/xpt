// web-server 启动入口。
//
// 职责：加载配置 -> 初始化日志 -> 打开 sqlite -> 执行迁移 ->
//
//	建立 tracker/dht 的 gRPC 客户端 -> 启动 gin HTTP 服务。
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xiezc/xpt/pkg/apppath"
	"github.com/xiezc/xpt/pkg/config"
	"github.com/xiezc/xpt/pkg/db"
	"github.com/xiezc/xpt/pkg/logger"
	"github.com/xiezc/xpt/services/web-server/handler"
	"github.com/xiezc/xpt/services/web-server/migrate"
	"github.com/xiezc/xpt/services/web-server/repository"
	"github.com/xiezc/xpt/services/web-server/router"
	"github.com/xiezc/xpt/services/web-server/service"
)

func main() {
	// 统一路径方案：向上查找 go.mod 定位项目根，之后全部用绝对路径。
	cfg, err := config.Load(apppath.Config("web-server"))
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}

	logger.InitLogger(cfg.IsDebug())
	log := logger.WithComponent("web-server")
	log.Info("starting web-server")

	sqlDB, err := db.NewSQLite(apppath.Data("web.db"))
	if err != nil {
		log.Error("open sqlite failed", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := db.RunMigrations(sqlDB, migrate.FS); err != nil {
		log.Error("run migrations failed", "err", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepo(sqlDB)
	torrentRepo := repository.NewTorrentRepo(sqlDB)

	svc, err := service.New(userRepo, torrentRepo,
		cfg.GetString("tracker_grpc_addr"), cfg.GetString("dht_grpc_addr"))
	if err != nil {
		log.Error("init web service failed", "err", err)
		os.Exit(1)
	}
	defer svc.Close()

	jwtSecret := cfg.GetString("jwt_secret")
	if jwtSecret == "" || jwtSecret == "change-me-secret" {
		log.Warn("jwt_secret is default, please change in production")
	}

	h := handler.New(svc, jwtSecret)
	r := router.New(h, jwtSecret)

	httpAddr := cfg.GetString("http_listen")
	srv := &http.Server{
		Addr:    httpAddr,
		Handler: r,
	}
	log.Info("web http server listening", "addr", httpAddr)

	// 启动 HTTP 服务。
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http serve error", "err", err)
			os.Exit(1)
		}
	}()

	// 优雅退出。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down web-server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
