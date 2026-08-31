// 处理DHT的消息接收 和 响应的逻辑
package service

import (
	"encoding/binary"
	"log/slog"
	"net"
	"time"

	"github.com/xiezc/xpt/services/dht-service/migrate"
	"github.com/xiezc/xpt/services/dht-service/repository"
)

func (s *DHTService) handleResponseMessage(pkt *Packet) {
	// 骨架：暂不实现，等你学完 bencode 后填充。
	data := pkt.data
	t, err := data.GetDictValue("t")
	txid, err := t.ToByteString()
	if err != nil {
		slog.Error("handleQueryMessage", "err", err)
		return
	}

	_, err = data.GetDictStrValue("r")
	if err != nil {
		slog.Error("handleQueryMessage", "err", err)
		s.sendError(pkt.addr, txid, 201, "无效的")
		return
	}

	req := pkt.pendingRequest
	switch req.Method {
	case "ping":
		//更新数据库信息
		s.upsertRespDhtNode(pkt)
	case "find_node":
		s.upsertRespDhtNode(pkt)
		s.handleFindNodeResp(pkt) //处理findNode包
	case "get_peers":
		s.upsertRespDhtNode(pkt)
		s.handleGetPeersResp(pkt)
	case "announce_peers":
		s.upsertRespDhtNode(pkt)

	}
}

func (s *DHTService) handleAnnouncePeersResp(pkt *Packet) {
	r, _ := pkt.data.GetDictValue("r")
	id, _ := r.GetDictValue("id")
	if id != nil {
		values66, _ := id.ToHexString()
		slog.Info("handleAnnouncePeersResp 不处理响应", "id", values66)
	}
}

func (s *DHTService) handleGetPeersResp(pkt *Packet) {
	r, _ := pkt.data.GetDictValue("r")
	nodes, _ := r.GetDictValue("nodes")
	if nodes != nil {
		nodes4, _ := nodes.ToByteString()
		size := len(nodes4) / 26
		for i := range size {
			aa := nodes4[26*(i-1) : 26*i]
			id := aa[0:20]
			ip := aa[20:24]
			port := int(binary.BigEndian.Uint16(aa[24:26]))

			nodeId := *(*migrate.Hash)(id)
			dhtNode := &repository.DHTNode{
				NodeID:   nodeId,
				KBucket:  xorBucket(s.selfID, nodeId),
				IP:       net.IPv4(ip[0], ip[1], ip[2], ip[3]),
				Port:     port,
				Token:    nil,
				LastSeen: time.Now(),
			}
			err := s.nodeRepo.UpsertNode(dhtNode)
			if err != nil {
				continue
			}
		}

	}

	nodes6, _ := r.GetDictValue("nodes6")
	if nodes6 != nil {
		nodes6B, _ := nodes.ToByteString()
		size := len(nodes6B) / 38
		for i := range size {
			aa := nodes6B[38*(i-1) : 38*i]
			id := aa[0:20]
			ip := aa[20:36]
			port := int(binary.BigEndian.Uint16(aa[36:38]))

			nodeId := *(*migrate.Hash)(id)
			dhtNode := &repository.DHTNode{
				NodeID:   nodeId,
				KBucket:  xorBucket(s.selfID, nodeId),
				IP:       ip,
				Port:     port,
				Token:    nil,
				LastSeen: time.Now(),
			}
			err := s.nodeRepo.UpsertNode(dhtNode)
			if err != nil {
				continue
			}
		}
	}

	values, _ := r.GetDictValue("values")
	if values != nil {
		values4, _ := values.ToHexString()
		slog.Info("handleGetPeersResp 暂不处理", "values", values4)
	}
	values6, _ := r.GetDictValue("values6")
	if values6 != nil {
		values66, _ := values6.ToHexString()
		slog.Info("handleGetPeersResp 暂不处理", "values6", values66)
	}

}

func (s *DHTService) handleFindNodeResp(pkt *Packet) {
	r, _ := pkt.data.GetDictValue("r")
	nodes, _ := r.GetDictValue("nodes")
	if nodes != nil {
		nodes4, _ := nodes.ToByteString()
		size := len(nodes4) / 26
		for i := range size {
			aa := nodes4[26*(i-1) : 26*i]
			id := aa[0:20]
			ip := aa[20:24]
			port := int(binary.BigEndian.Uint16(aa[24:26]))

			nodeId := *(*migrate.Hash)(id)
			dhtNode := &repository.DHTNode{
				NodeID:   nodeId,
				KBucket:  xorBucket(s.selfID, nodeId),
				IP:       net.IPv4(ip[0], ip[1], ip[2], ip[3]),
				Port:     port,
				Token:    nil,
				LastSeen: time.Now(),
			}
			err := s.nodeRepo.UpsertNode(dhtNode)
			if err != nil {
				continue
			}
		}

	}

	nodes6, _ := r.GetDictValue("nodes6")
	if nodes6 != nil {
		nodes6B, _ := nodes.ToByteString()
		size := len(nodes6B) / 38
		for i := range size {
			aa := nodes6B[38*(i-1) : 38*i]
			id := aa[0:20]
			ip := aa[20:36]
			port := int(binary.BigEndian.Uint16(aa[36:38]))

			nodeId := *(*migrate.Hash)(id)
			dhtNode := &repository.DHTNode{
				NodeID:   nodeId,
				KBucket:  xorBucket(s.selfID, nodeId),
				IP:       ip,
				Port:     port,
				Token:    nil,
				LastSeen: time.Now(),
			}
			err := s.nodeRepo.UpsertNode(dhtNode)
			if err != nil {
				continue
			}
		}

	}
}

func (s *DHTService) upsertRespDhtNode(pkt *Packet) {
	node := pkt.data
	t, err := pkt.data.GetDictValue("t")
	if err != nil {
		slog.Error("sendPingResponse 不回复无法解析txid的ping响应", "err", err)
	}
	_, err = t.ToByteString()
	if err != nil {
		slog.Error("sendPingResponse 不回复无法解析txid的ping响应", "err", err)
	}
	a, err := node.GetDictValue("r")
	if err != nil {
		slog.Error("upsertDhtNode", "err", err)
		return
	}

	reqId, err := a.GetDictValue("id")
	if err != nil {
		slog.Error("upsertDhtNode", "err", err)
		return
	}
	bytes, err := reqId.ToByteString()
	if err != nil {
		slog.Error("upsertDhtNode", "err", err)
		return
	}
	if len(bytes) != 20 {
		slog.Error("upsertDhtNode  reqId len != 20 ", "err", err)
		return
	}

	token, _ := a.GetDictValue("token")
	tokenB, _ := token.ToByteString()

	// 核心赋值：把切片内容拷贝到固定数组
	var nodeId = *(*migrate.Hash)(bytes)

	ip := pkt.addr.IP.To16()
	port := pkt.addr.Port

	kBucket := xorBucket(s.selfID, nodeId)
	dhtNode := repository.DHTNode{
		NodeID:   nodeId,
		KBucket:  kBucket,
		IP:       ip,
		Port:     port,
		Token:    tokenB,
		LastSeen: time.Now(),
	}
	err = s.nodeRepo.UpsertNode(&dhtNode)
	if err != nil {
		slog.Error("upsertDhtNode 出现异常, 保存数据库数据报错", err)
		return
	}
}
