// tracker-service 启动入口。
//
// 职责：加载配置 -> 初始化日志 -> 打开 sqlite -> 执行迁移 ->
//
//	启动 UDP tracker 协程（BEP 15）-> 启动 gRPC 服务（供 web 查询）。
package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/xiezc/xpt/api/gen/tracker"
	"github.com/xiezc/xpt/pkg/apppath"
	"github.com/xiezc/xpt/pkg/config"
	"github.com/xiezc/xpt/pkg/db"
	"github.com/xiezc/xpt/pkg/logger"
	"github.com/xiezc/xpt/services/tracker-service/handler"
	"github.com/xiezc/xpt/services/tracker-service/migrate"
	"github.com/xiezc/xpt/services/tracker-service/repository"
	"github.com/xiezc/xpt/services/tracker-service/service"
)

func main() {
	// 统一路径方案：向上查找 go.mod 定位项目根，之后全部用绝对路径。
	cfg, err := config.Load(apppath.Config("tracker-service"))
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}

	logger.InitLogger(cfg.IsDebug())
	log := logger.WithComponent("tracker-service")
	log.Info("starting tracker-service")

	sqlDB, err := db.NewSQLite(apppath.Data("tracker.db"))
	if err != nil {
		log.Error("open sqlite failed", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := db.RunMigrations(sqlDB, migrate.FS); err != nil {
		log.Error("run migrations failed", "err", err)
		os.Exit(1)
	}

	repo := repository.NewPeerRepo(sqlDB)
	core := service.NewCore(repo, cfg.GetInt("announce_interval"))

	// 启动 UDP tracker（BEP 15），后台协程。
	udpTracker, err := service.NewUDPTracker(core, cfg.GetString("udp_listen"))
	if err != nil {
		log.Error("init udp tracker failed", "err", err)
		os.Exit(1)
	}
	go udpTracker.Run()
	defer udpTracker.Close()

	// 启动 peer 清理协程。
	done := make(chan struct{})
	defer close(done)
	go core.CleanupLoop(5*time.Minute, done)

	// 启动 gRPC 服务。
	lis, err := net.Listen("tcp", cfg.GetString("grpc_listen"))
	if err != nil {
		log.Error("grpc listen failed", "err", err)
		os.Exit(1)
	}
	gs := grpc.NewServer()
	tracker.RegisterTrackerServiceServer(gs, handler.NewTrackerHandler(core))
	log.Info("tracker grpc server listening", "addr", cfg.GetString("grpc_listen"))
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Error("grpc serve error", "err", err)
			os.Exit(1)
		}
	}()

	// 优雅退出。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down tracker-service")
	gs.GracefulStop()
}
