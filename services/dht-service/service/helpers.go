package service

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
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

// hexID 将节点 ID 转为十六进制字符串（用于日志）。
func hexID(id []byte) string {
	return hex.EncodeToString(id)
}
