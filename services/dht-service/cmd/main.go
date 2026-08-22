// dht-service 启动入口。
//
// 职责：加载配置 -> 初始化日志 -> 打开 sqlite -> 执行迁移 ->
//
//	启动 DHT 爬虫协程 -> 启动 gRPC 服务（供 web 查询 infohash）。
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/xiezc/xpt/api/gen/dht"
	"github.com/xiezc/xpt/pkg/config"
	"github.com/xiezc/xpt/pkg/db"
	"github.com/xiezc/xpt/pkg/logger"
	"github.com/xiezc/xpt/services/dht-service/handler"
	"github.com/xiezc/xpt/services/dht-service/migrate"
	"github.com/xiezc/xpt/services/dht-service/repository"
	"github.com/xiezc/xpt/services/dht-service/service"
)

func main() {
	// 多路径探测：兼容项目根运行（make run-dht）与 VSCode 调试（cwd=cmd 目录）。
	cfg, err := config.LoadAny(
		"./services/dht-service/config.yaml", // 从项目根运行
		"../config.yaml",                     // 从 cmd 目录运行/调试
	)
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}

	logger.InitLogger(cfg.IsDebug())
	log := logger.WithComponent("dht-service")
	log.Info("starting dht-service")

	// 多路径探测 db 位置：兼容项目根运行与 cmd 目录调试。
	sqlDB, err := db.NewSQLite(db.ResolveDBPath(
		cfg.GetString("dht_sqlite_path"),
		"../../../data/dht.db",
	))
	if err != nil {
		log.Error("open sqlite failed", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// 执行数据库迁移（migrate 包内嵌 001_init.sql）。
	if err := db.RunMigrations(sqlDB, migrate.FS); err != nil {
		log.Error("run migrations failed", "err", err)
		os.Exit(1)
	}

	nodeRepo := repository.NewNodeRepo(sqlDB)
	hashRepo := repository.NewInfoHashRepo(sqlDB)

	// 启动 DHT 爬虫（后台协程）。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dhtSvc, err := service.New(nodeRepo, hashRepo, cfg.GetString("dht_udp_addr"))
	if err != nil {
		log.Error("init dht service failed", "err", err)
		os.Exit(1)
	}
	go func() {
		if err := dhtSvc.Run(ctx); err != nil {
			log.Error("dht service run error", "err", err)
		}
	}()

	// 启动 gRPC 服务。
	lis, err := net.Listen("tcp", cfg.GetString("grpc_listen"))
	if err != nil {
		log.Error("grpc listen failed", "err", err)
		os.Exit(1)
	}
	gs := grpc.NewServer()
	dht.RegisterDhtServiceServer(gs, handler.NewDhtHandler(hashRepo))
	log.Info("dht grpc server listening", "addr", cfg.GetString("grpc_listen"))
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Error("grpc serve error", "err", err)
			os.Exit(1)
		}
	}()

	// 优雅退出：Ctrl+C -> 停止 grpc 与 dht。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down dht-service")
	gs.GracefulStop()
	cancel()
	dhtSvc.Close()
}
