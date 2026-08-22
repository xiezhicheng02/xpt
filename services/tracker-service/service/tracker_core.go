// Package service 实现 tracker 核心逻辑：announce 处理与 peer 管理。
package service

import (
	"log/slog"
	"time"

	"github.com/xiezc/xpt/internal/shared"
	"github.com/xiezc/xpt/pkg/model"
	"github.com/xiezc/xpt/services/tracker-service/repository"
)

// AnnounceEvent 是客户端上报的事件类型（对应 BitTorrent 协议 event 参数）。
type AnnounceEvent int

const (
	// EventNone 常规周期上报。
	EventNone AnnounceEvent = iota
	// EventCompleted 下载完成（成为做种者）。
	EventCompleted
	// EventStarted 开始下载/做种。
	EventStarted
	// EventStopped 停止。
	EventStopped
)

// AnnounceRequest 是一次 announce 请求的解析结果。
type AnnounceRequest struct {
	InfoHash   string        // 20 字节 infohash（以 hex 存储）
	PeerID     string        // 20 字节 peer id
	IP         string        // 客户端 IP
	Port       int           // 客户端监听端口
	Uploaded   int64         // 已上传字节
	Downloaded int64         // 已下载字节
	Left       int64         // 剩余待下载字节
	Event      AnnounceEvent // 事件类型
}

// TrackerCore 是 tracker 的业务核心，封装 peer 管理逻辑。
// 它同时被 HTTP/grpc handler 和 UDP tracker 复用。
type TrackerCore struct {
	repo *repository.PeerRepo

	// announceInterval 是建议客户端下次上报的间隔（秒）。
	announceInterval int
}

// NewCore 构造 TrackerCore。
func NewCore(repo *repository.PeerRepo, announceInterval int) *TrackerCore {
	if announceInterval <= 0 {
		announceInterval = shared.DefaultAnnounceInterval
	}
	return &TrackerCore{repo: repo, announceInterval: announceInterval}
}

// Announce 处理一次 announce 请求。
//
// TODO(学习): 完整实现要点——
//  1. 校验 infohash / peer_id 长度（20 字节）；
//  2. 把 peer 记录 upsert 进 peers 表；
//  3. 根据 torrent_id 查询活跃 peer 列表返回给客户端；
//  4. Event==Stopped 时删除该 peer 记录；
//  5. 返回建议的 announce interval。
//
// 当前返回 peers 列表与 interval（interval 已实现，其余待填充）。
func (c *TrackerCore) Announce(req *AnnounceRequest) ([]model.Peer, int, error) {
	// 1. 校验（TODO: 补充 infohash/peer_id 长度校验）。
	if req.InfoHash == "" || req.PeerID == "" {
		return nil, 0, errBadAnnounce
	}

	// 2-4. TODO: 事件处理与 upsert（需要 torrent 表关联，先留空）。
	// 学习提示：peers 表用 torrent_id 关联 torrents，当前 001_init.sql
	// 只有 peers 表，你需要在实现时决定 torrent 的登记方式。

	// 5. 返回 peer 列表与 interval。
	peers, err := c.repo.ListPeersByTorrent(0, shared.MaxPeersPerAnnounce)
	if err != nil {
		slog.Error("list peers failed", "err", err)
		return nil, 0, err
	}
	return peers, c.announceInterval, nil
}

// GetPeers 返回指定 torrent 的活跃 peer 列表（grpc 查询用）。
func (c *TrackerCore) GetPeers(torrentID int64, limit int) ([]model.Peer, error) {
	if limit <= 0 {
		limit = shared.MaxPeersPerAnnounce
	}
	return c.repo.ListPeersByTorrent(torrentID, limit)
}

// GetStats 返回全局统计（grpc 查询用）。
func (c *TrackerCore) GetStats() (torrents, seeders, leechers int64, err error) {
	seeders, leechers, err = c.repo.CountStats()
	if err != nil {
		return 0, 0, 0, err
	}
	torrents, err = c.repo.CountTorrents()
	return torrents, seeders, leechers, err
}

// CleanupLoop 周期清理超时 peer，应作为 goroutine 启动。
func (c *TrackerCore) CleanupLoop(interval time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			n, err := c.repo.DeleteStale(shared.PeerTimeout)
			if err != nil {
				slog.Warn("cleanup stale peers failed", "err", err)
				continue
			}
			if n > 0 {
				slog.Info("cleaned stale peers", "count", n)
			}
		}
	}
}
