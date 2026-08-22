package repository

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/xiezc/xpt/pkg/model"
)

// ErrTorrentNotFound 表示种子不存在。
var ErrTorrentNotFound = errors.New("torrent not found")

// TorrentRepo 负责 torrents 表的读写。
type TorrentRepo struct {
	db *sqlx.DB
}

func NewTorrentRepo(db *sqlx.DB) *TorrentRepo {
	return &TorrentRepo{db: db}
}

// Create 登记一个新种子（info_hash 唯一）。
func (r *TorrentRepo) Create(t *model.Torrent) error {
	_, err := r.db.Exec(`
INSERT INTO torrents(info_hash, name, size, uploaded_by)
VALUES (?, ?, ?, ?)`,
		t.InfoHash, t.Name, t.Size, t.UploadedBy)
	return err
}

// GetByID 按 ID 查询种子。
func (r *TorrentRepo) GetByID(id int64) (*model.Torrent, error) {
	var t model.Torrent
	err := r.db.Get(&t, `
SELECT id, info_hash, name, size, uploaded_by, is_deleted, created_at
FROM torrents WHERE id = ? AND is_deleted = 0`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTorrentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetByInfoHash 按 info_hash 查询种子。
func (r *TorrentRepo) GetByInfoHash(hash string) (*model.Torrent, error) {
	var t model.Torrent
	err := r.db.Get(&t, `
SELECT id, info_hash, name, size, uploaded_by, is_deleted, created_at
FROM torrents WHERE info_hash = ? AND is_deleted = 0`, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTorrentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// List 分页列出种子（按创建时间倒序）。
func (r *TorrentRepo) List(offset, limit int) ([]model.Torrent, error) {
	list := []model.Torrent{}
	err := r.db.Select(&list, `
SELECT id, info_hash, name, size, uploaded_by, is_deleted, created_at
FROM torrents
WHERE is_deleted = 0
ORDER BY created_at DESC
LIMIT ? OFFSET ?`, limit, offset)
	if err == sql.ErrNoRows {
		return list, nil
	}
	return list, err
}

// SoftDelete 软删除种子（标记 is_deleted，不物理删除）。
func (r *TorrentRepo) SoftDelete(id int64) error {
	_, err := r.db.Exec(`UPDATE torrents SET is_deleted = 1 WHERE id = ?`, id)
	return err
}

// Count 返回未删除种子总数。
func (r *TorrentRepo) Count() (int, error) {
	var n int
	err := r.db.Get(&n, `SELECT COUNT(*) FROM torrents WHERE is_deleted = 0`)
	return n, err
}
