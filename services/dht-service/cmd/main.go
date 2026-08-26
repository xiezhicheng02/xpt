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

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"

	"github.com/xiezc/xpt/api/gen/dht"
	"github.com/xiezc/xpt/pkg/apppath"
	"github.com/xiezc/xpt/pkg/config"
	"github.com/xiezc/xpt/pkg/db"
	"github.com/xiezc/xpt/pkg/logger"
	"github.com/xiezc/xpt/services/dht-service/handler"
	"github.com/xiezc/xpt/services/dht-service/migrate"
	"github.com/xiezc/xpt/services/dht-service/repository"
	"github.com/xiezc/xpt/services/dht-service/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service exited with error", "err", err)
		os.Exit(1)
	}
}

// run 主流程编排：按顺序初始化所有组件、启动服务、阻塞等待退出信号、优雅关闭资源。
// 所有组件初始化错误统一向上返回，main 统一处理退出码。
func run() error {
	// 1. 加载配置
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// 2. 初始化日志
	log := initLogger(cfg)

	// 3. 初始化数据库并执行迁移
	sqlDB, err := initDatabase(cfg)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// 4. 初始化数据仓库层
	nodeRepo, hashRepo := initRepositories(sqlDB)

	// 5. 初始化上下文与取消
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 6. 初始化并启动 DHT 服务
	dhtSvc, err := startDHTService(ctx, nodeRepo, hashRepo, cfg)
	if err != nil {
		return err
	}
	defer dhtSvc.Close()

	// 7. 初始化并启动 gRPC 服务
	grpcServer, _, err := startGRPCServer(cfg, hashRepo)
	if err != nil {
		return err
	}
	defer grpcServer.GracefulStop()

	log.Info("dht-service started successfully")

	// 8. 阻塞等待退出信号，触发优雅关闭
	waitForShutdownSignal()
	log.Info("shutting down dht-service")

	// 触发上下文取消，通知 DHT 协程退出
	cancel()

	return nil
}

// loadConfig 加载服务配置，统一路径解析。
func loadConfig() (*config.Config, error) {
	// 统一路径方案：向上查找 go.mod 定位项目根，之后全部用绝对路径，
	// 与 cwd 无关（命令行运行 / VSCode 调试 / 任意目录启动都正确）。
	cfg, err := config.Load(apppath.Config("dht-service"))
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// initLogger 初始化全局日志并返回带组件标识的日志实例。
func initLogger(cfg *config.Config) *slog.Logger {
	logger.InitLogger(cfg.IsDebug())
	return logger.WithComponent("dht-service")
}

// initDatabase 打开 SQLite 连接并执行数据库迁移。
func initDatabase(cfg *config.Config) (*sqlx.DB, error) {
	sqlDB, err := db.NewSQLite(apppath.Data("dht.db"))
	if err != nil {
		return nil, err
	}

	// 执行数据库迁移（migrate 包内嵌 001_init.sql）。
	if err := db.RunMigrations(sqlDB, migrate.FS); err != nil {
		sqlDB.Close()
		return nil, err
	}

	return sqlDB, nil
}

// initRepositories 初始化所有数据仓库实例。
func initRepositories(sqlDB *sqlx.DB) (*repository.NodeRepo, *repository.InfoHashRepo) {
	nodeRepo := repository.NewNodeRepo(sqlDB)
	hashRepo := repository.NewInfoHashRepo(sqlDB)
	return nodeRepo, hashRepo
}

// startDHTService 初始化 DHT 服务并后台启动爬虫协程。
func startDHTService(ctx context.Context,
	nodeRepo *repository.NodeRepo,
	hashRepo *repository.InfoHashRepo,
	cfg *config.Config) (*service.DHTService, error) {
	dhtSvc, err := service.New(nodeRepo, hashRepo, cfg.GetString("dht_udp_addr"))
	if err != nil {
		return nil, err
	}

	go func() {
		if err := dhtSvc.Run(ctx); err != nil {
			slog.Error("dht service run error", "err", err)
		}
	}()

	return dhtSvc, nil
}

// startGRPCServer 初始化 gRPC 服务，注册处理器并启动监听。
func startGRPCServer(cfg *config.Config, hashRepo *repository.InfoHashRepo) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", cfg.GetString("grpc_listen"))
	if err != nil {
		return nil, nil, err
	}

	gs := grpc.NewServer()
	dht.RegisterDhtServiceServer(gs, handler.NewDhtHandler(hashRepo))

	slog.Info("dht grpc server listening", "addr", cfg.GetString("grpc_listen"))

	go func() {
		if err := gs.Serve(lis); err != nil {
			slog.Error("grpc serve error", "err", err)
		}
	}()

	return gs, lis, nil
}

// waitForShutdownSignal 阻塞等待系统中断信号，用于触发优雅退出流程。
func waitForShutdownSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
