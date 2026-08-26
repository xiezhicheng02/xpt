package repository

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// DHTNode 表示 DHT 网络中的一个已知节点。
// 字段与 dht_nodes 表对应。
type DHTNode struct {
	NodeID   [20]byte   `db:"node_id"`
	KBucket  byte       `db:"k_bucket "`
	IP4      *string    `db:"ip4"`
	Port4    *int       `db:"port4"`
	Token4   *string    `db:"token4"`
	IP6      *string    `db:"ip6"`
	Port6    *int       `db:"port6"`
	Token6   *string    `db:"token6"`
	LastSeen *time.Time `db:"last_seen"`
}

// NodeRepo 负责 dht_nodes 表的读写。
type NodeRepo struct {
	db *sqlx.DB
}

func NewNodeRepo(db *sqlx.DB) *NodeRepo {
	return &NodeRepo{db: db}
}

// UpsertNode 插入或更新一个已知节点（以 node_id 去重）。
// TODO: 实现时注意并发写同一 node_id 的场景，可用 INSERT ... ON CONFLICT。
func (r *NodeRepo) UpsertNode(n *DHTNode) error {

	_, err := r.db.Exec(`
		INSERT INTO dht_nodes(node_id, k_bucket, ip4, port4, token4, ip6, port6, token6, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?,?,? )
		ON CONFLICT(node_id) DO UPDATE SET
			k_bucket = COALESCE(excluded.k_bucket, dht_nodes.k_bucket),
			ip4 = COALESCE(excluded.ip4, dht_nodes.ip4),
			port4 = COALESCE(excluded.port4, dht_nodes.port4),
			token4 = COALESCE(excluded.token4, dht_nodes.token4),
			ip6 = COALESCE(excluded.ip6, dht_nodes.ip6),
			port6 = COALESCE(excluded.port6, dht_nodes.port6),
			token6 = COALESCE(excluded.token6, dht_nodes.token6),
		    last_seen = excluded.last_seen`,
		n.NodeID, n.KBucket, n.IP4, n.Port4, n.Token4, n.IP6, n.Port6, n.Token6, n.LastSeen)
	return err
}

// ListRecentNodes 返回最近 last_seen 的节点，用于引导新节点加入网络。
func (r *NodeRepo) ListRecentNodes(limit int) ([]DHTNode, error) {
	nodes := []DHTNode{}
	err := r.db.Select(&nodes, `
SELECT id, node_id, ip, port, last_seen
FROM dht_nodes
ORDER BY last_seen DESC
LIMIT ?`, limit)
	if err == sql.ErrNoRows {
		return nodes, nil
	}
	return nodes, err
}

// CountNodes 返回已知节点总数。
func (r *NodeRepo) CountNodes() (int, error) {
	var n int
	err := r.db.Get(&n, `SELECT COUNT(*) FROM dht_nodes`)
	return n, err
}
