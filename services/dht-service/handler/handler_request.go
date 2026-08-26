// 处理DHT的消息接收 和 响应的逻辑
package handler

import (
	"log/slog"
	"net"
	"time"

	"github.com/xiezc/xpt/pkg/bencode"
	"github.com/xiezc/xpt/services/dht-service/repository"
	"github.com/xiezc/xpt/services/dht-service/service"
)

func (s *service.DHTService) handleResponseMessage(addr *net.UDPAddr, data []byte) {
	// 骨架：暂不实现，等你学完 bencode 后填充。

}

func (s *service.DHTService) handleQueryMessage(addr *net.UDPAddr, node *bencode.BNode) {

	q := node.GetString("q")

	switch q {
	case "ping":
		//更新数据库信息
		upsertDhtNode(addr, node, s)
		//发送响应信息
		sendPingResponse(addr, node, s)
	case "find_node":
		//组装find_node响应，填充nodes   {"t":"原txid","y":"r","r":{"id":"我方id", "nodes":"26字节*N压缩节点列表"}}

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

func sendPingResponse(addr *net.UDPAddr, node *bencode.BNode, s *service.DHTService) {
	t := node.GetString("t")
	//{"t":"原txid","y":"r","r":{"id":"我们自己的selfID(20字节)"}}
	response := &bencode.BNode{
		Type: bencode.BDict,
		Dict: map[string]*bencode.BNode{
			"t": {Type: bencode.BString, Str: t},
			"y": {Type: bencode.BString, Str: "r"},
			"r": {Type: bencode.BDict, Dict: map[string]*bencode.BNode{
				"id": {Type: bencode.BString, Str: string(s.selfID)},
			},
			},
		},
	}
	bytes := bencode.Encode(response)
	_, err := s.udpConn.WriteToUDP(bytes, addr)
	if err != nil {
		slog.Error("ping消息恢复出错,丢弃消息", err)
		return
	}
}

func upsertDhtNode(addr *net.UDPAddr, node *bencode.BNode, s *service.DHTService) {
	a := node.Get("a")
	reqId := a.GetString("id")
	var ip4 string
	var port4 int
	if len(addr.IP) == net.IPv4len {
		ip4 = addr.IP.To4().String()
		port4 = addr.Port
	}
	var ip6 string
	var port6 int
	if len(addr.IP) == net.IPv6len {
		ip6 = addr.IP.To16().String()
		port6 = addr.Port
	}
	now := time.Now()
	dhtNode := repository.DHTNode{
		NodeID:   &reqId,
		IP4:      &ip4,
		Port4:    &port4,
		IP6:      &ip6,
		Port6:    &port6,
		LastSeen: &now,
	}
	err := s.nodeRepo.UpsertNode(&dhtNode)
	if err != nil {
		slog.Error("upsertDhtNode 出现异常, 保存数据库数据报错", err)
		return
	}
}
