package service

import (
	"bytes"
	"encoding/binary"
	"log/slog"
	"math/bits"
	"net"
	"time"

	"github.com/xiezc/xpt/pkg/bencode"
	"github.com/xiezc/xpt/services/dht-service/migrate"
	"github.com/xiezc/xpt/services/dht-service/repository"
)

func (s *DHTService) handleQueryMessage(pkt *Packet) {
	node := pkt.data
	q, err := node.GetDictStrValue("q")
	if err != nil {
		slog.Error("handleQueryMessage", "err", err)
	}
	switch q {
	case "ping":
		//更新数据库信息
		s.upsertQueryDhtNode(pkt)
		//发送响应信息
		s.sendPingResponse(pkt)
	case "find_node":
		//组装find_node响应，填充nodes   {"t":"原txid","y":"r","r":{"id":"我方id", "nodes":"26字节*N压缩节点列表"}}
		//更新数据库信息
		s.upsertQueryDhtNode(pkt)
		//发送响应
		s.sendFindNodeResponse(pkt)
	case "get_peers":
		//生成token，返回token+nodes
		//{"t":"原txid","y":"r","r":{
		//"id":"我方id",
		//"token":"8字节随机token",
		//"values":["6字节peer1","6字节peer2"...]
		//}}
		s.handleGetPeers(pkt)
	case "announce_peer":
		//校验token，提取info_hash存入hashRepo，返回简单response
		s.handleAnnouncePeer(pkt)
	}

}

func (s *DHTService) handleAnnouncePeer(pkt *Packet) {
	// 1. 提取事务ID
	t, err := pkt.data.GetDictValue("t")
	if err != nil {
		return
	}
	txid, err := t.ToByteString()
	if err != nil {
		return
	}

	// 2. 提取请求参数字典
	a, err := pkt.data.GetDictValue("a")
	if err != nil {
		s.sendError(pkt.addr, txid, 201, "missing a")
		return
	}

	// 3. 校验 info_hash
	ihNode, err := a.GetDictValue("info_hash")
	if err != nil {
		s.sendError(pkt.addr, txid, 201, "missing info_hash")
		return
	}
	infoHash, err := ihNode.ToByteString()
	if err != nil || len(infoHash) != 20 {
		s.sendError(pkt.addr, txid, 201, "invalid info_hash")
		return
	}

	// 3. 校验 info_hash
	reqId, err := a.GetDictValue("id")
	if err != nil {
		s.sendError(pkt.addr, txid, 201, "missing info_hash")
		return
	}
	idBytes, err := reqId.ToByteString()
	if err != nil {
		s.sendError(pkt.addr, txid, 201, "missing id")
		return
	}
	if len(idBytes) != 20 {
		s.sendError(pkt.addr, txid, 201, "missing id")
		return
	}
	id := *(*migrate.Hash)(idBytes)
	// 4. 校验 token
	tokenNode, err := a.GetDictValue("token")
	if err != nil {
		s.sendError(pkt.addr, txid, 201, "missing token")
		return
	}
	token, err := tokenNode.ToByteString()
	if err != nil {
		s.sendError(pkt.addr, txid, 201, "invalid token")
		return
	}
	if !s.verifyTokenAndIp(id, pkt.addr, token) {
		s.sendError(pkt.addr, txid, 203, "invalid token")
		return
	}

	// 5. 解析端口
	port := 0
	// 优先判断 implied_port
	implied, err := a.GetDictIntValue("implied_port")
	if err == nil {
		if implied == 1 {
			port = pkt.addr.Port
		}
	}
	// 没有 implied_port 则读取 port 字段
	if port == 0 {
		p, err := a.GetDictIntValue("port")
		if err != nil {
			s.sendError(pkt.addr, txid, 202, "missing port")
			return
		}
		if p <= 0 || p > 65535 {
			s.sendError(pkt.addr, txid, 202, "invalid port")
			return
		}
		port = int(p)
	}

	// 6. 存储 peer
	var ih migrate.Hash
	copy(ih[:], infoHash)

	hashPo := &repository.InfoHash{
		PeerId:      &id,
		InfoHash:    &ih,
		Port:        &port,
		ImpliedPort: &implied,
		LastSeen:    time.Now(),
	}

	err = s.hashRepo.UpsertInfoHash(hashPo)
	if err != nil {
		s.sendError(pkt.addr, txid, 202, "invalid request")
		return
	}

	// 7. 返回成功响应
	response := &bencode.BNode{
		Type: bencode.BDict,
		Dict: map[string]*bencode.BNode{
			"t": {Type: bencode.BString, Str: txid},
			"y": {Type: bencode.BString, Str: []byte("r")},
			"r": {
				Type: bencode.BDict,
				Dict: map[string]*bencode.BNode{
					"id": {Type: bencode.BString, Str: s.selfID[:]},
				},
			},
		},
	}
	sendPktToChannel(response, pkt.addr, s)
}

