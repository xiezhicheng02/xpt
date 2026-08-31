// Package service 实现 DHT 爬虫核心逻辑。
//
// 学习目标：理解 Kademlia DHT 协议如何工作——
//  1. 节点通过 UDP 互相发送 find_node / get_peers / announce_peer 消息；
//  2. 爬虫不断向已知节点发 find_node，借返回的节点列表扩散；
//  3. 当收到 announce_peer 时，其中携带的 infohash 就是"有人在分享的种子"。
//
// 架构说明：采用双通道读写分离模型
//
//	recvLoop 接收协程 → recvCh 接收队列 → handleLoop 处理Worker → sendCh 发送队列 → sendLoop 发送协程
//	职责分层：IO层只做收发与格式校验，业务层只做逻辑处理，互不阻塞
package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"runtime"
	"time"

	"github.com/xiezc/xpt/pkg/bencode"
	"github.com/xiezc/xpt/pkg/util"
	"github.com/xiezc/xpt/services/dht-service/repository"
)

// -------------------- 内部消息结构体 --------------------

// Packet 接收队列中的报文：已完成基础解码校验
type Packet struct {
	addr           *net.UDPAddr   // 对端地址
	data           *bencode.BNode // 原始报文数据（后续替换为解析后的KRPC结构体）
	pendingRequest *PendingRequest
}

// DHTService 是 DHT 爬虫服务的主结构。
type DHTService struct {
	nodeRepo *repository.NodeRepo
	hashRepo *repository.InfoHashRepo

	udpConn *net.UDPConn
	// selfID 是本节点在 DHT 网络中的 20 字节随机 ID。
	selfID [20]byte

	// 双通道队列
	recvCh chan *Packet // 接收队列：IO层 → 业务层
	sendCh chan *Packet // 发送队列：业务层 → IO层

	failCount    int //没有进入通道而丢弃的包的数量,
	successCount int //送入通道的数量

	PendingTable *PendingTable
}

// New 构造 DHTService，并准备 UDP 监听。
func New(
	nodeRepo *repository.NodeRepo,
	hashRepo *repository.InfoHashRepo,
	udpAddr string,
) (*DHTService, error) {
	udp, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv6zero, Port: portOf(udpAddr)})
	if err != nil {
		return nil, err
	}

	id := util.SHA1([]byte(udpAddr))

	return &DHTService{
		nodeRepo: nodeRepo,
		hashRepo: hashRepo,
		udpConn:  udp,
		selfID:   id,

		// 队列缓冲：2048 是 DHT 场景的经验值，兼顾抗峰值与内存占用
		recvCh: make(chan *Packet, 2048),
		sendCh: make(chan *Packet, 2048),

		failCount:    0,
		successCount: 0,
		PendingTable: &PendingTable{
			items: make(map[string]*PendingRequest),
		},
	}, nil
}

// Run 启动 DHT 爬虫主循环，直到 ctx 被取消。
func (s *DHTService) Run(ctx context.Context) error {
	log := slog.With("component", "dht")
	log.Info("dht service started", "self_id", hexID(s.selfID))

	// 1. 启动发送协程：单协程负责所有UDP写入，保证原子性
	go s.sendLoop(ctx)

	// 2. 启动接收协程：单协程负责所有UDP读取，避免多协程抢包
	go s.receiveLoop(ctx)

	// 3. 启动N个处理Worker：CPU核数个，并行处理报文业务逻辑
	workerNum := runtime.NumCPU()
	for i := 0; i < workerNum; i++ {
		go s.handleLoop(ctx)
	}

	//3.1 发送引导请求
	for i := range BootstrapAddrs {
		s.SendFindNodeBootstrap(BootstrapAddrs[i])
	}

	// 4. 主协程阻塞等待取消信号
	<-ctx.Done()

	// 5. 关闭连接，触发读写协程快速退出
	err := s.udpConn.Close()
	if err != nil {
		slog.Error("failed to close UDP connection", "err", err)
		return err
	}
	return nil
}

var BootstrapAddrs = []string{
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
	"router.utorrent.com:6881",
	"dht.libtorrent.org:25401",
}

// SendFindNodeBootstrap 向指定引导节点发送 find_node 请求，用于冷启动填充路由表
// 参数 bootstrapAddr: 引导节点地址，格式 "host:port"，例如 "router.bittorrent.com:6881"
func (s *DHTService) SendFindNodeBootstrap(bootstrapAddr string) error {
	// 1. 解析域名到 UDP 地址（默认 IPv4，IPv6 改为 "udp6"）
	udpAddr, err := net.ResolveUDPAddr("udp4", bootstrapAddr)
	if err != nil {
		return fmt.Errorf("解析引导节点地址失败 %s: %w", bootstrapAddr, err)
	}

	// 2. 生成 2 字节随机事务 ID（DHT 标准长度）
	txID := make([]byte, 2)
	if _, err := rand.Read(txID); err != nil {
		return fmt.Errorf("生成事务ID失败: %w", err)
	}

	findId := randomNodeID()

	// 3. 构造标准 BEP 5 find_node 请求报文
	req := &bencode.BNode{
		Type: bencode.BDict,
		Dict: map[string]*bencode.BNode{
			"t": {
				Type: bencode.BString,
				Str:  txID,
			},
			"y": {
				Type: bencode.BString,
				Str:  []byte("q"),
			},
			"q": {
				Type: bencode.BString,
				Str:  []byte("find_node"),
			},
			"a": {
				Type: bencode.BDict,
				Dict: map[string]*bencode.BNode{
					"id": {
						Type: bencode.BString,
						Str:  findId, // 本节点 ID
					},
					"target": {
						Type: bencode.BString,
						Str:  findId, // 引导阶段查询自身ID，获取离自己最近的节点
					},
				},
			},
		},
	}

	// 4. bencode 编码为二进制报文
	data := req.Encode()

	// 5. 提前存入待处理请求表，后续响应通过 txID 匹配
	s.PendingTable.Put(txID, &PendingRequest{
		Method:     "find_node",
		Target:     findId,
		RemoteAddr: udpAddr,
		SentAt:     time.Now(),
		Retry:      0,
	})

	// 6. 通过 UDP 连接发送报文
	if _, err := s.udpConn.WriteTo(data, udpAddr); err != nil {
		// 发送失败则回滚清理待处理表
		s.PendingTable.Remove(txID)
		return fmt.Errorf("发送 find_node 到 %s 失败: %w", bootstrapAddr, err)
	}

	return nil
}

