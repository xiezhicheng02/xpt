package repository

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// InfoHash 表示 DHT 网络中发现的 infohash。
// 字段与 dht_infohash 表对应。
type InfoHash struct {
	ID         int64     `db:"id"`
	InfoHash   string    `db:"info_hash"`
	Discovered time.Time `db:"discover_at"`
}

// InfoHashRepo 负责 dht_infohash 表的读写。
type InfoHashRepo struct {
	db *sqlx.DB
}

func NewInfoHashRepo(db *sqlx.DB) *InfoHashRepo {
	return &InfoHashRepo{db: db}
}

// Insert 记录一个新发现的 infohash（重复则忽略）。
func (r *InfoHashRepo) Insert(hash string) error {
	_, err := r.db.Exec(`
INSERT OR IGNORE INTO dht_infohash(info_hash, discover_at)
VALUES (?, ?)`, hash, time.Now())
	return err
}

// ListAll 返回全部已知 infohash。
// TODO: 数据量大后需要加分页，当前仅用于学习阶段。
func (r *InfoHashRepo) ListAll() ([]InfoHash, error) {
	rows := []InfoHash{}
	err := r.db.Select(&rows, `
SELECT id, info_hash, discover_at
FROM dht_infohash
ORDER BY discover_at DESC`)
	if err == sql.ErrNoRows {
		return rows, nil
	}
	return rows, err
}

// Count 返回已发现 infohash 的总数。
func (r *InfoHashRepo) Count() (int, error) {
	var n int
	err := r.db.Get(&n, `SELECT COUNT(*) FROM dht_infohash`)
	return n, err
}
