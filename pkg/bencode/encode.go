package bencode

import (
	"sort"
	"strconv"
)

// Encode 将 BNode 树编码为 bencode 字节流。
// 若节点带有 Raw 原始字节（解析生成的节点），会直接复用 Raw 提升性能；
// 手动构造的节点无 Raw 字段时自动生成编码内容。
func (n *BNode) Encode() []byte {
	buf := make([]byte, 0, 64)
	return appendNode(buf, n)
}

// -------------------- 四种独立类型编码方法 --------------------

// EncodeInt 编码一个整数值为 bencode 格式。
func EncodeInt(v int64) []byte {
	buf := make([]byte, 0, 16)
	buf = append(buf, 'i')
	buf = strconv.AppendInt(buf, v, 10)
	buf = append(buf, 'e')
	return buf
}

// EncodeString 编码一个字符串为 bencode 格式。
func EncodeString(s string) []byte {
	buf := make([]byte, 0, len(s)+8)
	buf = strconv.AppendInt(buf, int64(len(s)), 10)
	buf = append(buf, ':')
	buf = append(buf, s...)
	return buf
}

// EncodeBytes 【额外新增】编码二进制字节数组为 bencode 字符串格式。
// 适用于非文本二进制内容，避免 string 转换开销。
func EncodeBytes(b []byte) []byte {
	buf := make([]byte, 0, len(b)+8)
	buf = strconv.AppendInt(buf, int64(len(b)), 10)
	buf = append(buf, ':')
	buf = append(buf, b...)
	return buf
}

// EncodeList 编码 BNode 列表为 bencode 列表格式。
func EncodeList(list []*BNode) []byte {
	buf := make([]byte, 0, 64)
	buf = append(buf, 'l')
	for _, item := range list {
		buf = appendNode(buf, item)
	}
	buf = append(buf, 'e')
	return buf
}

// EncodeDict 编码 map 为 bencode 字典格式。
// 自动按字节字典序对 key 排序，符合 bencode 标准规范。
func EncodeDict(dict map[string]*BNode) []byte {
	buf := make([]byte, 0, 128)
	buf = append(buf, 'd')

	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// 编码 key
		buf = strconv.AppendInt(buf, int64(len(k)), 10)
		buf = append(buf, ':')
		buf = append(buf, k...)
		// 编码 value
		buf = appendNode(buf, dict[k])
	}
	buf = append(buf, 'e')
	return buf
}

// -------------------- 内部递归辅助函数 --------------------

// appendNode 递归向 buf 追加节点编码内容。
func appendNode(buf []byte, n *BNode) []byte {
	if n == nil {
		return buf
	}

	switch n.Type {
	case BInt:
		// 优先复用原始字节，无 Raw 则重新编码
		if len(n.Raw) > 0 {
			buf = append(buf, n.Raw...)
		} else {
			buf = append(buf, 'i')
			buf = strconv.AppendInt(buf, int64(n.Int), 10)
			buf = append(buf, 'e')
		}

	case BString:
		if len(n.Raw) > 0 {
			buf = append(buf, n.Raw...)
		} else {
			buf = strconv.AppendInt(buf, int64(len(n.Str)), 10)
			buf = append(buf, ':')
			buf = append(buf, n.Str...)
		}

	case BList:
		buf = append(buf, 'l')
		for _, item := range n.List {
			buf = appendNode(buf, item)
		}
		buf = append(buf, 'e')

	case BDict:
		buf = append(buf, 'd')
		// 按字节字典序排序 key，保证输出确定性
		keys := make([]string, 0, len(n.Dict))
		for k := range n.Dict {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			buf = strconv.AppendInt(buf, int64(len(k)), 10)
			buf = append(buf, ':')
			buf = append(buf, k...)
			buf = appendNode(buf, n.Dict[k])
		}
		buf = append(buf, 'e')
	}
	return buf
}