// receiveLoop 接收协程：只做 UDP 读取 + 前置解码校验 + 入队
// 核心原则：尽可能快地清空内核缓冲区，不做业务逻辑
func (s *DHTService) receiveLoop(ctx context.Context) {
	buf := make([]byte, 1025) // 单缓冲区复用，减少GC
	for {
		// 非阻塞检查取消信号
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 设置读取超时，保证最多1秒就能响应取消
		err := s.udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err != nil {
			slog.Error("udp SetReadDeadline error", "err", err)
			return
		}
		n, addr, err := s.udpConn.ReadFromUDP(buf)
		slog.Info("receive Pkd size", "size", n)
		if err != nil || n == 1025 {
			// 超时是正常现象，直接下一轮
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			// 上下文已取消，正常退出
			if ctx.Err() != nil {
				return
			}
			slog.Warn("udp read error", "err", err, n)
			continue
		}

		data, err := bencode.Decode(buf)
		if err != nil {
			slog.Warn("decode error", "err", err)
			continue
		}

		pkt := &Packet{
			addr: addr,
			data: data,
		}
		slog.Info("receive Pkd addr", "addr", pkt.addr)
		if s.checkReceivePacket(pkt) {
			// 非阻塞入队：队列满直接丢弃，DHT本身允许丢包
			select {
			case s.recvCh <- pkt:
				s.successCount++
			default:
				s.failCount++
			}
		}
	}
}

// 校验检查接受到的包的格式. 不正确的格式直接丢弃
func (s *DHTService) checkReceivePacket(pkt *Packet) bool {
	resp := pkt.data
	//收到的是响应, 检查事务的长度
	t, err := resp.GetDictValue("t")
	if err != nil {
		return false
	}
	y, err := resp.GetDictStrValue("y")
	if err != nil {
		//无法解析出来
		return false
	}
	switch y {
	case "e":
		slog.Error("受到错误响应, 直接丢弃")
		return false
	case "r":
		if len(t.Str) > 2 {
			slog.Error(" 收到响应的t长度不对, 直接丢弃 ")
			return false
		}
		//需要校验请求映射表, 只处理自己发送的请求的响应, 不是自己请求的直接抛弃
		txid, _ := t.ToByteString()
		pr, ok := s.PendingTable.Match(txid, pkt.addr)
		if !ok {
			// 匹配失败：不是自己的请求、已过期、地址不匹配，直接丢弃
			return false
		}
		pkt.pendingRequest = pr
		//返回true, 校验通过, 进入处理逻辑
		return true
	case "q":
		_, err := resp.GetDictValue("a")
		if err != nil {
			//没有请求体, 丢弃
			return false
		}
		if len(t.Str) > 4 {
			//不处理事务长度超过4的请求
			slog.Error(" 收到请求的t长度不对, 直接丢弃 ")
			return false
		}
		return true
	default:
		//不符合格式的请求, 直接丢弃
		return false
	}
}

// handleLoop 业务处理Worker：从接收队列取报文，执行业务逻辑
// 数量为CPU核数个，CPU密集型处理最优
func (s *DHTService) handleLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pkt := <-s.recvCh:
			// 调用业务处理入口
			s.handleMessage(pkt)
		}
	}
}

// sendLoop 发送协程：从发送队列取报文，统一写UDP
// 单协程写入，避免多协程并发写导致的报文交错与系统调用开销
func (s *DHTService) sendLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case pkt := <-s.sendCh:
			// 发送失败直接忽略，DHT不保证可靠交付
			_, _ = s.udpConn.WriteToUDP(pkt.data.Encode(), pkt.addr)
		}
	}
}

// Send 对外发送接口：业务层只需要调用此方法发消息，不用关心UDP细节
// 非阻塞发送，队列满直接丢弃
func (s *DHTService) Send(addr *net.UDPAddr, data *bencode.BNode) {
	pkt := &Packet{
		addr: addr,
		data: data,
	}

	select {
	case s.sendCh <- pkt:
	default:
		s.failCount++
	}
}

func (s *DHTService) handleMessage(pkt *Packet) {
	// 骨架：暂不实现，等你学完 bencode 后填充。
	slog.Debug("udp packet", "from", pkt.addr.String(), "type", pkt.data.Type)
	data := pkt.data
	y, err := data.GetDictStrValue("y")
	if err != nil {
		slog.Warn("get y error", "err", err)
		return
	}
	if y == "q" {
		s.handleQueryMessage(pkt)
	}
	if y == "r" {
		s.handleResponseMessage(pkt)
	}
	if y == "e" {
		e, _ := data.GetDictStrValue("e")
		slog.Info("收到错误信息响应", "e", e)
	}

}

// Close 释放资源。
func (s *DHTService) Close() {
	if s.udpConn != nil {
		s.udpConn.Close()
	}
}
