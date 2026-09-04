package service

//定时清除路由表
import (
	"log/slog"
	"net"
	"time"

	"github.com/xiezc/xpt/pkg/bencode"
	"github.com/xiezc/xpt/services/dht-service/repository"
)

// 定时清除过期的仓库中 node 和 info信息, 并且node过少的时候,主动寻找更多节点,  平衡K桶的数量
// CleanExpiredNodes 清理所有桶中过期的节点
// expire: 过期阈值，比如 30*time.Minute
func (s *DHTService) startCleanExpiredNodes() {
	cronTime := s.cfg.GetInt("k_bucket.cron_time")
	ticker := time.NewTicker(time.Duration(cronTime) * time.Second)
	defer ticker.Stop() // 退出时必须停止，防止资源泄漏

	// range 遍历 ticker.C，每次定时触发都会收到信号
	for range ticker.C {
		err := s.cleanExpiredNode()
		if err != nil {
			slog.Error(err.Error())
			continue
		}
	}
}

// 对逐个桶进行巡检的方法
// 1. 逐个获取每个桶的数据
// 2. 超出数量的数据, 按照最后活跃时间进行删除
// 3. 然后对生于的超出巡检周期的数据, 进行存活检测
// 4. 对于桶中缺少的数据数据进行补充
func (s *DHTService) cleanExpiredNode() error {
	expireTime := s.cfg.GetInt("k_bucket.expire_time")
	for k := range 160 {
		nodes, err := s.nodeRepo.ListRecentNodes(k, 100000)
		slog.Info("Cleaned expired nodes:", "k", k, "size", len(nodes))
		if err != nil {
			slog.Error("当前桶数据查询保存, k", k, err.Error())
			continue
		}
		//超出数量的节点直接删除
		kNodeLen := len(nodes)
		if kNodeLen > 8 {
			deadNodes := nodes[8:]
			deleteNodes(s, deadNodes)
		} else if kNodeLen > 4 {
			//分离只是超出巡检节点. 进行存活检测
			keepAliveNodes := nodes[:8]
			deadTime := time.Now().Add(time.Second * time.Duration(-expireTime))
			for i := range keepAliveNodes {
				node := keepAliveNodes[i]
				if node.LastSeen.Before(deadTime) {
					sendFindNodeV2(s, k, node)
				}
			}
		} else if kNodeLen > 0 { //对于桶中的数量过少的节点发送查找信息
			for _, node := range nodes {
				sendFindNodeV2(s, k, node)
			}
		} else { //对于桶中的数量过少的节点发送查找信息
			for _, addr := range BootstrapAddrs {
				udpAddr, err := net.ResolveUDPAddr("udp6", addr)
				if err != nil {
					slog.Error(err.Error())
					continue
				}
				slog.Info("Cleaned expired node:", "k", k, "IP", udpAddr.IP, "PORT", udpAddr.Port)
				sendFindNode(s, k, udpAddr)
			}
		}
	}
	return nil
}

var BootstrapAddrs = []string{
	"[240e:b8f:812:1d00:790d:543b:2146:cb1a]:41991",
	"[2408:822e:6e3:e80:58a3:ea04:cd8:7268]:30414",
	"[240e:b8f:812:1d00:bdf4:6f4d:c018:745]:41991",
}

// 维护稳定节点的方法. 会动态根据节点的响应决定使用那个节点
func findBootstrapAddr() {

}

func sendFindNode(s *DHTService, k int, addr *net.UDPAddr) bool {
	nodeId, err := GenerateNodeIDInBucket(s.selfID, k)
	if err != nil {
		slog.Error(err.Error())
		return false
	}

	response := &bencode.BNode{
		Type: bencode.BDict,
		Dict: map[string]*bencode.BNode{
			"t": {Type: bencode.BString, Str: randomTxID(2)},
			"y": {Type: bencode.BString, Str: []byte("q")},
			"q": {Type: bencode.BString, Str: []byte("find_node")},
			"a": {
				Type: bencode.BDict,
				Dict: map[string]*bencode.BNode{
					"id": {
						Type: bencode.BString,
						Str:  s.selfID[:],
					},
					"target": {
						Type: bencode.BString,
						Str:  nodeId[:],
					},
				},
			},
		},
	}

	err = s.DoSend(response, "find_node", addr)
	if err != nil {
		slog.Error("DoSend err", "err", err)
	}
	return true

}

func sendFindNodeV2(s *DHTService, k int, node repository.DHTNode) bool {
	addr := net.UDPAddr{
		IP:   node.IP,
		Port: node.Port,
	}
	return sendFindNode(s, k, &addr)
}

func sendPing(s *DHTService, node *repository.DHTNode) {
	//超过巡检周期的的数据需要发送成活检测的请求
	data := &bencode.BNode{
		Type: bencode.BDict,
		Dict: map[string]*bencode.BNode{
			"t": {
				Type: bencode.BString,
				Str:  randomTxID(2),
			},
			"y": {
				Type: bencode.BString,
				Str:  []byte("q"),
			},
			"q": {
				Type: bencode.BString,
				Str:  []byte("ping"),
			},
			"a": {
				Type: bencode.BDict,
				Dict: map[string]*bencode.BNode{
					"id": {
						Type: bencode.BString,
						Str:  node.NodeID[:],
					},
				},
			},
		},
	}
	addr := net.UDPAddr{
		IP:   node.IP,
		Port: node.Port,
	}
	err := s.DoSend(data, "ping", &addr)
	if err != nil {
		slog.Error("Failed to send ping", err)
	}
}

func deleteNodes(s *DHTService, nodes []repository.DHTNode) {
	//死亡节点删除
	for i := range nodes {
		//先删除infohash 再删除节点
		err := s.nodeRepo.DeleteById(nodes[i].NodeID)
		if err != nil {
			slog.Error("Failed to delete dead node", err)
			continue
		}

		err = s.hashRepo.DeleteByNodeId(nodes[i].NodeID)
		if err != nil {
			slog.Error("Failed to delete dead node", err)
			continue
		}
	}
}
