// Package handler 实现 tracker-service 的 gRPC 接口。
// 供 web 服务查询 peer 列表与统计信息。
package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xiezc/xpt/api/gen/tracker"
	"github.com/xiezc/xpt/services/tracker-service/service"
)

// TrackerHandler 实现 tracker.TrackerServiceServer。
type TrackerHandler struct {
	tracker.UnimplementedTrackerServiceServer
	core *service.TrackerCore
}

func NewTrackerHandler(core *service.TrackerCore) *TrackerHandler {
	return &TrackerHandler{core: core}
}

// GetTorrentPeers 返回指定 torrent 的 peer 列表。
// TODO(学习): info_hash 传的是 hex 字符串，需要与 torrent 表关联出 torrent_id。
func (h *TrackerHandler) GetTorrentPeers(ctx context.Context, req *tracker.GetTorrentPeersRequest) (*tracker.GetTorrentPeersResponse, error) {
	if req.GetInfoHash() == "" {
		return nil, status.Error(codes.InvalidArgument, "info_hash required")
	}
	// TODO: 由 info_hash 查 torrent_id（当前 peers 表以 torrent_id 关联）。
	peers, err := h.core.GetPeers(0, 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get peers: %v", err)
	}
	resp := &tracker.GetTorrentPeersResponse{}
	for _, p := range peers {
		resp.Peers = append(resp.Peers, &tracker.Peer{
			PeerId:   p.PeerID,
			Ip:       p.IP,
			Port:     int32(p.Port),
			IsSeeder: p.IsSeeder,
		})
	}
	return resp, nil
}

// GetStats 返回全局统计。
func (h *TrackerHandler) GetStats(ctx context.Context, _ *tracker.GetStatsRequest) (*tracker.GetStatsResponse, error) {
	torrents, seeders, leechers, err := h.core.GetStats()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get stats: %v", err)
	}
	return &tracker.GetStatsResponse{
		TotalTorrents: torrents,
		TotalSeeders:  seeders,
		TotalLeechers: leechers,
	}, nil
}
