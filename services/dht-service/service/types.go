package service

import (
	"net"
	"sync"
	"time"

	"github.com/xiezc/xpt/pkg/bencode"
)

// -------------------- 内部消息结构体 --------------------

// Packet 接收队列中的报文：已完成基础解码校验
type Packet struct {
	addr *net.UDPAddr   // 对端地址
	data *bencode.BNode // 原始报文数据（后续替换为解析后的KRPC结构体）
}

// PendingRequest 待处理的主动请求
type PendingRequest struct {
	Method     string       // 请求方法：find_node/get_peers/announce_peer
	Target     []byte       // 目标：节点ID 或 infohash
	RemoteAddr *net.UDPAddr // 目标节点地址
	SentAt     time.Time    // 发送时间
	Retry      int          // 已重试次数
	RespChan   chan *Packet
}

// PendingTable 待处理请求表，并发安全
type PendingTable struct {
	mu    sync.RWMutex
	items map[string]*PendingRequest // key: 事务ID字符串
}
