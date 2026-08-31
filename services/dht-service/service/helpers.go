package service

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xiezc/xpt/services/dht-service/migrate"
)

// portOf 从 "host:port" 形式的地址中解析端口号。
func portOf(addr string) int {
	_, portStr, err := netSplitHostPort(addr)
	if err != nil {
		return 6881 // 默认 DHT 端口
	}
	p, _ := strconv.Atoi(portStr)
	return p
}

// netSplitHostPort 是 net.SplitHostPort 的薄封装，便于替换默认值。
func netSplitHostPort(addr string) (string, string, error) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, "", errBadAddr
	}
	return addr[:idx], addr[idx+1:], nil
}

// errBadAddr 表示地址格式非法。
var errBadAddr = &addrError{}

type addrError struct{}

func (e *addrError) Error() string { return "invalid address" }

// randomNodeID 生成 DHT 节点 ID（20 字节随机）。
func randomNodeID() []byte {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return b
}

// hexID 将节点 ID 转为十六进制字符串（用于日志）。
func hexID(id migrate.Hash) string {
	return hex.EncodeToString(id[:])
}

// PendingRequest 待处理的主动请求
type PendingRequest struct {
	Method     string       // 请求方法：find_node/get_peers/announce_peer
	Target     []byte       // 目标：节点ID 或 infohash
	RemoteAddr *net.UDPAddr // 目标节点地址
	SentAt     time.Time    // 发送时间
	Retry      int          // 已重试次数
}

// PendingTable 待处理请求表，并发安全
type PendingTable struct {
	mu    sync.RWMutex
	items map[string]*PendingRequest // key: 事务ID字符串
}

// NewPendingTable 初始化待处理表
func NewPendingTable() *PendingTable {
	return &PendingTable{
		items: make(map[string]*PendingRequest),
	}
}

// Match 匹配响应，匹配成功则返回并删除缓存
func (pt *PendingTable) Match(txID []byte, from *net.UDPAddr) (*PendingRequest, bool) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := string(txID)
	req, ok := pt.items[key]
	if !ok {
		return nil, false
	}

	// 双重校验：源地址必须匹配，防止串包
	if req.RemoteAddr.IP.Equal(from.IP) && req.RemoteAddr.Port == from.Port {
		delete(pt.items, key)
		return req, true
	}

	return nil, false
}

// CleanExpired 清理过期请求，建议每秒调用一次
func (pt *PendingTable) CleanExpired() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	for key, req := range pt.items {
		if now.Sub(req.SentAt) > 15*time.Second {
			delete(pt.items, key)
		}
	}
}
