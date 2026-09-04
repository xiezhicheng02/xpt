package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"math/bits"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/xiezc/xpt/services/dht-service/migrate"
)

// portOf 从 "host:port" 形式的地址中解析端口号。
func portOf(addr string) int {
	_, portStr, err := netSplitHostPort(addr)
	if err != nil {
		return 6881 // 默认 DHT 端口
	}
	p, _ := strconv.Atoi(portStr)
	return p
}

// netSplitHostPort 是 net.SplitHostPort 的薄封装，便于替换默认值。
func netSplitHostPort(addr string) (string, string, error) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, "", errBadAddr
	}
	return addr[:idx], addr[idx+1:], nil
}

// errBadAddr 表示地址格式非法。
var errBadAddr = &addrError{}

type addrError struct{}

func (e *addrError) Error() string { return "invalid address" }

// randomNodeID 生成 DHT 节点 ID（20 字节随机）。
func randomNodeID() []byte {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return b
}

func randomTxID(len int) []byte {
	b := make([]byte, len)
	_, _ = rand.Read(b)
	return b
}

// hexID 将节点 ID 转为十六进制字符串（用于日志）。
func hexID(id migrate.Hash) string {
	return hex.EncodeToString(id[:])
}

// Match 匹配响应，匹配成功则返回并删除缓存
func (pt *PendingTable) Put(txID []byte, pr *PendingRequest) bool {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	key := hex.EncodeToString(txID)
	slog.Info("放入路由表的txid", "txid", key)
	pt.items[key] = pr
	return true
}
func (pt *PendingTable) Remove(txID []byte) bool {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := string(txID)
	delete(pt.items, key)
	return true
}

// Match 匹配响应，匹配成功则返回并删除缓存
func (pt *PendingTable) Match(txID []byte, from *net.UDPAddr) (*PendingRequest, bool) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := hex.EncodeToString(txID)
	slog.Info("接收到路由表, txid", "txid", key)
	req, ok := pt.items[key]
	if !ok {
		return nil, false
	}

	// 双重校验：源地址必须匹配，防止串包
	if req.RemoteAddr.IP.Equal(from.IP) && req.RemoteAddr.Port == from.Port {
		delete(pt.items, key)
		return req, true
	}

	return nil, false
}

// CleanExpired 清理过期请求，建议每秒调用一次
func (pt *PendingTable) CleanExpired() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	for key, req := range pt.items {
		if now.Sub(req.SentAt) > 15*time.Second {
			delete(pt.items, key)
		}
	}
}

// xorBucket 计算两个节点ID异或距离对应的桶编号（0~159）
// 返回160表示是节点自身
// 约定：第0字节是最高位字节，桶号=前缀零的个数，0最远，159最近
func xorBucket(selfID, nodeID [20]byte) int {
	for i := 0; i < 20; i++ {
		b := selfID[i] ^ nodeID[i]
		if b != 0 {
			// i个全零字节 ×8 + 当前字节前导零数 = 总前缀零个数
			return i*8 + bits.LeadingZeros8(b)
		}
	}
	return 160 // 距离为0，节点自身
}

// GenerateNodeIDInBucket 生成与自身节点落在同一K桶的随机节点ID
// bucketIdx: 桶索引（前缀零个数），范围0~159，0最远，159最近
func GenerateNodeIDInBucket(selfID [20]byte, bucketIdx int) ([20]byte, error) {
	if bucketIdx < 0 || bucketIdx >= 160 {
		return [20]byte{}, errors.New("桶索引超出范围 [0, 159]")
	}

	var mask [20]byte

	// 计算目标位的字节位置和位位置
	byteIdx := bucketIdx / 8
	bitIdx := 7 - (bucketIdx % 8)

	// 将目标位设为1：决定距离层级
	mask[byteIdx] |= 1 << bitIdx

	// 填充当前字节剩余低位（一次随机生成，掩码保留低位）
	if bitIdx > 0 {
		var randBuf [1]byte
		_, _ = rand.Read(randBuf[:])
		// 保留低bitIdx位
		mask[byteIdx] |= randBuf[0] & ((1 << bitIdx) - 1)
	}

	// 填充后续所有字节（完全随机）
	if byteIdx < 19 {
		remaining := make([]byte, 19-byteIdx)
		_, _ = rand.Read(remaining)
		copy(mask[byteIdx+1:], remaining)
	}

	// 自身ID XOR 掩码 = 目标节点ID
	var targetID [20]byte
	for i := 0; i < 20; i++ {
		targetID[i] = selfID[i] ^ mask[i]
	}

	return targetID, nil
}
