package repository

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/xiezc/xpt/pkg/model"
)

// PeerRepo 负责 peers 表的读写。
// 一个 torrent 下同一 peer_id 只保留一行（upsert 更新统计数据与最后上报时间）。
type PeerRepo struct {
	db *sqlx.DB
}

func NewPeerRepo(db *sqlx.DB) *PeerRepo {
	return &PeerRepo{db: db}
}

// UpsertPeer 记录一次 announce：存在则更新，不存在则插入。
// 同一 torrent 下以 peer_id 唯一。
func (r *PeerRepo) UpsertPeer(p *model.Peer) error {
	_, err := r.db.Exec(`
INSERT INTO peers(torrent_id, peer_id, ip, port, uploaded, downloaded, "left", is_seeder, last_announce)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(torrent_id, peer_id) DO UPDATE SET
  ip = excluded.ip,
  port = excluded.port,
  uploaded = excluded.uploaded,
  downloaded = excluded.downloaded,
  "left" = excluded."left",
  is_seeder = excluded.is_seeder,
  last_announce = excluded.last_announce`,
		p.TorrentID, p.PeerID, p.IP, p.Port, p.Uploaded, p.Downloaded, p.Left, p.IsSeeder, p.LastAnnounce)
	return err
}

// ListPeersByTorrent 返回指定 torrent 的活跃 peer 列表（按最近上报排序）。
func (r *PeerRepo) ListPeersByTorrent(torrentID int64, limit int) ([]model.Peer, error) {
	peers := []model.Peer{}
	err := r.db.Select(&peers, `
SELECT id, torrent_id, peer_id, ip, port, uploaded, downloaded, "left", is_seeder, last_announce
FROM peers
WHERE torrent_id = ?
ORDER BY last_announce DESC
LIMIT ?`, torrentID, limit)
	if err == sql.ErrNoRows {
		return peers, nil
	}
	return peers, err
}

// DeleteStale 删除超过 timeout 未 announce 的 peer（由后台协程周期调用）。
func (r *PeerRepo) DeleteStale(timeout time.Duration) (int64, error) {
	res, err := r.db.Exec(`
DELETE FROM peers
WHERE last_announce < ?`,
		time.Now().Add(-timeout).Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// CountStats 返回统计信息：种子数与做种/下载人数。
// 返回 (seeders, leechers, error)。
func (r *PeerRepo) CountStats() (int64, int64, error) {
	var seeders, leechers int64
	err := r.db.Get(&seeders, `SELECT COUNT(*) FROM peers WHERE is_seeder = 1`)
	if err != nil {
		return 0, 0, err
	}
	err = r.db.Get(&leechers, `SELECT COUNT(*) FROM peers WHERE is_seeder = 0`)
	if err != nil {
		return 0, 0, err
	}
	return seeders, leechers, nil
}

// CountTorrents 返回有活跃 peer 的 torrent 数量（供 GetStats 使用）。
func (r *PeerRepo) CountTorrents() (int64, error) {
	var n int64
	err := r.db.Get(&n, `SELECT COUNT(DISTINCT torrent_id) FROM peers`)
	return n, err
}