func (s *DHTService) sendError(addr *net.UDPAddr, t []byte, code int, msg string) {
	response := &bencode.BNode{
		Type: bencode.BDict,
		Dict: map[string]*bencode.BNode{
			"t": {Type: bencode.BString, Str: t},
			"y": {Type: bencode.BString, Str: []byte("e")},
			"e": {
				Type: bencode.BList,
				List: []*bencode.BNode{
					{Type: bencode.BInt, Int: 203},                        // 第1位：整数错误码
					{Type: bencode.BString, Str: []byte("invalid token")}, // 第2位：字符串错误信息
				},
			},
		},
	}
	sendPktToChannel(response, addr, s)
}

func (s *DHTService) verifyTokenAndIp(nodeId [20]byte, addr *net.UDPAddr, token []byte) bool {
	bNode, err := s.nodeRepo.GetNode(nodeId)
	if err != nil {
		slog.Error("verifyToken", "err", err)
		return false
	}
	t := bNode.Token
	if !bytes.Equal(t, token) {
		return false
	}
	//开始校验IP
	ip := addr.IP.To16()
	if !bytes.Equal(bNode.IP, ip) {
		return false
	}
	return true
}

func (s *DHTService) handleGetPeers(pkt *Packet) {
	t, err := pkt.data.GetDictValue("t")
	//响应：{"id" : "<查询节点 ID>", "token" : "<不透明写入令牌>", "values" : ["<对等节点 1 信息字符串>", "<对等节点 2 信息字符串>"]}
	//或者：{"id" : "<查询节点 ID>", "token" : "<不透明写入令牌>", "nodes" : "<精简节点信息>"}
	if err != nil {
		slog.Info("handleGetPeers 不回复无法解析txid的findNode响应", "err", err)
		return
	}
	txid, err := t.ToByteString()
	if err != nil {
		slog.Info("handleGetPeers 不回复无法解析txid的findNode响应", "err", err)
		return
	}

	a, err := pkt.data.GetDictValue("a")
	if err != nil {
		slog.Info("handleGetPeers 不回复无法解析a的findNode响应", "err", err)
		return
	}

	reqId, err := a.GetDictValue("id")
	if err != nil {
		slog.Info("handleGetPeers 不回复无法解析请求id的get_peers响应", "err", err)
		s.sendError(pkt.addr, txid, 202, "invalid id")
		return
	}
	info_hash, err := a.GetDictValue("info_hash")
	if err != nil {
		slog.Info("handleGetPeers 不回复无法解析info_hash的get_peers响应", "err", err)
		s.sendError(pkt.addr, txid, 202, "invalid info_hash")
		return
	}
	idBytes, err := reqId.ToByteString()
	if len(idBytes) != 20 || err != nil {
		slog.Info("handleGetPeers 不回复,id的长度不对", "err", err)
		s.sendError(pkt.addr, txid, 202, "invalid Id")
	}
	id := (*migrate.Hash)(idBytes)
	hashBytes, err := info_hash.ToByteString()
	if len(hashBytes) != 20 || err != nil {
		slog.Info("handleGetPeers 不回复,hash的长度不对", "err", err)
		s.sendError(pkt.addr, txid, 202, "invalid info_hash")
	}
	hash := (*migrate.Hash)(hashBytes)

	infoHashs, err := s.hashRepo.ListByInfoHash(hash, 8)
	if err != nil {
		s.sendError(pkt.addr, txid, 202, "invalid info_hash")
		return
	}
	tokenNode, err := a.GetDictValue("token")
	if err != nil {
		s.sendError(pkt.addr, txid, 201, "missing token")
		return
	}
	token, err := tokenNode.ToByteString()
	if err != nil {
		s.sendError(pkt.addr, txid, 201, "invalid token")
		return
	}
	// 5. 构造响应字典
	rDict := map[string]*bencode.BNode{
		"id":    {Type: bencode.BString, Str: s.selfID[:]},
		"token": {Type: bencode.BString, Str: token},
	}
	if len(infoHashs) > 0 {
		ids := make([]*migrate.Hash, 0, len(infoHashs))
		for i := range infoHashs {
			hash1 := infoHashs[i]
			ids = append(ids, hash1.PeerId)
		}
		nodes, err := s.nodeRepo.GetNodes(ids)
		if err != nil {
			s.sendError(pkt.addr, txid, 202, "invalid info_hash")
			return
		}
		if len(nodes) == 0 {
			s.sendError(pkt.addr, txid, 202, "invalid info_hash")
		}
		var nodeIdMap map[migrate.Hash]repository.DHTNode
		for _, node := range nodes {
			nodeIdMap[node.NodeID] = node
		}

		//存在内容, 需要返回
		// 有peer：构造values列表
		valuesList := make([]*bencode.BNode, 0, len(infoHashs))
		valuesList6 := make([]*bencode.BNode, 0, len(infoHashs))
		var peerBuf [6]byte
		var peerBuf6 [18]byte
		for _, p := range infoHashs {
			node := nodeIdMap[*p.PeerId]
			ip4 := node.IP.To4()
			if ip4 == nil {
				ip6 := node.IP.To16()
				copy(peerBuf6[:16], ip6)
				binary.BigEndian.PutUint16(peerBuf6[16:18], uint16(*p.Port))
				valuesList6 = append(valuesList, &bencode.BNode{
					Type: bencode.BString,
					Str:  peerBuf6[:],
				})
			} else {
				copy(peerBuf[:4], ip4)
				binary.BigEndian.PutUint16(peerBuf[4:6], uint16(*p.Port))
				valuesList = append(valuesList, &bencode.BNode{
					Type: bencode.BString,
					Str:  peerBuf[:],
				})
			}
		}
		rDict["values"] = &bencode.BNode{Type: bencode.BList, List: valuesList}
		rDict["values6"] = &bencode.BNode{Type: bencode.BList, List: valuesList6}
	} else {
		kBucket := xorBucket(s.selfID, *id)
		nodes, err := s.nodeRepo.ListRecentNodes(kBucket, 8)
		if err != nil {
			s.sendError(pkt.addr, txid, 202, "invalid info_hash")
			return
		}
		nodesBuf := make([]byte, 0, 26)
		var buf [26]byte
		nodesBuf6 := make([]byte, 0, 38)
		var buf6 [38]byte
		for _, node := range nodes {
			ip4 := node.IP.To4()
			if ip4 == nil {
				copy(buf[:20], node.NodeID[:])
				copy(buf[20:24], ip4)
				binary.BigEndian.PutUint16(buf[24:26], uint16(node.Port))
				nodesBuf = append(nodesBuf, buf[:]...)
			} else {
				ip6 := node.IP.To16()
				copy(buf6[:20], node.NodeID[:])
				copy(buf6[20:36], ip6)
				binary.BigEndian.PutUint16(buf6[36:38], uint16(node.Port))
				nodesBuf6 = append(nodesBuf6, buf6[:]...)
			}
		}
		if len(nodesBuf) > 0 {
			rDict["nodes"] = &bencode.BNode{Type: bencode.BString, Str: nodesBuf}
		}
		if len(nodesBuf6) > 0 {
			rDict["nodes6"] = &bencode.BNode{Type: bencode.BString, Str: nodesBuf6}
		}
	}

	// 5. 组装完整KRPC响应
	resp := &bencode.BNode{
		Type: bencode.BDict,
		Dict: map[string]*bencode.BNode{
			"t": {Type: bencode.BString, Str: txid},
			"y": {Type: bencode.BString, Str: []byte("r")},
			"r": {Type: bencode.BDict, Dict: rDict},
		},
	}

	sendPktToChannel(resp, pkt.addr, s)
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

	idPtr := (*[20]byte)(targetId)
	kBucket := xorBucket(s.selfID, *idPtr)

	nodes, err := s.nodeRepo.ListRecentNodes(kBucket, 8)
	if err != nil {
		slog.Info("sendFindNodeResponse 无法获取target的数据库中同K桶的数据, 不回复响应了", err)
		return
	}

	var nodes4 = make([]byte, 0, 8*26)
	var nodes6 = make([]byte, 0, 8*38)
	for i := range nodes {
		node := nodes[i]
		nodeId := node.NodeID
		if node.Port <= 0 || node.Port > 65535 {
			//端口号不正确
			continue
		}
		var portbuf [2]byte
		portbuf[0] = byte(node.Port >> 8)   // 高 8 位
		portbuf[1] = byte(node.Port & 0xFF) // 低 8 位

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

// xorBucket 计算两个节点ID异或距离对应的桶编号（0~159）
func xorBucket(selfID, nodeID migrate.Hash) int {
	for i := 0; i < 20; i++ {
		b := selfID[i] ^ nodeID[i]
		if b != 0 {
			// 字节位置×8 + 字节内前导零个数 = 最高有效位位置
			return (19-i)*8 + bits.LeadingZeros8(b)
		}
	}
	return 0 // 距离为0（节点自身）
}

func sendPktToChannel(sendData *bencode.BNode, addr *net.UDPAddr, s *DHTService) {
	sendPkt := &Packet{
		addr,
		sendData,
		nil,
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

func (s *DHTService) upsertQueryDhtNode(pkt *Packet) {
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
	var nodeId = *(*migrate.Hash)(bytes)

	ip := pkt.addr.IP.To16()
	port := pkt.addr.Port

	kBucket := xorBucket(s.selfID, nodeId)
	dhtNode := repository.DHTNode{
		NodeID:   nodeId,
		KBucket:  kBucket,
		IP:       ip,
		Port:     port,
		Token:    nil,
		LastSeen: time.Now(),
	}
	err = s.nodeRepo.UpsertNode(&dhtNode)
	if err != nil {
		slog.Error("upsertDhtNode 出现异常, 保存数据库数据报错", err)
		return
	}
}
