package model

import "time"

// Torrent 表示一个被上传的种子文件及其统计信息。
// 字段与 web-server 的 torrents 表对应。
type Torrent struct {
	ID         int64     `db:"id" json:"id"`
	InfoHash   string    `db:"info_hash" json:"info_hash"`
	Name       string    `db:"name" json:"name"`
	Size       int64     `db:"size" json:"size"`
	UploadedBy int64     `db:"uploaded_by" json:"uploaded_by"`
	IsDeleted  bool      `db:"is_deleted" json:"is_deleted"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`

	// 以下字段由 tracker 统计接口填充，不入库。
	Seeders  int `db:"-" json:"seeders"`
	Leechers int `db:"-" json:"leechers"`
}
