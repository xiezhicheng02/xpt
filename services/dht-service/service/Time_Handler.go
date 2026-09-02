package service

//定时清除路由表
import (
	"log/slog"
	"net"
	"time"

	"github.com/xiezc/xpt/pkg/bencode"
	"github.com/xiezc/xpt/services/dht-service/repository"
)

func (s *DHTService) startCleanPendingTable() {
	clean_job_second := s.cfg.GetInt("pending_table.clean_job_second")
	// 创建一个每秒触发一次的定时器
	ticker := time.NewTicker(time.Duration(clean_job_second) * time.Second)
	defer ticker.Stop() // 退出时必须停止，防止资源泄漏

	// range 遍历 ticker.C，每次定时触发都会收到信号
	for range ticker.C {
		// 执行定时任务：清理过期待处理请求
		s.PendingTable.CleanExpired()
	}
}

// 定时清除过期的仓库中 node 和 info信息, 并且node过少的时候,主动寻找更多节点,  平衡K桶的数量
// CleanExpiredNodes 清理所有桶中过期的节点
// expire: 过期阈值，比如 30*time.Minute
func (s *DHTService) startCleanExpiredNodes() {

	clean_job_time := s.cfg.GetInt("kbucket.clean_job_time")
	ticker := time.NewTicker(time.Duration(clean_job_time) * time.Second)
	defer ticker.Stop() // 退出时必须停止，防止资源泄漏

	// range 遍历 ticker.C，每次定时触发都会收到信号
	for range ticker.C {

	}

}

func cleanExpiredNode(s *DHTService, cleanTime time.Time) error {
	expire_time := s.cfg.GetInt("kbucket.clean_job_time")
	nodes, err := s.nodeRepo.GetExpireNodes(cleanTime)
	if err != nil {
		return err
	}

	deadTime := time.Now().Add(time.Second * time.Duration(-expire_time))
	var deadNode []repository.DHTNode
	var expireNode []repository.DHTNode
	for i := range nodes {
		node := nodes[i]
		if node.LastSeen.After(deadTime) {
			expireNode = append(expireNode, node)
		} else {
			deadNode = append(deadNode, node)
		}
	}
	//过期的数据需要发送成活检测的请求
	for i := range expireNode {
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
							Str:  expireNode[i].NodeID[:],
						},
					},
				},
			},
		}
		addr := net.UDPAddr{
			expireNode[i].IP,
			expireNode[i].Port,
			nil,
		}
		err := s.DoSend(data, "ping", &addr)
		if err != nil {
			slog.Error("Failed to send ping", err)
			continue
		}

	}

	//死亡节点删除
	for i := range deadNode {
		s.hashRepo.de()

	}
}
