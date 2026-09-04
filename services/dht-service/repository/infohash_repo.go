package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/xiezc/xpt/services/dht-service/migrate"
)

// InfoHash 表示 DHT 网络中发现的 infohash。
// 字段与 dht_infohash 表对应。
type InfoHash struct {
	PeerId      *migrate.Hash `db:"peer_id"`
	InfoHash    *migrate.Hash `db:"info_hash"`
	Port        *int          `db:"port"`
	ImpliedPort *int          `db:"implied_port"`
	LastSeen    time.Time     `db:"last_seen"`
}

// InfoHashRepo 负责 dht_infohash 表的读写。
type InfoHashRepo struct {
	db *sqlx.DB
}

func NewInfoHashRepo(db *sqlx.DB) *InfoHashRepo {
	return &InfoHashRepo{db: db}
}

func (r *InfoHashRepo) DeleteByNodeId(nodeId migrate.Hash) error {
	_, err := r.db.Exec(
		`DELETE FROM  dht_infohash where peer_id = ?`, nodeId)
	return err
}

// Insert 记录一个新发现的 infohash（重复则忽略）。
func (r *InfoHashRepo) UpsertInfoHash(infoHash *InfoHash) error {
	_, err := r.db.Exec(`
		INSERT INTO dht_infohash(peer_id, info_hash, port, implied_port, last_seen)
		VALUES (?, ?, ?, ?, ? )
		ON CONFLICT(peer_id, info_hash) DO UPDATE SET
			peer_id = COALESCE(excluded.peer_id, dht_infohash.peer_id),
			info_hash = COALESCE(excluded.info_hash, dht_infohash.info_hash),
			port = COALESCE(excluded.port, dht_infohash.port),
			implied_port = COALESCE(excluded.implied_port, dht_infohash.implied_port),
		    last_seen = excluded.last_seen`,
		infoHash.PeerId, infoHash.InfoHash, infoHash.Port, infoHash.ImpliedPort, infoHash.LastSeen)
	return err
}

func (r *InfoHashRepo) ListByInfoHash(hash *migrate.Hash, klimit int) ([]InfoHash, error) {
	var rows []InfoHash
	err := r.db.Select(&rows, `
		SELECT peer_id, info_hash, port, implied_port, last_seen
		FROM dht_infohash where info_hash = ?
		ORDER BY last_seen DESC limit ?`, hash, klimit)
	if errors.Is(err, sql.ErrNoRows) {
		return rows, nil
	}
	return rows, err
}
