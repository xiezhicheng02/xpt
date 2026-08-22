package service

import (
	"encoding/binary"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/xiezc/xpt/internal/shared"
	"github.com/xiezc/xpt/pkg/netutil"
)

// UDPTracker 实现 BEP 15 UDP tracker 协议。
//
// 学习目标：
//  1. connect 握手：客户端发固定魔数，服务器回复 connection_id；
//  2. announce：携带 connection_id + infohash + peer 信息，服务器返回 peer 列表；
//  3. 用 connection_id 缓存防止伪造（本骨架用简单 map 缓存）。
//
// TODO(学习): 完整实现细节——
//   - announce 报文 98 字节的逐字段解析（netutil 已提供打包函数）；
//   - scrape 请求处理；
//   - connection 缓存过期清理；
//   - 错误响应（action=3）返回。
type UDPTracker struct {
	core *TrackerCore
	conn *net.UDPConn

	mu           sync.Mutex
	connections  map[int64]time.Time // connection_id -> 创建时间
	nextConnID   int64
	announceWait time.Duration
}

// NewUDPTracker 监听 udpAddr 并返回 UDPTracker。
func NewUDPTracker(core *TrackerCore, udpAddr string) (*UDPTracker, error) {
	addr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return &UDPTracker{
		core:         core,
		conn:         conn,
		connections:  make(map[int64]time.Time),
		nextConnID:   1,
		announceWait: 5 * time.Minute,
	}, nil
}

// Run 启动 UDP 读取循环，直到 conn 关闭。
func (t *UDPTracker) Run() {
	log := slog.With("component", "udp-tracker")
	log.Info("udp tracker listening", "addr", t.conn.LocalAddr().String())

	buf := make([]byte, 2048)
	for {
		n, addr, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			log.Warn("udp read error", "err", err)
			return
		}
		pkt := make([]byte, n)
		copy(pkt, buf[:n])
		go t.handlePacket(addr, pkt)
	}
}

// Close 关闭 UDP 监听。
func (t *UDPTracker) Close() {
	if t.conn != nil {
		t.conn.Close()
	}
}

// handlePacket 根据包长度与 action 分派处理。
func (t *UDPTracker) handlePacket(addr *net.UDPAddr, pkt []byte) {
	if len(pkt) < 16 {
		return
	}
	action := binary.BigEndian.Uint32(pkt[8:12])

	switch action {
	case shared.UDPActionConnect:
		t.handleConnect(addr, pkt)
	case shared.UDPActionAnnounce, shared.UDPActionAnnounce6:
		t.handleAnnounce(addr, pkt)
	case shared.UDPActionScrape:
		t.handleScrape(addr, pkt)
	default:
		slog.Warn("unknown udp action", "action", action, "from", addr.String())
	}
}

// handleConnect 处理 connect 请求，分配 connection_id。
func (t *UDPTracker) handleConnect(addr *net.UDPAddr, pkt []byte) {
	if len(pkt) < shared.UDPConnectRequestLen {
		return
	}
	// 校验协议魔数。
	protocolID := int64(binary.BigEndian.Uint64(pkt[0:8]))
	if protocolID != shared.UDPProtocolID {
		return
	}
	txID := binary.BigEndian.Uint32(pkt[12:16])

	t.mu.Lock()
	connID := t.nextConnID
	t.nextConnID++
	t.connections[connID] = time.Now()
	t.mu.Unlock()

	resp := make([]byte, 16)
	binary.BigEndian.PutUint32(resp[0:4], shared.UDPActionConnect)
	binary.BigEndian.PutUint32(resp[4:8], txID)
	binary.BigEndian.PutUint64(resp[8:16], uint64(connID))
	t.conn.WriteToUDP(resp, addr)
}

// handleAnnounce 处理 announce 请求。
// TODO(学习): 解析 98 字节报文，调 core.Announce，回包 peer 列表。
func (t *UDPTracker) handleAnnounce(addr *net.UDPAddr, pkt []byte) {
	if len(pkt) < shared.UDPAnnounceRequestLen {
		return
	}
	connID := int64(binary.BigEndian.Uint64(pkt[0:8]))
	txID := binary.BigEndian.Uint32(pkt[12:16])

	// 校验 connection_id 是否有效。
	t.mu.Lock()
	_, ok := t.connections[connID]
	t.mu.Unlock()
	if !ok {
		// TODO: 返回 action=3 错误包。
		return
	}

	// TODO(学习): 提取 infohash/peer_id/port/uploaded/downloaded/left/event，
	// 构造 AnnounceRequest 调 t.core.Announce，把返回的 peers 打包回发。
	_ = txID
	slog.Debug("announce received", "from", addr.String(), "conn_id", connID)
}

// handleScrape 处理 scrape 请求。
// TODO(学习): 返回多个 torrent 的种子/下载者统计。
func (t *UDPTracker) handleScrape(addr *net.UDPAddr, pkt []byte) {
	slog.Debug("scrape received", "from", addr.String())
	// TODO: 实现 scrape 响应。
}

// 编译期断言：netutil 的打包函数在骨架中保持可用。
var _ = netutil.PackConnectRequest
var _ = netutil.PackAnnounceRequest
