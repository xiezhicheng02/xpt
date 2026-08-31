package repository

import (
	"net"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/xiezc/xpt/services/dht-service/migrate"
)

// DHTNode 表示 DHT 网络中的一个已知节点。
// 字段与 dht_nodes 表对应。
type DHTNode struct {
	NodeID   migrate.Hash `db:"node_id"`
	KBucket  int          `db:"k_bucket "`
	IP       net.IP       `db:"ip"`
	Port     int          `db:"port"`
	Token    []byte       `db:"token"`
	LastSeen time.Time    `db:"last_seen"`
}

// NodeRepo 负责 dht_nodes 表的读写。
type NodeRepo struct {
	db *sqlx.DB
}

func NewNodeRepo(db *sqlx.DB) *NodeRepo {
	return &NodeRepo{db: db}
}

// UpsertNode 插入或更新一个已知节点（以 node_id 去重）。
func (r *NodeRepo) UpsertNode(n *DHTNode) error {
	_, err := r.db.Exec(`
		INSERT INTO dht_nodes(node_id, k_bucket, ip, port, token,  last_seen)
		VALUES (?, ?, ?, ?, ?,? )
		ON CONFLICT(node_id) DO UPDATE SET
			k_bucket = COALESCE(excluded.k_bucket, dht_nodes.k_bucket),
			ip = COALESCE(excluded.ip, dht_nodes.ip),
			port = COALESCE(excluded.port, dht_nodes.port),
			token = COALESCE(excluded.token, dht_nodes.token),
		    last_seen = excluded.last_seen`,
		n.NodeID, n.KBucket, n.IP, n.Port, n.Token, n.LastSeen)
	return err
}

// ListRecentNodes 返回最近 last_seen 的节点，用于引导新节点加入网络。
func (r *NodeRepo) ListRecentNodes(kBucket int, limit int) ([]DHTNode, error) {
	nodes := make([]DHTNode, 0, limit)
	err := r.db.Select(&nodes,
		`SELECT node_id, ip, port,token, last_seen
		FROM dht_nodes where k_bucket = ? 
		ORDER BY last_seen DESC
		LIMIT ?`, kBucket, limit)
	return nodes, err
}

func (r *NodeRepo) GetNode(nodeID [20]byte) (*DHTNode, error) {
	nodes := make([]DHTNode, 0, 1)
	err := r.db.Select(&nodes,
		`SELECT node_id, ip, port,token, last_seen
		FROM dht_nodes where node_id = ? 
		ORDER BY last_seen DESC`, nodeID)
	return &nodes[0], err
}

func (r *NodeRepo) GetNodes(nodeID []*migrate.Hash) ([]DHTNode, error) {
	// 1. 空输入直接返回，避免生成非法SQL
	if len(nodeID) == 0 {
		return []DHTNode{}, nil
	}
	// 2. 固定数组转字节切片，适配SQL驱动参数绑定
	ids := make([][]byte, len(nodeID))
	for i := range nodeID {
		ids[i] = nodeID[i][:] // 零拷贝切片视图
	}

	// 3. 展开IN子句：生成带多个占位符的SQL和参数列表
	query, args, err := sqlx.In(
		`SELECT node_id, ip, port, token, last_seen
         FROM dht_nodes 
         WHERE node_id IN (?)
         ORDER BY last_seen DESC`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	// 4. 适配驱动的占位符格式（如$1、?等）
	query = r.db.Rebind(query)

	// 5. 预分配容量，避免查询后扩容
	nodes := make([]DHTNode, 0, len(nodeID))

	// 6. 执行查询
	err = r.db.Select(&nodes, query, args...)
	return nodes, err
}
