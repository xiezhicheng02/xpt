package bencode

import (
	"sort"
	"strconv"
)

// Encode 将 BNode 树编码为 bencode 字节流。
func Encode(n *BNode) []byte {
	buf := make([]byte, 0, 64)
	return appendNode(buf, n)
}

// EncodeString 便捷方法：编码一个字符串值。
func EncodeString(s string) []byte {
	return []byte(strconv.Itoa(len(s)) + ":" + s)
}

// EncodeInt 便捷方法：编码一个整数值。
func EncodeInt(v int64) []byte {
	return []byte("i" + strconv.FormatInt(v, 10) + "e")
}

// appendNode 递归编码节点。
func appendNode(buf []byte, n *BNode) []byte {
	if n == nil {
		return buf
	}
	switch n.Type {
	case BInt:
		buf = append(buf, 'i')
		buf = strconv.AppendInt(buf, n.Int, 10)
		buf = append(buf, 'e')
	case BString:
		buf = strconv.AppendInt(buf, int64(len(n.Str)), 10)
		buf = append(buf, ':')
		buf = append(buf, n.Str...)
	case BList:
		buf = append(buf, 'l')
		for _, item := range n.List {
			buf = appendNode(buf, item)
		}
		buf = append(buf, 'e')
	case BDict:
		buf = append(buf, 'd')
		// bencode 规范要求字典键按字典序排序，保证编码结果确定。
		keys := make([]string, 0, len(n.Dict))
		for k := range n.Dict {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			buf = append(buf, strconv.Itoa(len(k))...)
			buf = append(buf, ':')
			buf = append(buf, k...)
			buf = appendNode(buf, n.Dict[k])
		}
		buf = append(buf, 'e')
	}
	return buf
}
