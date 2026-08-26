// Package service 实现 DHT 爬虫核心逻辑。
//
// 学习目标：理解 Kademlia DHT 协议如何工作——
//  1. 节点通过 UDP 互相发送 find_node / get_peers / announce_peer 消息；
//  2. 爬虫不断向已知节点发 find_node，借返回的节点列表扩散；
//  3. 当收到 announce_peer 时，其中携带的 infohash 就是"有人在分享的种子"。
//
// 本文件提供可运行的骨架：UDP 收发、消息解析入口都已搭好，
// 但具体协议细节（KRPC 消息编解码、路由表、find_node 扩散）留给你逐步实现。
package service

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/xiezc/xpt/pkg/util"
	"github.com/xiezc/xpt/services/dht-service/repository"
)

// DHTService 是 DHT 爬虫服务的主结构。
type DHTService struct {
	nodeRepo *repository.NodeRepo
	hashRepo *repository.InfoHashRepo

	udpConn *net.UDPConn
	// selfID 是本节点在 DHT 网络中的 20 字节随机 ID。
	selfID []byte

	// bootstrapNodes 是初始引导节点（如 dht.transmissionbt.com）。
	// TODO: 从配置读取，填入知名 DHT 引导节点。
	bootstrapNodes []string

	// routingTable 是 Kademlia 路由表。
	// TODO: 实现桶结构（按距离分桶，桶容量 8）。
	routingTable map[string]bool
}

// New 构造 DHTService，并准备 UDP 监听。
func New(nodeRepo *repository.NodeRepo, hashRepo *repository.InfoHashRepo, udpAddr string) (*DHTService, error) {
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
	}, nil
}

// Run 启动 DHT 爬虫主循环，直到 ctx 被取消。
func (s *DHTService) Run(ctx context.Context) error {
	log := slog.With("component", "dht")
	log.Info("dht service started", "self_id", hexID(s.selfID))

	// TODO(学习): 启动时向 bootstrapNodes 发送 find_node，填充路由表。
	// 之后进入 UDP 读取循环：
	//   - find_node 响应 -> 把返回的节点写入 nodeRepo、加入路由表
	//   - get_peers 响应  -> 可能携带 token，暂不处理
	//   - announce_peer   -> 提取 infohash 写入 hashRepo

	go s.loop(ctx)
	<-ctx.Done()
	s.udpConn.Close()
	return nil
}

// loop 是 UDP 读取循环骨架。
func (s *DHTService) loop(ctx context.Context) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		s.udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := s.udpConn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			slog.Warn("udp read error", "err", err)
			continue
		}
		// TODO(学习): 调用 handleMessage 解析 KRPC 消息并分派。
		slog.Debug("udp packet", "from", addr.String(), "len", n)

	}
}

// handleMessage 处理一条 UDP 消息。
// TODO(学习): 用 bencode 解码消息，根据 y(类型) 字段分派：
//
//	y == "r" -> handleResponse
//	y == "q" -> handleQuery
func (s *DHTService) handleMessage(addr *net.UDPAddr, data []byte) {
	// 骨架：暂不实现，等你学完 bencode 后填充。

}

// Close 释放资源。
func (s *DHTService) Close() {
	if s.udpConn != nil {
		s.udpConn.Close()
	}
}
