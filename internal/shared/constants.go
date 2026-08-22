// Package shared 存放三个服务共用的常量与协议常量。
package shared

import "time"

// BitTorrent 协议常量。
const (
	// InfoHashLen 是 info hash（SHA1）的字节长度。
	InfoHashLen = 20
	// PeerIDLen 是 peer id 的字节长度。
	PeerIDLen = 20

	// DefaultAnnounceInterval 是 tracker 建议客户端两次 announce 的间隔。
	DefaultAnnounceInterval = 1800
	// MinAnnounceInterval 是 tracker 允许的最小间隔。
	MinAnnounceInterval = 900
	// PeerTimeout 是 peer 超过该时间未 announce 即视为离开。
	PeerTimeout = 30 * time.Minute

	// MaxPeersPerAnnounce 单次 announce 返回的最大 peer 数。
	MaxPeersPerAnnounce = 50
)

// UDP tracker 协议常量（BEP 15）。
const (
	// UDP 协议魔数，客户端握手时使用。
	UDPProtocolID int64 = 0x41727101980

	// UDP action 类型。
	UDPActionConnect   uint32 = 0
	UDPActionAnnounce  uint32 = 1
	UDPActionScrape    uint32 = 2
	UDPActionError     uint32 = 3
	UDPActionAnnounce6 uint32 = 4

	// UDP 包长度。
	UDPConnectRequestLen  = 16
	UDPConnectResponseLen = 16
	UDPAnnounceRequestLen = 98
)

// 服务端口默认值（与 config.yaml 保持一致）。
const (
	TrackerGRPCDefault = "127.0.0.1:50051"
	DhtGRPCDefault     = "127.0.0.1:50052"
	WebHTTPDefault     = "0.0.0.0:8080"
)
