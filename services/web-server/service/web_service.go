// Package service 是 web-server 的业务层。
// 它组合 repository 与 gRPC 客户端（tracker/dht），向 handler 提供用例。
package service

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/xiezc/xpt/api/gen/dht"
	"github.com/xiezc/xpt/api/gen/tracker"
	"github.com/xiezc/xpt/pkg/errcode"
	"github.com/xiezc/xpt/pkg/model"
	"github.com/xiezc/xpt/services/web-server/repository"
)

// WebService 是 web-server 的业务入口。
type WebService struct {
	userRepo    *repository.UserRepo
	torrentRepo *repository.TorrentRepo

	trackerConn *grpc.ClientConn
	dhtConn     *grpc.ClientConn

	trackerClient tracker.TrackerServiceClient
	dhtClient     dht.DhtServiceClient
}

// New 构造 WebService，并建立到 tracker/dht 的 gRPC 连接。
func New(userRepo *repository.UserRepo, torrentRepo *repository.TorrentRepo,
	trackerAddr, dhtAddr string) (*WebService, error) {

	trackerConn, err := grpc.NewClient(trackerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	dhtConn, err := grpc.NewClient(dhtAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		trackerConn.Close()
		return nil, err
	}
	return &WebService{
		userRepo:      userRepo,
		torrentRepo:   torrentRepo,
		trackerConn:   trackerConn,
		dhtConn:       dhtConn,
		trackerClient: tracker.NewTrackerServiceClient(trackerConn),
		dhtClient:     dht.NewDhtServiceClient(dhtConn),
	}, nil
}

// Close 释放 gRPC 连接。
func (s *WebService) Close() {
	if s.trackerConn != nil {
		s.trackerConn.Close()
	}
	if s.dhtConn != nil {
		s.dhtConn.Close()
	}
}

// Register 注册新用户。
// TODO(学习): 密码应做 bcrypt 哈希后入库，并校验用户名/邮箱格式。
func (s *WebService) Register(username, password, email string) (*model.User, error) {
	if username == "" || password == "" {
		return nil, errcode.ErrBadRequest
	}
	// TODO: 检查用户是否已存在，然后哈希密码。
	u := &model.User{Username: username, Email: email}
	if err := s.userRepo.Create(u); err != nil {
		return nil, errcode.Wrap(errcode.CodeConflict, err)
	}
	return u, nil
}

// Login 校验用户名密码，返回用户信息。
// TODO(学习): 校验 password_hash，签发 JWT 在 handler 层完成。
func (s *WebService) Login(username, password string) (*model.User, error) {
	u, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, errcode.ErrBadLogin
	}
	// TODO: 比对密码哈希。
	_ = password
	return u, nil
}

// ListTorrents 分页列出种子。
func (s *WebService) ListTorrents(page, size int) ([]model.Torrent, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	return s.torrentRepo.List((page-1)*size, size)
}

// UploadTorrent 登记种子元数据。
func (s *WebService) UploadTorrent(t *model.Torrent) error {
	if t.InfoHash == "" {
		return errcode.ErrBadInfoHash
	}
	return s.torrentRepo.Create(t)
}

// GetTorrentPeers 通过 gRPC 查询 tracker 的 peer 列表。
func (s *WebService) GetTorrentPeers(ctx context.Context, infoHash string) (*tracker.GetTorrentPeersResponse, error) {
	return s.trackerClient.GetTorrentPeers(ctx, &tracker.GetTorrentPeersRequest{InfoHash: infoHash})
}

// GetTrackerStats 通过 gRPC 查询 tracker 全局统计。
func (s *WebService) GetTrackerStats(ctx context.Context) (*tracker.GetStatsResponse, error) {
	return s.trackerClient.GetStats(ctx, &tracker.GetStatsRequest{})
}

// GetKnownInfoHashes 通过 gRPC 查询 DHT 网络发现的 infohash。
func (s *WebService) GetKnownInfoHashes(ctx context.Context) (*dht.GetKnownInfoHashResponse, error) {
	return s.dhtClient.GetKnownInfoHash(ctx, &dht.GetKnownInfoHashRequest{})
}

// 编译期断言：保持 time 依赖用于后续 token 过期等逻辑。
var _ = time.Minute
