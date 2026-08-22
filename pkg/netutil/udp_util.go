// Package netutil 提供 UDP tracker 协议（BEP 15）的报文构造与解析工具。
package netutil

import (
	"encoding/binary"
	"errors"
	"net"
)

// ErrShortPacket 表示收到的 UDP 包长度不足。
var ErrShortPacket = errors.New("udp: packet too short")

// UDPConnectRequest 是客户端发起的连接请求（固定 16 字节）。
type UDPConnectRequest struct {
	ProtocolID int64  // 固定 0x41727101980
	Action     uint32 // 0 = connect
	TxID       uint32 // 事务 ID，客户端自选
}

// UDPConnectResponse 是服务器对 connect 的响应（固定 16 字节）。
type UDPConnectResponse struct {
	Action     uint32
	TxID       uint32
	Connection int64 // 服务器下发的连接 ID，后续 announce/scrape 必须携带
}

// UDPAnnounceRequest 是 announce 请求（IPv4，固定 98 字节）。
// 字段顺序即线上字节序，实现时按 BEP 15 严格打包。
type UDPAnnounceRequest struct {
	ConnectionID int64
	Action       uint32
	TxID         uint32
	InfoHash     [20]byte
	PeerID       [20]byte
	Downloaded   int64
	Left         int64
	Uploaded     int64
	Event        uint32
	IPAddress    uint32
	Key          uint32
	NumWant      int32
	Port         uint16
}

// PackConnectRequest 序列化 connect 请求。
func PackConnectRequest(txID uint32) []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:8], uint64(0x41727101980))
	binary.BigEndian.PutUint32(buf[8:12], 0) // action=connect
	binary.BigEndian.PutUint32(buf[12:16], txID)
	return buf
}

// UnpackConnectResponse 解析 connect 响应。
func UnpackConnectResponse(buf []byte) (UDPConnectResponse, error) {
	var r UDPConnectResponse
	if len(buf) < 16 {
		return r, ErrShortPacket
	}
	r.Action = binary.BigEndian.Uint32(buf[0:4])
	r.TxID = binary.BigEndian.Uint32(buf[4:8])
	r.Connection = int64(binary.BigEndian.Uint64(buf[8:16]))
	return r, nil
}

// PackAnnounceRequest 序列化 announce 请求（IPv4）。
func PackAnnounceRequest(req UDPAnnounceRequest) []byte {
	buf := make([]byte, 98)
	binary.BigEndian.PutUint64(buf[0:8], uint64(req.ConnectionID))
	binary.BigEndian.PutUint32(buf[8:12], req.Action)
	binary.BigEndian.PutUint32(buf[12:16], req.TxID)
	copy(buf[16:36], req.InfoHash[:])
	copy(buf[36:56], req.PeerID[:])
	binary.BigEndian.PutUint64(buf[56:64], uint64(req.Downloaded))
	binary.BigEndian.PutUint64(buf[64:72], uint64(req.Left))
	binary.BigEndian.PutUint64(buf[72:80], uint64(req.Uploaded))
	binary.BigEndian.PutUint32(buf[80:84], req.Event)
	binary.BigEndian.PutUint32(buf[84:88], req.IPAddress)
	binary.BigEndian.PutUint32(buf[88:92], req.Key)
	binary.BigEndian.PutUint32(buf[92:96], uint32(req.NumWant))
	binary.BigEndian.PutUint16(buf[96:98], req.Port)
	return buf
}

// PackPeerList 将 peer 地址列表打包为 announce 响应的 peer 段
// （每个 peer 6 字节：IPv4 4 字节 + port 2 字节）。
func PackPeerList(peers []net.TCPAddr) []byte {
	buf := make([]byte, 0, 6*len(peers))
	for _, p := range peers {
		ip4 := p.IP.To4()
		if ip4 == nil {
			continue
		}
		buf = append(buf, ip4...)
		buf = append(buf, byte(p.Port>>8), byte(p.Port))
	}
	return buf
}
