// 处理DHT的消息接收 和 响应的逻辑
package service

import (
	"log/slog"
	"net"
	"time"

	"github.com/xiezc/xpt/pkg/bencode"
	"github.com/xiezc/xpt/services/dht-service/repository"
)

func (s *DHTService) handleResponseMessage(addr *net.UDPAddr, data []byte) {
	// 骨架：暂不实现，等你学完 bencode 后填充。

}

func (s *DHTService) handleQueryMessage(pkt *Packet) {
	node := pkt.data
	q, err := node.GetDictStrValue("q")
	if err != nil {
		slog.Error("handleQueryMessage", "err", err)
	}

	switch q {
	case "ping":
		//更新数据库信息
		s.upsertDhtNode(pkt)
		//发送响应信息
		s.sendPingResponse(pkt)
	case "find_node":
		//组装find_node响应，填充nodes   {"t":"原txid","y":"r","r":{"id":"我方id", "nodes":"26字节*N压缩节点列表"}}
		//更新数据库信息
		s.upsertDhtNode(pkt)

	case "get_peers":
		//生成token，返回token+nodes
		//{"t":"原txid","y":"r","r":{
		//"id":"我方id",
		//"token":"8字节随机token",
		//"values":["6字节peer1","6字节peer2"...]
		//}}

	case "announce_peer":
		//校验token，提取info_hash存入hashRepo，返回简单response
	}

}

func (s *DHTService) sendFindNodeResponse(pkt *Packet) {
	t, err := pkt.data.GetDictValue("t")
	//{"t":"原txid","y":"r","r":{"id":"我们自己的selfID(20字节)"}}
	if err != nil {
		slog.Info("sendFindNodeResponse 不回复无法解析txid的findNode响应", "err", err)
		return
	}
	txid, err := t.ToByteString()
	if err != nil {
		slog.Info("sendFindNodeResponse 不回复无法解析txid的findNode响应", "err", err)
		return
	}

	a, err := pkt.data.GetDictValue("a")
	if err != nil {
		slog.Info("sendFindNodeResponse 不回复无法解析a的findNode响应", "err", err)
		return
	}
	//获取请求的目标数据
	target, err := a.GetDictValue("target")
	if err != nil {
		slog.Info("sendFindNodeResponse 不回复无法解析target的findNode响应", "err", err)
		return
	}
	targetId, err := target.ToByteString()
	if err != nil {
		slog.Info("sendFindNodeResponse 无法获取target的数组格式, 不回复响应了", err)
		return
	}
	if len(targetId) != 20 {
		slog.Info("sendFindNodeResponse 获取target的长度不是20, 不回复响应了", "len", len(targetId))
		return
	}

	nodes, err := s.nodeRepo.ListRecentNodes(targetId, 8)
	if err != nil {
		slog.Info("sendFindNodeResponse 无法获取target的数据库中同K桶的数据, 不回复响应了", err)
		return
	}

	var nodes4 = make([]byte, 0, 8*26)
	var nodes6 = make([]byte, 0, 8*38)
	for i := range nodes {
		node := nodes[i]
		nodeId := node.NodeID
		if node.Port == nil || *node.Port <= 0 || *node.Port > 65535 {
			//端口号不正确
			continue
		}
		var portbuf [2]byte
		portbuf[0] = byte(*node.Port >> 8)   // 高 8 位
		portbuf[1] = byte(*node.Port & 0xFF) // 低 8 位

		ip4 := node.IP.To4()
		if ip4 != nil {
			nodes4 = append(nodes4, nodeId[:]...)
			nodes4 = append(nodes4, ip4[:]...)
			nodes4 = append(nodes4, portbuf[:]...)
		} else {
			ip6 := node.IP.To16()
			if ip6 != nil {
				nodes6 = append(nodes6, nodeId[:]...)
				nodes6 = append(nodes6, ip6[:]...)
				nodes6 = append(nodes6, portbuf[:]...)
			}
		}
	}
	r := &bencode.BNode{Type: bencode.BDict, Dict: map[string]*bencode.BNode{
		"id": {Type: bencode.BString, Str: s.selfID[:]},
	},
	}
	if len(nodes4) > 0 {
		r.Dict["nodes"] = &bencode.BNode{Type: bencode.BString, Str: nodes4}
	}

	if len(nodes6) > 0 {
		r.Dict["nodes6"] = &bencode.BNode{Type: bencode.BString, Str: nodes6}
	}
	response := &bencode.BNode{
		Type: bencode.BDict,
		Dict: map[string]*bencode.BNode{
			"t": {Type: bencode.BString, Str: txid},
			"y": {Type: bencode.BString, Str: []byte("r")},
			"r": r,
		},
	}

	//发送到通道中
	sendPktToChannel(response, pkt.addr, s)
}

func sendPktToChannel(sendData *bencode.BNode, addr *net.UDPAddr, s *DHTService) {
	sendPkt := &Packet{
		addr,
		sendData,
	}
	select {
	case s.sendCh <- sendPkt:
		s.successCount++
	default:
		// 队列满，丢弃，可加计数指标
		s.failCount++
	}
}

func (s *DHTService) sendPingResponse(pkt *Packet) {
	t, err := pkt.data.GetDictValue("t")
	//{"t":"原txid","y":"r","r":{"id":"我们自己的selfID(20字节)"}}
	if err != nil {
		slog.Error("sendPingResponse 不回复无法解析txid的ping响应", "err", err)
		return
	}
	txid, err := t.ToByteString()
	if err != nil {
		slog.Error("sendPingResponse 不回复无法解析txid的ping响应", "err", err)
		return
	}
	response := &bencode.BNode{
		Type: bencode.BDict,
		Dict: map[string]*bencode.BNode{
			"t": {Type: bencode.BString, Str: txid},
			"y": {Type: bencode.BString, Str: []byte("r")},
			"r": {Type: bencode.BDict, Dict: map[string]*bencode.BNode{
				"id": {Type: bencode.BString, Str: s.selfID[:]},
			},
			},
		},
	}

	//发送消息
	//发送到通道中
	sendPktToChannel(response, pkt.addr, s)
}

func (s *DHTService) upsertDhtNode(pkt *Packet) {
	node := pkt.data
	t, err := pkt.data.GetDictValue("t")
	if err != nil {
		slog.Error("sendPingResponse 不回复无法解析txid的ping响应", "err", err)
	}
	_, err = t.ToByteString()
	if err != nil {
		slog.Error("sendPingResponse 不回复无法解析txid的ping响应", "err", err)
	}
	a, err := node.GetDictValue("a")
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
	// 核心赋值：把切片内容拷贝到固定数组
	var nodeId [20]byte
	copy(nodeId[:], bytes)

	ip := pkt.addr.IP.To16()
	port := pkt.addr.Port

	now := time.Now()
	dhtNode := repository.DHTNode{
		NodeID:   &nodeId,
		KBucket:  bytes[0],
		IP:       ip,
		Port:     &port,
		Token:    nil,
		LastSeen: &now,
	}
	err = s.nodeRepo.UpsertNode(&dhtNode)
	if err != nil {
		slog.Error("upsertDhtNode 出现异常, 保存数据库数据报错", err)
		return
	}
}
